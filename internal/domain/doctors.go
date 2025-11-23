package domain

import "time"

type Doctor struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Name        string    `json:"name"`
	Designation string    `json:"designation"` // cardiologist, ophthalmologist, etc.
	CreatedAt   time.Time `json:"created_at"`
}
