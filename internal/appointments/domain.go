package appointments

import (
	"context"
	"time"
)

type AppointmentStatus string

const (
	StatusBooked    AppointmentStatus = "BOOKED"
	StatusCompleted AppointmentStatus = "COMPLETED"
	StatusCancelled AppointmentStatus = "CANCELLED"
)

type Appointment struct {
	ID              string            `json:"id"`
	PatientID       string            `json:"patient_id"`
	DoctorID        string            `json:"doctor_id"`
	HospitalID      string            `json:"hospital_id"`
	AppointmentTime time.Time         `json:"appointment_time"`
	Status          AppointmentStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
}

type CreateAppointmentReq struct {
	PatientID       string    `json:"patient_id" validate:"required,uuid"`
	HospitalID      string    `json:"hospital_id" validate:"required,uuid"`
	AppointmentTime time.Time `json:"appointment_time" validate:"required"`
}

type UpdateStatusReq struct {
	Status AppointmentStatus `json:"status" validate:"required,oneof=BOOKED COMPLETED CANCELLED"`
}

type Repository interface {
	Create(ctx context.Context, a *Appointment) error
	GetByID(ctx context.Context, id string) (*Appointment, error)
	ListByDoctor(ctx context.Context, doctorID string, limit, offset int) ([]Appointment, int64, error)
	UpdateStatus(ctx context.Context, id string, status AppointmentStatus) error
}

type Service interface {
	CreateAppointment(ctx context.Context, doctorID string, req CreateAppointmentReq) (*Appointment, error)
	GetDoctorAppointments(ctx context.Context, doctorID string, limit, offset int) ([]Appointment, int64, error)
	UpdateStatus(ctx context.Context, doctorID string, appointmentID string, req UpdateStatusReq) (*Appointment, error)
}
