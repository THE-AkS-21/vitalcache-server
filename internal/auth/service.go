package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

const (
	refreshKeyPrefix = "refresh:"
	refreshTTL       = 7 * 24 * time.Hour
	accessTTL        = 15 * time.Minute
)

type Service struct {
	users    UserRepository
	profiles ProfileRepository
	rdb      *redis.Client
	ks       pkgjwt.JWTKeySource
	log      *zap.Logger
}

func NewService(users UserRepository, profiles ProfileRepository, rdb *redis.Client, ks pkgjwt.JWTKeySource) *Service {
	return &Service{
		users:    users,
		profiles: profiles,
		rdb:      rdb,
		ks:       ks,
		log:      logger.Named("auth.service"),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Register & Login
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) Register(ctx context.Context, req RegisterRequest) error {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal(fmt.Errorf("hash password: %w", err))
	}

	u := &User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PhoneNumber:  req.PhoneNumber,
		DateOfBirth:  req.DateOfBirth,
		Gender:       req.Gender,
	}

	// ✅ Self-registration is PATIENT-only.
	// Role and Designation from the request body are intentionally ignored.
	// Doctor/staff accounts are created through the admin invite flow.
	if err := s.users.CreateUserTransaction(ctx, u, "PATIENT", "PATIENT"); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "already registered") {
			return apperr.Conflict("email or phone number already registered", err)
		}
		return apperr.BadRequest(err.Error())
	}

	s.log.Info("user registered", zap.String("email", req.Email), zap.String("role", "PATIENT"))
	return nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (TokenPair, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil || u == nil {
		return TokenPair{}, apperr.Unauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return TokenPair{}, apperr.Unauthorized("invalid credentials")
	}

	profile, err := s.profiles.GetProfileAndPermissions(ctx, u.ID)
	if err != nil {
		s.log.Error("login: profile fetch error", zap.String("user_id", u.ID), zap.Error(err))
		return TokenPair{}, apperr.Internal(err)
	}

	claims := Claims{
		UserID:      u.ID,
		Role:        profile.Role,
		Designation: profile.Designation,
		Permissions: profile.Permissions,
		DoctorID:    profile.DoctorID,
		PatientID:   profile.PatientID,
	}

	access, err := s.signAccess(claims)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	refresh, err := s.issueRefresh(ctx, u.ID)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	s.log.Info("user logged in", zap.String("user_id", u.ID), zap.String("role", profile.Role))
	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Refresh & Logout
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) Refresh(ctx context.Context, rawRefresh string) (TokenPair, error) {
	if rawRefresh == "" {
		return TokenPair{}, apperr.Unauthorized("missing refresh token")
	}

	userID, err := s.validateRefreshToken(ctx, rawRefresh)
	if err != nil {
		return TokenPair{}, apperr.Unauthorized("invalid or expired refresh token")
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil || u == nil {
		return TokenPair{}, apperr.Unauthorized("user not found")
	}

	profile, err := s.profiles.GetProfileAndPermissions(ctx, u.ID)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	_ = s.revokeRefresh(ctx, rawRefresh)

	claims := Claims{
		UserID:      u.ID,
		Role:        profile.Role,
		Designation: profile.Designation,
		Permissions: profile.Permissions,
		DoctorID:    profile.DoctorID,
		PatientID:   profile.PatientID,
	}

	access, err := s.signAccess(claims)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	newRefresh, err := s.issueRefresh(ctx, u.ID)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	return TokenPair{AccessToken: access, RefreshToken: newRefresh}, nil
}

func (s *Service) Logout(ctx context.Context, rawRefresh string) {
	if rawRefresh != "" {
		_ = s.revokeRefresh(ctx, rawRefresh)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// JWT & Redis Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) activeKey() ([]byte, error) {
	_, key := s.ks.ActiveKey()
	if len(key) == 0 {
		return nil, fmt.Errorf("no active jwt key")
	}
	return key, nil
}

func (s *Service) signAccess(c Claims) (string, error) {
	key, err := s.activeKey()
	if err != nil {
		return "", err
	}

	mapClaims := jwtv4.MapClaims{
		"sub":         c.UserID,
		"user_id":     c.UserID,
		"role":        c.Role,
		"designation": c.Designation,
		"exp":         time.Now().Add(accessTTL).Unix(),
		"iat":         time.Now().Unix(),
	}
	if len(c.Permissions) > 0 {
		mapClaims["permissions"] = c.Permissions
	}
	if c.DoctorID != nil {
		mapClaims["doctor_id"] = *c.DoctorID
	}
	if c.PatientID != nil {
		mapClaims["patient_id"] = *c.PatientID
	}

	tok := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, mapClaims)
	return tok.SignedString(key)
}

func (s *Service) issueRefresh(ctx context.Context, userID string) (string, error) {
	raw := uuid.New().String()
	h := hashToken(raw)
	key := refreshKeyPrefix + h

	if err := s.rdb.Set(ctx, key, userID, refreshTTL).Err(); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}
	return raw, nil
}

func (s *Service) validateRefreshToken(ctx context.Context, raw string) (string, error) {
	h := hashToken(raw)
	key := refreshKeyPrefix + h

	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return "", fmt.Errorf("get refresh token: %w", err)
	}

	return val, nil
}

func (s *Service) revokeRefresh(ctx context.Context, raw string) error {
	h := hashToken(raw)
	return s.rdb.Del(ctx, refreshKeyPrefix+h).Err()
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
