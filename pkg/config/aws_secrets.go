package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfgv2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Keyring struct {
	ActiveKID        string            `json:"active_kid"`
	RotatesEveryDays int               `json:"rotates_every_days"`
	Keys             map[string]string `json:"keys"`
	CreatedAt        map[string]string `json:"created_at"`
}

type SecretPayload struct {
	// Legacy Supabase HTTP client credentials (used during transition).
	// Will be removed once all stores are migrated to pgx/mongo-driver.
	SupabaseURL string `json:"SUPABASE_URL"`
	SupabaseKey string `json:"SUPABASE_KEY"`

	// PostgresDSN is the full Supabase connection URL for pgx.
	// Format: postgresql://postgres:[pw]@db.[ref].supabase.co:5432/postgres
	// Leave empty to fall back to the POSTGRES_DSN environment variable.
	PostgresDSN string `json:"POSTGRES_DSN,omitempty"`

	JWTKeyring Keyring `json:"JWT_KEYRING"`
}

type SecretsClient struct {
	sm     *secretsmanager.Client
	secret string
}

// NewSecretsClient returns nil (no error) if AWS is not configured, enabling local .env fallback.
func NewSecretsClient(ctx context.Context) (*SecretsClient, error) {
	region := os.Getenv("AWS_REGION")
	secretID := os.Getenv("AWS_SECRET_ID")

	// Local/dev fallback: if either is missing, we disable AWS secrets.
	if region == "" || secretID == "" {
		return nil, nil
	}

	awscfg, err := cfgv2.LoadDefaultConfig(ctx, cfgv2.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &SecretsClient{
		sm:     secretsmanager.NewFromConfig(awscfg),
		secret: secretID,
	}, nil
}

func (c *SecretsClient) Get(ctx context.Context) (*SecretPayload, string, error) {
	out, err := c.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(c.secret),
	})
	if err != nil {
		return nil, "", err
	}
	if out.SecretString == nil {
		return nil, "", fmt.Errorf("secret has no SecretString")
	}
	var sp SecretPayload
	if err := json.Unmarshal([]byte(*out.SecretString), &sp); err != nil {
		return nil, "", err
	}
	ver := ""
	if out.VersionId != nil {
		ver = *out.VersionId
	}
	return &sp, ver, nil
}

func (c *SecretsClient) Put(ctx context.Context, payload *SecretPayload) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.sm.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(c.secret),
		SecretString: aws.String(string(buf)),
	})
	return err
}

func (kr *Keyring) ActiveKeyAge() (time.Duration, error) {
	created, ok := kr.CreatedAt[kr.ActiveKID]
	if !ok {
		return 0, fmt.Errorf("created_at missing for active kid %s", kr.ActiveKID)
	}
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return 0, err
	}
	return time.Since(t), nil
}
