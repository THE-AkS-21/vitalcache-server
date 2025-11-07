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
	SupabaseURL string  `json:"SUPABASE_URL"`
	SupabaseKey string  `json:"SUPABASE_KEY"`
	SMTPHost    string  `json:"SMTP_HOST"`
	SMTPPort    string  `json:"SMTP_PORT"`
	SMTPUser    string  `json:"SMTP_USER"`
	SMTPPass    string  `json:"SMTP_PASS"`
	JWTKeyring  Keyring `json:"JWT_KEYRING"`
}

type SecretsClient struct {
	sm     *secretsmanager.Client
	secret string
}

func NewSecretsClient(ctx context.Context) (*SecretsClient, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return nil, fmt.Errorf("AWS_REGION is required")
	}
	secretID := os.Getenv("AWS_SECRET_ID")
	if secretID == "" {
		return nil, fmt.Errorf("AWS_SECRET_ID is required")
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
