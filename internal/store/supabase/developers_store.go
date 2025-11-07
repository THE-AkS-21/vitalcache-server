package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	supa "github.com/supabase-community/supabase-go"
)

type DevelopersStore struct{ c *supa.Client }

func NewDevelopersStore(c *supa.Client) *DevelopersStore { return &DevelopersStore{c: c} }

func (s *DevelopersStore) Create(ctx context.Context, d domain.Developer) (domain.Developer, error) {
	data, _, err := s.c.From("developers").Insert(d, false, "representation", "", "public").Execute()
	if err != nil {
		return domain.Developer{}, err
	}
	var out []domain.Developer
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.Developer{}, err
	}
	if len(out) == 0 {
		return domain.Developer{}, fmt.Errorf("no developer returned")
	}
	return out[0], nil
}

func (s *DevelopersStore) GetByUserID(userID uint) (*domain.Developer, error) {
	data, _, err := s.c.From("developers").Select("*", "", false).Eq("user_id", fmt.Sprintf("%d", userID)).Limit(1, "").Execute()
	if err != nil {
		return nil, err
	}
	var out []domain.Developer
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("developer profile not found")
	}
	return &out[0], nil
}
