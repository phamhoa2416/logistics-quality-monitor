package utils

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func ValidatePassword(password string) error {
	var (
		hasMinLength = false
		hasUpper     = false
		hasLower     = false
		hasNumber    = false
		hasSpecial   = false
	)

	if len(password) >= 8 {
		hasMinLength = true
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasMinLength || !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return errors.New("password must be at least 8 characters and contain uppercase, " +
			"lowercase, number and special symbol")
	}

	return nil
}
