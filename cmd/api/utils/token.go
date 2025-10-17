package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateToken(doctorID uint, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": doctorID,                                  // Subject of the token
		"iat": time.Now().Unix(),                         // Issued At
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // Expires in 7 days
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

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
