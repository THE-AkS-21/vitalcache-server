package middlewares

import (
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain/policies"
	"github.com/gin-gonic/gin"
)

// RequireRole ensures the caller has one of the allowed roles.
// Use like: v1.Use(RequireRole("doctor")) or RequireRole(policies.RoleDoctor, policies.RoleDeveloper)
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := c.GetString(CtxUserRole)
		if role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role not found on context"})
			return
		}
		if _, ok := allowedSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

// Sugar helpers, if you prefer readable guards
func RequireDoctor() gin.HandlerFunc    { return RequireRole(policies.RoleDoctor) }
func RequireDeveloper() gin.HandlerFunc { return RequireRole(policies.RoleDeveloper) }
