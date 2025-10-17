package models

import "time"

type Patient struct {
	ID             uint      `json:"id,omitempty"`
	Name           string    `json:"name"`
	Age            int       `json:"age"`
	Sex            string    `json:"sex"`
	MobileNumber   string    `json:"mobile_number"`
	Email          string    `json:"email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	NumberOfVisits int       `json:"number_of_visits"`
	DoctorID       uint      `json:"doctor_id"`
}
