package reports

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetReportFormat(ctx context.Context, hospitalID string) (*ReportFormat, error) {
	fmt, err := s.repo.GetFormat(ctx, hospitalID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if fmt == nil {
		// return a default empty format instead of 404
		return &ReportFormat{HospitalID: hospitalID}, nil
	}
	return fmt, nil
}

func (s *service) UpdateReportFormat(ctx context.Context, hospitalID string, req UpdateFormatReq) (*ReportFormat, error) {
	f := &ReportFormat{
		HospitalID:  hospitalID,
		HeaderText:  req.HeaderText,
		AddressText: req.AddressText,
		FooterText:  req.FooterText,
		LogoURL:     req.LogoURL,
	}
	if err := s.repo.UpsertFormat(ctx, f); err != nil {
		return nil, apperr.Internal(err)
	}
	return f, nil
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/v1/reports/format
func (h *Handler) GetFormat(c *gin.Context) {
	// Only doctors/staff associated with a hospital can see the format.
	// We'll use doctorID as hospitalID for now as discussed.
	did, ok := c.Get(middleware.CtxDoctorID)
	if !ok || did.(string) == "" {
		apperr.Abort(c, apperr.Unauthorized("No hospital association found"))
		return
	}
	hospitalID := did.(string)

	fmt, err := h.svc.GetReportFormat(c.Request.Context(), hospitalID)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, fmt)
}

// PUT /api/v1/reports/format
func (h *Handler) UpdateFormat(c *gin.Context) {
	did, ok := c.Get(middleware.CtxDoctorID)
	if !ok || did.(string) == "" {
		apperr.Abort(c, apperr.Unauthorized("No hospital association found"))
		return
	}
	hospitalID := did.(string)

	var req UpdateFormatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON"))
		return
	}

	fmt, err := h.svc.UpdateReportFormat(c.Request.Context(), hospitalID, req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, fmt)
}
