package usecase

import (
	"context"
	"fmt"
	"strings"

	"doctor-service/internal/model"
)

type DoctorRepository interface {
	Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	List(ctx context.Context) ([]model.Doctor, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	NextID(ctx context.Context) string
}

type DoctorUsecase interface {
	CreateDoctor(ctx context.Context, fullName, specialization, email string) (model.Doctor, error)
	GetDoctor(ctx context.Context, id string) (model.Doctor, error)
	ListDoctors(ctx context.Context) ([]model.Doctor, error)
}

type doctorUsecase struct {
	repo DoctorRepository
}

func NewDoctorUsecase(repo DoctorRepository) DoctorUsecase {
	return &doctorUsecase{repo: repo}
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
	return u.repo.Create(ctx, doctor)
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
