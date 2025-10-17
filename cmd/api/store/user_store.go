package store

import (
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	supa "github.com/supabase-community/supabase-go"
)

type UserStore struct {
	client *supa.Client
}

func NewUserStore(client *supa.Client) *UserStore {
	return &UserStore{client: client}
}

// Create a new user in the central users table
func (s *UserStore) Create(user models.User) (models.User, error) {
	data, _, err := s.client.From("users").
		Insert(user, false, "representation", "", "public").
		Execute()
	if err != nil {
		return models.User{}, err
	}

	var result []models.User
	if err := json.Unmarshal(data, &result); err != nil {
		return models.User{}, err
	}

	if len(result) == 0 {
		return models.User{}, fmt.Errorf("failed to create user, no result returned")
	}

	return result[0], nil
}

// GetByEmail fetches a user by their email for login
func (s *UserStore) GetByEmail(email string) (*models.User, error) {
	data, _, err := s.client.From("users").
		Select("*", "", true).
		Eq("email", email).
		Limit(1, "").
		Execute()
	if err != nil {
		return nil, err
	}

	var result []models.User
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("user with email %s not found", email)
	}

	return &result[0], nil
}
