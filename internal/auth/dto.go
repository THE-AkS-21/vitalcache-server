package auth

type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email"`
	Password    string  `json:"password" validate:"required,min=8"`
	FirstName   string  `json:"first_name" validate:"required,min=2"`
	LastName    string  `json:"last_name" validate:"required,min=2"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	DateOfBirth *string `json:"date_of_birth,omitempty"` // format: YYYY-MM-DD
	Gender      *string `json:"gender,omitempty" validate:"omitempty,oneof=MALE FEMALE OTHER"`
	Role        string  `json:"role" validate:"required"` // e.g., "PATIENT", "DOCTOR"
	Designation string  `json:"designation"`              // e.g., "GENERAL_PHYSICIAN"
	InviteToken string  `json:"invite_token,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse is the JSON body returned for both /login and /refresh.
// ❌ NO refresh_token field — it is delivered exclusively via HttpOnly cookie.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

// RefreshRequest kept for backward compat with non-browser clients (CLI tools, mobile).
// The handler prefers the cookie; this is a fallback only.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type InviteRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Role        string `json:"role" validate:"required"`
	Designation string `json:"designation,omitempty"`
}

type AcceptInviteRequest struct {
	Token       string  `json:"token" validate:"required"`
	Password    string  `json:"password" validate:"required,min=8"`
	FirstName   string  `json:"first_name" validate:"required,min=2"`
	LastName    string  `json:"last_name" validate:"required,min=2"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	DateOfBirth *string `json:"date_of_birth,omitempty"` // format: YYYY-MM-DD
	Gender      *string `json:"gender,omitempty" validate:"omitempty,oneof=MALE FEMALE OTHER"`
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type GoogleLoginRequest struct {
	Token string `json:"token" validate:"required"`
}
