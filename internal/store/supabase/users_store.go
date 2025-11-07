package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	supa "github.com/supabase-community/supabase-go"
)

type UsersStore struct{ c *supa.Client }

func NewUsersStore(c *supa.Client) *UsersStore { return &UsersStore{c: c} }

func (s *UsersStore) Create(ctx context.Context, u domain.User) (domain.User, error) {
	data, _, err := s.c.From("users").Insert(u, false, "representation", "", "public").Execute()
	if err != nil {
		return domain.User{}, err
	}
	var out []domain.User
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.User{}, err
	}
	if len(out) == 0 {
		return domain.User{}, fmt.Errorf("no user returned")
	}
	return out[0], nil
}

func (s *UsersStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	data, _, err := s.c.From("users").Select("*", "", false).Eq("email", email).Limit(1, "").Execute()
	if err != nil {
		return nil, err
	}
	var out []domain.User
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return &out[0], nil
}

// Calls your RPC if present; returns new user id when successful
func (s *UsersStore) RegisterViaRPC(ctx context.Context, email, hash, role, name, specificRole, profileType string) (uint, error) {
	payload := map[string]any{
		"user_email":            email,
		"user_password_hash":    hash,
		"user_primary_role":     role,
		"profile_name":          name,
		"profile_specific_role": specificRole,
		"profile_type":          profileType,
	}
	res := s.c.Rpc("handle_new_user_registration", "public", payload)
	if res != "" { // v0.0.4 returns string error
		return 0, fmt.Errorf("registration rpc failed: %s", res)
	}
	// If your RPC can be changed, return the id to avoid a second query.
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	return u.ID, nil
}
