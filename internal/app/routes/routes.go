package routes

import (
	"net/http"

	"github.com/THE-AkS-21/vitalcache-server/internal/app/middlewares"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/handlers/hdeps"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/prescriptions"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/gin-gonic/gin"
	supa "github.com/supabase-community/supabase-go"
)

func Register(r *gin.Engine, db *supa.Client, ks jwt.JWTKeySource, secrets *config.SecretPayload, q queue.Client) {
	// Stores
	usersStore := supabase.NewUsersStore(db)
	doctorsStore := supabase.NewDoctorsStore(db)
	developersStore := supabase.NewDevelopersStore(db)
	patientsStore := supabase.NewPatientsStore(db)
	medicinesStore := supabase.NewMedicinesStore(db)

	// Services
	authSvc := auth.NewService(usersStore, doctorsStore, developersStore, ks)
	patSvc := patients.NewService(patientsStore)
	presSvc := prescriptions.NewService(patientsStore, db, q) // in-memory queue; no email

	// Handler deps
	deps := hdeps.Deps{
		Auth:          authSvc,
		Patients:      patSvc,
		Prescriptions: presSvc,
		Users:         usersStore,
		Doctors:       doctorsStore,
		Developers:    developersStore,
		Medicines:     medicinesStore,
		KeySource:     ks,
	}

	// Public
	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/register", handlers.Register(deps))
		auth.POST("/login", handlers.Login(deps))

		api.GET("/medicines", handlers.ListMedicines(deps))
	}

	// Protected
	v1 := api.Group("/v1")
	v1.Use(middlewares.Auth(ks))
	{
		// doctor-only routes
		doctor := v1.Group("/")
		doctor.Use(middlewares.RequireDoctor())

		doctor.GET("/profiles/me", handlers.MyProfile(deps))
		doctor.POST("/patients", handlers.CreatePatient(deps))
		doctor.GET("/patients/search", handlers.SearchPatients(deps))
		doctor.GET("/patients/:id", handlers.GetPatientByID(deps))
		doctor.PATCH("/patients/:id", handlers.UpdatePatient(deps))
		doctor.GET("/patients/:id/prescriptions", handlers.PatientPrescriptionHistory(deps))
		doctor.POST("/prescriptions/send", handlers.SendPrescription(deps))
	}

}

func RegisterPublicRoutes(r *gin.Engine) {

	// --- Health Endpoints ---
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/healthz/ready", func(c *gin.Context) {
		// TODO: Inject real checks via deps (db, secrets, queue, etc.)
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}
