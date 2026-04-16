package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgxRepo{db: db}
}

func (r *pgxRepo) CreateUserTransaction(ctx context.Context, u *User, roleName, designationName string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get Role ID
	var roleID string
	err = tx.QueryRow(ctx, "SELECT id FROM roles WHERE name = $1", roleName).Scan(&roleID)
	if err != nil {
		return errors.New("invalid role specified")
	}

	// 2. Get Designation ID (Fallback to a default if empty)
	var designationID string
	if designationName != "" {
		err = tx.QueryRow(ctx, "SELECT id FROM designations WHERE name = $1 AND role_id = $2", designationName, roleID).Scan(&designationID)
	} else {
		err = tx.QueryRow(ctx, "SELECT id FROM designations WHERE role_id = $1 LIMIT 1", roleID).Scan(&designationID)
	}
	if err != nil {
		return errors.New("invalid designation specified for role")
	}

	// 3. Insert User
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, phone_number, date_of_birth, gender)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, full_name, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.PhoneNumber, u.DateOfBirth, u.Gender).
		Scan(&u.ID, &u.FullName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return err
	}

	// 4. Map User Role
	_, err = tx.Exec(ctx, "INSERT INTO user_roles (user_id, role_id, designation_id) VALUES ($1, $2, $3)", u.ID, roleID, designationID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgxRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, full_name, phone_number, date_of_birth, gender, is_active 
		FROM users WHERE email = $1 AND is_active = true
	`
	var u User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.FullName, &u.PhoneNumber, &u.DateOfBirth, &u.Gender, &u.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (r *pgxRepo) GetByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, email, password_hash, first_name, last_name, full_name, is_active FROM users WHERE id = $1 AND is_active = true`
	var u User
	err := r.db.QueryRow(ctx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.FullName, &u.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}
