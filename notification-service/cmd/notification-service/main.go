package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"notification-service/internal/subscriber"
)

func main() {
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

	subjects := []string{"doctors.created", "appointments.created", "appointments.status_updated"}
	
	sub := subscriber.NewSubscriber(nc)
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
