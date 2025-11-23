// internal/domain/prescriptions.go
package domain

import "time"

type Prescription struct {
	ID        int64     `json:"id,omitempty"`
	PatientID int       `json:"patient_id"`
	DoctorID  int       `json:"doctor_id"`
	BundleID  *int      `json:"bundle_id,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	FileURL   *string   `json:"file_url,omitempty"`
	SentAt    time.Time `json:"sent_at"`
	CreatedAt time.Time `json:"created_at"`
}
