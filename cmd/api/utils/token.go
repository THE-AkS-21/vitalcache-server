package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// GenerateToken now includes the user's role
func GenerateToken(userID uint, role string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID, // Subject of the token
		"role": role,   // User's role
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ... ValidateToken remains the same for now, but you could extract the role from it too ...
func ValidateToken(tokenString string, secret string) (float64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if sub, ok := claims["sub"].(float64); ok {
			return sub, nil
		}
	}
	return 0, fmt.Errorf("invalid token")
}
