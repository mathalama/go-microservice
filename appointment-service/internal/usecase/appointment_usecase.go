package usecase

import (
	"appointment-service/internal/domain"
	"fmt"
	"log"
	"strings"
	"time"
)

type AppointmentUseCase interface {
	CreateAppointment(title, description, doctorID string) (domain.Appointment, error)
	GetAppointment(id string) (domain.Appointment, error)
	ListAppointments() ([]domain.Appointment, error)
	UpdateStatus(id string, status domain.Status) (domain.Appointment, error)
}

type appointmentUseCase struct {
	repo         domain.AppointmentRepository
	doctorClient domain.DoctorClient
}

func NewAppointmentUseCase(repo domain.AppointmentRepository, doctorClient domain.DoctorClient) AppointmentUseCase {
	return &appointmentUseCase{
		repo:         repo,
		doctorClient: doctorClient,
	}
}

func (u *appointmentUseCase) CreateAppointment(title, description, doctorID string) (domain.Appointment, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	doctorID = strings.TrimSpace(doctorID)

	if title == "" {
		return domain.Appointment{}, fmt.Errorf("title is required")
	}
	if doctorID == "" {
		return domain.Appointment{}, fmt.Errorf("doctor_id is required")
	}

	if err := u.ensureDoctorExists(doctorID); err != nil {
		return domain.Appointment{}, err
	}

	now := time.Now().UTC()
	appointment := domain.Appointment{
		ID:          u.repo.NextID(),
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      domain.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return u.repo.Create(appointment)
}

func (u *appointmentUseCase) GetAppointment(id string) (domain.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Appointment{}, fmt.Errorf("id is required")
	}
	return u.repo.GetByID(id)
}

func (u *appointmentUseCase) ListAppointments() ([]domain.Appointment, error) {
	return u.repo.List()
}

func (u *appointmentUseCase) UpdateStatus(id string, status domain.Status) (domain.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Appointment{}, fmt.Errorf("id is required")
	}
	if !status.IsValid() {
		return domain.Appointment{}, domain.ErrInvalidStatus
	}

	current, err := u.repo.GetByID(id)
	if err != nil {
		return domain.Appointment{}, err
	}

	if err := u.ensureDoctorExists(current.DoctorID); err != nil {
		return domain.Appointment{}, err
	}

	if current.Status == domain.StatusDone && status == domain.StatusNew {
		return domain.Appointment{}, domain.ErrForbiddenStatusTransit
	}

	return u.repo.UpdateStatus(id, status, time.Now().UTC())
}

func (u *appointmentUseCase) ensureDoctorExists(doctorID string) error {
	exists, err := u.doctorClient.DoctorExists(doctorID)
	if err != nil {
		log.Printf("doctor validation failed: doctor_id=%s error=%v", doctorID, err)
		return domain.ErrDependencyUnavailable
	}
	if !exists {
		return domain.ErrDoctorNotFound
	}
	return nil
}
