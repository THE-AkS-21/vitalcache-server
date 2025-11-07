package handlers

import (
	"net/http"
	"strconv"

	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/gin-gonic/gin"
)

func CreatePatient(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.CreatePatientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userIDVal, _ := c.Get("user_id")
		doctor, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}
		p, err := d.Patients.Create(c.Request.Context(), req, doctor.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create patient"})
			return
		}
		c.JSON(http.StatusCreated, p)
	}
}

func SearchPatients(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		mobile := c.Query("mobile")
		if mobile == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mobile query parameter is required"})
			return
		}
		userIDVal, _ := c.Get("user_id")
		doctor, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}
		list, err := d.Patients.GetByMobile(c.Request.Context(), mobile, doctor.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve patients"})
			return
		}
		c.JSON(http.StatusOK, list)
	}
}

func GetPatientByID(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
			return
		}
		userIDVal, _ := c.Get("user_id")
		doctor, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}
		p, err := d.Patients.GetByID(c.Request.Context(), uint(id), doctor.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func UpdatePatient(d hdeps.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
			return
		}
		var req dto.UpdatePatientRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userIDVal, _ := c.Get("user_id")
		doctor, err := d.Doctors.GetByUserID(uint(userIDVal.(uint)))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "doctor profile not found"})
			return
		}
		p, err := d.Patients.Update(c.Request.Context(), uint(id), doctor.ID, req)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}
