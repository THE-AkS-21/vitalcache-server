package controllers

import (
	"log/slog"
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/utils"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	store *store.DoctorStore
	cfg   *config.Config
}

func NewAuthController(store *store.DoctorStore, cfg *config.Config) *AuthController {
	return &AuthController{store: store, cfg: cfg}
}

type RegisterRequest struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Designation string `json:"designation"`
}

func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process registration"})
		return
	}

	newDoctor := models.Doctor{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Designation:  req.Designation,
	}

	createdDoctor, err := ac.store.Create(newDoctor)
	if err != nil {
		slog.Error("Failed to create doctor in store", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Email may already be in use"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Doctor registered successfully", "doctor_id": createdDoctor.ID})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doctor, err := ac.store.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, doctor.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(doctor.ID, ac.cfg.JWTSecret)
	if err != nil {
		slog.Error("Failed to generate JWT", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
