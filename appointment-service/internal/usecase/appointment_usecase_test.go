package usecase

import (
	"context"
	"errors"
	"testing"

	"appointment-service/internal/model"
	"appointment-service/internal/repository"
)

type stubDoctorClient struct {
	exists bool
	err    error
}

func (s stubDoctorClient) DoctorExists(ctx context.Context, doctorID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.exists, nil
}

func TestCreateAppointmentRequiresTitleAndDoctorID(t *testing.T) {
	t.Parallel()

	repo := repository.NewAppointmentMemoryRepository()
	uc := NewAppointmentUseCase(repo, stubDoctorClient{exists: true})

	if _, err := uc.CreateAppointment(context.Background(), "", "desc", "doctor-1"); err == nil {
		t.Fatal("CreateAppointment() with empty title error = nil, want validation error")
	}

	if _, err := uc.CreateAppointment(context.Background(), "Visit", "desc", ""); err == nil {
		t.Fatal("CreateAppointment() with empty doctor_id error = nil, want validation error")
	}
}

func TestCreateAppointmentChecksDoctorExistence(t *testing.T) {
	t.Parallel()

	repo := repository.NewAppointmentMemoryRepository()
	uc := NewAppointmentUseCase(repo, stubDoctorClient{exists: false})

	_, err := uc.CreateAppointment(context.Background(), "Visit", "desc", "doctor-404")
	if err == nil {
		t.Fatal("CreateAppointment() error = nil, want doctor validation error")
	}
	if !errors.Is(err, model.ErrDoctorNotFound) {
		t.Fatalf("CreateAppointment() error = %v, want %v", err, model.ErrDoctorNotFound)
	}
}

func TestCreateAppointmentReturnsDependencyUnavailableWhenDoctorServiceFails(t *testing.T) {
	t.Parallel()

	repo := repository.NewAppointmentMemoryRepository()
	uc := NewAppointmentUseCase(repo, stubDoctorClient{err: errors.New("connection refused")})

	_, err := uc.CreateAppointment(context.Background(), "Visit", "desc", "doctor-1")
	if err == nil {
		t.Fatal("CreateAppointment() error = nil, want dependency unavailable")
	}
	if !errors.Is(err, model.ErrDependencyUnavailable) {
		t.Fatalf("CreateAppointment() error = %v, want %v", err, model.ErrDependencyUnavailable)
	}
}

func TestUpdateStatusValidations(t *testing.T) {
	t.Parallel()

	repo := repository.NewAppointmentMemoryRepository()
	uc := NewAppointmentUseCase(repo, stubDoctorClient{exists: true})

	created, err := uc.CreateAppointment(context.Background(), "Visit", "desc", "doctor-1")
	if err != nil {
		t.Fatalf("CreateAppointment() error = %v", err)
	}

	updated, err := uc.UpdateStatus(context.Background(), created.ID, model.StatusInProgress)
	if err != nil {
		t.Fatalf("UpdateStatus() to in_progress error = %v", err)
	}
	if updated.Status != model.StatusInProgress {
		t.Fatalf("UpdateStatus() status = %q, want %q", updated.Status, model.StatusInProgress)
	}

	done, err := uc.UpdateStatus(context.Background(), created.ID, model.StatusDone)
	if err != nil {
		t.Fatalf("UpdateStatus() to done error = %v", err)
	}
	if done.Status != model.StatusDone {
		t.Fatalf("UpdateStatus() final status = %q, want %q", done.Status, model.StatusDone)
	}

	_, err = uc.UpdateStatus(context.Background(), created.ID, model.StatusNew)
	if err == nil {
		t.Fatal("UpdateStatus() done->new error = nil, want forbidden transition")
	}
	if !errors.Is(err, model.ErrForbiddenStatusTransit) {
		t.Fatalf("UpdateStatus() done->new error = %v, want %v", err, model.ErrForbiddenStatusTransit)
	}

	_, err = uc.UpdateStatus(context.Background(), created.ID, model.Status("paused"))
	if err == nil {
		t.Fatal("UpdateStatus() invalid status error = nil, want invalid status")
	}
	if !errors.Is(err, model.ErrInvalidStatus) {
		t.Fatalf("UpdateStatus() invalid status error = %v, want %v", err, model.ErrInvalidStatus)
	}
}
