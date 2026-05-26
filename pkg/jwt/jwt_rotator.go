package jwt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
)

type JWTKeySource interface {
	ActiveKey() (kid string, secret []byte)
	AllKeys() map[string][]byte
}

type keySource struct {
	kid     string
	keys    map[string][]byte
	rotDays int
	payload *config.SecretPayload
	save    func(ctx context.Context, p *config.SecretPayload) error
}

func (ks *keySource) ActiveKey() (string, []byte) { return ks.kid, ks.keys[ks.kid] }
func (ks *keySource) AllKeys() map[string][]byte  { return ks.keys }

func NewKeySource(payload *config.SecretPayload, saver func(context.Context, *config.SecretPayload) error) (*keySource, error) {
	keys := make(map[string][]byte, len(payload.JWTKeyring.Keys))
	for kid, b64 := range payload.JWTKeyring.Keys {
		raw, err := base64.RawURLEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decode key %s: %w", kid, err)
		}
		keys[kid] = raw
	}
	rot := payload.JWTKeyring.RotatesEveryDays
	if rot == 0 {
		rot = 7
	}
	return &keySource{
		kid:     payload.JWTKeyring.ActiveKID,
		keys:    keys,
		rotDays: rot,
		payload: payload,
		save:    saver,
	}, nil
}

func RotateIfNeeded(ctx context.Context, ks *keySource) error {
	age, err := ks.payload.JWTKeyring.ActiveKeyAge()
	if err != nil {
		return err
	}
	window := time.Duration(ks.rotDays) * 24 * time.Hour
	if age < window {
		return nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	b64 := base64.RawURLEncoding.EncodeToString(raw)
	kid := "kid-" + time.Now().UTC().Format("2006-01-02")

	kr := &ks.payload.JWTKeyring
	kr.ActiveKID = kid
	if kr.Keys == nil {
		kr.Keys = map[string]string{}
	}
	if kr.CreatedAt == nil {
		kr.CreatedAt = map[string]string{}
	}
	kr.Keys[kid] = b64
	kr.CreatedAt[kid] = time.Now().UTC().Format(time.RFC3339)

	// keep only last 2 keys
	if len(kr.Keys) > 2 {
		oldestKID := ""
		oldest := time.Now()
		for k, ts := range kr.CreatedAt {
			t, e := time.Parse(time.RFC3339, ts)
			if e != nil {
				continue
			}
			if t.Before(oldest) && k != kid {
				oldest = t
				oldestKID = k
			}
		}
		if oldestKID != "" {
			delete(kr.Keys, oldestKID)
			delete(kr.CreatedAt, oldestKID)
		}
	}

	if err := ks.save(ctx, ks.payload); err != nil {
		return err
	}
	ks.kid = kid
	ks.keys[kid] = raw
	slog.Info("JWT key rotated", "kid", kid)
	return nil
}
