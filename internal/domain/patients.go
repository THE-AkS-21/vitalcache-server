package domain

import "time"

type Patient struct {
	ID             uint
	Name           string
	Age            int
	Sex            string
	MobileNumber   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	NumberOfVisits int
	DoctorID       uint
	UserID         *uint
}
