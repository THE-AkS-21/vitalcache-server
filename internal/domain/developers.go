package domain

import (
	"encoding/json"
	"time"
)

type Developer struct {
	ID          uint            `json:"id"`
	UserID      uint            `json:"user_id"`
	Name        string          `json:"name"`
	Role        string          `json:"role"`        // godfather, senior, junior, intern
	Permissions json.RawMessage `json:"permissions"` // JSONB for dynamic Junior permissions
	CreatedAt   time.Time       `json:"created_at"`
}
