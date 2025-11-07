package patients

import (
	"context"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
)

type Service struct{ store *supabase.PatientsStore }

func NewService(s *supabase.PatientsStore) *Service { return &Service{store: s} }

func (s *Service) Create(ctx context.Context, req dto.CreatePatientRequest, doctorID uint) (domain.Patient, error) {
	p := domain.Patient{
		Name:         req.Name,
		Age:          req.Age,
		Sex:          req.Sex,
		MobileNumber: req.MobileNumber,
		Email:        req.Email,
		DoctorID:     doctorID,
		CreatedAt:    time.Now(),
	}
	return s.store.Create(ctx, p)
}

func (s *Service) GetByMobile(ctx context.Context, mobile string, doctorID uint) ([]domain.Patient, error) {
	return s.store.GetByMobile(ctx, mobile, doctorID)
}

func (s *Service) GetByID(ctx context.Context, id, doctorID uint) (*domain.Patient, error) {
	return s.store.GetByID(ctx, id, doctorID)
}

func (s *Service) Update(ctx context.Context, id, doctorID uint, req dto.UpdatePatientRequest) (domain.Patient, error) {
	updates := map[string]any{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Age > 0 {
		updates["age"] = req.Age
	}
	if req.Sex != "" {
		updates["sex"] = req.Sex
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	updates["updated_at"] = time.Now()
	return s.store.Update(ctx, id, doctorID, updates)
}
