package supabase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	postgrest "github.com/supabase-community/postgrest-go"
)

type Client struct {
	Base string
	Key  string
}

// url: "https://<ref>.supabase.co", key: anon/service key
func NewClient(url, key string) *Client {
	base := strings.TrimRight(url, "/") + "/rest/v1"
	return &Client{Base: base, Key: key}
}

// core returns a client with only apikey + Authorization headers (no RLS identity).
func (s *Client) core() *postgrest.Client {
	return postgrest.NewClient(s.Base, "", map[string]string{
		"apikey":        s.Key,
		"Authorization": "Bearer " + s.Key,
	})
}

// ForUser returns a client with per-request RLS identity headers.
func (s *Client) ForUser(userID int, role string) *postgrest.Client {
	return postgrest.NewClient(s.Base, "", map[string]string{
		"apikey":        s.Key,
		"Authorization": "Bearer " + s.Key,
		"X-User-Id":     strconv.Itoa(userID),
		"X-Role":        role,
	})
}

// WithToken returns a client using the provided JWT for RLS.
func (s *Client) WithToken(token string) *postgrest.Client {
	return postgrest.NewClient(s.Base, "", map[string]string{
		"apikey":        s.Key,
		"Authorization": "Bearer " + token,
	})
}

// WithRLS returns a client with JWT and RLS context headers.
func (s *Client) WithRLS(ctx context.Context, token string) *postgrest.Client {
	headers := map[string]string{
		"apikey":        s.Key,
		"Authorization": "Bearer " + token,
	}
	if role, ok := ctx.Value("user_role").(string); ok && role != "" {
		headers["X-Role"] = role
	}
	if uid, ok := ctx.Value("user_id").(int); ok {
		headers["X-User-Id"] = fmt.Sprintf("%d", uid)
	}
	if did, ok := ctx.Value("doctor_id").(int); ok {
		headers["X-Doctor-Id"] = fmt.Sprintf("%d", did)
	}
	return postgrest.NewClient(s.Base, "", headers)
}

func (s *Client) Ping(ctx context.Context) error {
	// Lightweight connectivity check
	_, _, err := s.core().From("medicines").Select("id", "", true).Limit(1, "").Execute()
	return err
}
