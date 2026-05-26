package config

import "time"

// NewLocalKeyring returns a static dev key so the app can run without AWS.
func NewLocalKeyring() Keyring {
	now := time.Now().UTC().Format(time.RFC3339)
	return Keyring{
		ActiveKID:        "dev-1",
		RotatesEveryDays: 9999,
		Keys: map[string]string{
			"dev-1": "dev-secret-key-change-me",
		},
		CreatedAt: map[string]string{
			"dev-1": now,
		},
	}
}
