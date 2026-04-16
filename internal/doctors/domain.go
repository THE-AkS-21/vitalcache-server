// Package doctors defines the VitalCache doctor domain.
package doctors

import (
	"context"
	"time"
)

// Doctor is the canonical representation of a doctor profile in PostgreSQL.
type Doctor struct {
	ID                 int64     `json:"id"`
	UserID             int64     `json:"user_id"`
	Name               string    `json:"name"`
	Designation        string    `json:"designation"`
	Specialisation     string    `json:"specialisation"`
	RegistrationNumber *string   `json:"registration_number,omitempty"`
	HospitalID         *int64    `json:"hospital_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Repository is the persistence contract for the doctors module.
type Repository interface {
	GetByID(ctx context.Context, id int64) (*Doctor, error)
	GetByUserID(ctx context.Context, userID int64) (*Doctor, error)
	List(ctx context.Context, limit, offset int) ([]Doctor, int64, error)
}
