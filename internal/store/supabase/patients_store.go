package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
)

type PatientsStore struct{ db *Client }

func NewPatientsStore(db *Client) *PatientsStore { return &PatientsStore{db: db} }

// Insert(data, upsert, onConflict, returning, count)
func (s *PatientsStore) Create(ctx context.Context, token string, p *domain.Patient) (*domain.Patient, error) {
	pc := s.db.WithRLS(ctx, token)
	data, _, err := pc.From("patients").Insert(p, false, "", "representation", "").Execute()
	if err != nil {
		return nil, fmt.Errorf("insert patient: %w", err)
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil || len(out) == 0 {
		return nil, fmt.Errorf("decode patient")
	}
	return &out[0], nil
}

func (s *PatientsStore) GetByID(ctx context.Context, token string, id int) (*domain.Patient, error) {
	pc := s.db.WithRLS(ctx, token)
	data, _, err := pc.From("patients").
		Select("id,name,age,sex,mobile_number,created_at,updated_at,number_of_visits,doctor_id,user_id", "", true).
		Eq("id", strconv.Itoa(id)).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("get patient: %w", err)
	}
	var out domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode patient: %w", err)
	}
	return &out, nil
}

func (s *PatientsStore) SearchByMobile(
	ctx context.Context,
	token string,
	mobile string,
	limit, offset int,
) ([]domain.Patient, error) {
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
	q := pc.From("patients").
		Select("id,name,age,sex,mobile_number,created_at,updated_at,number_of_visits,doctor_id,user_id", "", true).
		Like("mobile_number", "%"+mobile+"%").
		Range(offset, offset+limit-1, "")

	data, _, err := q.Execute()
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode patients: %w", err)
	}
	return out, nil
}

// Update(data, returning, count)
func (s *PatientsStore) UpdatePartial(ctx context.Context, token string, id int, patch map[string]any) (*domain.Patient, error) {
	pc := s.db.WithRLS(ctx, token)
	data, _, err := pc.From("patients").
		Update(patch, "representation", "").
		Eq("id", strconv.Itoa(id)).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("update patient: %w", err)
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil || len(out) == 0 {
		return nil, fmt.Errorf("decode patient")
	}
	return &out[0], nil
}
