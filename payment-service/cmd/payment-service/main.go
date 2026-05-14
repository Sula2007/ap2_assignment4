package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	"payment-service/internal/transport/handler"
	"payment-service/internal/usecase"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := getEnv("DATABASE_URL",
		"host=localhost port=5433 user=postgres password=postgres dbname=payment_db sslmode=disable")
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	for i := 0; i < 10; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("DB not ready, retrying (%d/10)...", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatal("DB ping failed:", err)
	}

	publisher, err := messaging.NewRabbitMQPublisher(rabbitmqURL)
	if err != nil {
		log.Fatal("Failed to create publisher:", err)
	}
	defer publisher.Close()

	paymentRepo := repository.NewPaymentRepository(db)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, publisher)
	paymentHandler := handler.NewPaymentHandler(paymentUsecase)

	router := gin.Default()
	api := router.Group("/api/v1")
	{
		api.POST("/payments", paymentHandler.CreatePayment)
		api.GET("/payments/:order_id", paymentHandler.GetPayment)
	}

	srv := &http.Server{Addr: ":8081", Handler: router}

	// ── Graceful Shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Payment service started on :8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe error:", err)
		}
	}()

	<-quit
	log.Println("[Payment] Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("[Payment] Server stopped.")
}
