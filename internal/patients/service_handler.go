package patients

import (
	"context"
	"net/http"
	"strconv"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

// --- Service ---
type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetPatients(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, total, err := s.repo.ListByDoctor(ctx, doctorID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return list, total, nil
}

// --- Handler ---
type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	// JWT payload is now strings
	userID, _ := c.Get("user_id")
	doctorID, ok := userID.(string)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("Invalid user ID in token"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := h.svc.GetPatients(c.Request.Context(), doctorID, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusOK, gin.H{
		"data":  list,
		"total": total,
	})
}
