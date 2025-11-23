package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type DoctorsStore struct{ c *postgrest.Client }

func NewDoctorsStore(c *Client) *DoctorsStore { return &DoctorsStore{c: c.core()} }

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
	println("DEBUG [DoctorsStore]: GetByUserID called with userID:", userID)
	data, _, err := s.c.From("doctors").Select("*", "", false).Eq("user_id", fmt.Sprintf("%d", userID)).Limit(1, "").Execute()
	if err != nil {
		println("DEBUG [DoctorsStore]: Query error:", err.Error())
		return nil, err
	}
	println("DEBUG [DoctorsStore]: Query successful, data:", string(data))
	var out []domain.Doctor
	if err := json.Unmarshal(data, &out); err != nil {
		println("DEBUG [DoctorsStore]: Unmarshal error:", err.Error())
		return nil, err
	}
	if len(out) == 0 {
		println("DEBUG [DoctorsStore]: No doctor found for userID:", userID)
		return nil, fmt.Errorf("doctor profile not found")
	}
	println("DEBUG [DoctorsStore]: Doctor found, ID:", out[0].ID, "Designation:", out[0].Designation)
	return &out[0], nil
}
