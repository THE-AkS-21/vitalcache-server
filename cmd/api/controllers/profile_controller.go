package controllers

import (
	"log/slog"
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/gin-gonic/gin"
)

type ProfileController struct {
	doctorStore    *store.DoctorStore
	developerStore *store.DeveloperStore
}

func NewProfileController(ds *store.DoctorStore, dev_s *store.DeveloperStore) *ProfileController {
	return &ProfileController{doctorStore: ds, developerStore: dev_s}
}

func (pc *ProfileController) GetMyProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	var profile interface{}
	var err error

	switch userRole {
	case "doctor":
		// Corrected: The method now exists on the store
		profile, err = pc.doctorStore.GetByUserID(userID.(uint))
	case "developer":
		// Corrected: The method now exists on the store
		profile, err = pc.developerStore.GetByUserID(userID.(uint))
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "No profile available for this user role"})
		return
	}

	if err != nil {
		slog.Error("Failed to get user profile", "user_id", userID, "role", userRole, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
