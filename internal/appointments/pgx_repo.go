package appointments

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepo{db: db}
}

func (r *pgxRepo) Create(ctx context.Context, a *Appointment) error {
	query := `
		INSERT INTO appointments (patient_id, doctor_id, hospital_id, appointment_time, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, a.PatientID, a.DoctorID, a.HospitalID, a.AppointmentTime, a.Status).
		Scan(&a.ID, &a.CreatedAt)
}

func (r *pgxRepo) GetByID(ctx context.Context, id string) (*Appointment, error) {
	query := `
		SELECT id, patient_id, doctor_id, hospital_id, appointment_time, status, created_at
		FROM appointments WHERE id = $1
	`
	var a Appointment
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.PatientID, &a.DoctorID, &a.HospitalID, &a.AppointmentTime, &a.Status, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &a, err
}

func (r *pgxRepo) ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Appointment, int64, error) {
	const countQ = `SELECT COUNT(id) FROM appointments WHERE doctor_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQ, doctorID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, patient_id, doctor_id, hospital_id, appointment_time, status, created_at
		FROM appointments WHERE doctor_id = $1
		ORDER BY appointment_time DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, doctorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []Appointment

	for rows.Next() {
		var a Appointment
		if err := rows.Scan(
			&a.ID, &a.PatientID, &a.DoctorID, &a.HospitalID, &a.AppointmentTime, &a.Status, &a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, a)
	}
	if result == nil {
		result = []Appointment{}
	}
	return result, total, nil
}

func (r *pgxRepo) UpdateStatus(ctx context.Context, id string, status AppointmentStatus) error {
	query := `UPDATE appointments SET status = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}
