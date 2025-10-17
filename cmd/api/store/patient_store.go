package store

import (
	"encoding/json"
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

// Create a new patient
func (s *PatientStore) Create(patient models.Patient) (models.Patient, error) {
	data, _, err := s.client.From("patients").Insert(patient, false, "representation", "", "public").Execute()
	if err != nil {
		return models.Patient{}, err
	}

	var result []models.Patient
	if err := json.Unmarshal(data, &result); err != nil {
		return models.Patient{}, err
	}
	if len(result) == 0 {
		return models.Patient{}, fmt.Errorf("failed to create patient, no result returned")
	}
	return result[0], nil
}

// GetByMobile fetches patients by mobile number and doctor
func (s *PatientStore) GetByMobile(mobileNumber string, doctorID uint) ([]models.Patient, error) {
	data, _, err := s.client.From("patients").
		Select("*", "", true).
		Eq("mobile_number", mobileNumber).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return nil, err
	}

	var result []models.Patient
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetByID fetches a single patient by ID and doctor
func (s *PatientStore) GetByID(id uint, doctorID uint) (*models.Patient, error) {
	data, _, err := s.client.From("patients").
		Select("*", "", true).
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return nil, err
	}

	var result []models.Patient
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("patient not found or access denied")
	}

	return &result[0], nil
}

// Update updates a patient's details
func (s *PatientStore) Update(id uint, doctorID uint, updates map[string]interface{}) (models.Patient, error) {
	updates["updated_at"] = time.Now()

	data, _, err := s.client.From("patients").
		Update(updates, "representation", "public").
		Eq("id", fmt.Sprintf("%d", id)).
		Eq("doctor_id", fmt.Sprintf("%d", doctorID)).
		Execute()
	if err != nil {
		return models.Patient{}, err
	}

	var result []models.Patient
	if err := json.Unmarshal(data, &result); err != nil {
		return models.Patient{}, err
	}

	if len(result) == 0 {
		return models.Patient{}, fmt.Errorf("patient not found or access denied")
	}

	return result[0], nil
}
