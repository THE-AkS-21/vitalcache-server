package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	apperr "github.com/THE-AkS-21/vitalcache-server/internal/pkg/errors"
	"github.com/THE-AkS-21/vitalcache-server/internal/pkg/logger"
	pkgjwt "github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
)

const (
	refreshKeyPrefix = "refresh:"
	refreshTTL       = 7 * 24 * time.Hour
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

func (s *Service) Register(ctx context.Context, req RegisterRequest, hospitalID *string) error {
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

	role := strings.ToUpper(req.Role)
	if role != "PATIENT" && role != "DOCTOR" && role != "STAFF" {
		role = "PATIENT" // default fallback
	}

	var roleName, designationName string
	if role == "DOCTOR" {
		roleName = "Doctor"
		designationName = req.Designation
		if designationName == "" || designationName == "GENERAL_PHYSICIAN" || designationName == "DOCTOR" {
			designationName = "General Physician" // sensible default
		}
	} else if role == "STAFF" {
		roleName = "Staff"
		designationName = req.Designation
		if designationName == "" {
			designationName = "Staff"
		}
	} else {
		roleName = "Patient"
		designationName = "Patient"
	}

	// Create user with the requested role
	if err := s.users.CreateUserTransaction(ctx, u, roleName, designationName, hospitalID); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "already registered") {
			return apperr.Conflict("email or phone number already registered", err)
		}
		return apperr.BadRequest(err.Error())
	}

	s.log.Info("user registered", zap.String("email", req.Email), zap.String("role", role))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Invite Flow
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) GenerateInvite(ctx context.Context, req InviteRequest) (string, error) {
	key, err := s.activeKey()
	if err != nil {
		return "", err
	}

	mapClaims := jwtv4.MapClaims{
		"email":       strings.ToLower(strings.TrimSpace(req.Email)),
		"role":        strings.ToUpper(req.Role),
		"designation": req.Designation,
		"exp":         time.Now().Add(24 * time.Hour).Unix(), // 24-hour expiry
		"iat":         time.Now().Unix(),
		"type":        "invite",
	}

	tok := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, mapClaims)
	return tok.SignedString(key)
}

func (s *Service) AcceptInvite(ctx context.Context, req AcceptInviteRequest) error {
	key, err := s.activeKey()
	if err != nil {
		return err
	}

	// 1. Verify and decode the invite token
	token, err := jwtv4.Parse(req.Token, func(token *jwtv4.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil || !token.Valid {
		return apperr.Unauthorized("invalid or expired invite link")
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok || claims["type"] != "invite" {
		return apperr.Unauthorized("invalid token type")
	}

	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)
	designation, _ := claims["designation"].(string)

	if email == "" || role == "" {
		return apperr.BadRequest("invite token is missing required fields")
	}

	// 2. Hash password and prepare user
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal(fmt.Errorf("hash password: %w", err))
	}

	u := &User{
		Email:        email, // Email strictly from the token
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PhoneNumber:  req.PhoneNumber,
		DateOfBirth:  req.DateOfBirth,
		Gender:       req.Gender,
	}

	// 3. Create the user with the role guaranteed by the signed token
	if err := s.users.CreateUserTransaction(ctx, u, role, designation, nil); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "already registered") {
			return apperr.Conflict("email is already registered", err)
		}
		return apperr.BadRequest(err.Error())
	}

	s.log.Info("user accepted invite", zap.String("email", email), zap.String("role", role))
	return nil
}

func (s *Service) UpdatePassword(ctx context.Context, userID string, req UpdatePasswordRequest) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil || u == nil {
		return apperr.Unauthorized("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperr.Unauthorized("incorrect old password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.Internal(fmt.Errorf("hash password: %w", err))
	}

	if err := s.users.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return apperr.Internal(err)
	}

	s.log.Info("user updated password", zap.String("user_id", userID))
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

	ttl := 30 * time.Minute
	refreshDur := 7 * 24 * time.Hour

	if strings.EqualFold(profile.Role, "Tester") {
		ttl = 5 * time.Minute
		refreshDur = 5 * time.Minute
	} else if strings.EqualFold(profile.Role, "Doctor") || strings.EqualFold(profile.Role, "Admin") || strings.EqualFold(profile.Designation, "Godfather") {
		ttl = 24 * time.Hour
		refreshDur = 24 * time.Hour
	}

	access, err := s.signAccess(claims, ttl)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	refresh, err := s.issueRefresh(ctx, u.ID, refreshDur)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	s.log.Info("user logged in", zap.String("user_id", u.ID), zap.String("role", profile.Role))
	return TokenPair{AccessToken: access, RefreshToken: refresh, RefreshTTL: int(refreshDur.Seconds())}, nil
}

