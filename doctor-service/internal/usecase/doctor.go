package usecase

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"doctor-service/internal/model"
)

type DoctorRepository interface {
	Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	List(ctx context.Context) ([]model.Doctor, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	NextID(ctx context.Context) string
}

type EventPublisher interface {
	Publish(ctx context.Context, subject string, event interface{}) error
}

type CacheRepository interface {
	Get(ctx context.Context, key string, target interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type DoctorUsecase interface {
	CreateDoctor(ctx context.Context, fullName, specialization, email string) (model.Doctor, error)
	GetDoctor(ctx context.Context, id string) (model.Doctor, error)
	ListDoctors(ctx context.Context) ([]model.Doctor, error)
}

type doctorUsecase struct {
	repo      DoctorRepository
	publisher EventPublisher
	cache     CacheRepository
	cacheTTL  time.Duration
}

func NewDoctorUsecase(repo DoctorRepository, publisher EventPublisher, cacheRepo CacheRepository) DoctorUsecase {
	ttlStr := os.Getenv("CACHE_TTL_SECONDS")
	ttlSec, err := strconv.Atoi(ttlStr)
	if err != nil {
		ttlSec = 60 // Default 60s
	}

	return &doctorUsecase{
		repo:      repo,
		publisher: publisher,
		cache:     cacheRepo,
		cacheTTL:  time.Duration(ttlSec) * time.Second,
	}
}

func (u *doctorUsecase) CreateDoctor(ctx context.Context, fullName, specialization, email string) (model.Doctor, error) {
	fullName = strings.TrimSpace(fullName)
	email = strings.TrimSpace(strings.ToLower(email))
	specialization = strings.TrimSpace(specialization)

	if fullName == "" {
		return model.Doctor{}, fmt.Errorf("full_name is required")
	}
	if email == "" {
		return model.Doctor{}, fmt.Errorf("email is required")
	}

	exists, err := u.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return model.Doctor{}, err
	}
	if exists {
		return model.Doctor{}, model.ErrEmailExists
	}

	doctor := model.Doctor{
		ID:             u.repo.NextID(ctx),
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}

	createdDoctor, err := u.repo.Create(ctx, doctor)
	if err != nil {
		return model.Doctor{}, err
	}

	// Publish event
	event := map[string]interface{}{
		"event_type":     "doctors.created",
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
		"id":             createdDoctor.ID,
		"full_name":      createdDoctor.FullName,
		"specialization": createdDoctor.Specialization,
		"email":          createdDoctor.Email,
	}
	if err := u.publisher.Publish(ctx, "doctors.created", event); err != nil {
		log.Printf("failed to publish doctor.created event: %v", err)
	}

	// Invalidate list cache (Write-Through strategy as per assignment table)
	if err := u.cache.Delete(ctx, "doctors:list"); err != nil {
		log.Printf("failed to invalidate doctors:list cache: %v", err)
	}

	return createdDoctor, nil
}

func (u *doctorUsecase) GetDoctor(ctx context.Context, id string) (model.Doctor, error) {
	if strings.TrimSpace(id) == "" {
		return model.Doctor{}, fmt.Errorf("id is required")
	}

	key := fmt.Sprintf("doctor:%s", id)
	var doctor model.Doctor
	found, err := u.cache.Get(ctx, key, &doctor)
	if err == nil && found {
		return doctor, nil
	}

	doctor, err = u.repo.GetByID(ctx, id)
	if err != nil {
		return model.Doctor{}, err
	}

	if err := u.cache.Set(ctx, key, doctor, u.cacheTTL); err != nil {
		log.Printf("failed to set doctor cache for %s: %v", id, err)
	}

	return doctor, nil
}

func (u *doctorUsecase) ListDoctors(ctx context.Context) ([]model.Doctor, error) {
	key := "doctors:list"
	var doctors []model.Doctor
	found, err := u.cache.Get(ctx, key, &doctors)
	if err == nil && found {
		return doctors, nil
	}

	doctors, err = u.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.cache.Set(ctx, key, doctors, u.cacheTTL); err != nil {
		log.Printf("failed to set doctors:list cache: %v", err)
	}

	return doctors, nil
}
