package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"doctor-service/internal/cache"
	"doctor-service/internal/event"
	"doctor-service/internal/middleware"
	"doctor-service/internal/repository"
	grpctransport "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run() error {
	// Load .env file from service root
	_ = godotenv.Load()

	port := os.Getenv("DOCTOR_SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	dbURL := os.Getenv("DOCTOR_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DOCTOR_DATABASE_URL environment variable is required")
	}

	// 1. Connect to Database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	defer db.Close()

	// 2. Run Migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create migration driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("failed to initialize migrations: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")

	// 3. Connect to NATS
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err := nats.Connect(natsURL)
	var publisher usecase.EventPublisher
	if err != nil {
		log.Printf("WARNING: failed to connect to NATS: %v. Publishing will be disabled.", err)
		// Mock publisher that does nothing if NATS is down
		publisher = &noopPublisher{}
	} else {
		defer nc.Close()
		publisher = event.NewNatsPublisher(nc)
	}

	repo := repository.NewPostgresRepository(db)

	// 4. Connect to Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	
	var cacheRepo usecase.CacheRepository
	var redisClient *redis.Client

	redisCache, err := cache.NewRedisCache(redisURL)
	if err != nil {
		log.Printf("WARNING: failed to connect to Redis: %v. Caching will be disabled.", err)
		cacheRepo = &noopCache{}
	} else {
		cacheRepo = redisCache
		// We need the client for the rate limiter too
		opts, _ := redis.ParseURL(redisURL)
		redisClient = redis.NewClient(opts)
	}

	uc := usecase.NewDoctorUsecase(repo, publisher, cacheRepo)

	// 5. Rate Limiter
	var interceptors []grpc.UnaryServerInterceptor
	if redisClient != nil {
		limiter := middleware.NewRateLimiter(redisClient)
		interceptors = append(interceptors, limiter.UnaryInterceptor())
	}

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
	handler := grpctransport.NewDoctorServer(uc)
	handler.Register(server)
	reflection.Register(server)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	log.Printf("Doctor service started on port %s", port)
	return server.Serve(lis)
}

type noopPublisher struct{}

func (p *noopPublisher) Publish(ctx context.Context, subject string, event interface{}) error {
	log.Printf("NATS unavailable, skipping publish to %s", subject)
	return nil
}

type noopCache struct{}

func (c *noopCache) Get(ctx context.Context, key string, target interface{}) (bool, error) {
	return false, nil
}

func (c *noopCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (c *noopCache) Delete(ctx context.Context, key string) error {
	return nil
}
