package repository

import (
	"context"
	"sync"

	"doctor-service/internal/model"
)

type DoctorMemoryRepository struct {
	mu     sync.RWMutex
	items  map[string]model.Doctor
	emails map[string]string
}

func NewDoctorMemoryRepository() *DoctorMemoryRepository {
	return &DoctorMemoryRepository{
		items:  make(map[string]model.Doctor),
		emails: make(map[string]string),
	}
}

func (r *DoctorMemoryRepository) Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[doctor.ID] = doctor
	r.emails[doctor.Email] = doctor.ID
	return doctor, nil
}

func (r *DoctorMemoryRepository) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doctor, ok := r.items[id]
	if !ok {
		return model.Doctor{}, model.ErrDoctorNotFound
	}
	return doctor, nil
}

func (r *DoctorMemoryRepository) List(ctx context.Context) ([]model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]model.Doctor, 0, len(r.items))
	for _, doctor := range r.items {
		result = append(result, doctor)
	}
	return result, nil
}

func (r *DoctorMemoryRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.emails[email]
	return ok, nil
}