func (s *Service) GoogleLogin(ctx context.Context, req GoogleLoginRequest) (TokenPair, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = "placeholder-client-id"
	}

	// For local dev without a real token/client ID, we might allow a bypass if DEV_INSECURE_COOKIES is set,
	// but the user wants actual validation to happen.
	payload, err := idtoken.Validate(ctx, req.Token, clientID)
	if err != nil {
		s.log.Error("google token validation failed", zap.Error(err))
		return TokenPair{}, apperr.Unauthorized("invalid google token")
	}

	emailRaw, ok := payload.Claims["email"].(string)
	if !ok || emailRaw == "" {
		return TokenPair{}, apperr.Unauthorized("google token missing email")
	}
	email := strings.ToLower(strings.TrimSpace(emailRaw))

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	if u == nil {
		// Auto-register Patient if not found
		// Generating a random password since they login via Google
		u = &User{
			Email:     email,
			FirstName: strings.Split(email, "@")[0],
			LastName:  "",
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
		u.PasswordHash = string(hash)

		if err := s.users.CreateUserTransaction(ctx, u, "Patient", "Patient", nil); err != nil {
			return TokenPair{}, apperr.Internal(fmt.Errorf("auto-register patient: %v", err))
		}
	}

	profile, err := s.profiles.GetProfileAndPermissions(ctx, u.ID)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	// Prevent auto-registering uninvited doctors/staff (they would have the Patient role if just auto-registered)
	// But since we just auto-registered them as Patient, it's fine.

	claims := Claims{
		UserID:      u.ID,
		Role:        profile.Role,
		Designation: profile.Designation,
		Permissions: profile.Permissions,
		DoctorID:    profile.DoctorID,
		PatientID:   profile.PatientID,
	}

	ttl := 30 * time.Minute
	refreshDur := 7 * 24 * time.Hour

	if strings.EqualFold(profile.Role, "Tester") {
		ttl = 5 * time.Minute
		refreshDur = 5 * time.Minute
	} else if strings.EqualFold(profile.Role, "Doctor") || strings.EqualFold(profile.Role, "Admin") || strings.EqualFold(profile.Designation, "Godfather") {
		ttl = 24 * time.Hour
		refreshDur = 24 * time.Hour
	}

	access, err := s.signAccess(claims, ttl)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	refresh, err := s.issueRefresh(ctx, u.ID, refreshDur)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	s.log.Info("user logged in via google", zap.String("user_id", u.ID), zap.String("role", profile.Role))
	return TokenPair{AccessToken: access, RefreshToken: refresh, RefreshTTL: int(refreshDur.Seconds())}, nil
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

	ttl := 30 * time.Minute
	refreshDur := 7 * 24 * time.Hour

	if strings.EqualFold(profile.Role, "Tester") {
		ttl = 5 * time.Minute
		refreshDur = 5 * time.Minute
	} else if strings.EqualFold(profile.Role, "Doctor") || strings.EqualFold(profile.Role, "Admin") || strings.EqualFold(profile.Designation, "Godfather") {
		ttl = 24 * time.Hour
		refreshDur = 24 * time.Hour
	}

	access, err := s.signAccess(claims, ttl)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	newRefresh, err := s.issueRefresh(ctx, u.ID, refreshDur)
	if err != nil {
		return TokenPair{}, apperr.Internal(err)
	}

	return TokenPair{AccessToken: access, RefreshToken: newRefresh, RefreshTTL: int(refreshDur.Seconds())}, nil
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

func (s *Service) signAccess(c Claims, ttl time.Duration) (string, error) {
	key, err := s.activeKey()
	if err != nil {
		return "", err
	}

	mapClaims := jwtv4.MapClaims{
		"sub":         c.UserID,
		"user_id":     c.UserID,
		"role":        c.Role,
		"designation": c.Designation,
		"exp":         time.Now().Add(ttl).Unix(),
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

func (s *Service) issueRefresh(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	raw := uuid.New().String()
	h := hashToken(raw)
	key := refreshKeyPrefix + h

	if err := s.rdb.Set(ctx, key, userID, ttl).Err(); err != nil {
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
