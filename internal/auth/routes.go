package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/middleware"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

// simpleKeySource adapts a static JWT secret string into the JWTKeySource interface.
// Used when the full key-rotation keyring (pkg/config.SecretPayload) is not available.
type simpleKeySource struct {
	secret []byte
}

func (s simpleKeySource) ActiveKey() (string, []byte) { return "v1", s.secret }
func (s simpleKeySource) AllKeys() map[string][]byte {
	return map[string][]byte{"v1": s.secret}
}

// RegisterRoutes mounts all auth + profile routes.
// Returns the JWTKeySource so main.go can pass it to middleware.Auth.
func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client, jwtSecret string) pkgjwt.JWTKeySource {
	ks := simpleKeySource{secret: []byte(jwtSecret)}

	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	svc := NewService(userRepo, profileRepo, rdb, ks)
	h := NewHandler(svc)
	meH := NewMeHandler()

	// ── Public auth routes (no JWT required) ─────────────────────────────────
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/register", h.Register) // temporary doctor + patient registration
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
		authGroup.POST("/invites/accept", h.AcceptInvite) // public invite acceptance
		authGroup.POST("/google", h.GoogleLogin)
	}

	// ── Protected auth routes ────────────────────────────────────────────────
	authProtectedGroup := router.Group("/auth")
	authProtectedGroup.Use(middleware.Auth(ks))
	{
		// Only Admins, Doctors, Developers can generate staff invites (GodFather designation bypassed via auth middleware)
		authProtectedGroup.POST("/invites", middleware.BlockRole("TESTER"), middleware.RequireRole("ADMIN", "DOCTOR", "DEVELOPER"), h.GenerateInvite)
		// Users can update their own password
		authProtectedGroup.PUT("/password", h.UpdatePassword)
	}

	// ── Protected profile route ───────────────────────────────────────────────
	// GET /api/v1/profiles/me — called by the Next.js session proxy on every boot
	profileGroup := router.Group("/profiles")
	profileGroup.Use(middleware.Auth(ks))
	{
		profileGroup.GET("/me", meH.GetMe)
	}

	return ks
}
