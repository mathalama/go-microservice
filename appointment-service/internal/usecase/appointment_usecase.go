package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"appointment-service/internal/model"
)

type AppointmentUseCase interface {
	CreateAppointment(ctx context.Context, title, description, doctorID string) (model.Appointment, error)
	GetAppointment(ctx context.Context, id string) (model.Appointment, error)
	ListAppointments(ctx context.Context) ([]model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error)
}

type appointmentUseCase struct {
	repo         model.AppointmentRepository
	doctorClient model.DoctorClient
}

func NewAppointmentUseCase(repo model.AppointmentRepository, doctorClient model.DoctorClient) AppointmentUseCase {
	return &appointmentUseCase{
		repo:         repo,
		doctorClient: doctorClient,
	}
}

func (u *appointmentUseCase) CreateAppointment(ctx context.Context, title, description, doctorID string) (model.Appointment, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	doctorID = strings.TrimSpace(doctorID)

	if title == "" {
		return model.Appointment{}, fmt.Errorf("title is required")
	}
	if doctorID == "" {
		return model.Appointment{}, fmt.Errorf("doctor_id is required")
	}

	if err := u.ensureDoctorExists(ctx, doctorID); err != nil {
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

func (u *appointmentUseCase) GetAppointment(ctx context.Context, id string) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}
	return u.repo.GetByID(id)
}

func (u *appointmentUseCase) ListAppointments(ctx context.Context) ([]model.Appointment, error) {
	return u.repo.List()
}

func (u *appointmentUseCase) UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}
	if !status.IsValid() {
		return model.Appointment{}, model.ErrInvalidStatus
	}

	current, err := u.repo.GetByID(id)
	if err != nil {
		return model.Appointment{}, err
	}

	if err := u.ensureDoctorExists(ctx, current.DoctorID); err != nil {
		return model.Appointment{}, err
	}

	if current.Status == model.StatusDone && status == model.StatusNew {
		return model.Appointment{}, model.ErrForbiddenStatusTransit
	}

	return u.repo.UpdateStatus(id, status, time.Now().UTC())
}

func (u *appointmentUseCase) ensureDoctorExists(ctx context.Context, doctorID string) error {
	exists, err := u.doctorClient.DoctorExists(ctx, doctorID)
	if err != nil {
		log.Printf("doctor validation failed: doctor_id=%s error=%v", doctorID, err)
		return model.ErrDependencyUnavailable
	}
	if !exists {
		return model.ErrDoctorNotFound
	}
	return nil
}
