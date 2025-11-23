package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type DevelopersStore struct{ c *postgrest.Client }

func NewDevelopersStore(c *Client) *DevelopersStore {
	return &DevelopersStore{c: c.core()}
}

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
	println("DEBUG [DevelopersStore]: GetByUserID called with userID:", userID)
	data, _, err := s.c.From("developers").Select("*", "", false).Eq("user_id", fmt.Sprintf("%d", userID)).Limit(1, "").Execute()
	if err != nil {
		println("DEBUG [DevelopersStore]: Query error:", err.Error())
		return nil, err
	}
	println("DEBUG [DevelopersStore]: Query successful, data:", string(data))
	var out []domain.Developer
	if err := json.Unmarshal(data, &out); err != nil {
		println("DEBUG [DevelopersStore]: Unmarshal error:", err.Error())
		return nil, err
	}
	if len(out) == 0 {
		println("DEBUG [DevelopersStore]: No developer found for userID:", userID)
		return nil, fmt.Errorf("developer profile not found")
	}
	println("DEBUG [DevelopersStore]: Developer found, ID:", out[0].ID, "Role:", out[0].Role)
	return &out[0], nil
}
