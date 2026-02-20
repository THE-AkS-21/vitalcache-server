package middlewares

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt" // <-- change
	"github.com/gin-gonic/gin"
)

const (
	CtxUserID      = "user_id"
	CtxUserRole    = "user_role"
	CtxDesignation = "designation"
	CtxDoctorID    = "doctor_id"
	CtxRawToken    = "raw_token"
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
		c.Set(CtxUserID, userID)
		if role, ok := claims["role"]; ok {
			c.Set(CtxUserRole, role)
		}
		if designation, ok := claims["designation"]; ok {
			c.Set(CtxDesignation, designation)
		}
		if doctorID, ok := claims["doctor_id"]; ok {
			c.Set(CtxDoctorID, doctorID)
		}
		c.Set(CtxRawToken, parts[1])
		c.Next()
	}
}

// Helper functions to extract values from context

func GetUserID(c *gin.Context) int {
	if v, ok := c.Get(CtxUserID); ok {
		if id, ok := v.(int); ok {
			return id
		}
	}
	return 0
}

func GetRole(c *gin.Context) string {
	if v, ok := c.Get(CtxUserRole); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}

func GetDesignation(c *gin.Context) string {
	if v, ok := c.Get(CtxDesignation); ok {
		if designation, ok := v.(string); ok {
			return designation
		}
	}
	return ""
}

func GetDoctorID(c *gin.Context) (int, bool) {
	if v, ok := c.Get(CtxDoctorID); ok {
		if id, ok := v.(int); ok {
			return id, true
		}
		// Handle float64 from JWT claims
		if id, ok := v.(float64); ok {
			return int(id), true
		}
	}
	return 0, false
}

func GetRawToken(c *gin.Context) string {
	if v, ok := c.Get(CtxRawToken); ok {
		if token, ok := v.(string); ok {
			return token
		}
	}
	return ""
}
