package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// simpleKeySource adapts a static JWT secret string into the JWTKeySource interface
type simpleKeySource struct {
	secret []byte
}

func (s simpleKeySource) ActiveKey() (string, []byte) {
	return "v1", s.secret
}

// ✅ ADDED: Implement the missing AllKeys method to satisfy pkg/jwt.JWTKeySource
func (s simpleKeySource) AllKeys() map[string][]byte {
	return map[string][]byte{
		"v1": s.secret,
	}
}

func RegisterRoutes(router *gin.RouterGroup, db *pgxpool.Pool, rdb *redis.Client, jwtSecret string) {
	userRepo := NewUserRepository(db)
	profileRepo := NewProfileRepository(db)
	ks := simpleKeySource{secret: []byte(jwtSecret)}

	svc := NewService(userRepo, profileRepo, rdb, ks)
	h := NewHandler(svc)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", h.Login)
		authGroup.POST("/register", h.Register)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
	}
}
