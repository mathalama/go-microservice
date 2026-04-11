package usecase

import (
	"context"
	"testing"

	"doctor-service/internal/model"
	"doctor-service/internal/repository"
)

func TestCreateDoctorAssignsSequentialIDs(t *testing.T) {
	t.Parallel()

	repo := repository.NewDoctorMemoryRepository()
	uc := NewDoctorUsecase(repo)

	first, err := uc.CreateDoctor(context.Background(), "Dr. Aigerim Sarsen", "Cardiology", "aigerim@example.com")
	if err != nil {
		t.Fatalf("CreateDoctor() error = %v", err)
	}

	second, err := uc.CreateDoctor(context.Background(), "Dr. Marat Omarov", "Therapy", "marat@example.com")
	if err != nil {
		t.Fatalf("CreateDoctor() second error = %v", err)
	}

	if first.ID != "doctor-1" {
		t.Fatalf("first doctor ID = %q, want %q", first.ID, "doctor-1")
	}
	if second.ID != "doctor-2" {
		t.Fatalf("second doctor ID = %q, want %q", second.ID, "doctor-2")
	}
}

func TestCreateDoctorRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := repository.NewDoctorMemoryRepository()
	uc := NewDoctorUsecase(repo)

	if _, err := uc.CreateDoctor(context.Background(), "Dr. Aigerim Sarsen", "Cardiology", "AIGERIM@example.com"); err != nil {
		t.Fatalf("CreateDoctor() seed error = %v", err)
	}

	_, err := uc.CreateDoctor(context.Background(), "Dr. Another", "Dermatology", "aigerim@example.com")
	if err == nil {
		t.Fatal("CreateDoctor() error = nil, want duplicate email error")
	}
	if err != model.ErrEmailExists {
		t.Fatalf("CreateDoctor() error = %v, want %v", err, model.ErrEmailExists)
	}
}

func TestCreateDoctorRequiresFullNameAndEmail(t *testing.T) {
	t.Parallel()

	repo := repository.NewDoctorMemoryRepository()
	uc := NewDoctorUsecase(repo)

	if _, err := uc.CreateDoctor(context.Background(), "", "Cardiology", "doctor@example.com"); err == nil {
		t.Fatal("CreateDoctor() with empty full_name error = nil, want validation error")
	}

	if _, err := uc.CreateDoctor(context.Background(), "Dr. Aigerim Sarsen", "Cardiology", ""); err == nil {
		t.Fatal("CreateDoctor() with empty email error = nil, want validation error")
	}
}
