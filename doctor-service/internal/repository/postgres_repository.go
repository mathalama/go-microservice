package repository

import (
	"context"
	"database/sql"
	"fmt"

	"doctor-service/internal/model"
	"github.com/google/uuid"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error) {
	query := `INSERT INTO doctors (id, full_name, specialization, email) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, doctor.ID, doctor.FullName, doctor.Specialization, doctor.Email)
	if err != nil {
		return model.Doctor{}, fmt.Errorf("failed to insert doctor: %w", err)
	}
	return doctor, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	query := `SELECT id, full_name, specialization, email FROM doctors WHERE id = $1`
	var doctor model.Doctor
	err := r.db.QueryRowContext(ctx, query, id).Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email)
	if err == sql.ErrNoRows {
		return model.Doctor{}, model.ErrDoctorNotFound
	}
	if err != nil {
		return model.Doctor{}, fmt.Errorf("failed to get doctor: %w", err)
	}
	return doctor, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Doctor, error) {
	query := `SELECT id, full_name, specialization, email FROM doctors ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list doctors: %w", err)
	}
	defer rows.Close()

	var doctors []model.Doctor
	for rows.Next() {
		var doctor model.Doctor
		if err := rows.Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email); err != nil {
			return nil, fmt.Errorf("failed to scan doctor: %w", err)
		}
		doctors = append(doctors, doctor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning rows: %w", err)
	}
	return doctors, nil
}

func (r *PostgresRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM doctors WHERE email = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) NextID(ctx context.Context) string {
	return uuid.New().String()
}
