package domain

import "time"

type Developer struct {
	ID        uint
	Name      string
	Role      string
	UserID    uint
	CreatedAt time.Time
}
