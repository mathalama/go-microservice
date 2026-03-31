package repository

import (
	"fmt"
	"sync"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
)

type DoctorMemoryRepository struct {
	mu       sync.RWMutex
	items    map[string]model.Doctor
	emails   map[string]string
	sequence int
}

func NewDoctorMemoryRepository() *DoctorMemoryRepository {
	return &DoctorMemoryRepository{
		items:  make(map[string]model.Doctor),
		emails: make(map[string]string),
	}
}

func (r *DoctorMemoryRepository) NextID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sequence++
	return fmt.Sprintf("doctor-%d", r.sequence)
}

func (r *DoctorMemoryRepository) Create(doctor model.Doctor) (model.Doctor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[doctor.ID] = doctor
	r.emails[doctor.Email] = doctor.ID
	return doctor, nil
}

func (r *DoctorMemoryRepository) GetByID(id string) (model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doctor, ok := r.items[id]
	if !ok {
		return model.Doctor{}, usecase.ErrDoctorNotFound
	}
	return doctor, nil
}

func (r *DoctorMemoryRepository) List() ([]model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]model.Doctor, 0, len(r.items))
	for _, doctor := range r.items {
		result = append(result, doctor)
	}
	return result, nil
}

func (r *DoctorMemoryRepository) ExistsByEmail(email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.emails[email]
	return ok, nil
}
