package domain

import "time"

type User struct {
	ID           uint
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}
