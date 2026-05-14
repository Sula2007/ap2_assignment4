package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"notification-service/internal/messaging"
	"notification-service/internal/provider"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"
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
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	redisAddr := getEnv("REDIS_URL", "localhost:6379")
	providerMode := getEnv("PROVIDER_MODE", "SIMULATED")
	maxRetries := getEnvInt("MAX_RETRIES", 3)

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	for i := 0; i < 10; i++ {
		if err := rdb.Ping(context.Background()).Err(); err == nil {
			break
		}
		log.Printf("Redis not ready, retrying (%d/10)...", i+1)
		time.Sleep(3 * time.Second)
	}
	defer rdb.Close()

	idempotencyStore := repository.NewRedisIdempotencyStore(rdb)

	var emailSender provider.EmailSender
	switch providerMode {
	case "REAL":
		apiKey := getEnv("MAILJET_API_KEY", "")
		secretKey := getEnv("MAILJET_SECRET_KEY", "")
		senderEmail := getEnv("MAILJET_SENDER_EMAIL", "no-reply@example.com")
		emailSender = provider.NewMailjetEmailSender(apiKey, secretKey, senderEmail)
		log.Println("[Notification] Using REAL Mailjet provider.")
	default:
		emailSender = provider.NewMockEmailSender()
		log.Println("[Notification] Using SIMULATED mock provider.")
	}

	notifUsecase := usecase.NewNotificationUsecase(idempotencyStore, emailSender, maxRetries)

	consumer, err := messaging.NewConsumer(rabbitmqURL, notifUsecase)
	if err != nil {
		log.Fatal("Failed to create consumer:", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[Notification] Shutting down gracefully...")
		cancel()
	}()

	log.Println("Notification service started.")
	if err := consumer.Start(ctx); err != nil {
		log.Fatal("Consumer error:", err)
	}
	log.Println("[Notification] Service stopped.")
}
