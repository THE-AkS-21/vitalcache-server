package patients

import (
	"context"
	"time"
)

// Patient blends the 'patients' and 'users' table for the frontend
type Patient struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	PhoneNumber *string   `json:"phone_number,omitempty"`
	Gender      *string   `json:"gender,omitempty"`
	DateOfBirth *string   `json:"date_of_birth,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Repository interface {
	ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error)
}

type Service interface {
	GetPatients(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error)
}
