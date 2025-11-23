package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func CreatePatient(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreatePatientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.WriteBadRequest(c, "invalid request", err.Error())
			return
		}
		docID, ok := middlewares.GetDoctorID(c)
		if !ok {
			errors.WriteForbidden(c, "doctor id missing in token", nil)
			return
		}
		req.DoctorID = &docID // ensure ownership under RLS
		token := middlewares.GetRawToken(c)
		resp, err := d.Patients.Create(c.Request.Context(), token, req)
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func GetPatientByID(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			errors.WriteBadRequest(c, "invalid patient id", nil)
			return
		}

		// Cache key: patient:{id}
		cacheKey := "patient:" + strconv.Itoa(id)
		if d.Redis != nil {
			val, err := d.Redis.Client.Get(c.Request.Context(), cacheKey).Result()
			if err == nil {
				c.Header("X-Cache", "HIT")
				c.Data(http.StatusOK, "application/json", []byte(val))
				return
			}
		}

		token := middlewares.GetRawToken(c)
		item, err := d.Patients.GetByID(c.Request.Context(), token, id)
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}

		if d.Redis != nil {
			// Cache for 5 minutes
			if b, err := json.Marshal(item); err == nil {
				d.Redis.Client.Set(c.Request.Context(), cacheKey, b, 5*time.Minute)
			}
		}

		c.JSON(http.StatusOK, item)
	}
}

func UpdatePatient(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			errors.WriteBadRequest(c, "invalid patient id", nil)
			return
		}
		var req dto.UpdatePatientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errors.WriteBadRequest(c, "invalid request", err.Error())
			return
		}
		token := middlewares.GetRawToken(c)
		item, err := d.Patients.UpdatePartial(c.Request.Context(), token, id, req)
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}
		c.JSON(http.StatusOK, item)
	}
}

func SearchPatients(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		mobile := c.Query("mobile")
		if mobile == "" {
			errors.WriteBadRequest(c, "mobile query param required", nil)
			return
		}

		token := middlewares.GetRawToken(c)

		items, err := d.Patients.SearchByMobile(
			c.Request.Context(), token, mobile,
			clampLimit(c), clampOffset(c),
		)
		if err != nil {
			errors.WriteInternal(c, err)
			return
		}

		c.JSON(http.StatusOK, items)
	}
}
