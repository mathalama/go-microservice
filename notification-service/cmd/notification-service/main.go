package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/jobqueue"
	"notification-service/internal/logger"
	"notification-service/internal/subscriber"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	var nc *nats.Conn
	var err error

	// Exponential backoff for NATS connection
	backoff := 1 * time.Second
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to NATS (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	if err != nil {
		log.Fatalf("Could not connect to NATS after %d attempts: %v", maxRetries, err)
	}
	defer nc.Close()

	log.Printf("Connected to NATS at %s", natsURL)

	// Redis connection for Job Queue
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	queue, err := jobqueue.NewJobQueue(redisURL)
	if err != nil {
		log.Fatalf("Failed to initialize job queue: %v", err)
	}
	queue.Start(context.Background())
	defer queue.Stop()

	l := logger.NewLogger()
	subjects := []string{"doctors.created", "appointments.created", "appointments.status_updated"}

	sub := subscriber.NewSubscriber(nc, l, queue)
	if err := sub.SubscribeToAll(subjects); err != nil {
		log.Fatalf("Subscription failed: %v", err)
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Notification Service...")
	nc.Drain()
}
