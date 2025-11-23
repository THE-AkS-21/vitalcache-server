package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/THE-AkS-21/vitalcache-server/internal/http/dto"
	"github.com/THE-AkS-21/vitalcache-server/internal/store/supabase"
	"github.com/THE-AkS-21/vitalcache-server/pkg/config"
	"github.com/THE-AkS-21/vitalcache-server/pkg/jwt"
	"github.com/THE-AkS-21/vitalcache-server/pkg/utils"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type Service struct {
	secrets       *config.SecretPayload
	jwt           jwt.JWTKeySource
	users         *supabase.UsersStore
	doctors       *supabase.DoctorsStore
	developers    *supabase.DevelopersStore
	hospitalStaff *supabase.HospitalStaffStore
}

// NewService accepts all role-specific stores for RBAC
func NewService(
	secrets *config.SecretPayload,
	jwt jwt.JWTKeySource,
	users *supabase.UsersStore,
	doctors *supabase.DoctorsStore,
	developers *supabase.DevelopersStore,
	hospitalStaff *supabase.HospitalStaffStore,
) *Service {
	return &Service{
		secrets:       secrets,
		jwt:           jwt,
		users:         users,
		doctors:       doctors,
		developers:    developers,
		hospitalStaff: hospitalStaff,
	}
}

// Register hashes the password and calls the RPC to create User + Profile atomically
func (s *Service) Register(ctx context.Context, req dto.RegisterRequest) error {
	// 1. Hash the password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}

	// 2. Default to "patient" role if not specified
	role := req.Role
	if role == "" {
		role = "patient"
	}

	// 3. Determine designation based on role
	designation := req.Designation
	if role == "patient" {
		designation = "patient" // Patients have fixed designation
	} else if designation == "" {
		// Default designations for other roles
		switch role {
		case "developer":
			designation = "intern"
		case "doctor":
			designation = "general_physician"
		case "hospital_staff":
			designation = "clerk"
		}
	}

	// 4. Call Supabase RPC (handle_new_user_registration)
	_, err = s.users.RegisterViaRPC(
		ctx,
		req.Email,
		hash,
		role,
		req.Name,
		designation,
		role, // profileType
	)
	return err
}

// Login verifies email+password and fetches role+designation from database
func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (Tokens, error) {
	// 1. Fetch User by Email
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		// DEBUG: Log the actual error
		println("DEBUG: GetByEmail error:", err.Error())
		return Tokens{}, errors.New("invalid credentials")
	}
	if u == nil {
		println("DEBUG: User is nil")
		return Tokens{}, errors.New("invalid credentials")
	}
	println("DEBUG: Found user:", u.Email, "role:", u.Role, "hash:", u.PasswordHash[:20]+"...")

	// 2. Check Password Hash
	println("DEBUG: Attempting password check for:", req.Password)
	println("DEBUG: Against hash:", u.PasswordHash)
	checkResult := utils.CheckPasswordHash(req.Password, u.PasswordHash)
	println("DEBUG: Check result:", checkResult)
	if !checkResult {
		println("DEBUG: Password check failed")
		return Tokens{}, errors.New("invalid credentials")
	}
	println("DEBUG: Password check passed")

	// 3. Fetch designation from role-specific table
	println("DEBUG: Starting designation fetch for role:", u.Role)
	var designation string
	var permissions map[string]bool
	var doctorID *int

	switch u.Role {
	case "doctor":
		println("DEBUG: Fetching doctor profile...")
		doc, err := s.doctors.GetByUserID(u.ID)
		if err != nil {
			println("DEBUG: Doctor fetch error:", err.Error())
			return Tokens{}, errors.New("doctor profile not found")
		}
		println("DEBUG: Doctor fetched, designation:", doc.Designation)
		designation = doc.Designation // e.g., "cardiologist" (lowercase)
		dID := int(doc.ID)
		doctorID = &dID

	case "developer":
		println("DEBUG: Fetching developer profile...")
		dev, err := s.developers.GetByUserID(u.ID)
		if err != nil {
			println("DEBUG: Developer fetch error:", err.Error())
			return Tokens{}, errors.New("developer profile not found")
		}
		println("DEBUG: Developer fetched, role:", dev.Role)
		designation = dev.Role // e.g., "godfather", "junior"

		// If Junior, load dynamic permissions from JSONB
		if designation == "junior" && len(dev.Permissions) > 0 {
			if err := json.Unmarshal(dev.Permissions, &permissions); err != nil {
				println("DEBUG: Permissions unmarshal error:", err.Error())
				// Log error but don't fail login
				permissions = make(map[string]bool)
			} else {
				println("DEBUG: Loaded permissions:", permissions)
			}
		}

	case "hospital_staff":
		println("DEBUG: Fetching hospital staff profile...")
		staff, err := s.hospitalStaff.GetByUserID(u.ID)
		if err != nil {
			println("DEBUG: Hospital staff fetch error:", err.Error())
			return Tokens{}, errors.New("hospital staff profile not found")
		}
		println("DEBUG: Staff fetched, designation:", staff.Designation)
		designation = staff.Designation // e.g., "admin", "receptionist"

	case "patient":
		println("DEBUG: Patient role, setting designation to 'patient'")
		designation = "patient" // Patients have fixed designation

	default:
		println("DEBUG: Unknown role:", u.Role)
		return Tokens{}, errors.New("unknown user role")
	}

	println("DEBUG: Final designation:", designation)
	println("DEBUG: Generating tokens...")
	// 4. Generate Tokens with role + designation + permissions (for juniors)
	access, err := s.signAccess(int(u.ID), u.Role, designation, permissions, doctorID, 15*time.Minute)
	if err != nil {
		println("DEBUG: signAccess error:", err.Error())
		return Tokens{}, err
	}
	println("DEBUG: Access token generated")
	refresh, err := s.signRefresh(int(u.ID), 30*24*time.Hour)
	if err != nil {
		println("DEBUG: signRefresh error:", err.Error())
		return Tokens{}, err
	}
	println("DEBUG: Refresh token generated, login successful!")

	return Tokens{AccessToken: access, RefreshToken: refresh}, err
}

