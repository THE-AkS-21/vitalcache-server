package store

import (
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type MedicineStore struct {
	client *supa.Client
}

func NewMedicineStore(client *supa.Client) *MedicineStore {
	return &MedicineStore{client: client}
}

// New Function: Get all medicines for dropdowns
func (s *MedicineStore) GetAll() ([]models.Medicine, error) {
	var result []models.Medicine
	err := s.client.From("medicines").Select("*").Execute(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
