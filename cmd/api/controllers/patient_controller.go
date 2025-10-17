package controllers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/gin-gonic/gin"
)

type PatientController struct {
	store *store.PatientStore
}

func NewPatientController(store *store.PatientStore) *PatientController {
	return &PatientController{store: store}
}

// ... CreatePatient function (no changes) ...
type CreatePatientRequest struct {
	Name         string `json:"name" binding:"required"`
	Age          int    `json:"age" binding:"required,gt=0"`
	Sex          string `json:"sex"`
	MobileNumber string `json:"mobile_number" binding:"required"`
	Email        string `json:"email,omitempty,email"`
}

func (pc *PatientController) CreatePatient(c *gin.Context) {
	var req CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	doctorID, _ := c.Get("doctor_id")
	newPatient := models.Patient{
		Name:         req.Name,
		Age:          req.Age,
		Sex:          req.Sex,
		MobileNumber: req.MobileNumber,
		Email:        req.Email,
		DoctorID:     doctorID.(uint),
	}
	createdPatient, err := pc.store.Create(newPatient)
	if err != nil {
		slog.Error("Failed to create patient in store", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create patient"})
		return
	}
	c.JSON(http.StatusCreated, createdPatient)
}

// Updated SearchPatients to use doctorID from context
func (pc *PatientController) SearchPatients(c *gin.Context) {
	mobile := c.Query("mobile")
	if mobile == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mobile query parameter is required"})
		return
	}
	doctorID, _ := c.Get("doctor_id")
	patients, err := pc.store.GetByMobile(mobile, doctorID.(uint))
	if err != nil {
		slog.Error("Failed to search patients in store", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve patients"})
		return
	}
	c.JSON(http.StatusOK, patients)
}

// New Handler: Get a single patient by their ID
func (pc *PatientController) GetPatientByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid patient ID"})
		return
	}
	doctorID, _ := c.Get("doctor_id")
	patient, err := pc.store.GetByID(uint(id), doctorID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, patient)
}

// New Handler: Update a patient's details
type UpdatePatientRequest struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Sex   string `json:"sex"`
	Email string `json:"email,omitempty,email"`
}

func (pc *PatientController) UpdatePatient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid patient ID"})
		return
	}

	var req UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert request to a map for partial updates
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Age > 0 {
		updates["age"] = req.Age
	}
	if req.Sex != "" {
		updates["sex"] = req.Sex
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No update fields provided"})
		return
	}

	doctorID, _ := c.Get("doctor_id")
	updatedPatient, err := pc.store.Update(uint(id), doctorID.(uint), updates)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedPatient)
}