// Refresh validates the refresh token and issues a new pair
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	if refreshToken == "" {
		return Tokens{}, errors.New("missing refresh token")
	}
	token, err := s.parseHS256(refreshToken)
	if err != nil {
		return Tokens{}, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok {
		return Tokens{}, errors.New("invalid claims")
	}

	subFloat, _ := claims["sub"].(float64)
	sub := int(subFloat)

	// Note: In a strict system, you might re-fetch the user here to ensure they aren't banned.
	// For refresh, we use empty designation and nil permissions since it's not critical for refresh flow
	access, err := s.signAccess(sub, "user", "", nil, nil, 15*time.Minute)
	if err != nil {
		return Tokens{}, err
	}

	newRefresh, err := s.signRefresh(sub, 30*24*time.Hour)
	return Tokens{AccessToken: access, RefreshToken: newRefresh}, err
}

// --- Helpers (JWT Signing) ---

func (s *Service) activeKey() ([]byte, error) {
	if s.jwt == nil {
		return nil, errors.New("jwt key source not initialized")
	}
	_, key := s.jwt.ActiveKey()
	if len(key) == 0 {
		return nil, errors.New("no active jwt key")
	}
	return key, nil
}

func (s *Service) signAccess(sub int, role string, designation string, permissions map[string]bool, doctorID *int, ttl time.Duration) (string, error) {
	key, err := s.activeKey()
	if err != nil {
		return "", err
	}
	claims := jwtv4.MapClaims{
		"sub":         sub,
		"role":        role,
		"designation": designation,
		"exp":         time.Now().Add(ttl).Unix(),
		"iat":         time.Now().Unix(),
	}

	// Add dynamic permissions for Junior developers
	if len(permissions) > 0 {
		claims["perms"] = permissions
	}

	if doctorID != nil {
		claims["doctor_id"] = *doctorID
	}
	t := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
	return t.SignedString(key)
}

func (s *Service) signRefresh(sub int, ttl time.Duration) (string, error) {
	key, err := s.activeKey()
	if err != nil {
		return "", err
	}
	claims := jwtv4.MapClaims{
		"sub": sub,
		"typ": "refresh",
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	}
	t := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
	return t.SignedString(key)
}

func (s *Service) parseHS256(raw string) (*jwtv4.Token, error) {
	key, err := s.activeKey()
	if err != nil {
		return nil, err
	}
	return jwtv4.Parse(raw, func(t *jwtv4.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return key, nil
	})
}
