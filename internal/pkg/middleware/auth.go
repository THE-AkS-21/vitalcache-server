package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	appErrors "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			appErrors.Abort(c, appErrors.Unauthorized("Authentication failed or not provided"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			appErrors.Abort(c, appErrors.Unauthorized("Invalid or expired token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			appErrors.Abort(c, appErrors.Unauthorized("Invalid token claims structure"))
			return
		}

		// Set context variables for handlers and RLS
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Set("permissions", claims["permissions"])

		c.Next()
	}
}

// RequireRole enforces Role-Based Access Control (RBAC)
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			appErrors.Abort(c, appErrors.New("FORBIDDEN", "You do not have permission to perform this action"))
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			appErrors.Abort(c, appErrors.New("FORBIDDEN", "You do not have permission to perform this action"))
			return
		}

		for _, role := range allowedRoles {
			if role == roleStr {
				c.Next()
				return
			}
		}

		appErrors.Abort(c, appErrors.New("FORBIDDEN", "You do not have permission to perform this action"))
	}
}
