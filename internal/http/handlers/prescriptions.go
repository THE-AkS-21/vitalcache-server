package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func CreatePrescription(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreatePrescriptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.WriteBadRequest(c, "invalid request", err.Error())
			return
		}
		doctorID, ok := middlewares.GetDoctorID(c)
		if !ok {
			errors.WriteForbidden(c, "doctor id missing in token", nil)
			return
		}
		req.DoctorID = doctorID // server fills from token
		token := middlewares.GetRawToken(c)
		res, err := d.Prescriptions.Create(c.Request.Context(), token, req)
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}
		c.JSON(http.StatusCreated, res)
	}
}

func PatientPrescriptionHistory(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			errors.WriteBadRequest(c, "invalid patient id", nil)
			return
		}
		var startPtr, endPtr *time.Time
		if s := c.Query("start"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				startPtr = &t
			}
		}
		if s := c.Query("end"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				endPtr = &t
			}
		}

		token := middlewares.GetRawToken(c)
		items, err := d.Prescriptions.ListByPatient(c.Request.Context(), token, patientID, startPtr, endPtr, clampLimit(c), clampOffset(c))
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	}
}
