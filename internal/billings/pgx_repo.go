package billings

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxRepo struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepo{db: db}
}

func (r *pgxRepo) Create(ctx context.Context, b *Billing) error {
	query := `
		INSERT INTO billings (patient_id, doctor_id, medical_report_id, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, b.PatientID, b.DoctorID, b.MedicalReportID, b.Amount, b.Status).
		Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	return err
}

func (r *pgxRepo) GetByID(ctx context.Context, id string) (*Billing, error) {
	query := `
		SELECT b.id, b.patient_id, b.doctor_id, b.medical_report_id, b.amount, b.status, b.created_at, b.updated_at,
		       up.first_name || ' ' || up.last_name as patient_name,
		       ud.first_name || ' ' || ud.last_name as doctor_name
		FROM billings b
		JOIN users up ON b.patient_id = (SELECT user_id FROM patients WHERE id = b.patient_id)
		JOIN users ud ON b.doctor_id = (SELECT user_id FROM doctors WHERE id = b.doctor_id)
		WHERE b.id = $1
	`
	var b Billing
	err := r.db.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.PatientID, &b.DoctorID, &b.MedicalReportID, &b.Amount, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		&b.PatientName, &b.DoctorName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &b, err
}

func (r *pgxRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE billings SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

func (r *pgxRepo) List(ctx context.Context, doctorID, patientID string, limit, offset int) ([]Billing, int64, error) {
	query := `
		SELECT b.id, b.patient_id, b.doctor_id, b.medical_report_id, b.amount, b.status, b.created_at, b.updated_at,
		       up.first_name || ' ' || up.last_name as patient_name,
		       ud.first_name || ' ' || ud.last_name as doctor_name,
		       COUNT(*) OVER() AS total
		FROM billings b
		LEFT JOIN patients p ON b.patient_id = p.id
		LEFT JOIN users up ON p.user_id = up.id
		LEFT JOIN doctors d ON b.doctor_id = d.id
		LEFT JOIN users ud ON d.user_id = ud.id
		WHERE ($1::uuid IS NULL OR b.doctor_id = $1::uuid)
		  AND ($2::uuid IS NULL OR b.patient_id = $2::uuid)
		ORDER BY b.created_at DESC
		LIMIT $3 OFFSET $4
	`

	var docIDParam, patIDParam interface{}
	if doctorID != "" {
		docIDParam = doctorID
	}
	if patientID != "" {
		patIDParam = patientID
	}

	rows, err := r.db.Query(ctx, query, docIDParam, patIDParam, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var billings []Billing
	var total int64
	for rows.Next() {
		var b Billing
		if err := rows.Scan(
			&b.ID, &b.PatientID, &b.DoctorID, &b.MedicalReportID, &b.Amount, &b.Status, &b.CreatedAt, &b.UpdatedAt,
			&b.PatientName, &b.DoctorName, &total,
		); err != nil {
			return nil, 0, err
		}
		billings = append(billings, b)
	}
	return billings, total, nil
}

func (r *pgxRepo) GetAnalytics(ctx context.Context, doctorID string) (*Analytics, error) {
	// PENDING and PAID are both considered in patient counts, but maybe only PAID in earnings.
	// We'll just do a simple aggregation.
	query := `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE DATE(created_at) = CURRENT_DATE AND status = 'PAID'), 0) as today_earnings,
			COUNT(*) FILTER (WHERE DATE(created_at) = CURRENT_DATE) as today_patients,
			
			COALESCE(SUM(amount) FILTER (WHERE created_at >= date_trunc('week', CURRENT_DATE) AND status = 'PAID'), 0) as weekly_earnings,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('week', CURRENT_DATE)) as weekly_patients,
			
			COALESCE(SUM(amount) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE) AND status = 'PAID'), 0) as monthly_earnings,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE)) as monthly_patients,
			
			COALESCE(SUM(amount) FILTER (WHERE created_at >= date_trunc('year', CURRENT_DATE) AND status = 'PAID'), 0) as yearly_earnings,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('year', CURRENT_DATE)) as yearly_patients
		FROM billings
		WHERE doctor_id = $1
	`

	var a Analytics
	err := r.db.QueryRow(ctx, query, doctorID).Scan(
		&a.TodayEarnings, &a.TodayPatients,
		&a.WeeklyEarnings, &a.WeeklyPatients,
		&a.MonthlyEarnings, &a.MonthlyPatients,
		&a.YearlyEarnings, &a.YearlyPatients,
	)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *pgxRepo) GetHeatmap(ctx context.Context, doctorID string) (*HeatmapData, error) {
	query := `
		SELECT TO_CHAR(DATE(created_at), 'YYYY-MM-DD') as date, COUNT(*) as count
		FROM billings
		WHERE doctor_id = $1 AND created_at >= CURRENT_DATE - INTERVAL '365 days'
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`
	rows, err := r.db.Query(ctx, query, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []DailyStat
	for rows.Next() {
		var d DailyStat
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	return &HeatmapData{Data: data}, nil
}
