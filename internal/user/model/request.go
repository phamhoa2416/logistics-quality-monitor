package model

type RegisterRequest struct {
	Username        string  `json:"username" validate:"required,min=3,max=100"`
	Email           string  `json:"email" validate:"required,email"`
	Password        string  `json:"password" validate:"required,min=8"`
	ConfirmPassword string  `json:"confirm_password" validate:"required,eqfield=Password"`
	FullName        string  `json:"full_name" validate:"required,min=2,max=255"`
	PhoneNumber     *string `json:"phone_number" validate:"omitempty,phone"`
	Role            string  `json:"role" validate:"required,user_role"`
	Address         *string `json:"address" validate:"omitempty,max=500"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type ChangePasswordRequest struct {
	OldPassword     string `json:"old_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type UpdateProfileRequest struct {
	FullName    *string `json:"full_name" validate:"omitempty,min=2,max=255"`
	PhoneNumber *string `json:"phone_number" validate:"omitempty,phone"`
	Address     *string `json:"address" validate:"omitempty,max=500"`
}
