package supabase

import (
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type HospitalStaffStore struct{ c *postgrest.Client }

func NewHospitalStaffStore(c *Client) *HospitalStaffStore {
	return &HospitalStaffStore{c: c.core()}
}

// GetByUserID fetches a hospital staff profile by user_id from the "staff" table
func (s *HospitalStaffStore) GetByUserID(userID uint) (*domain.HospitalStaff, error) {
	println("DEBUG [HospitalStaffStore]: GetByUserID called with userID:", userID)
	data, _, err := s.c.From("staff").
		Select("*", "", false).
		Eq("user_id", fmt.Sprintf("%d", userID)).
		Limit(1, "").
		Execute()
	if err != nil {
		println("DEBUG [HospitalStaffStore]: Query error:", err.Error())
		return nil, err
	}
	println("DEBUG [HospitalStaffStore]: Query successful, data:", string(data))
	var out []domain.HospitalStaff
	if err := json.Unmarshal(data, &out); err != nil {
		println("DEBUG [HospitalStaffStore]: Unmarshal error:", err.Error())
		return nil, err
	}
	if len(out) == 0 {
		println("DEBUG [HospitalStaffStore]: No staff found for userID:", userID)
		return nil, fmt.Errorf("hospital staff profile not found")
	}
	println("DEBUG [HospitalStaffStore]: Staff found, ID:", out[0].ID, "Designation:", out[0].Designation)
	return &out[0], nil
}
