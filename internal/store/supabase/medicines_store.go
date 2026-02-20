package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type MedicinesStore struct{ db *Client }

func NewMedicinesStore(db *Client) *MedicinesStore { return &MedicinesStore{db: db} }

func (s *MedicinesStore) List(ctx context.Context, token string, limit, offset int) ([]domain.Medicine, error) {
	slog.Debug("fetching medicines list", "limit", limit, "offset", offset)
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

	data, _, err := q.Execute()
	if err != nil {
		slog.Error("failed to execute medicines query", "err", err)
		return nil, fmt.Errorf("medicines list: %w", err)
	}

	// json.Unmarshal handles empty arrays "[]" and nil slices automatically
	var out []domain.Medicine
	if err := json.Unmarshal(data, &out); err != nil {
		slog.Error("failed to decode medicines", "err", err)
		return nil, fmt.Errorf("decode medicines: %w", err)
	}

	slog.Debug("fetched medicines successfully", "count", len(out))
	return out, nil
}
