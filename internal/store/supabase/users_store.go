package supabase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	postgrest "github.com/supabase-community/postgrest-go"
)

type UsersStore struct{ c *postgrest.Client }

func NewUsersStore(c *Client) *UsersStore { return &UsersStore{c: c.core()} }

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
	println("DEBUG [UsersStore]: GetByEmail called with email:", email)
	data, _, err := s.c.From("users").Select("*", "", false).Eq("email", email).Limit(1, "").Execute()
	if err != nil {
		println("DEBUG [UsersStore]: Query error:", err.Error())
		return nil, err
	}
	println("DEBUG [UsersStore]: Query successful, data length:", len(data))
	var out []domain.User
	if err := json.Unmarshal(data, &out); err != nil {
		println("DEBUG [UsersStore]: Unmarshal error:", err.Error())
		return nil, err
	}
	if len(out) == 0 {
		println("DEBUG [UsersStore]: No user found for email:", email)
		return nil, fmt.Errorf("user not found")
	}
	println("DEBUG [UsersStore]: User found, ID:", out[0].ID, "Role:", out[0].Role)
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
	res := s.c.Rpc("handle_new_user_registration", "", payload)
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
