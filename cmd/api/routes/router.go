package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	supa "github.com/supabase-community/supabase-go"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/config"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/controllers"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/middleware"
	"github.com/THE-AkS-21/vitalcache-server/cmd/api/store"
)

func SetupRouter(db *supa.Client, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// ✅ Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // Restrict this in production!
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	r.Use(cors.New(corsConfig))

	// ✅ Initialize stores
	userStore := store.NewUserStore(db)
	doctorStore := store.NewDoctorStore(db)
	developerStore := store.NewDeveloperStore(db)
	patientStore := store.NewPatientStore(db)
	medicineStore := store.NewMedicineStore(db)

	// ✅ Initialize controllers
	authController := controllers.NewAuthController(userStore, doctorStore, developerStore, cfg)
	patientController := controllers.NewPatientController(patientStore)
	doctorController := controllers.NewDoctorController(doctorStore)
	medicineController := controllers.NewMedicineController(medicineStore)

	// ✅ Group API routes
	api := r.Group("/api")
	{
		// 🔓 Public Authentication Routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
		}

		// 🔓 Public Medicine Route
		//api.GET("/medicines", medicineController.GetAllMedicines)

		// 🔒 Protected Routes (JWT Required)
		v1 := api.Group("/v1")
		v1.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			// 👨‍⚕️ Doctor Routes
			v1.GET("/doctors/me", doctorController.GetMyProfile)

			// 🧑‍⚕️ Patient Routes
			v1.POST("/patients", patientController.CreatePatient)
			v1.GET("/patients/search", patientController.SearchPatients)
			v1.GET("/patients/:id", patientController.GetPatientByID)
			v1.PATCH("/patients/:id", patientController.UpdatePatient)

			// 🔓 Medicine Route
			api.GET("/medicines", medicineController.GetAllMedicines)
		}
	}

	return r
}
