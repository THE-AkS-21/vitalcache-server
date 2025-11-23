package domain

import "time"

// HospitalStaff represents a staff member (admin, receptionist, pharmacist, clerk)
type HospitalStaff struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	Name        string    `json:"name"`
	Designation string    `json:"designation"` // admin, receptionist, pharmacist, clerk
	CreatedAt   time.Time `json:"created_at"`
}
