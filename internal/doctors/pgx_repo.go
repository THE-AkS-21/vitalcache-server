// Package repo provides the pgx-backed implementation of doctors.Repository.
package doctors

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxDoctorRepo implements doctors.Repository against Supabase Postgres.
type PgxDoctorRepo struct {
	pool *pgxpool.Pool
}

// New returns a production-ready doctor repository.
func New(pool *pgxpool.Pool) *PgxDoctorRepo {
	return &PgxDoctorRepo{pool: pool}
}

// GetByID fetches a single doctor by primary key.
// Returns nil, nil if not found.
func (r *PgxDoctorRepo) GetByID(ctx context.Context, id int64) (*Doctor, error) {
	const q = `
		SELECT d.id, d.user_id, u.name, d.designation, d.specialisation,
		       d.registration_number, d.hospital_id, d.created_at
		FROM   doctors d
		JOIN   users   u ON u.id = d.user_id
		WHERE  d.id = $1
		LIMIT  1`

	row := r.pool.QueryRow(ctx, q, id)
	return scanDoctor(row)
}

// GetByUserID fetches a doctor profile by the linked users.id (used post-login).
func (r *PgxDoctorRepo) GetByUserID(ctx context.Context, userID int64) (*Doctor, error) {
	const q = `
		SELECT d.id, d.user_id, u.name, d.designation, d.specialisation,
		       d.registration_number, d.hospital_id, d.created_at
		FROM   doctors d
		JOIN   users   u ON u.id = d.user_id
		WHERE  d.user_id = $1
		LIMIT  1`

	row := r.pool.QueryRow(ctx, q, userID)
	return scanDoctor(row)
}

// List returns a paginated list of all doctors ordered by name.
func (r *PgxDoctorRepo) List(ctx context.Context, limit, offset int) ([]Doctor, int64, error) {
	const q = `
		SELECT d.id, d.user_id, u.name, d.designation, d.specialisation,
		       d.registration_number, d.hospital_id, d.created_at,
		       COUNT(*) OVER() AS total
		FROM   doctors d
		JOIN   users   u ON u.id = d.user_id
		ORDER  BY u.name
		LIMIT  $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list doctors: %w", err)
	}
	defer rows.Close()

	var (
		result []Doctor
		total  int64
	)
	for rows.Next() {
		var d Doctor
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Name, &d.Designation, &d.Specialisation,
			&d.RegistrationNumber, &d.HospitalID, &d.CreatedAt, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan doctor row: %w", err)
		}
		result = append(result, d)
	}
	return result, total, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────

func scanDoctor(row pgx.Row) (*Doctor, error) {
	var d Doctor
	err := row.Scan(
		&d.ID, &d.UserID, &d.Name, &d.Designation, &d.Specialisation,
		&d.RegistrationNumber, &d.HospitalID, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan doctor: %w", err)
	}
	return &d, nil
}
