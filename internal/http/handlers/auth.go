package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

const (
	defaultRefreshDays = 30
	refreshCookieName  = "refresh_token"
	refreshCookiePath  = "/api/auth"
)

// ---------- helpers ----------

func isAuthWired(d hdeps.Deps) bool { return d.Auth != nil }

func refreshTTL() time.Duration {
	d := defaultRefreshDays
	if v := strings.TrimSpace(os.Getenv("REFRESH_TTL_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 180 {
			d = n
		}
	}
	return time.Duration(d) * 24 * time.Hour
}

func cookieDomain() string { return strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")) }

func isSecureCookie() bool {
	// Default secure in non-debug; allow DEV_INSECURE_COOKIES to disable for local
	if os.Getenv("DEV_INSECURE_COOKIES") != "" {
		return false
	}
	// If you run behind HTTPS in prod, this should be true.
	return true
}

// ---------- handlers ----------

func Register(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthWired(d) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": gin.H{"code": "not_implemented", "message": "auth service not wired"},
			})
			return
		}

		var req dto.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.WriteBadRequest(c, "invalid request", err.Error())
			return
		}
		// normalize basics
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		if err := d.Auth.Register(c.Request.Context(), req); err != nil {
			errors.WriteConflict(c, "registration failed", err.Error())
			return
		}
		c.Status(http.StatusCreated)
	}
}

func Login(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		println("DEBUG [Handler]: Login request received")
		if !isAuthWired(d) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": gin.H{"code": "not_implemented", "message": "auth service not wired"},
			})
			return
		}

		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			println("DEBUG [Handler]: JSON binding error:", err.Error())
			errors.WriteBadRequest(c, "invalid request", err.Error())
			return
		}
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		println("DEBUG [Handler]: Request bound, email:", req.Email)

		println("DEBUG [Handler]: Calling auth service Login...")
		tokens, err := d.Auth.Login(c.Request.Context(), req)
		if err != nil {
			println("DEBUG [Handler]: Auth service returned error:", err.Error())
			errors.WriteUnauthorized(c, "invalid credentials", err.Error())
			return
		}
		println("DEBUG [Handler]: Auth service returned success!")
		println("DEBUG [Handler]: AccessToken length:", len(tokens.AccessToken))
		println("DEBUG [Handler]: RefreshToken length:", len(tokens.RefreshToken))

		// Set httpOnly refresh cookie
		c.SetSameSite(http.SameSiteLaxMode) // safe default for APIs
		c.SetCookie(
			refreshCookieName,
			tokens.RefreshToken,
			int(refreshTTL().Seconds()),
			refreshCookiePath,
			cookieDomain(),
			isSecureCookie(),
			true, // HttpOnly
		)

		c.JSON(http.StatusOK, gin.H{"accessToken": tokens.AccessToken})
	}
}

func Refresh(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthWired(d) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": gin.H{"code": "not_implemented", "message": "auth service not wired"},
			})
			return
		}

		// Prefer cookie; optionally accept JSON body {"refreshToken": "..."} for non-cookie clients
		rt, _ := c.Cookie(refreshCookieName)
		if rt == "" {
			var body struct {
				RefreshToken string `json:"refreshToken"`
			}
			_ = c.ShouldBindJSON(&body)
			rt = strings.TrimSpace(body.RefreshToken)
		}
		if rt == "" {
			errors.WriteUnauthorized(c, "missing refresh token", nil)
			return
		}

		tokens, err := d.Auth.Refresh(c.Request.Context(), rt)
		if err != nil {
			errors.WriteUnauthorized(c, "refresh failed", err.Error())
			return
		}

		// Rotate cookie
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			refreshCookieName,
			tokens.RefreshToken,
			int(refreshTTL().Seconds()),
			refreshCookiePath,
			cookieDomain(),
			isSecureCookie(),
			true,
		)

		c.JSON(http.StatusOK, gin.H{"accessToken": tokens.AccessToken})
	}
}

func Logout(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Even if Auth isn’t wired, clearing the cookie is still useful
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			refreshCookieName,
			"",
			-1, // expire immediately
			refreshCookiePath,
			cookieDomain(),
			isSecureCookie(),
			true,
		)
		c.Status(http.StatusNoContent)
	}
}
