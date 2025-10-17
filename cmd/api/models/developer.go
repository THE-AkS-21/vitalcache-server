package models

import "time"

type Developer struct {
	ID        uint      `json:"id,omitempty"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
