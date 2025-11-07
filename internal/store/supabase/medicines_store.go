package supabase

import (
	"context"
	"encoding/json"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go" // ← add this
	supa "github.com/supabase-community/supabase-go"
)

type MedicinesStore struct{ c *supa.Client }

func NewMedicinesStore(c *supa.Client) *MedicinesStore { return &MedicinesStore{c: c} }

func (s *MedicinesStore) List(ctx context.Context, limit, offset int) ([]domain.Medicine, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	data, _, err := s.c.From("medicines").
		Select("id,name,dose,duration_days,frequency,recommended_brands,created_at", "", true).
		Order("id", &postgrest.OrderOpts{Ascending: true}). // ← change here
		Range(offset, offset+limit-1, "").
		Execute()
	if err != nil {
		return nil, err
	}

	var out []domain.Medicine
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
