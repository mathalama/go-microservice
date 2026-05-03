package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"doctor-service/internal/event"
	"doctor-service/internal/repository"
	grpctransport "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
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
	uc := usecase.NewDoctorUsecase(repo, publisher)

	server := grpc.NewServer()
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
