package usecase

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"appointment-service/internal/model"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, event interface{}) error
}

type CacheRepository interface {
	Get(ctx context.Context, key string, target interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type AppointmentUseCase interface {
	CreateAppointment(ctx context.Context, title, description, doctorID string) (model.Appointment, error)
	GetAppointment(ctx context.Context, id string) (model.Appointment, error)
	ListAppointments(ctx context.Context) ([]model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error)
}

type appointmentUseCase struct {
	repo         model.AppointmentRepository
	doctorClient model.DoctorClient
	publisher    EventPublisher
	cache        CacheRepository
	cacheTTL     time.Duration
}

func NewAppointmentUseCase(repo model.AppointmentRepository, doctorClient model.DoctorClient, publisher EventPublisher, cacheRepo CacheRepository) AppointmentUseCase {
	ttlStr := os.Getenv("CACHE_TTL_SECONDS")
	ttlSec, err := strconv.Atoi(ttlStr)
	if err != nil {
		ttlSec = 60 // Default 60s
	}

	return &appointmentUseCase{
		repo:         repo,
		doctorClient: doctorClient,
		publisher:    publisher,
		cache:        cacheRepo,
		cacheTTL:     time.Duration(ttlSec) * time.Second,
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
		ID:          u.repo.NextID(ctx),
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	created, err := u.repo.Create(ctx, appointment)
	if err != nil {
		return model.Appointment{}, err
	}

	// Publish event
	event := map[string]interface{}{
		"event_type":  "appointments.created",
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"id":          created.ID,
		"title":       created.Title,
		"doctor_id":   created.DoctorID,
		"status":      string(created.Status),
	}
	if err := u.publisher.Publish(ctx, "appointments.created", event); err != nil {
		log.Printf("failed to publish appointments.created event: %v", err)
	}

	// Invalidate list cache (Write-Around)
	if err := u.cache.Delete(ctx, "appointments:list"); err != nil {
		log.Printf("failed to invalidate appointments:list cache: %v", err)
	}

	return created, nil
}

func (u *appointmentUseCase) GetAppointment(ctx context.Context, id string) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}

	key := fmt.Sprintf("appointment:%s", id)
	var appointment model.Appointment
	found, err := u.cache.Get(ctx, key, &appointment)
	if err == nil && found {
		return appointment, nil
	}

	appointment, err = u.repo.GetByID(ctx, id)
	if err != nil {
		return model.Appointment{}, err
	}

	if err := u.cache.Set(ctx, key, appointment, u.cacheTTL); err != nil {
		log.Printf("failed to set appointment cache for %s: %v", id, err)
	}

	return appointment, nil
}

func (u *appointmentUseCase) ListAppointments(ctx context.Context) ([]model.Appointment, error) {
	key := "appointments:list"
	var appointments []model.Appointment
	found, err := u.cache.Get(ctx, key, &appointments)
	if err == nil && found {
		return appointments, nil
	}

	appointments, err = u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.cache.Set(ctx, key, appointments, u.cacheTTL); err != nil {
		log.Printf("failed to set appointments:list cache: %v", err)
	}

	return appointments, nil
}

func (u *appointmentUseCase) UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error) {
	if strings.TrimSpace(id) == "" {
		return model.Appointment{}, fmt.Errorf("id is required")
	}
	if !status.IsValid() {
		return model.Appointment{}, model.ErrInvalidStatus
	}

	// We still need to fetch the appointment to check the doctor_id for the gRPC call.
	// This "double read" (one here, one in UpdateStatus transaction) is a trade-off 
	// to avoid holding a database lock during an external gRPC call.
	current, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return model.Appointment{}, err
	}

	if err := u.ensureDoctorExists(ctx, current.DoctorID); err != nil {
		return model.Appointment{}, err
	}

	updated, oldStatus, err := u.repo.UpdateStatus(ctx, id, status, time.Now().UTC())
	if err != nil {
		return model.Appointment{}, err
	}

	// Publish event
	event := map[string]interface{}{
		"event_type":  "appointments.status_updated",
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"id":          updated.ID,
		"doctor_id":   updated.DoctorID,
		"old_status":  string(oldStatus),
		"new_status":  string(updated.Status),
	}
	if err := u.publisher.Publish(ctx, "appointments.status_updated", event); err != nil {
		log.Printf("failed to publish appointments.status_updated event: %v", err)
	}

	// Write-Through: update/invalidate cache
	key := fmt.Sprintf("appointment:%s", id)
	if err := u.cache.Set(ctx, key, updated, u.cacheTTL); err != nil {
		log.Printf("failed to update appointment cache for %s: %v", id, err)
	}
	if err := u.cache.Delete(ctx, "appointments:list"); err != nil {
		log.Printf("failed to invalidate appointments:list cache: %v", err)
	}

	return updated, nil
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
