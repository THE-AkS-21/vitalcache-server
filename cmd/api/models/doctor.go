package models

import "time"

type Doctor struct {
	ID           uint      `json:"id,omitempty"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Omit from all JSON responses for security
	Designation  string    `json:"designation"`
	CreatedAt    time.Time `json:"created_at"`
}
