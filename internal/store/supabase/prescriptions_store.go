package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/supabase-community/postgrest-go"
	supa "github.com/supabase-community/supabase-go"
)

type PrescriptionsStore struct{ c *supa.Client }

func NewPrescriptionsStore(c *supa.Client) *PrescriptionsStore { return &PrescriptionsStore{c: c} }

// Create inserts a prescription row and returns the stored record.
func (s *PrescriptionsStore) Create(ctx context.Context, p domain.Prescription) (domain.Prescription, error) {
	if p.SentAt.IsZero() {
		p.SentAt = time.Now().UTC()
	}
	data, _, err := s.c.From("prescriptions").Insert(p, false, "representation", "", "public").Execute()
	if err != nil {
		return domain.Prescription{}, err
	}
	var out []domain.Prescription
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.Prescription{}, err
	}
	if len(out) == 0 {
		return domain.Prescription{}, fmt.Errorf("no prescription returned")
	}
	return out[0], nil
}

// ListByPatientVisibleToDoctor returns prescriptions for a patient visible to the doctor.
// Use link-table patient_doctors if available; otherwise fallback to ownership on patients.
func (s *PrescriptionsStore) ListByPatientVisibleToDoctorRange(
	ctx context.Context,
	patientID, doctorID uint, // doctorID unused here; RLS handles visibility
	startRFC3339, endRFC3339 string, // optional
	limit, offset int,
) ([]domain.Prescription, error) {

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := s.c.From("prescriptions").
		Select("*", "", true).
		Eq("patient_id", fmt.Sprintf("%d", patientID)).
		Order("sent_at", &postgrest.OrderOpts{Ascending: false})

	if startRFC3339 != "" {
		q = q.Gte("sent_at", startRFC3339)
	}
	if endRFC3339 != "" {
		q = q.Lte("sent_at", endRFC3339)
	}

	data, _, err := q.Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, err
	}

	var out []domain.Prescription
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
