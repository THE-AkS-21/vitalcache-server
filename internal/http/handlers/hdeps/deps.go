package hdeps

import (
	"context"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/kafka"
	"github.com/THE-AkS-21/vitalcache-server/internal/infra/redis"
	"github.com/THE-AkS-21/vitalcache-server/internal/queue"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/prescriptions"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

type Deps struct {
	DB  *supabase.Client
	JWT jwt.JWTKeySource

	Auth          *auth.Service
	Patients      PatientsService
	Prescriptions PrescriptionsService
	Medicines     MedicinesStore

	Secrets *config.SecretPayload
	Q       queue.Client
	Redis   *redis.Client
	Kafka   *kafka.Producer
}

func New(db *supabase.Client, ks jwt.JWTKeySource, secrets *config.SecretPayload, q queue.Client, rdb *redis.Client, kp *kafka.Producer) Deps {
	// stores
	patientsStore := supabase.NewPatientsStore(db)
	prescriptionsStore := supabase.NewPrescriptionsStore(db)
	medStore := supabase.NewMedicinesStore(db)
	usersStore := supabase.NewUsersStore(db)
	docStore := supabase.NewDoctorsStore(db)
	developersStore := supabase.NewDevelopersStore(db)       // NEW: For developer RBAC
	hospitalStaffStore := supabase.NewHospitalStaffStore(db) // NEW: For hospital staff RBAC

	// services
	patientsSvc := patients.NewService(patientsStore)
	prescriptionsSvc := prescriptions.NewService(patientsStore, prescriptionsStore, q, kp)
	authSvc := auth.NewService(secrets, ks, usersStore, docStore, developersStore, hospitalStaffStore) // UPDATED

	return Deps{
		DB:            db,
		JWT:           ks,
		Secrets:       secrets,
		Q:             q,
		Medicines:     medStore,
		Patients:      patientsSvc,
		Prescriptions: prescriptionsSvc,
		Auth:          authSvc,
		Redis:         rdb,
		Kafka:         kp,
	}
}

type PatientsService interface {
	Create(ctx context.Context, token string, req dto.CreatePatientRequest) (domain.Patient, error)
	GetByID(ctx context.Context, token string, id int) (*domain.Patient, error)
	UpdatePartial(ctx context.Context, token string, id int, req dto.UpdatePatientRequest) (domain.Patient, error)
	SearchByMobile(ctx context.Context, token string, mobile string, limit, offset int) ([]domain.Patient, error)
}

type PrescriptionsService interface {
	Create(ctx context.Context, token string, req dto.CreatePrescriptionRequest) (domain.Prescription, error)
	ListByPatient(ctx context.Context, token string, patientID int, start, end *time.Time, limit, offset int) ([]domain.Prescription, error)
}

type MedicinesStore interface {
	List(ctx context.Context, token string, limit, offset int) ([]domain.Medicine, error)
}
