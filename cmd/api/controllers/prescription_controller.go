package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/services"
	"github.com/gin-gonic/gin"
)

type PrescriptionController struct {
	prescriptionService *services.PrescriptionService
}

func NewPrescriptionController(ps *services.PrescriptionService) *PrescriptionController {
	// Corrected: Return an instance of the struct "PrescriptionController"
	return &PrescriptionController{prescriptionService: ps}
}

func (pc *PrescriptionController) SendPrescription(c *gin.Context) {
	patientIDStr := c.PostForm("patientId")
	patientID, err := strconv.ParseUint(patientIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid patientId provided"})
		return
	}

	file, err := c.FormFile("prescriptionFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	userID, _ := c.Get("user_id")

	// Create a unique temporary path to save the file
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d-%s", time.Now().UnixNano(), file.Filename))
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file"})
		return
	}

	err = pc.prescriptionService.ProcessAndSendPrescription(uint(patientID), userID.(uint), tempPath, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process prescription: %v", err)})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Prescription has been queued for sending"})
}
