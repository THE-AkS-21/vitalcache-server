package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	Sub       int    `json:"sub"`                 // users.id
	Role      string `json:"role"`                // doctor|developer|patient
	DoctorID  *int   `json:"doctor_id,omitempty"` // set for doctors
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

func GenerateToken(userID uint, role string, ks JWTKeySource, ttl time.Duration) (string, error) {
	kid, key := ks.ActiveKey()
	if len(key) == 0 {
		return "", errors.New("no active key")
	}
	claims := jwt.MapClaims{
		"sub":  int64(userID),
		"role": role,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = kid
	return t.SignedString(key)
}

func GenerateTokenWithCustomClaims(ks JWTKeySource, customClaims map[string]interface{}) (string, error) {
	kid, key := ks.ActiveKey()
	if len(key) == 0 {
		return "", errors.New("no active key")
	}
	claims := jwt.MapClaims{}
	for k, v := range customClaims {
		claims[k] = v
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = kid
	return t.SignedString(key)
}

func ValidateToken(token string, ks JWTKeySource) (uint, jwt.MapClaims, error) {
	parser := &jwt.Parser{}
	// peek KID
	tok, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return 0, nil, err
	}
	kid, ok := tok.Header["kid"].(string)
	if !ok || kid == "" {
		return 0, nil, errors.New("malformed token: missing kid")
	}

	keys := ks.AllKeys()

	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		if k, ok := keys[kid]; ok {
			return k, nil
		}
		return nil, errors.New("kid not found")
	}

	tok, err = jwt.ParseWithClaims(token, jwt.MapClaims{}, keyFunc)
	if err != nil {
		return 0, nil, err
	}
	if claims, ok := tok.Claims.(jwt.MapClaims); ok && tok.Valid {
		switch v := claims["sub"].(type) {
		case float64:
			return uint(v), claims, nil
		case int64:
			return uint(v), claims, nil
		case int:
			return uint(v), claims, nil
		default:
			return 0, nil, fmt.Errorf("invalid sub type")
		}
	}
	return 0, nil, fmt.Errorf("invalid token")
}

func (c *Claims) IsExpired() bool {
	return time.Now().Unix() >= c.ExpiresAt
}
