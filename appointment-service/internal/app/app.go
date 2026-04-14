package app

import (
	"fmt"
	"net"
	"os"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	grpctransport "appointment-service/internal/transport/grpc"
	"appointment-service/internal/usecase"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
)

func Run() error {
	port := getEnv("APPOINTMENT_SERVICE_PORT", "8080")
	doctorServiceAddr := getEnv("DOCTOR_SERVICE_ADDR", "localhost:8081")

	repo := repository.NewAppointmentMemoryRepository()
	conn, err := grpc.NewClient(doctorServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	doctorClient := client.NewDoctorGRPCClient(conn)
	uc := usecase.NewAppointmentUseCase(repo, doctorClient)
	server := grpc.NewServer()
	handler := grpctransport.NewAppointmentServer(uc)
	handler.Register(server)
	reflection.Register(server)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	log.Printf("Appointment service started on port %s (connecting to Doctor service at %s)", port, doctorServiceAddr)
	return server.Serve(lis)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
