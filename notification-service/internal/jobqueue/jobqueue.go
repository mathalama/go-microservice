package jobqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Job struct {
	IdempotencyKey string `json:"idempotency_key"`
	AppointmentID  string `json:"appointment_id"`
	DoctorID       string `json:"doctor_id"`
	OccurredAt     string `json:"occurred_at"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
	Attempt        int    `json:"attempt"`
}

type JobLog struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	JobID   string `json:"job_id"`
	Attempt int    `json:"attempt"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type JobQueue struct {
	redisClient *redis.Client
	jobChan     chan Job
	gatewayURL  string
	poolSize    int
	wg          sync.WaitGroup
}

func NewJobQueue(redisURL string) (*JobQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}
	client := redis.NewClient(opts)

	poolSizeStr := os.Getenv("WORKER_POOL_SIZE")
	poolSize, err := strconv.Atoi(poolSizeStr)
	if err != nil {
		poolSize = 3
	}

	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	return &JobQueue{
		redisClient: client,
		jobChan:     make(chan Job, 100), // Buffered channel
		gatewayURL:  gatewayURL,
		poolSize:    poolSize,
	}, nil
}

func (q *JobQueue) Start(ctx context.Context) {
	for i := 0; i < q.poolSize; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i+1)
	}
}

func (q *JobQueue) Stop() {
	close(q.jobChan)
	q.wg.Wait()
}

func (q *JobQueue) Enqueue(eventType, id, occurredAt, appointmentID, doctorID string) {
	// 1. Generate Idempotency Key
	keySource := eventType + id + occurredAt
	hash := sha256.Sum256([]byte(keySource))
	idempotencyKey := hex.EncodeToString(hash[:])

	job := Job{
		IdempotencyKey: idempotencyKey,
		AppointmentID:  appointmentID,
		DoctorID:       doctorID,
		OccurredAt:     occurredAt,
		Channel:        "email",
		Recipient:      "patient@clinic.kz",
		Message:        fmt.Sprintf("Your appointment %s with doctor %s is complete.", appointmentID, doctorID),
		Attempt:        1,
	}

	// 2. Check Idempotency in Redis
	val, err := q.redisClient.Get(context.Background(), idempotencyKey).Result()
	if err == nil && val == "done" {
		q.log(JobLog{Level: "info", JobID: idempotencyKey, Status: "dropped", Error: "already processed"})
		return
	}

	q.jobChan <- job
	q.log(JobLog{Level: "info", JobID: idempotencyKey, Status: "enqueued", Attempt: 1})
}

func (q *JobQueue) worker(ctx context.Context, id int) {
	defer q.wg.Done()
	for job := range q.jobChan {
		q.processJob(ctx, job)
	}
}

func (q *JobQueue) processJob(ctx context.Context, job Job) {
	q.log(JobLog{Level: "info", JobID: job.IdempotencyKey, Status: "processing", Attempt: job.Attempt})

	success, err := q.callGateway(job)
	if success {
		// Mark as done in Redis with 24h TTL
		q.redisClient.Set(ctx, job.IdempotencyKey, "done", 24*time.Hour)
		q.log(JobLog{Level: "info", JobID: job.IdempotencyKey, Status: "success", Attempt: job.Attempt})
		return
	}

	// Retry logic
	if job.Attempt < 3 {
		backoff := time.Duration(1<<uint(job.Attempt-1)) * time.Second
		q.log(JobLog{Level: "warn", JobID: job.IdempotencyKey, Status: "retry", Attempt: job.Attempt, Error: err.Error()})
		
		time.Sleep(backoff)
		job.Attempt++
		
		select {
		case q.jobChan <- job:
		default:
			q.logErr(JobLog{
				Level:   "error",
				JobID:   job.IdempotencyKey,
				Status:  "dead_letter",
				Attempt: job.Attempt,
				Error:   "job channel full, dropping job",
			})
		}
		return
	}

	// Dead letter
	q.logErr(JobLog{Level: "error", JobID: job.IdempotencyKey, Status: "dead_letter", Attempt: job.Attempt, Error: fmt.Sprintf("max retries reached: %v", err)})
}

func (q *JobQueue) callGateway(job Job) (bool, error) {
	payload, _ := json.Marshal(map[string]string{
		"idempotency_key": job.IdempotencyKey,
		"channel":         job.Channel,
		"recipient":       job.Recipient,
		"message":         job.Message,
	})

	req, err := http.NewRequest("POST", q.gatewayURL+"/notify", bytes.NewBuffer(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, fmt.Errorf("gateway returned status %d", resp.StatusCode)
}

func (q *JobQueue) log(entry JobLog) {
	entry.Time = time.Now().UTC().Format(time.RFC3339)
	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}

func (q *JobQueue) logErr(entry JobLog) {
	entry.Time = time.Now().UTC().Format(time.RFC3339)
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
