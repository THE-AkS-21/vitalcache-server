package domain

import "time"

type Doctor struct {
	ID          uint
	Name        string
	Designation string
	UserID      uint
	CreatedAt   time.Time
}
