package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"order-service/internal/repository"
	"order-service/internal/transport/handler"
	"order-service/internal/transport/middleware"
	"order-service/internal/usecase"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func main() {
	dsn := getEnv("DATABASE_URL",
		"host=localhost port=5432 user=postgres password=postgres dbname=order_db sslmode=disable")
	paymentURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")
	redisAddr := getEnv("REDIS_URL", "localhost:6379")
	cacheTTLSec := getEnvInt("CACHE_TTL_SECONDS", 300)
	rateLimitReqs := getEnvInt("RATE_LIMIT_REQUESTS", 10)
	rateLimitWindow := getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)

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

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Redis ping failed:", err)
	}
	defer rdb.Close()

	cacheTTL := time.Duration(cacheTTLSec) * time.Second
	orderCache := repository.NewRedisOrderCache(rdb)
	orderRepo := repository.NewOrderRepository(db)
	paymentClient := repository.NewPaymentHTTPClient(paymentURL)
	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient, orderCache, cacheTTL)
	orderHandler := handler.NewOrderHandler(orderUsecase)

	router := gin.Default()

	windowDur := time.Duration(rateLimitWindow) * time.Second
	router.Use(middleware.RateLimiter(rdb, rateLimitReqs, windowDur))

	api := router.Group("/api/v1")
	{
		api.POST("/orders", orderHandler.CreateOrder)
		api.GET("/orders/:id", orderHandler.GetOrder)
		api.PATCH("/orders/:id/cancel", orderHandler.CancelOrder)
	}

	srv := &http.Server{Addr: ":8082", Handler: router}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Order service started on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe error:", err)
		}
	}()

	<-quit
	log.Println("[Order] Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("[Order] Server stopped.")
}
