package prescriptions

import (
	"net/http"
	"strconv"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/validator"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreatePrescriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Abort(c, apperr.BadRequest("invalid JSON payload"))
		return
	}
	if err := validator.Check(&req); err != nil {
		apperr.Abort(c, err)
		return
	}

	userID, _ := c.Get("user_id")
	doctorID, ok := userID.(string)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("Invalid user ID in token"))
		return
	}

	p, err := h.svc.CreatePrescription(c.Request.Context(), doctorID, req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusCreated, p)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		apperr.Abort(c, apperr.BadRequest("prescription ID is required"))
		return
	}

	p, err := h.svc.GetPrescription(c.Request.Context(), id)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusOK, p)
}

func (h *Handler) ListPatientHistory(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		apperr.Abort(c, apperr.BadRequest("invalid patient ID"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := h.svc.GetPatientHistory(c.Request.Context(), patientID, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusOK, gin.H{
		"data":  list,
		"total": total,
	})
}
