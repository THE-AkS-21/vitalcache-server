package invites

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

// NOTE: Instead of a full database table for invites, we can just issue a special JWT
// since the only thing it needs to contain is the clinic/hospital ID and role (DOCTOR)
// This keeps it stateless and simple.

type Service struct {
	ks  pkgjwt.JWTKeySource
	log *zap.Logger
}

func NewService(ks pkgjwt.JWTKeySource) *Service {
	return &Service{ks: ks, log: logger.Named("invites.service")}
}

func (s *Service) GenerateInviteToken(hospitalID string, role string) (string, error) {
	// A special JWT with a 7-day expiration
	claims := map[string]interface{}{
		"type":        "invite",
		"hospital_id": hospitalID,
		"role":        role, // Typically "Doctor"
		"exp":         time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	token, err := pkgjwt.GenerateTokenWithCustomClaims(s.ks, claims)
	if err != nil {
		return "", err
	}
	return token, nil
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type GenerateInviteRequest struct {
	HospitalID string `json:"hospital_id" validate:"required"`
	Role       string `json:"role" validate:"required,oneof=Doctor Staff"`
}

func (h *Handler) GenerateInvite(c *gin.Context) {
	var req GenerateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid json"))
		return
	}

	// Only Developer, GodFather, Admin can generate invites
	role, _ := c.Get(middleware.CtxRole)
	roleStr, _ := role.(string)

	if roleStr != "Developer" && roleStr != "GodFather" && roleStr != "Admin" {
		apperr.Abort(c, apperr.Unauthorized("only admins can generate invites"))
		return
	}

	token, err := h.svc.GenerateInviteToken(req.HospitalID, req.Role)
	if err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	apperr.WriteOK(c, http.StatusOK, gin.H{"invite_token": token})
}

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, ks pkgjwt.JWTKeySource) {
	svc := NewService(ks)
	h := NewHandler(svc)

	group := router.Group("/invites")
	group.Use(middleware.Auth(ks))
	{
		group.POST("", middleware.RequireRole("DEVELOPER", "GODFATHER", "ADMIN"), h.GenerateInvite)
	}
}
