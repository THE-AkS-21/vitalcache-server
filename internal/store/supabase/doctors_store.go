package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	supa "github.com/supabase-community/supabase-go"
)

type DoctorsStore struct{ c *supa.Client }

func NewDoctorsStore(c *supa.Client) *DoctorsStore { return &DoctorsStore{c: c} }

func (s *DoctorsStore) Create(ctx context.Context, d domain.Doctor) (domain.Doctor, error) {
	data, _, err := s.c.From("doctors").Insert(d, false, "representation", "", "public").Execute()
	if err != nil {
		return domain.Doctor{}, err
	}
	var out []domain.Doctor
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.Doctor{}, err
	}
	if len(out) == 0 {
		return domain.Doctor{}, fmt.Errorf("no doctor returned")
	}
	return out[0], nil
}

func (s *DoctorsStore) GetByUserID(userID uint) (*domain.Doctor, error) {
	data, _, err := s.c.From("doctors").Select("*", "", false).Eq("user_id", fmt.Sprintf("%d", userID)).Limit(1, "").Execute()
	if err != nil {
		return nil, err
	}
	var out []domain.Doctor
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("doctor profile not found")
	}
	return &out[0], nil
}
