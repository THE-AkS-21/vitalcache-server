package routes

import (
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/controllers"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/middleware"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	supa "github.com/supabase-community/supabase-go"
)

func SetupRouter(db *supa.Client, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.LoggerMiddleware())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	r.Use(cors.New(corsConfig))

	// Stores
	userStore := store.NewUserStore(db)
	doctorStore := store.NewDoctorStore(db)
	developerStore := store.NewDeveloperStore(db)
	patientStore := store.NewPatientStore(db)
	medicineStore := store.NewMedicineStore(db)

	// Controllers
	authController := controllers.NewAuthController(userStore, db, cfg)
	profileController := controllers.NewProfileController(doctorStore, developerStore)
	patientController := controllers.NewPatientController(patientStore)
	medicineController := controllers.NewMedicineController(medicineStore)

	api := r.Group("/api")
	{
		// Public routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
		}
		// Corrected: Moved medicine route to be public
		api.GET("/medicines", medicineController.GetAllMedicines)

		// Protected routes
		v1 := api.Group("/v1")
		v1.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			v1.GET("/profiles/me", profileController.GetMyProfile)

			// Patient management (ensure logic in controller checks for 'doctor' role if needed)
			v1.POST("/patients", patientController.CreatePatient)
			v1.GET("/patients/search", patientController.SearchPatients)
			v1.GET("/patients/:id", patientController.GetPatientByID)
			v1.PATCH("/patients/:id", patientController.UpdatePatient)
		}
	}
	return r
}
