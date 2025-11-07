package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

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

func ValidateToken(token string, ks JWTKeySource) (uint, jwt.MapClaims, error) {
	parser := &jwt.Parser{}
	// peek KID
	tok, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return 0, nil, err
	}
	kid, _ := tok.Header["kid"].(string)
	keys := ks.AllKeys()

	keyFunc := func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		if kid != "" {
			if k, ok := keys[kid]; ok {
				return k, nil
			}
			return nil, errors.New("kid not found")
		}
		return nil, errors.New("no kid")
	}
	// try with kid
	tok, err = jwt.ParseWithClaims(token, jwt.MapClaims{}, keyFunc)
	if err != nil {
		// brute force legacy tokens (pre-kid)
		var lastErr error
		for _, k := range keys {
			tok, lastErr = jwt.ParseWithClaims(token, jwt.MapClaims{}, func(_ *jwt.Token) (interface{}, error) { return k, nil })
			if lastErr == nil && tok.Valid {
				break
			}
		}
		if lastErr != nil {
			return 0, nil, lastErr
		}
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
