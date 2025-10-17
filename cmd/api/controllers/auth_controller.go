package controllers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/models"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/utils"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userStore      *store.UserStore
	doctorStore    *store.DoctorStore
	developerStore *store.DeveloperStore
	cfg            *config.Config
}

func NewAuthController(userStore *store.UserStore, doctorStore *store.DoctorStore, developerStore *store.DeveloperStore, cfg *config.Config) *AuthController {
	return &AuthController{
		userStore:      userStore,
		doctorStore:    doctorStore,
		developerStore: developerStore,
		cfg:            cfg,
	}
}

type RegisterRequest struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        string `json:"role" binding:"required"` // e.g., "doctor", "god"
	Designation string `json:"designation,omitempty"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error during registration"})
		return
	}

	// Corrected Logic: Determine the primary role for the 'users' table
	primaryRole := strings.ToLower(req.Role)
	isDeveloperRole := false
	switch primaryRole {
	case "god", "godfather", "senior", "junior", "intern":
		primaryRole = "developer" // Set the primary role to 'developer'
		isDeveloperRole = true
	}

	// Step 1: Create the central user account with the correct primary role
	newUser := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         primaryRole,
	}
	createdUser, err := ac.userStore.Create(newUser)
	if err != nil {
		slog.Error("Failed to create user", "error", err)
		c.JSON(http.StatusConflict, gin.H{"error": "Email may already be in use"})
		return
	}

	// Step 2: Create the role-specific profile
	switch createdUser.Role {
	case "doctor":
		newDoctorProfile := models.Doctor{
			Name:        req.Name,
			Designation: req.Designation,
			UserID:      createdUser.ID,
		}
		_, err := ac.doctorStore.Create(newDoctorProfile)
		if err != nil {
			slog.Error("Failed to create doctor profile", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create doctor profile"})
			return
		}
	case "developer":
		if !isDeveloperRole {
			// Handle case where user just specifies "developer"
			req.Role = "junior" // Default to junior if no specific role is given
		}
		newDevProfile := models.Developer{
			Name:   req.Name,
			Role:   req.Role, // Use the original, specific role (e.g., "god")
			UserID: createdUser.ID,
		}
		_, err := ac.developerStore.Create(newDevProfile)
		if err != nil {
			slog.Error("Failed to create developer profile", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create developer profile"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role specified"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user_id": createdUser.ID})
}

// ... Login function remains the same ...
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
	user, err := ac.userStore.GetByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	token, err := utils.GenerateToken(user.ID, user.Role, ac.cfg.JWTSecret)
	if err != nil {
		slog.Error("Failed to generate JWT", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process login"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role})
}
