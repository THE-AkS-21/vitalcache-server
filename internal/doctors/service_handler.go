// Package doctors — service + Gin handlers.
package doctors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Service
// ─────────────────────────────────────────────────────────────────────────────

// Service implements the doctor use-cases.
type Service struct {
	repo Repository
	log  *zap.Logger
}

// NewService wires the doctor service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Named("doctors.service")}
}

func (s *Service) GetByID(ctx context.Context, id string) (*Doctor, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("doctor.GetByID: %w", err))
	}
	if d == nil {
		return nil, apperr.NotFound("doctor", nil)
	}
	return d, nil
}

func (s *Service) GetByUserID(ctx context.Context, userID string) (*Doctor, error) {
	d, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("doctor.GetByUserID: %w", err))
	}
	if d == nil {
		return nil, apperr.NotFound("doctor profile", nil)
	}
	return d, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Doctor, int64, error) {
	ds, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return ds, total, nil
}

func (s *Service) UpdateFee(ctx context.Context, id string, fee float64) error {
	if err := s.repo.UpdateFee(ctx, id, fee); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────────────────────────────────────

// Handler holds the doctors service.
type Handler struct{ svc *Service }

// NewHandler creates the doctors HTTP handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// GET /api/v1/doctors?limit=&offset=
func (h *Handler) List(c *gin.Context) {
	pg := pagination.FromContext(c)
	ds, total, err := h.svc.List(c.Request.Context(), pg.Limit, pg.Offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WritePaginated(c, http.StatusOK, ds, apperr.PaginationMeta{
		Limit: pg.Limit, Offset: pg.Offset, Total: total,
	})
}

// GET /api/v1/doctors/:id
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apperr.Abort(c, apperr.BadRequest("invalid id"))
		return
	}
	d, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, d)
}

// PUT /api/v1/doctors/:id/fee
type UpdateFeeRequest struct {
	Fee float64 `json:"fee" validate:"required,min=0"`
}

func (h *Handler) UpdateFee(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apperr.Abort(c, apperr.BadRequest("invalid id"))
		return
	}

	var req UpdateFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON"))
		return
	}

	if err := h.svc.UpdateFee(c.Request.Context(), id, req.Fee); err != nil {
		apperr.Abort(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
