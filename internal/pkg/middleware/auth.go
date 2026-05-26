// Package middleware provides authentication and RBAC middleware for VitalCache.
// Auth uses the pkg/jwt.JWTKeySource — supporting key rotation — instead of a
// raw static secret string.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	appErrors "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

// Context keys for values set by Auth middleware.
const (
	CtxUserID      = "user_id"
	CtxRole        = "role"
	CtxDesignation = "designation"
	CtxPermissions = "permissions"
	CtxDoctorID    = "doctor_id"
	CtxPatientID   = "patient_id"
)

// Auth validates the Bearer JWT using the full key-rotation-aware JWTKeySource.
// On success it sets user_id, role, designation, permissions, doctor_id, patient_id
// in the Gin context so downstream handlers and RBAC middleware can use them.
func Auth(ks pkgjwt.JWTKeySource) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			appErrors.Abort(c, appErrors.Unauthorized("missing or malformed Authorization header"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ValidateToken handles kid-based lookup and legacy brute-force fallback.
		_, claims, err := pkgjwt.ValidateToken(tokenString, ks)
		if err != nil {
			appErrors.Abort(c, appErrors.Unauthorized("invalid or expired token"))
			return
		}

		// ── Extract standard claims ──────────────────────────────────────────
		setStr := func(key string, v interface{}) {
			if s, ok := v.(string); ok && s != "" {
				c.Set(key, s)
			}
		}

		setStr(CtxUserID, claims["user_id"])
		setStr(CtxRole, claims["role"])
		setStr(CtxDesignation, claims["designation"])
		setStr(CtxDoctorID, claims["doctor_id"])
		setStr(CtxPatientID, claims["patient_id"])

		// Permissions is a []interface{} in MapClaims — convert to []string.
		if perms, ok := claims["permissions"].([]interface{}); ok {
			out := make([]string, 0, len(perms))
			for _, p := range perms {
				if s, ok := p.(string); ok {
					out = append(out, s)
				}
			}
			c.Set(CtxPermissions, out)
		} else {
			c.Set(CtxPermissions, []string{})
		}

		c.Next()
	}
}

// RequireRole enforces that the authenticated user has one of the given roles.
// Must be used AFTER Auth middleware.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(CtxRole)
		if !exists {
			appErrors.Abort(c, appErrors.Forbidden("role not set in token"))
			return
		}
		roleStr, ok := role.(string)
		if !ok {
			appErrors.Abort(c, appErrors.Forbidden("invalid role claim"))
			return
		}

		// Universal access for GodFather designation
		designation, existsDesig := c.Get(CtxDesignation)
		if existsDesig {
			if desigStr, ok := designation.(string); ok && strings.EqualFold(desigStr, "GodFather") {
				c.Next()
				return
			}
		}

		// Check against allowed roles
		for _, r := range allowedRoles {
			if strings.EqualFold(r, roleStr) {
				c.Next()
				return
			}
		}
		appErrors.Abort(c, appErrors.Forbidden("insufficient role"))
	}
}

// RequirePermission enforces that the authenticated user holds a specific permission.
// Must be used AFTER Auth middleware.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(CtxPermissions)
		if !exists {
			appErrors.Abort(c, appErrors.Forbidden("permissions not set"))
			return
		}
		perms, ok := val.([]string)
		if !ok {
			appErrors.Abort(c, appErrors.Forbidden("invalid permissions claim"))
			return
		}
		for _, p := range perms {
			if p == permission {
				c.Next()
				return
			}
		}
		appErrors.Abort(c, appErrors.Forbidden("missing required permission: "+permission))
	}
}

// BlockRole enforces that the authenticated user does NOT have the specified role.
// Used to prevent TESTER roles from making mutations (POST, PUT, PATCH, DELETE).
// Note: GodFather designation bypasses this block.
func BlockRole(blockedRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// GodFather designation bypasses all role blocks
		designation, existsDesig := c.Get(CtxDesignation)
		if existsDesig {
			if desigStr, ok := designation.(string); ok && strings.EqualFold(desigStr, "GodFather") {
				c.Next()
				return
			}
		}
		role, exists := c.Get(CtxRole)
		if exists {
			if roleStr, ok := role.(string); ok && strings.EqualFold(roleStr, blockedRole) {
				appErrors.Abort(c, appErrors.Forbidden("this action is not allowed for your role"))
				return
			}
		}
		c.Next()
	}
}

// RequireDesignation enforces that the authenticated user has a specific designation.
// Used for GodFather-only endpoints (e.g., invite another GodFather).
func RequireDesignation(allowedDesignations ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		designation, exists := c.Get(CtxDesignation)
		if !exists {
			appErrors.Abort(c, appErrors.Forbidden("designation not set in token"))
			return
		}
		desigStr, ok := designation.(string)
		if !ok {
			appErrors.Abort(c, appErrors.Forbidden("invalid designation claim"))
			return
		}
		for _, d := range allowedDesignations {
			if strings.EqualFold(d, desigStr) {
				c.Next()
				return
			}
		}
		appErrors.Abort(c, appErrors.Forbidden("insufficient designation privileges"))
	}
}
