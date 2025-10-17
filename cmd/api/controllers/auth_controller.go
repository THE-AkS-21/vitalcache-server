package controllers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/utils"
	"github.com/gin-gonic/gin"
	supa "github.com/supabase-community/supabase-go"
)

type AuthController struct {
	userStore *store.UserStore
	db        *supa.Client
	cfg       *config.Config
}

func NewAuthController(userStore *store.UserStore, db *supa.Client, cfg *config.Config) *AuthController {
	return &AuthController{userStore: userStore, db: db, cfg: cfg}
}

type RegisterRequest struct {
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        string `json:"role" binding:"required"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}

	primaryRole := strings.ToLower(req.Role)
	profileType := ""
	profileSpecificRole := ""

	switch primaryRole {
	case "doctor":
		profileType = "doctor"
		profileSpecificRole = req.Designation
	case "god", "godfather", "senior", "junior", "intern":
		profileType = "developer"
		profileSpecificRole = primaryRole
		primaryRole = "developer"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role specified"})
		return
	}

	payload := map[string]interface{}{
		"user_email":            req.Email,
		"user_password_hash":    hashedPassword,
		"user_primary_role":     primaryRole,
		"profile_name":          req.Name,
		"profile_specific_role": profileSpecificRole,
		"profile_type":          profileType,
	}

	// ✅ v0.0.4: RPC returns string, not error
	result := ac.db.Rpc("handle_new_user_registration", "public", payload)
	slog.Info("RPC result", "data", result)

	newUser, err := ac.userStore.GetByEmail(req.Email)
	if err != nil {
		slog.Error("Failed to fetch newly created user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User created but failed to fetch details"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user_id": newUser.ID,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"role":  user.Role,
	})
}
