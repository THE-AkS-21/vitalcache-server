package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	supa "github.com/supabase-community/supabase-go"
)

type PatientsStore struct{ c *supa.Client }

func NewPatientsStore(c *supa.Client) *PatientsStore { return &PatientsStore{c: c} }

func (s *PatientsStore) Create(ctx context.Context, p domain.Patient) (domain.Patient, error) {
	data, _, err := s.c.From("patients").Insert(p, false, "representation", "", "public").Execute()
	if err != nil {
		return domain.Patient{}, err
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.Patient{}, err
	}
	if len(out) == 0 {
		return domain.Patient{}, fmt.Errorf("no patient returned")
	}
	return out[0], nil
}

func (s *PatientsStore) GetByMobile(ctx context.Context, mobile string, doctorID uint) ([]domain.Patient, error) {
	data, _, err := s.c.From("patients").Select("*", "", true).
		Eq("mobile_number", mobile).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return nil, err
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PatientsStore) GetByID(ctx context.Context, id, doctorID uint) (*domain.Patient, error) {
	data, _, err := s.c.From("patients").Select("*", "", true).
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return nil, err
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("patient not found or access denied")
	}
	return &out[0], nil
}

func (s *PatientsStore) Update(ctx context.Context, id, doctorID uint, updates map[string]any) (domain.Patient, error) {
	data, _, err := s.c.From("patients").Update(updates, "representation", "public").
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return domain.Patient{}, err
	}
	var out []domain.Patient
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.Patient{}, err
	}
	if len(out) == 0 {
		return domain.Patient{}, fmt.Errorf("patient not found or access denied")
	}
	return out[0], nil
}
