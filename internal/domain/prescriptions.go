package domain

import "time"

type Prescription struct {
	ID        uint
	PatientID uint
	DoctorID  uint
	FileURL   string
	SentAt    time.Time
}
