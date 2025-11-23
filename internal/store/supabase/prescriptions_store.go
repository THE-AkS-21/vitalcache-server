package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type PrescriptionsStore struct{ db *Client }

func NewPrescriptionsStore(db *Client) *PrescriptionsStore { return &PrescriptionsStore{db: db} }

func (s *PrescriptionsStore) Create(ctx context.Context, token string, p *domain.Prescription) (*domain.Prescription, error) {
	pc := s.db.WithRLS(ctx, token)
	data, _, err := pc.From("prescriptions").Insert(p, false, "", "representation", "").Execute()
	if err != nil {
		return nil, fmt.Errorf("insert prescription: %w", err)
	}
	var out []domain.Prescription
	if err := json.Unmarshal(data, &out); err != nil || len(out) == 0 {
		return nil, fmt.Errorf("decode prescription")
	}
	return &out[0], nil
}

func (s *PrescriptionsStore) ListByPatient(ctx context.Context, token string, patientID int, start, end *time.Time, limit, offset int) ([]domain.Prescription, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	pc := s.db.WithRLS(ctx, token)
	q := pc.From("prescriptions").
		Select("id,patient_id,doctor_id,bundle_id,notes,file_url,sent_at,created_at", "", true).
		Eq("patient_id", strconv.Itoa(patientID)).
		Order("sent_at", &postgrest.OrderOpts{Ascending: false}).
		Range(offset, offset+limit-1, "")

	if start != nil {
		q = q.Gte("sent_at", start.UTC().Format(time.RFC3339))
	}
	if end != nil {
		q = q.Lte("sent_at", end.UTC().Format(time.RFC3339))
	}

	data, _, err := q.Execute()
	if err != nil {
		return nil, fmt.Errorf("list prescriptions: %w", err)
	}
	var out []domain.Prescription
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode prescriptions: %w", err)
	}
	return out, nil
}
