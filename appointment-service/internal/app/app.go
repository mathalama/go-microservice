package app

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	httptransport "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

func Run() error {
	port := getEnv("APPOINTMENT_SERVICE_PORT", "8082")
	doctorServiceURL := getEnv("DOCTOR_SERVICE_URL", "http://localhost:8081")
	timeoutMS := getEnvInt("DOCTOR_SERVICE_TIMEOUT_MS", 2000)

	repo := repository.NewAppointmentMemoryRepository()
	doctorClient := client.NewDoctorHTTPClient(doctorServiceURL, time.Duration(timeoutMS)*time.Millisecond)
	uc := usecase.NewAppointmentUsecase(repo, doctorClient)
	handler := httptransport.NewAppointmentHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	return router.Run(fmt.Sprintf(":%s", port))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
