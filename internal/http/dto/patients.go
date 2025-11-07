package dto

type CreatePatientRequest struct {
	Name         string `json:"name" binding:"required"`
	Age          int    `json:"age" binding:"required,gt=0"`
	Sex          string `json:"sex"`
	MobileNumber string `json:"mobile_number" binding:"required"`
	Email        string `json:"email,omitempty,email"`
}

type UpdatePatientRequest struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Sex   string `json:"sex"`
	Email string `json:"email,omitempty,email"`
}
