// Package auth — Gin HTTP handlers for the auth module.
// All business logic lives in Service; handlers only bind, validate, and respond.
package auth

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
)

const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/"
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

	var hospitalID *string

	if req.InviteToken != "" {
		// Just parse the unverified token to extract hospital_id and role
		// We could fully verify it if we have the ks, but we don't have it in this handler directly.
		// Alternatively, we pass InviteToken to the service and let it verify.
		// Let's pass it to the service.
	} else if req.Role == "DOCTOR" || req.Role == "STAFF" {
		apperr.Abort(c, apperr.Forbidden("an invite token is required to register as a doctor or staff member"))
		return
	}

	// Service signature will be updated to accept InviteToken
	if err := h.svc.Register(c.Request.Context(), req, hospitalID); err != nil {
		apperr.Abort(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

// ─────────────────────────────────────────────────────────────────────────────
// Invite Handlers
// ─────────────────────────────────────────────────────────────────────────────

// POST /api/auth/invites
// Protected: Only Admin/Doctor should call this to generate an invite link.
func (h *Handler) GenerateInvite(c *gin.Context) {
	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	// Enforce Godfather restriction
	if strings.EqualFold(req.Designation, "Godfather") {
		designation, exists := c.Get("designation")
		if !exists {
			apperr.Abort(c, apperr.Forbidden("only Godfathers can invite other Godfathers"))
			return
		}
		if desigStr, ok := designation.(string); !ok || !strings.EqualFold(desigStr, "Godfather") {
			apperr.Abort(c, apperr.Forbidden("only Godfathers can invite other Godfathers"))
			return
		}
	}

	token, err := h.svc.GenerateInvite(c.Request.Context(), req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	// For now, return the token so the client can construct the link and copy it
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"invite_token": token,
		},
	})
}

// POST /api/auth/invites/accept
// Public: The invited user submits their details + the token.
func (h *Handler) AcceptInvite(c *gin.Context) {
	var req AcceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	if err := h.svc.AcceptInvite(c.Request.Context(), req); err != nil {
		apperr.Abort(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/v1/auth/password
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) UpdatePassword(c *gin.Context) {
	userIDStr, ok := c.Get("user_id")
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("user not authenticated"))
		return
	}
	userID := userIDStr.(string)

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	if err := h.svc.UpdatePassword(c.Request.Context(), userID, req); err != nil {
		apperr.Abort(c, err)
		return
	}

	c.Status(http.StatusNoContent)
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
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		pair.RefreshToken,
		pair.RefreshTTL,
		refreshCookiePath,
		cookieDomain(),
		secureFlag(),
		true, // HttpOnly
	)

	apperr.WriteOK(c, http.StatusOK, LoginResponse{AccessToken: pair.AccessToken})
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/auth/google
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON: "+err.Error()))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	pair, err := h.svc.GoogleLogin(c.Request.Context(), req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		pair.RefreshToken,
		pair.RefreshTTL,
		refreshCookiePath,
		cookieDomain(),
		secureFlag(),
		true,
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
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		pair.RefreshToken,
		pair.RefreshTTL,
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
	c.SetSameSite(http.SameSiteStrictMode)
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

func cookieDomain() string { return strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")) }

func secureFlag() bool {
	// Disabled only when DEV_INSECURE_COOKIES is explicitly set (HTTP local dev).
	return os.Getenv("DEV_INSECURE_COOKIES") == ""
}
