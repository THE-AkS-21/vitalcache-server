package models

import "time"

type Doctor struct {
	ID          uint      `json:"id,omitempty"`
	Name        string    `json:"name"`
	Designation string    `json:"designation"`
	UserID      uint      `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}
