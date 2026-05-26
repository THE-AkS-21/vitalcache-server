package billings

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/pagination"
)

type Service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Named("billings.service")}
}

func (s *Service) Create(ctx context.Context, b *Billing) error {
	if b.Status == "" {
		b.Status = "PENDING"
	}
	return s.repo.Create(ctx, b)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) List(ctx context.Context, doctorID, patientID string, limit, offset int) ([]Billing, int64, error) {
	return s.repo.List(ctx, doctorID, patientID, limit, offset)
}

func (s *Service) GetAnalytics(ctx context.Context, doctorID string) (*Analytics, error) {
	return s.repo.GetAnalytics(ctx, doctorID)
}

func (s *Service) GetHeatmap(ctx context.Context, doctorID string) (*HeatmapData, error) {
	return s.repo.GetHeatmap(ctx, doctorID)
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type CreateBillingRequest struct {
	PatientID       string  `json:"patient_id" validate:"required"`
	DoctorID        string  `json:"doctor_id" validate:"required"`
	MedicalReportID string  `json:"medical_report_id"`
	Amount          float64 `json:"amount" validate:"required,min=0"`
	Status          string  `json:"status" validate:"omitempty,oneof=PENDING PAID CANCELLED"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateBillingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid json"))
		return
	}

	b := &Billing{
		PatientID:       req.PatientID,
		DoctorID:        req.DoctorID,
		MedicalReportID: req.MedicalReportID,
		Amount:          req.Amount,
		Status:          req.Status,
	}

	if err := h.svc.Create(c.Request.Context(), b); err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	apperr.WriteOK(c, http.StatusCreated, b)
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=PENDING PAID CANCELLED"`
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apperr.Abort(c, apperr.BadRequest("missing id"))
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid json"))
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	pg := pagination.FromContext(c)

	role, _ := c.Get(middleware.CtxRole)
	roleStr, _ := role.(string)
	desig, _ := c.Get(middleware.CtxDesignation)
	desigStr, _ := desig.(string)
	isGodFather := strings.EqualFold(desigStr, "GodFather")

	var doctorID, patientID string

	switch {
	case strings.EqualFold(roleStr, "Doctor"):
		docRaw, _ := c.Get(middleware.CtxDoctorID)
		doctorID, _ = docRaw.(string)
		if doctorID == "" {
			apperr.Abort(c, apperr.Unauthorized("no doctor profile"))
			return
		}
	case strings.EqualFold(roleStr, "Patient"):
		patRaw, _ := c.Get(middleware.CtxPatientID)
		patientID, _ = patRaw.(string)
		if patientID == "" {
			apperr.Abort(c, apperr.Unauthorized("no patient profile"))
			return
		}
	case isGodFather || strings.EqualFold(roleStr, "Admin") || strings.EqualFold(roleStr, "Developer"):
		// Can see all or filter by query param
		doctorID = c.Query("doctor_id")
		patientID = c.Query("patient_id")
	default:
		apperr.Abort(c, apperr.Forbidden("role not permitted"))
		return
	}

	list, total, err := h.svc.List(c.Request.Context(), doctorID, patientID, pg.Limit, pg.Offset)
	if err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	apperr.WritePaginated(c, http.StatusOK, list, apperr.PaginationMeta{
		Limit:  pg.Limit,
		Offset: pg.Offset,
		Total:  total,
	})
}

func (h *Handler) GetAnalytics(c *gin.Context) {
	role, _ := c.Get(middleware.CtxRole)
	roleStr, _ := role.(string)
	desig, _ := c.Get(middleware.CtxDesignation)
	desigStr, _ := desig.(string)
	isGodFather := strings.EqualFold(desigStr, "GodFather")

	var doctorID string

	if strings.EqualFold(roleStr, "Doctor") {
		docRaw, _ := c.Get(middleware.CtxDoctorID)
		doctorID, _ = docRaw.(string)
		if doctorID == "" {
			apperr.Abort(c, apperr.Unauthorized("no doctor profile found"))
			return
		}
	} else if isGodFather || strings.EqualFold(roleStr, "Admin") || strings.EqualFold(roleStr, "Developer") {
		// Admin/GodFather can query analytics for any doctor
		doctorID = c.Query("doctor_id")
		// If no doctor_id provided, return aggregate analytics
	} else {
		apperr.Abort(c, apperr.Forbidden("insufficient privileges to view analytics"))
		return
	}

	analytics, err := h.svc.GetAnalytics(c.Request.Context(), doctorID)
	if err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	apperr.WriteOK(c, http.StatusOK, analytics)
}

func (h *Handler) GetHeatmap(c *gin.Context) {
	role, _ := c.Get(middleware.CtxRole)
	roleStr, _ := role.(string)
	desig, _ := c.Get(middleware.CtxDesignation)
	desigStr, _ := desig.(string)
	isGodFather := strings.EqualFold(desigStr, "GodFather")

	var doctorID string

	if strings.EqualFold(roleStr, "Doctor") {
		docRaw, _ := c.Get(middleware.CtxDoctorID)
		doctorID, _ = docRaw.(string)
		if doctorID == "" {
			apperr.Abort(c, apperr.Unauthorized("no doctor profile found"))
			return
		}
	} else if isGodFather || strings.EqualFold(roleStr, "Admin") || strings.EqualFold(roleStr, "Developer") {
		doctorID = c.Query("doctor_id")
	} else {
		apperr.Abort(c, apperr.Forbidden("insufficient privileges to view heatmap"))
		return
	}

	heatmap, err := h.svc.GetHeatmap(c.Request.Context(), doctorID)
	if err != nil {
		apperr.Abort(c, apperr.Internal(err))
		return
	}

	apperr.WriteOK(c, http.StatusOK, heatmap)
}
