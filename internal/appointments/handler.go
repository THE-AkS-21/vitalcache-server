package appointments

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
	var req CreateAppointmentReq
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

	app, err := h.svc.CreateAppointment(c.Request.Context(), doctorID, req)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusCreated, app)
}

func (h *Handler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	doctorID, ok := userID.(string)
	if !ok {
		apperr.Abort(c, apperr.Unauthorized("Invalid user ID in token"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := h.svc.GetDoctorAppointments(c.Request.Context(), doctorID, limit, offset)
	if err != nil {
		apperr.Abort(c, err)
		return
	}

	apperr.WriteOK(c, http.StatusOK, gin.H{
		"data":  list,
		"total": total,
	})
}
