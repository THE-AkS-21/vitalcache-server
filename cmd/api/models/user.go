package models

import "time"

// User represents the central authentication entity.
type User struct {
	ID           uint      `json:"id,omitempty"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}
