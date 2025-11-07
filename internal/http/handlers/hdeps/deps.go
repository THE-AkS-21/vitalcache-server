package hdeps

import (
	"github.com/THE-AkS-21/vitalcache-server/internal/service/auth"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/patients"
	"github.com/THE-AkS-21/vitalcache-server/internal/service/prescriptions"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

type Deps struct {
	Auth          *auth.Service
	Patients      *patients.Service
	Prescriptions *prescriptions.Service

	// Repos (read-only use in handlers)
	Users      *supabase.UsersStore
	Doctors    *supabase.DoctorsStore
	Developers *supabase.DevelopersStore
	Medicines  *supabase.MedicinesStore

	KeySource jwt.JWTKeySource
}
