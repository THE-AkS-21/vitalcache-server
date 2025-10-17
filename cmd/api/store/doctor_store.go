package store

import (
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type DoctorStore struct {
	client *supa.Client
}

func NewDoctorStore(client *supa.Client) *DoctorStore {
	return &DoctorStore{client: client}
}

// Create a new doctor
func (s *DoctorStore) Create(doctor models.Doctor) (models.Doctor, error) {
	data, _, err := s.client.From("doctors").Insert(doctor, false, "representation", "", "public").Execute()
	if err != nil {
		return models.Doctor{}, err
	}

	var result []models.Doctor
	if err := json.Unmarshal(data, &result); err != nil {
		return models.Doctor{}, err
	}

	if len(result) == 0 {
		return models.Doctor{}, fmt.Errorf("failed to create doctor, no result returned")
	}

	return result[0], nil
}

// GetByEmail fetches a doctor by email
func (s *DoctorStore) GetByEmail(email string) (*models.Doctor, error) {
	data, _, err := s.client.From("doctors").Select("*", "", true).Eq("email", email).Execute()
	if err != nil {
		return nil, err
	}

	var result []models.Doctor
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("doctor with email %s not found", email)
	}

	return &result[0], nil
}

// GetByID fetches a doctor by ID
func (s *DoctorStore) GetByUserID(userID uint) (*models.Doctor, error) {
	data, _, err := s.client.From("doctors").Select("*", "", false).Eq("user_id", fmt.Sprintf("%d", userID)).Limit(1, "").Execute()
	if err != nil {
		return nil, err
	}
	var result []models.Doctor
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("doctor profile not found for user_id: %d", userID)
	}
	return &result[0], nil
}
