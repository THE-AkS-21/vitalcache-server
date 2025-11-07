package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/domain"
	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/THE-AkS-21/vitalcache-server/pkg/utils"
)

type Service struct {
	users      *supabase.UsersStore
	doctors    *supabase.DoctorsStore
	developers *supabase.DevelopersStore
	ks         jwt.JWTKeySource
}

func NewService(u *supabase.UsersStore, d *supabase.DoctorsStore, dev *supabase.DevelopersStore, ks jwt.JWTKeySource) *Service {
	return &Service{users: u, doctors: d, developers: dev, ks: ks}
}

func (s *Service) Register(ctx context.Context, req dto.RegisterRequest) (uint, error) {
	// Keep your role logic (doctor vs developer roles)
	role := strings.ToLower(req.Role)
	profileType := ""
	profileSpecificRole := ""

	switch role {
	case "doctor":
		profileType = "doctor"
		profileSpecificRole = req.Designation
	case "god", "godfather", "senior", "junior", "intern":
		profileType = "developer"
		profileSpecificRole = role
		role = "developer"
	default:
		return 0, errors.New("invalid role specified")
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return 0, err
	}

	// Prefer your existing RPC if available; otherwise insert user then profile
	uid, err := s.users.RegisterViaRPC(ctx, req.Email, hash, role, req.Name, profileSpecificRole, profileType)
	if err != nil {
		// Fallback: create directly
		u, err2 := s.users.Create(ctx, domain.User{Email: req.Email, PasswordHash: hash, Role: role})
		if err2 != nil {
			return 0, err
		}
		if role == "doctor" {
			_, _ = s.doctors.Create(ctx, domain.Doctor{Name: req.Name, Designation: req.Designation, UserID: u.ID})
		} else {
			_, _ = s.developers.Create(ctx, domain.Developer{Name: req.Name, Role: profileSpecificRole, UserID: u.ID})
		}
		return u.ID, nil
	}
	return uid, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (domain.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	if !utils.CheckPasswordHash(password, u.PasswordHash) {
		return domain.User{}, errors.New("invalid")
	}
	return *u, nil
}

func (s *Service) IssueToken(userID uint, role string, ttl time.Duration) (string, error) {
	return jwt.GenerateToken(userID, role, s.ks, ttl)
}

func (s *Service) Validate(token string) (uint, map[string]any, error) {
	uid, claims, err := jwt.ValidateToken(token, s.ks)
	return uid, claims, err
}
