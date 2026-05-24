package patients

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
)

// ─────────────────────────────────────────────────────────────────────────────
// Service
// ─────────────────────────────────────────────────────────────────────────────

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) GetPatients(ctx context.Context, doctorID string, limit, offset int) ([]Patient, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListByDoctor(ctx, doctorID, limit, offset)
}

func (s *service) SearchPatients(ctx context.Context, doctorID, phone string) ([]Patient, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		list, _, err := s.repo.ListByDoctor(ctx, doctorID, 50, 0)
		return list, err
	}
	return s.repo.SearchByPhone(ctx, doctorID, phone, 50)
}

func (s *service) GetPatient(ctx context.Context, id string) (*Patient, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if p == nil {
		return nil, apperr.NotFound("patient", nil)
	}
	return p, nil
}

func (s *service) CreatePatient(ctx context.Context, req CreatePatientReq, doctorID string) (*Patient, error) {
	p, err := s.repo.Create(ctx, req, doctorID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return p, nil
}

func (s *service) UpdatePatient(ctx context.Context, id string, req UpdatePatientReq) (*Patient, error) {
	p, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if p == nil {
		return nil, apperr.NotFound("patient", nil)
	}
	return p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────────────────────────────────────

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// getDoctorID extracts the doctor's own UUID from the JWT context.
// Falls back to user_id if doctor_id is not present (e.g., hospital staff).
func getDoctorID(c *gin.Context) (string, bool) {
	if did, ok := c.Get(middleware.CtxDoctorID); ok {
		if s, ok := did.(string); ok && s != "" {
			return s, true
		}
	}
	if uid, ok := c.Get(middleware.CtxUserID); ok {
		if s, ok := uid.(string); ok && s != "" {
			return s, true
		}
	}
	return "", false
}

// GET /api/v1/patients?limit=&offset=
func (h *Handler) List(c *gin.Context) {
	doctorID, ok := getDoctorID(c)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("doctor identity not found in token"))
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := h.svc.GetPatients(c.Request.Context(), doctorID, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WritePaginated(c, http.StatusOK, list, apperr.PaginationMeta{Limit: limit, Offset: offset, Total: total})
}

// GET /api/v1/patients/search?mobile=xxx
func (h *Handler) Search(c *gin.Context) {
	doctorID, ok := getDoctorID(c)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("doctor identity not found in token"))
		return
	}
	phone := c.Query("mobile")

	list, err := h.svc.SearchPatients(c.Request.Context(), doctorID, phone)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, gin.H{"data": list, "total": int64(len(list))})
}

// GET /api/v1/patients/:id
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	p, err := h.svc.GetPatient(c.Request.Context(), id)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, p)
}

// POST /api/v1/patients
func (h *Handler) Create(c *gin.Context) {
	var req CreatePatientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON payload"))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	doctorID, ok := getDoctorID(c)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("doctor identity not found in token"))
		return
	}

	p, err := h.svc.CreatePatient(c.Request.Context(), req, doctorID)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusCreated, p)
}

// PATCH /api/v1/patients/:id
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdatePatientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON payload"))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	p, err := h.svc.UpdatePatient(c.Request.Context(), id, req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, p)
}
