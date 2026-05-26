// Package doctors defines the VitalCache doctor domain.
package doctors

import (
	"context"
	"time"
)

// Doctor is the canonical representation of a doctor profile in PostgreSQL.
type Doctor struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Name               string    `json:"name"`
	Designation        string    `json:"designation"`
	Specialisation     string    `json:"specialisation"`
	RegistrationNumber *string   `json:"registration_number,omitempty"`
	HospitalID         *string   `json:"hospital_id,omitempty"`
	ConsultationFee    float64   `json:"consultation_fee"`
	CreatedAt          time.Time `json:"created_at"`
}

// Repository is the persistence contract for the doctors module.
type Repository interface {
	GetByID(ctx context.Context, id string) (*Doctor, error)
	GetByUserID(ctx context.Context, userID string) (*Doctor, error)
	List(ctx context.Context, limit, offset int) ([]Doctor, int64, error)
	UpdateFee(ctx context.Context, id string, fee float64) error
}
