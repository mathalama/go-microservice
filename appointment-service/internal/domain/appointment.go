package domain

import (
	"errors"
	"time"
)

var (
	ErrAppointmentNotFound    = errors.New("appointment not found")
	ErrDoctorNotFound         = errors.New("doctor does not exist")
	ErrDependencyUnavailable  = errors.New("doctor service is unavailable")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrForbiddenStatusTransit = errors.New("transition from done to new is not allowed")
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) IsValid() bool {
	return s == StatusNew || s == StatusInProgress || s == StatusDone
}

type Appointment struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DoctorID    string    `json:"doctor_id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AppointmentRepository interface {
	Create(appointment Appointment) (Appointment, error)
	GetByID(id string) (Appointment, error)
	List() ([]Appointment, error)
	UpdateStatus(id string, status Status, updatedAt time.Time) (Appointment, error)
	NextID() string
}

type DoctorClient interface {
	DoctorExists(doctorID string) (bool, error)
}
