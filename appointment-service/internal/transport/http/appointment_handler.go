package http

import (
	"errors"
	"net/http"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AppointmentHandler struct {
	uc *usecase.AppointmentUsecase
}

func NewAppointmentHandler(uc *usecase.AppointmentUsecase) *AppointmentHandler {
	return &AppointmentHandler{uc: uc}
}

type createAppointmentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DoctorID    string `json:"doctor_id"`
}

type updateStatusRequest struct {
	Status model.Status `json:"status"`
}

func (h *AppointmentHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/appointments", h.CreateAppointment)
	router.GET("/appointments/:id", h.GetAppointment)
	router.GET("/appointments", h.ListAppointments)
	router.PATCH("/appointments/:id/status", h.UpdateStatus)
}

func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {
	var req createAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	appointment, err := h.uc.CreateAppointment(req.Title, req.Description, req.DoctorID)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}
	c.JSON(http.StatusCreated, appointment)
}

func (h *AppointmentHandler) GetAppointment(c *gin.Context) {
	appointment, err := h.uc.GetAppointment(c.Param("id"))
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, appointment)
}

func (h *AppointmentHandler) ListAppointments(c *gin.Context) {
	appointments, err := h.uc.ListAppointments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, appointments)
}

func (h *AppointmentHandler) UpdateStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	appointment, err := h.uc.UpdateStatus(c.Param("id"), req.Status)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, appointment)
}

func (h *AppointmentHandler) handleUsecaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrAppointmentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrDoctorNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrDependencyUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "doctor service is unavailable, appointment operation cannot be completed",
		})
	case errors.Is(err, usecase.ErrInvalidStatus), errors.Is(err, usecase.ErrForbiddenStatusTransit):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
