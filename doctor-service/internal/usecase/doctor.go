package usecase

import (
	"errors"
	"fmt"
	"strings"

	"doctor-service/internal/model"
)

var (
	ErrDoctorNotFound = errors.New("doctor not found")
	ErrEmailExists    = errors.New("doctor with this email already exists")
)

type DoctorRepository interface {
	Create(doctor model.Doctor) (model.Doctor, error)
	GetByID(id string) (model.Doctor, error)
	List() ([]model.Doctor, error)
	ExistsByEmail(email string) (bool, error)
	NextID() string
}

type DoctorUsecase struct {
	repo DoctorRepository
}

func NewDoctorUsecase(repo DoctorRepository) *DoctorUsecase {
	return &DoctorUsecase{repo: repo}
}

func (u *DoctorUsecase) CreateDoctor(fullName, specialization, email string) (model.Doctor, error) {
	fullName = strings.TrimSpace(fullName)
	email = strings.TrimSpace(strings.ToLower(email))
	specialization = strings.TrimSpace(specialization)

	if fullName == "" {
		return model.Doctor{}, fmt.Errorf("full_name is required")
	}
	if email == "" {
		return model.Doctor{}, fmt.Errorf("email is required")
	}

	exists, err := u.repo.ExistsByEmail(email)
	if err != nil {
		return model.Doctor{}, err
	}
	if exists {
		return model.Doctor{}, ErrEmailExists
	}

	doctor := model.Doctor{
		ID:             u.repo.NextID(),
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}
	return u.repo.Create(doctor)
}

func (u *DoctorUsecase) GetDoctor(id string) (model.Doctor, error) {
	if strings.TrimSpace(id) == "" {
		return model.Doctor{}, fmt.Errorf("id is required")
	}
	return u.repo.GetByID(id)
}

func (u *DoctorUsecase) ListDoctors() ([]model.Doctor, error) {
	return u.repo.List()
}
