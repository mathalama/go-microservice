package repository

import (
	"fmt"
	"sync"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
)

type AppointmentMemoryRepository struct {
	mu       sync.RWMutex
	items    map[string]model.Appointment
	sequence int
}

func NewAppointmentMemoryRepository() *AppointmentMemoryRepository {
	return &AppointmentMemoryRepository{
		items: make(map[string]model.Appointment),
	}
}

func (r *AppointmentMemoryRepository) NextID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sequence++
	return fmt.Sprintf("appointment-%d", r.sequence)
}

func (r *AppointmentMemoryRepository) Create(appointment model.Appointment) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[appointment.ID] = appointment
	return appointment, nil
}

func (r *AppointmentMemoryRepository) GetByID(id string) (model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	appointment, ok := r.items[id]
	if !ok {
		return model.Appointment{}, usecase.ErrAppointmentNotFound
	}
	return appointment, nil
}

func (r *AppointmentMemoryRepository) List() ([]model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]model.Appointment, 0, len(r.items))
	for _, appointment := range r.items {
		result = append(result, appointment)
	}
	return result, nil
}

func (r *AppointmentMemoryRepository) UpdateStatus(id string, status model.Status, updatedAt time.Time) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	appointment, ok := r.items[id]
	if !ok {
		return model.Appointment{}, usecase.ErrAppointmentNotFound
	}
	appointment.Status = status
	appointment.UpdatedAt = updatedAt
	r.items[id] = appointment
	return appointment, nil
}
