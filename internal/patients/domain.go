package patients

import (
	"context"
	"time"
)

// Patient blends the 'patients' and 'users' tables for API responses.
type Patient struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	FullName    string    `json:"name"` // computed: first + last
	PhoneNumber *string   `json:"phone_number,omitempty"`
	Gender      *string   `json:"gender,omitempty"`
	DateOfBirth *string   `json:"date_of_birth,omitempty"`
	Age         *int      `json:"age,omitempty"` // computed from date_of_birth
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreatePatientReq is the request body for POST /patients.
// Creates both the auth user and the patient profile in a single transaction.
type CreatePatientReq struct {
	FirstName   string  `json:"first_name" validate:"required,min=1,max=100"`
	LastName    string  `json:"last_name"  validate:"required,min=1,max=100"`
	PhoneNumber string  `json:"phone_number" validate:"required,min=8,max=20"`
	Gender      *string `json:"gender,omitempty" validate:"omitempty,oneof=MALE FEMALE OTHER"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
	Age         *int    `json:"age,omitempty"`
}

// UpdatePatientReq is the request body for PATCH /patients/:id.
type UpdatePatientReq struct {
	FirstName   *string `json:"first_name,omitempty"  validate:"omitempty,min=1,max=100"`
	LastName    *string `json:"last_name,omitempty"   validate:"omitempty,min=1,max=100"`
	PhoneNumber *string `json:"phone_number,omitempty" validate:"omitempty,min=8,max=20"`
	Gender      *string `json:"gender,omitempty"      validate:"omitempty,oneof=MALE FEMALE OTHER"`
	DateOfBirth *string `json:"date_of_birth,omitempty"`
}

// Repository is the persistence contract for the patients module.
type Repository interface {
	// ListByDoctor returns patients who have had at least one appointment with doctorID.
	ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error)
	// Search finds patients by mobile number (doctor-scoped via appointments).
	SearchByPhone(ctx context.Context, doctorID, phone string, limit int) ([]Patient, error)
	// GetByID fetches a single patient by UUID.
	GetByID(ctx context.Context, id string) (*Patient, error)
	// Create inserts a new patient record (no auth user — doctor-created record).
	Create(ctx context.Context, req CreatePatientReq, createdByDoctorID string) (*Patient, error)
	// Update patches a patient's demographic fields.
	Update(ctx context.Context, id string, req UpdatePatientReq) (*Patient, error)
}

// Service is the use-case contract for the patients module.
type Service interface {
	GetPatients(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error)
	SearchPatients(ctx context.Context, doctorID, phone string) ([]Patient, error)
	GetPatient(ctx context.Context, id string) (*Patient, error)
	CreatePatient(ctx context.Context, req CreatePatientReq, doctorID string) (*Patient, error)
	UpdatePatient(ctx context.Context, id string, req UpdatePatientReq) (*Patient, error)
}
