package repository

import (
	"fmt"
	"sync"
	"time"

	"appointment-service/internal/domain"
)

type AppointmentMemoryRepository struct {
	mu       sync.RWMutex
	items    map[string]domain.Appointment
	sequence int
}

func NewAppointmentMemoryRepository() *AppointmentMemoryRepository {
	return &AppointmentMemoryRepository{
		items: make(map[string]domain.Appointment),
	}
}

func (r *AppointmentMemoryRepository) NextID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sequence++
	return fmt.Sprintf("appointment-%d", r.sequence)
}

func (r *AppointmentMemoryRepository) Create(appointment domain.Appointment) (domain.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[appointment.ID] = appointment
	return appointment, nil
}

func (r *AppointmentMemoryRepository) GetByID(id string) (domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	appointment, ok := r.items[id]
	if !ok {
		return domain.Appointment{}, domain.ErrAppointmentNotFound
	}
	return appointment, nil
}

func (r *AppointmentMemoryRepository) List() ([]domain.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Appointment, 0, len(r.items))
	for _, appointment := range r.items {
		result = append(result, appointment)
	}
	return result, nil
}

func (r *AppointmentMemoryRepository) UpdateStatus(id string, status domain.Status, updatedAt time.Time) (domain.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	appointment, ok := r.items[id]
	if !ok {
		return domain.Appointment{}, domain.ErrAppointmentNotFound
	}
	appointment.Status = status
	appointment.UpdatedAt = updatedAt
	r.items[id] = appointment
	return appointment, nil
}
