package patients

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepo{db: db}
}

// ─────────────────────────────────────────────────────────────────────────────
// List
// ─────────────────────────────────────────────────────────────────────────────

func (r *pgxRepo) ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error) {
	const countQ = `
		SELECT COUNT(DISTINCT p.id)
		FROM patients p
		JOIN appointments a ON a.patient_id = p.id
		WHERE a.doctor_id = $1
		  AND p.deleted_at IS NULL`

	var total int64
	if err := r.db.QueryRow(ctx, countQ, doctorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count patients by doctor: %w", err)
	}

	const q = `
		SELECT DISTINCT p.id, p.user_id, u.first_name, u.last_name,
		       u.phone_number, u.gender, u.date_of_birth, p.created_at, p.updated_at
		FROM patients p
		JOIN users u ON p.user_id = u.id
		JOIN appointments a ON a.patient_id = p.id
		WHERE a.doctor_id = $1
		  AND p.deleted_at IS NULL
		ORDER BY u.first_name ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, doctorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list patients by doctor: %w", err)
	}
	defer rows.Close()

	patients, err := scanPatients(rows)
	return patients, total, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Search
// ─────────────────────────────────────────────────────────────────────────────

func (r *pgxRepo) SearchByPhone(ctx context.Context, doctorID, phone string, limit int) ([]Patient, error) {
	const q = `
		SELECT p.id, p.user_id, u.first_name, u.last_name,
		       u.phone_number, u.gender, u.date_of_birth, p.created_at, p.updated_at
		FROM patients p
		JOIN users u ON p.user_id = u.id
		WHERE u.phone_number ILIKE $1
		  AND p.deleted_at IS NULL
		LIMIT $2`

	// Escape special ILIKE characters to prevent CPU exhaustion from full-table scans
	safePhone := strings.ReplaceAll(phone, "%", "\\%")
	safePhone = strings.ReplaceAll(safePhone, "_", "\\_")

	rows, err := r.db.Query(ctx, q, "%"+safePhone+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	return scanPatients(rows)
}

// ─────────────────────────────────────────────────────────────────────────────
// Get by ID
// ─────────────────────────────────────────────────────────────────────────────

func (r *pgxRepo) GetByID(ctx context.Context, id string) (*Patient, error) {
	const q = `
		SELECT p.id, p.user_id, u.first_name, u.last_name,
		       u.phone_number, u.gender, u.date_of_birth, p.created_at, p.updated_at
		FROM patients p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	row := r.db.QueryRow(ctx, q, id)

	var p Patient
	err := scanPatientRow(row, &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get patient by id: %w", err)
	}
	return &p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Create (doctor-created patient — no auth user created)
// ─────────────────────────────────────────────────────────────────────────────

func (r *pgxRepo) Create(ctx context.Context, req CreatePatientReq, createdByDoctorID string) (*Patient, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Create a placeholder user row (no auth — no password_hash)
	const insertUser = `
		INSERT INTO users (email, password_hash, first_name, last_name, phone_number, gender, date_of_birth)
		VALUES ($1, '', $2, $3, $4, $5, $6)
		RETURNING id, first_name, last_name, phone_number, gender, date_of_birth`

	// Generate a synthetic email to satisfy the unique constraint
	syntheticEmail := "patient+" + uuid.New().String() + "@vitalcache.internal"

	var userID, firstName, lastName string
	var phone, gender, dob *string
	err = tx.QueryRow(ctx, insertUser,
		syntheticEmail, req.FirstName, req.LastName, req.PhoneNumber, req.Gender, req.DateOfBirth,
	).Scan(&userID, &firstName, &lastName, &phone, &gender, &dob)
	if err != nil {
		return nil, fmt.Errorf("create patient user: %w", err)
	}

	// 2. Assign PATIENT role
	var patientRoleID string
	if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE name = 'PATIENT'`).Scan(&patientRoleID); err != nil {
		return nil, fmt.Errorf("fetch patient role: %w", err)
	}
	var patientDesigID string
	if err := tx.QueryRow(ctx, `SELECT id FROM designations WHERE role_id = $1 LIMIT 1`, patientRoleID).Scan(&patientDesigID); err != nil {
		return nil, fmt.Errorf("fetch patient designation: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role_id, designation_id) VALUES ($1, $2, $3)`,
		userID, patientRoleID, patientDesigID); err != nil {
		return nil, fmt.Errorf("assign patient role: %w", err)
	}

	// 3. Create patient record
	const insertPatient = `
		INSERT INTO patients (user_id, created_by_doctor_id)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	var p Patient
	p.FirstName = firstName
	p.LastName = lastName
	p.FullName = firstName + " " + lastName
	p.PhoneNumber = phone
	p.Gender = gender
	p.DateOfBirth = dob

	// Compute age if DOB provided
	if dob != nil {
		if birth, err := time.Parse("2006-01-02", *dob); err == nil {
			age := int(time.Since(birth).Hours() / 8766)
			p.Age = &age
		}
	}

	err = tx.QueryRow(ctx, insertPatient, userID, createdByDoctorID).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert patient: %w", err)
	}
	p.UserID = userID

	return &p, tx.Commit(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────────────────────────────────────

func (r *pgxRepo) Update(ctx context.Context, id string, req UpdatePatientReq) (*Patient, error) {
	// Partial update of the users table (patients row has no demographic columns)
	const q = `
		UPDATE users u
		SET
			first_name   = COALESCE($1, u.first_name),
			last_name    = COALESCE($2, u.last_name),
			phone_number = COALESCE($3, u.phone_number),
			gender       = COALESCE($4, u.gender),
			date_of_birth = COALESCE($5, u.date_of_birth),
			updated_at   = NOW()
		FROM patients p
		WHERE p.user_id = u.id
		  AND p.id = $6
		  AND p.deleted_at IS NULL
		RETURNING p.id, p.user_id, u.first_name, u.last_name,
		          u.phone_number, u.gender, u.date_of_birth, p.created_at, NOW()`

	row := r.db.QueryRow(ctx, q,
		req.FirstName, req.LastName, req.PhoneNumber, req.Gender, req.DateOfBirth, id)

	var p Patient
	err := scanPatientRow(row, &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Scan helpers
// ─────────────────────────────────────────────────────────────────────────────

func scanPatients(rows pgx.Rows) ([]Patient, error) {
	var result []Patient
	for rows.Next() {
		var p Patient
		var dob *string
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.FirstName, &p.LastName,
			&p.PhoneNumber, &p.Gender, &dob, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.DateOfBirth = dob
		p.FullName = p.FirstName + " " + p.LastName
		if dob != nil {
			if birth, err := time.Parse("2006-01-02", *dob); err == nil {
				age := int(time.Since(birth).Hours() / 8766)
				p.Age = &age
			}
		}
		result = append(result, p)
	}
	if result == nil {
		result = []Patient{}
	}
	return result, rows.Err()
}

func scanPatientRow(row pgx.Row, p *Patient) error {
	var dob *string
	if err := row.Scan(
		&p.ID, &p.UserID, &p.FirstName, &p.LastName,
		&p.PhoneNumber, &p.Gender, &dob, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return err
	}
	p.DateOfBirth = dob
	p.FullName = p.FirstName + " " + p.LastName
	if dob != nil {
		if birth, err := time.Parse("2006-01-02", *dob); err == nil {
			age := int(time.Since(birth).Hours() / 8766)
			p.Age = &age
		}
	}
	return nil
}
