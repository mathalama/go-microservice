package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type NotifyRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type NotifyResponse struct {
	Status string `json:"status"`
}

var (
	processedKeys = make(map[string]bool)
	mu            sync.Mutex
)

func main() {
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/notify", handleNotify)

	fmt.Printf("Mock Notification Gateway starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Gateway failed to start: %v", err)
	}
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simulate 20% transient failures
	if rand.Float32() < 0.2 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	var req NotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log request to stdout in JSON
	logEntry, _ := json.Marshal(map[string]interface{}{
		"time":    time.Now().UTC().Format(time.RFC3339),
		"request": req,
	})
	fmt.Println(string(logEntry))

	mu.Lock()
	defer mu.Unlock()

	status := "accepted"
	if processedKeys[req.IdempotencyKey] {
		status = "duplicate"
	} else {
		processedKeys[req.IdempotencyKey] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NotifyResponse{Status: status})
}
