package auth

type LoginRequest struct {
	Email    string `json:"email" validate:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type OTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type OTPResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	OTPCode  string `json:"otp" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}
