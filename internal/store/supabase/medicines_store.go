package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type MedicinesStore struct{ db *Client }

func NewMedicinesStore(db *Client) *MedicinesStore { return &MedicinesStore{db: db} }

func (s *MedicinesStore) List(ctx context.Context, token string, limit, offset int) ([]domain.Medicine, error) {
	println("DEBUG [MedicinesStore]: List called with limit:", limit, "offset:", offset)
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	q := s.db.WithRLS(ctx, token).
		From("medicines").
		Select("id,name,dose,duration_days,frequency,recommended_brands,created_at", "", true).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Range(offset, offset+limit-1, "")

	println("DEBUG [MedicinesStore]: Executing query...")
	data, _, err := q.Execute()
	if err != nil {
		println("DEBUG [MedicinesStore]: Query error:", err.Error())
		return nil, fmt.Errorf("medicines list: %w", err)
	}
	println("DEBUG [MedicinesStore]: Query successful, data length:", len(data))

	// Handle empty result
	if len(data) == 0 || string(data) == "[]" || string(data) == "" {
		println("DEBUG [MedicinesStore]: Empty result, returning empty slice")
		return []domain.Medicine{}, nil
	}

	var out []domain.Medicine
	if err := json.Unmarshal(data, &out); err != nil {
		println("DEBUG [MedicinesStore]: Unmarshal error:", err.Error())
		return nil, fmt.Errorf("decode medicines: %w", err)
	}
	println("DEBUG [MedicinesStore]: Unmarshaled", len(out), "medicines")
	return out, nil
}
