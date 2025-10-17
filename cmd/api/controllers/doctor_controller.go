package controllers

import (
	"log/slog"
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/gin-gonic/gin"
)

type DoctorController struct {
	store *store.DoctorStore
}

func NewDoctorController(store *store.DoctorStore) *DoctorController {
	return &DoctorController{store: store}
}

// New Handler: Get the profile of the currently logged-in doctor
func (dc *DoctorController) GetMyProfile(c *gin.Context) {
	doctorID, exists := c.Get("doctor_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No doctor ID found in token"})
		return
	}

	doctor, err := dc.store.GetByUserID(doctorID.(uint))
	if err != nil {
		slog.Error("Failed to get doctor profile", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Doctor profile not found"})
		return
	}

	c.JSON(http.StatusOK, doctor)
}
