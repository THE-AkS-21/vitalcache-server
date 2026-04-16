// Package auth — Gin HTTP handlers for the auth module.
// All business logic lives in Service; handlers only bind, validate, and respond.
package auth

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/api/auth"
	defaultRefreshTTL = 7 * 24 * time.Hour
)

// Handler holds the auth service dependency.
type Handler struct {
	svc *Service
}

// NewHandler creates the auth HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/auth/register
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	if err := h.svc.Register(c.Request.Context(), req); err != nil {
		apperr.Abort(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/auth/login
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	pair, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	// Deliver refresh token as an httpOnly cookie.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshCookieName,
		pair.RefreshToken,
		int(refreshCookieTTL().Seconds()),
		refreshCookiePath,
		cookieDomain(),
		secureFlag(),
		true, // HttpOnly
	)

	apperr.WriteOK(c, http.StatusOK, LoginResponse{AccessToken: pair.AccessToken})
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/auth/refresh
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Refresh(c *gin.Context) {
	// Prefer cookie; accept JSON body for non-browser clients.
	rt, _ := c.Cookie(refreshCookieName)
	if rt == "" {
		var body RefreshRequest
		_ = c.ShouldBindJSON(&body)
		rt = strings.TrimSpace(body.RefreshToken)
	}
	if rt == "" {
		apperr.Abort(c, apperr.Unauthorized("missing refresh token"))
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), rt)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	// Rotate cookie.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshCookieName,
		pair.RefreshToken,
		int(refreshCookieTTL().Seconds()),
		refreshCookiePath,
		cookieDomain(),
		secureFlag(),
		true,
	)

	apperr.WriteOK(c, http.StatusOK, LoginResponse{AccessToken: pair.AccessToken})
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/auth/logout
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) Logout(c *gin.Context) {
	rt, _ := c.Cookie(refreshCookieName)
	h.svc.Logout(c.Request.Context(), rt)

	// Clear the cookie regardless of whether the token was valid.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshCookieName,
		"",
		-1,
		refreshCookiePath,
		cookieDomain(),
		secureFlag(),
		true,
	)
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cookie helpers
// ─────────────────────────────────────────────────────────────────────────────

func refreshCookieTTL() time.Duration {
	if v := os.Getenv("REFRESH_TTL_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 180 {
			return time.Duration(n) * 24 * time.Hour
		}
	}
	return defaultRefreshTTL
}

func cookieDomain() string { return strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")) }

func secureFlag() bool {
	// Disabled only when DEV_INSECURE_COOKIES is explicitly set (HTTP local dev).
	return os.Getenv("DEV_INSECURE_COOKIES") == ""
}
