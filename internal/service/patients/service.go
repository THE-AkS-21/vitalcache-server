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

func (s *Service) Create(ctx context.Context, token string, req dto.CreatePatientRequest) (domain.Patient, error) {
	p := domain.Patient{
		Name:         req.Name,
		Age:          req.Age,
		Sex:          req.Sex,
		MobileNumber: req.MobileNumber,
		DoctorID:     uint(*req.DoctorID),
		CreatedAt:    time.Now(),
	}
	out, err := s.store.Create(ctx, token, &p)
	if err != nil {
		return domain.Patient{}, err
	}
	return *out, nil
}

//func (s *Service) GetByMobile(ctx context.Context, userID int, role, mobile string) ([]domain.Patient, error) {
//	return s.store.SearchByMobile(ctx, userID, role, mobile, 50, 0)
//}

func (s *Service) SearchByMobile(
	ctx context.Context,
	token string,
	mobile string,
	limit, offset int,
) ([]domain.Patient, error) {
	return s.store.SearchByMobile(ctx, token, mobile, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, token string, id int) (*domain.Patient, error) {
	return s.store.GetByID(ctx, token, id)
}

func (s *Service) UpdatePartial(ctx context.Context, token string, id int, req dto.UpdatePatientRequest) (domain.Patient, error) {
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
	updates["updated_at"] = time.Now()
	out, err := s.store.UpdatePartial(ctx, token, id, updates)
	if err != nil {
		return domain.Patient{}, err
	}
	return *out, nil
}
