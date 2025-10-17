package store

import (
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type DeveloperStore struct {
	client *supa.Client
}

// Constructor for DeveloperStore
func NewDeveloperStore(client *supa.Client) *DeveloperStore {
	return &DeveloperStore{client: client}
}

// Create inserts a new developer profile into the 'developers' table
func (s *DeveloperStore) Create(dev models.Developer) (models.Developer, error) {
	// Insert with explicit parameters as per latest Supabase Go v2 API
	data, _, err := s.client.
		From("developers").
		Insert(dev, false, "representation", "", "public").
		Execute()
	if err != nil {
		return models.Developer{}, fmt.Errorf("failed to insert developer: %v", err)
	}

	var result []models.Developer
	if err := json.Unmarshal(data, &result); err != nil {
		return models.Developer{}, fmt.Errorf("failed to unmarshal developer insert response: %v", err)
	}
	if len(result) == 0 {
		return models.Developer{}, fmt.Errorf("no developer record returned after insert")
	}

	return result[0], nil
}

// GetByUserID retrieves a developer profile by its associated user_id
func (s *DeveloperStore) GetByUserID(userID uint) (*models.Developer, error) {
	data, _, err := s.client.
		From("developers").
		Select("*", "", true).
		Eq("user_id", fmt.Sprintf("%d", userID)).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch developer: %v", err)
	}

	var result []models.Developer
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal developer select response: %v", err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("developer profile not found for user_id: %d", userID)
	}

	return &result[0], nil
}
