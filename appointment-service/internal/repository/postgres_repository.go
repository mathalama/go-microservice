package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"appointment-service/internal/model"
	"github.com/google/uuid"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, app model.Appointment) (model.Appointment, error) {
	query := `INSERT INTO appointments (id, title, description, doctor_id, status, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, app.ID, app.Title, app.Description, app.DoctorID, string(app.Status), app.CreatedAt, app.UpdatedAt)
	if err != nil {
		return model.Appointment{}, fmt.Errorf("failed to insert appointment: %w", err)
	}
	return app, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	query := `SELECT id, title, description, doctor_id, status, created_at, updated_at FROM appointments WHERE id = $1`
	var app model.Appointment
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&app.ID, &app.Title, &app.Description, &app.DoctorID, &status, &app.CreatedAt, &app.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.Appointment{}, model.ErrAppointmentNotFound
	}
	if err != nil {
		return model.Appointment{}, fmt.Errorf("failed to get appointment: %w", err)
	}
	app.Status = model.Status(status)
	return app, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]model.Appointment, error) {
	query := `SELECT id, title, description, doctor_id, status, created_at, updated_at FROM appointments ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list appointments: %w", err)
	}
	defer rows.Close()

	var apps []model.Appointment
	for rows.Next() {
		var app model.Appointment
		var status string
		if err := rows.Scan(&app.ID, &app.Title, &app.Description, &app.DoctorID, &status, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan appointment: %w", err)
		}
		app.Status = model.Status(status)
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning rows: %w", err)
	}
	return apps, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, status model.Status, updatedAt time.Time) (model.Appointment, model.Status, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Appointment{}, "", err
	}
	defer tx.Rollback()

	// 1. Fetch current record with lock to ensure atomicity
	var currentStatusStr string
	err = tx.QueryRowContext(ctx, "SELECT status FROM appointments WHERE id = $1 FOR UPDATE", id).Scan(&currentStatusStr)
	if err == sql.ErrNoRows {
		return model.Appointment{}, "", model.ErrAppointmentNotFound
	}
	if err != nil {
		return model.Appointment{}, "", fmt.Errorf("failed to lock appointment: %w", err)
	}

	oldStatus := model.Status(currentStatusStr)

	// Business logic: enforce transition rules within the transaction
	if oldStatus == model.StatusDone && status == model.StatusNew {
		return model.Appointment{}, "", model.ErrForbiddenStatusTransit
	}

	// 2. Perform the update
	query := `UPDATE appointments SET status = $1, updated_at = $2 WHERE id = $3 
	          RETURNING id, title, description, doctor_id, status, created_at, updated_at`
	var app model.Appointment
	var statusStr string
	err = tx.QueryRowContext(ctx, query, string(status), updatedAt, id).Scan(&app.ID, &app.Title, &app.Description, &app.DoctorID, &statusStr, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return model.Appointment{}, "", fmt.Errorf("failed to update appointment status: %w", err)
	}
	app.Status = model.Status(statusStr)

	if err := tx.Commit(); err != nil {
		return model.Appointment{}, "", err
	}

	return app, oldStatus, nil
}

func (r *PostgresRepository) NextID(ctx context.Context) string {
	return uuid.New().String()
}
