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
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken string `json:"accessToken"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}
