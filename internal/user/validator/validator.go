package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	err := validate.RegisterValidation("user_role", validateUserRole)
	if err != nil {
		return
	}
	err = validate.RegisterValidation("phone", validatePhone)
	if err != nil {
		return
	}
}

func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

func validateUserRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	validRoles := []string{"customer", "provider", "shipper", "admin"}

	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	re := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return re.MatchString(phone)
}

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(email)
}
