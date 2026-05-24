package prescriptions

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Medication struct {
	MedicineID   int    `bson:"medicine_id" json:"medicine_id"`
	Name         string `bson:"name" json:"name"`
	Dosage       string `bson:"dosage" json:"dosage"`
	Frequency    string `bson:"frequency" json:"frequency"`
	DurationDays int    `bson:"duration_days" json:"duration_days"`
	Instructions string `bson:"instructions" json:"instructions"`
}

type Prescription struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PrescriptionID string             `bson:"prescription_id" json:"prescription_id"`
	PatientID      string             `bson:"patient_id" json:"patient_id"`
	DoctorID       string             `bson:"doctor_id" json:"doctor_id"`
	HospitalID     string             `bson:"hospital_id" json:"hospital_id"`
	Medications    []Medication       `bson:"medications" json:"medications"`
	Notes          string             `bson:"notes" json:"notes"`
	Status         string             `bson:"status" json:"status"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

type CreatePrescriptionReq struct {
	PatientID   string       `json:"patient_id" validate:"required,uuid"`
	HospitalID  string       `json:"hospital_id" validate:"required,uuid"`
	Medications []Medication `json:"medications" validate:"required,min=1"`
	Notes       string       `json:"notes,omitempty"`
}

type Repository interface {
	Create(ctx context.Context, p *Prescription) error
	CreateWithOutbox(ctx context.Context, p *Prescription, jobBytes []byte) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*Prescription, error)
	ListByPatient(ctx context.Context, patientID string, limit, offset int) ([]Prescription, int64, error)
}

type Service interface {
	CreatePrescription(ctx context.Context, doctorID string, req CreatePrescriptionReq) (*Prescription, error)
	GetPrescription(ctx context.Context, id string) (*Prescription, error)
	GetPatientHistory(ctx context.Context, patientID string, limit, offset int) ([]Prescription, int64, error)
}
