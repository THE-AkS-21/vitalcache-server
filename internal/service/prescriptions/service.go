package prescriptions

import (
	"context"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/kafka"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
)

type Service struct {
	patients      *supabase.PatientsStore
	prescriptions *supabase.PrescriptionsStore
	q             queue.Client
	kafka         *kafka.Producer
}

func NewService(ps *supabase.PatientsStore, prs *supabase.PrescriptionsStore, q queue.Client, k *kafka.Producer) *Service {
	return &Service{
		patients:      ps,
		prescriptions: prs,
		q:             q,
		kafka:         k,
	}
}

// Create – JSON-based create used by handlers.
// NOTE: Handler injects DoctorID from token; client cannot set it.
func (s *Service) Create(ctx context.Context, token string, req dto.CreatePrescriptionRequest) (domain.Prescription, error) {
	now := time.Now().UTC()
	p := &domain.Prescription{
		PatientID: req.PatientID,
		DoctorID:  req.DoctorID,
		BundleID:  req.BundleID,
		Notes:     req.Notes,
		FileURL:   req.FileURL,
		SentAt:    now,
		CreatedAt: now,
	}
	out, err := s.prescriptions.Create(ctx, token, p)
	if err != nil {
		return domain.Prescription{}, err
	}

	if s.kafka != nil {
		_ = s.kafka.Publish("prescription_created", out)
	}

	// Enqueue for email sending (stubbed)
	if req.FileURL != nil && *req.FileURL != "" {
		// Using FileURL as both path and name for now, or derive name from URL
		_ = s.q.EnqueuePrescription(ctx, uint(out.PatientID), *req.FileURL, "prescription.pdf")
	}

	return *out, nil
}

// ListByPatient returns prescriptions for a patient, optionally filtered by time range.
func (s *Service) ListByPatient(
	ctx context.Context,
	token string,
	patientID int,
	start, end *time.Time,
	limit, offset int,
) ([]domain.Prescription, error) {
	return s.prescriptions.ListByPatient(ctx, token, patientID, start, end, limit, offset)
}
