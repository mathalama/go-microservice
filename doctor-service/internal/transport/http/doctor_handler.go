package http

import (
	"errors"
	"net/http"

	"doctor-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type DoctorHandler struct {
	uc *usecase.DoctorUsecase
}

func NewDoctorHandler(uc *usecase.DoctorUsecase) *DoctorHandler {
	return &DoctorHandler{uc: uc}
}

type createDoctorRequest struct {
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func (h *DoctorHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/doctors", h.CreateDoctor)
	router.GET("/doctors/:id", h.GetDoctor)
	router.GET("/doctors", h.ListDoctors)
}

func (h *DoctorHandler) CreateDoctor(c *gin.Context) {
	var req createDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	doctor, err := h.uc.CreateDoctor(req.FullName, req.Specialization, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, doctor)
}

func (h *DoctorHandler) GetDoctor(c *gin.Context) {
	doctor, err := h.uc.GetDoctor(c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrDoctorNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, doctor)
}

func (h *DoctorHandler) ListDoctors(c *gin.Context) {
	doctors, err := h.uc.ListDoctors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, doctors)
}
