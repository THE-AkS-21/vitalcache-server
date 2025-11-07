package middlewares

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt" // <-- change
	"github.com/gin-gonic/gin"
)

// Inject your key source when wiring the router
func Auth(ks jwt.JWTKeySource) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		userID, claims, err := jwt.ValidateToken(parts[1], ks) // <-- change
		if err != nil {
			slog.Warn("JWT validation failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.Set("user_id", userID)
		if role, ok := claims["role"]; ok {
			c.Set("user_role", role)
		}
		c.Next()
	}
}
