package medical_reports

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateReport(ctx context.Context, doctorID string, req CreateReportReq) (*MedicalReport, error) {
	report := &MedicalReport{
		ReportID:      "REP-" + uuid.New().String()[:8],
		PatientID:     req.PatientID,
		DoctorID:      doctorID,
		HospitalID:    req.HospitalID,
		DiseaseName:   req.DiseaseName,
		DiagnosisBody: req.DiagnosisBody,
		Medications:   req.Medications,
		Precautions:   req.Precautions,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, report); err != nil {
		return nil, apperr.Internal(err)
	}

	return report, nil
}

func (s *service) GetReport(ctx context.Context, id string) (*MedicalReport, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperr.BadRequest("invalid id format")
	}

	report, err := s.repo.GetByID(ctx, oid)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if report == nil {
		return nil, apperr.NotFound("medical_report", nil)
	}
	return report, nil
}

func (s *service) GetPatientHistory(ctx context.Context, patientID string, limit, offset int) ([]MedicalReport, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, total, err := s.repo.ListByPatient(ctx, patientID, limit, offset)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	return list, total, nil
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

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

func (h *Handler) Create(c *gin.Context) {
	var req CreateReportReq
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

	report, err := h.svc.CreateReport(c.Request.Context(), doctorID, req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusCreated, report)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	report, err := h.svc.GetReport(c.Request.Context(), id)
	if err != nil {
		apperr.Abort(c, err)
		return
	}
	apperr.WriteOK(c, http.StatusOK, report)
}

func (h *Handler) PatientHistory(c *gin.Context) {
	patientID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := h.svc.GetPatientHistory(c.Request.Context(), patientID, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WritePaginated(c, http.StatusOK, list, apperr.PaginationMeta{Limit: limit, Offset: offset, Total: total})
}
