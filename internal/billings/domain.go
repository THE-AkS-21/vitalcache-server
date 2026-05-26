package billings

import (
	"context"
	"time"
)

type Billing struct {
	ID              string    `json:"id"`
	PatientID       string    `json:"patient_id"`
	PatientName     string    `json:"patient_name"`
	DoctorID        string    `json:"doctor_id"`
	DoctorName      string    `json:"doctor_name"`
	MedicalReportID string    `json:"medical_report_id,omitempty"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"` // PENDING, PAID, CANCELLED
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Analytics struct {
	TodayEarnings   float64 `json:"today_earnings"`
	TodayPatients   int     `json:"today_patients"`
	WeeklyEarnings  float64 `json:"weekly_earnings"`
	WeeklyPatients  int     `json:"weekly_patients"`
	MonthlyEarnings float64 `json:"monthly_earnings"`
	MonthlyPatients int     `json:"monthly_patients"`
	YearlyEarnings  float64 `json:"yearly_earnings"`
	YearlyPatients  int     `json:"yearly_patients"`
}

type DailyStat struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

type HeatmapData struct {
	Data []DailyStat `json:"data"`
}

type Repository interface {
	Create(ctx context.Context, b *Billing) error
	GetByID(ctx context.Context, id string) (*Billing, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	List(ctx context.Context, doctorID, patientID string, limit, offset int) ([]Billing, int64, error)
	GetAnalytics(ctx context.Context, doctorID string) (*Analytics, error)
	GetHeatmap(ctx context.Context, doctorID string) (*HeatmapData, error)
}
