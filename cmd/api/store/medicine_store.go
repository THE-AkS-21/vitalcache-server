package store

import (
	"encoding/json"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type MedicineStore struct {
	client *supa.Client
}

func NewMedicineStore(client *supa.Client) *MedicineStore {
	return &MedicineStore{client: client}
}

// GetAll returns all medicines
func (s *MedicineStore) GetAll() ([]models.Medicine, error) {
	data, _, err := s.client.From("medicines").Select("*", "", true).Execute()
	if err != nil {
		return nil, err
	}

	var result []models.Medicine
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
