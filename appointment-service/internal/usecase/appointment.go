package usecase

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"appointment-service/internal/model"
)

var (
	ErrAppointmentNotFound    = errors.New("appointment not found")
	ErrDoctorNotFound         = errors.New("doctor does not exist")
	ErrDependencyUnavailable  = errors.New("doctor service is unavailable")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrForbiddenStatusTransit = errors.New("transition from done to new is not allowed")
)

type AppointmentRepository interface {
	Create(appointment model.Appointment) (model.Appointment, error)
	GetByID(id string) (model.Appointment, error)
	List() ([]model.Appointment, error)
	UpdateStatus(id string, status model.Status, updatedAt time.Time) (model.Appointment, error)
	NextID() string
}

type DoctorClient interface {
	DoctorExists(doctorID string) (bool, error)
}

type AppointmentUsecase struct {
	repo         AppointmentRepository
	doctorClient DoctorClient
}

func NewAppointmentUsecase(repo AppointmentRepository, doctorClient DoctorClient) *AppointmentUsecase {
	return &AppointmentUsecase{
		repo:         repo,
		doctorClient: doctorClient,
	}
}

func (u *AppointmentUsecase) CreateAppointment(title, description, doctorID string) (model.Appointment, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	doctorID = strings.TrimSpace(doctorID)

	if title == "" {
		return model.Appointment{}, fmt.Errorf("title is required")
	}
	if doctorID == "" {
		return model.Appointment{}, fmt.Errorf("doctor_id is required")
	}

	if err := u.ensureDoctorExists(doctorID); err != nil {
		return model.Appointment{}, err
	}

	now := time.Now().UTC()
	appointment := model.Appointment{
		ID:          u.repo.NextID(),
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return u.repo.Create(appointment)
}

func (u *AppointmentUsecase) GetAppointment(id string) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}
	return u.repo.GetByID(id)
}

func (u *AppointmentUsecase) ListAppointments() ([]model.Appointment, error) {
	return u.repo.List()
}

func (u *AppointmentUsecase) UpdateStatus(id string, status model.Status) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}
	if !status.IsValid() {
		return model.Appointment{}, ErrInvalidStatus
	}

	current, err := u.repo.GetByID(id)
	if err != nil {
		return model.Appointment{}, err
	}

	if err := u.ensureDoctorExists(current.DoctorID); err != nil {
		return model.Appointment{}, err
	}

	if current.Status == model.StatusDone && status == model.StatusNew {
		return model.Appointment{}, ErrForbiddenStatusTransit
	}

	return u.repo.UpdateStatus(id, status, time.Now().UTC())
}

func (u *AppointmentUsecase) ensureDoctorExists(doctorID string) error {
	exists, err := u.doctorClient.DoctorExists(doctorID)
	if err != nil {
		log.Printf("doctor validation failed: doctor_id=%s error=%v", doctorID, err)
		return ErrDependencyUnavailable
	}
	if !exists {
		return ErrDoctorNotFound
	}
	return nil
}
