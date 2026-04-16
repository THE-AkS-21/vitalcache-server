package patients

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepo{db: db}
}

func (r *pgxRepo) ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error) {
	// A doctor's patients are distinct patients who have booked appointments with them
	query := `
		SELECT DISTINCT p.id, p.user_id, u.first_name, u.last_name, u.phone_number, u.gender, u.date_of_birth, p.created_at,
		       COUNT(*) OVER() AS total
		FROM patients p
		JOIN users u ON p.user_id = u.id
		JOIN appointments a ON a.patient_id = p.id
		WHERE a.doctor_id = $1
		ORDER BY u.first_name ASC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, doctorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []Patient
	var total int64

	for rows.Next() {
		var p Patient
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.FirstName, &p.LastName, &p.PhoneNumber, &p.Gender, &p.DateOfBirth, &p.CreatedAt, &total,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, p)
	}

	if result == nil {
		result = []Patient{}
	}
	return result, total, nil
}
