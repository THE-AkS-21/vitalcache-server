package store

import (
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

func (s *DoctorStore) Create(doctor models.Doctor) (models.Doctor, error) {
	var result []models.Doctor
	err := s.client.From("doctors").Insert(doctor).Execute(&result)
	if err != nil {
		return models.Doctor{}, err
	}
	if len(result) == 0 {
		return models.Doctor{}, fmt.Errorf("failed to create doctor, no result returned")
	}
	return result[0], nil
}

func (s *DoctorStore) GetByEmail(email string) (*models.Doctor, error) {
	var result []models.Doctor
	err := s.client.From("doctors").
		Select("*").
		Eq("email", email).
		Limit(1).
		Execute(&result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("doctor with email %s not found", email)
	}
	return &result[0], nil
}

// New Function: Get a single doctor by their ID
func (s *DoctorStore) GetByID(id uint) (*models.Doctor, error) {
	var result []models.Doctor
	err := s.client.From("doctors").
		Select("*").
		Eq("id", fmt.Sprintf("%d", id)).
		Limit(1).
		Execute(&result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("doctor with id %d not found", id)
	}
	return &result[0], nil
}
