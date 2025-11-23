package routes

import (
	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/kafka"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/redis"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// Register wires all API routes and dependencies into the gin.Engine.
func Register(r *gin.Engine, db *supabase.Client, ks jwt.JWTKeySource, secrets *config.SecretPayload, q queue.Client, rdb *redis.Client, kp *kafka.Producer) {
	deps := hdeps.New(db, ks, secrets, q, rdb, kp)

	api := r.Group("/api")
	{
		// auth (server-validated, httpOnly refresh cookie)
		api.POST("/auth/register", handlers.Register(deps))
		api.POST("/auth/login", handlers.Login(deps))
		api.POST("/auth/refresh", handlers.Refresh(deps))
		api.POST("/auth/logout", handlers.Logout(deps))
	}

	v1 := r.Group("/api/v1")
	{
		// JWT auth middleware (sets user_id:int, user_role:string, doctor_id:int)
		v1.Use(middlewares.Auth(deps.JWT))

		v1.GET("/profiles/me", handlers.MyProfile(deps))

		// All authenticated users can view medicines
		v1.GET("/medicines", handlers.ListMedicines(deps))

		// ✅ shared search route (doctor + developer)
		search := v1.Group("/")
		search.Use(middlewares.RequireRole("doctor", "developer"))
		{
			search.GET("/patients/search", handlers.SearchPatients(deps))
		}

		// ✅ doctor-only restricted endpoints
		doctor := v1.Group("/")
		doctor.Use(middlewares.RequireDoctor())
		{
			doctor.POST("/patients", handlers.CreatePatient(deps))
			doctor.GET("/patients/:id", handlers.GetPatientByID(deps))
			doctor.PATCH("/patients/:id", handlers.UpdatePatient(deps))
			doctor.GET("/patients/:id/prescriptions", handlers.PatientPrescriptionHistory(deps))
			doctor.POST("/prescriptions", handlers.CreatePrescription(deps))
		}

		// developers-only admin endpoints (no PHI response)
		dev := v1.Group("/admin")
		dev.Use(middlewares.RequireDeveloper())
		{
			// future: manage patient_doctors links
		}
	}
}
