package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// --- Entities ---
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	FullName     string    `json:"full_name"`
	PhoneNumber  *string   `json:"phone_number,omitempty"`
	DateOfBirth  *string   `json:"date_of_birth,omitempty"` // YYYY-MM-DD
	Gender       *string   `json:"gender,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Profile struct {
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Designation string   `json:"designation"`
	Permissions []string `json:"permissions"`
	DoctorID    *string  `json:"doctor_id,omitempty"`
	PatientID   *string  `json:"patient_id,omitempty"`
}

// --- Interfaces ---
type UserRepository interface {
	CreateUserTransaction(ctx context.Context, u *User, roleName, designationName string) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}

type ProfileRepository interface {
	GetProfileAndPermissions(ctx context.Context, userID string) (*Profile, error)
}

// --- JWT Structs ---
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Designation string   `json:"designation"`
	Permissions []string `json:"permissions"`
	DoctorID    *string  `json:"doctor_id,omitempty"`
	PatientID   *string  `json:"patient_id,omitempty"`
	jwt.RegisteredClaims
}
