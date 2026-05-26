package medical_reports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Medication struct {
	MedicineID   string `bson:"medicine_id" json:"medicine_id"`
	Name         string `bson:"name" json:"name"`
	Dosage       string `bson:"dosage" json:"dosage"`
	Frequency    string `bson:"frequency" json:"frequency"`
	DurationDays int    `bson:"duration_days" json:"duration_days"`
	Instructions string `bson:"instructions" json:"instructions"`
}

type MedicalReport struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ReportID      string             `bson:"report_id" json:"report_id"`
	PatientID     string             `bson:"patient_id" json:"patient_id"`
	DoctorID      string             `bson:"doctor_id" json:"doctor_id"`
	HospitalID    string             `bson:"hospital_id" json:"hospital_id"`
	DiseaseName   string             `bson:"disease_name" json:"disease_name"`
	DiagnosisBody string             `bson:"diagnosis_body" json:"diagnosis_body"`
	Medications   []Medication       `bson:"medications" json:"medications"`
	Precautions   string             `bson:"precautions" json:"precautions"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

type CreateReportReq struct {
	PatientID     string       `json:"patient_id" validate:"required,uuid"`
	HospitalID    string       `json:"hospital_id" validate:"required,uuid"`
	DiseaseName   string       `json:"disease_name" validate:"required"`
	DiagnosisBody string       `json:"diagnosis_body"`
	Medications   []Medication `json:"medications"`
	Precautions   string       `json:"precautions"`
}

type Repository interface {
	Create(ctx context.Context, p *MedicalReport) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*MedicalReport, error)
	ListByPatient(ctx context.Context, patientID string, limit, offset int) ([]MedicalReport, int64, error)
}

type Service interface {
	CreateReport(ctx context.Context, doctorID string, req CreateReportReq) (*MedicalReport, error)
	GetReport(ctx context.Context, id string) (*MedicalReport, error)
	GetPatientHistory(ctx context.Context, patientID string, limit, offset int) ([]MedicalReport, int64, error)
}
