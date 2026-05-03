package usecase

import (
	"context"
	"fmt"
	"log"
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

type DoctorUsecase interface {
	CreateDoctor(ctx context.Context, fullName, specialization, email string) (model.Doctor, error)
	GetDoctor(ctx context.Context, id string) (model.Doctor, error)
	ListDoctors(ctx context.Context) ([]model.Doctor, error)
}

type doctorUsecase struct {
	repo      DoctorRepository
	publisher EventPublisher
}

func NewDoctorUsecase(repo DoctorRepository, publisher EventPublisher) DoctorUsecase {
	return &doctorUsecase{
		repo:      repo,
		publisher: publisher,
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

	return createdDoctor, nil
}

func (u *doctorUsecase) GetDoctor(ctx context.Context, id string) (model.Doctor, error) {
	if strings.TrimSpace(id) == "" {
		return model.Doctor{}, fmt.Errorf("id is required")
	}
	return u.repo.GetByID(ctx, id)
}

func (u *doctorUsecase) ListDoctors(ctx context.Context) ([]model.Doctor, error) {
	return u.repo.List(ctx)
}
