package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
)

// MeHandler serves GET /api/v1/profiles/me
// No DB call needed — all data is already decoded from the JWT claims
// by the Auth middleware and stored in the Gin context.
type MeHandler struct{}

func NewMeHandler() *MeHandler { return &MeHandler{} }

// ProfileResponse mirrors the frontend GoProfileResponse interface.
type ProfileResponse struct {
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	Designation string   `json:"designation,omitempty"`
	Permissions []string `json:"permissions"`
	DoctorID    *string  `json:"doctor_id,omitempty"`
	PatientID   *string  `json:"patient_id,omitempty"`
}

func (h *MeHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get(middleware.CtxUserID)
	role, _ := c.Get(middleware.CtxRole)
	designation, _ := c.Get(middleware.CtxDesignation)
	permsRaw, _ := c.Get(middleware.CtxPermissions)
	doctorIDRaw, _ := c.Get(middleware.CtxDoctorID)
	patientIDRaw, _ := c.Get(middleware.CtxPatientID)

	userIDStr, _ := userID.(string)
	if userIDStr == "" {
		apperr.Abort(c, apperr.Unauthorized("user_id not found in token"))
		return
	}

	perms, _ := permsRaw.([]string)
	if perms == nil {
		perms = []string{}
	}

	resp := ProfileResponse{
		UserID:      userIDStr,
		Role:        strOrEmpty(role),
		Designation: strOrEmpty(designation),
		Permissions: perms,
	}

	if did := strOrEmpty(doctorIDRaw); did != "" {
		resp.DoctorID = &did
	}
	if pid := strOrEmpty(patientIDRaw); pid != "" {
		resp.PatientID = &pid
	}

	apperr.WriteOK(c, http.StatusOK, resp)
}

func strOrEmpty(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
