package app

import (
	"fmt"
	"net"
	"os"

	"doctor-service/internal/repository"
	grpctransport "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run() error {
	port := os.Getenv("DOCTOR_SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	repo := repository.NewDoctorMemoryRepository()
	uc := usecase.NewDoctorUsecase(repo)
	server := grpc.NewServer()
	handler := grpctransport.NewDoctorServer(uc)
	handler.Register(server)
	reflection.Register(server)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	return server.Serve(lis)
}
