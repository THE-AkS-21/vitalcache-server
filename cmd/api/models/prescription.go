package models

import "time"

type Prescription struct {
	ID        uint      `json:"id,omitempty"`
	PatientID uint      `json:"patient_id"`
	DoctorID  uint      `json:"doctor_id"`
	FileUrl   string    `json:"file_url"` // URL of the prescription in Supabase Storage
	SentAt    time.Time `json:"sent_at"`
}
