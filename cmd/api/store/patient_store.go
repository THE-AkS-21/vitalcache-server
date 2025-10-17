package store

import (
	"fmt"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type PatientStore struct {
	client *supa.Client
}

func NewPatientStore(client *supa.Client) *PatientStore {
	return &PatientStore{client: client}
}

func (s *PatientStore) Create(patient models.Patient) (models.Patient, error) {
	var result []models.Patient
	err := s.client.From("patients").Insert(patient).Execute(&result)
	if err != nil {
		return models.Patient{}, err
	}
	if len(result) == 0 {
		return models.Patient{}, fmt.Errorf("failed to create patient, no result returned")
	}
	return result[0], nil
}

func (s *PatientStore) GetByMobile(mobileNumber string, doctorID uint) ([]models.Patient, error) {
	var result []models.Patient
	// Important: We also filter by doctor_id to respect RLS policies
	err := s.client.From("patients").
		Select("*").
		Eq("mobile_number", mobileNumber).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// New Function: Get a single patient by their ID
func (s *PatientStore) GetByID(id uint, doctorID uint) (*models.Patient, error) {
	var result []models.Patient
	err := s.client.From("patients").
		Select("*").
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Limit(1).
		Execute(&result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("patient not found or access denied")
	}
	return &result[0], nil
}

// New Function: Update a patient's details
func (s *PatientStore) Update(id uint, doctorID uint, updates map[string]interface{}) (models.Patient, error) {
	var result []models.Patient

	// Ensure the updated_at field is always set
	updates["updated_at"] = time.Now()

	err := s.client.From("patients").
		Update(updates).
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute(&result)

	if err != nil {
		return models.Patient{}, err
	}
	if len(result) == 0 {
		return models.Patient{}, fmt.Errorf("patient not found or access denied")
	}
	return result[0], nil
}
