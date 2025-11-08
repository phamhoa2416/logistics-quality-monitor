package utils

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

// SanitizeString removes potentially dangerous characters and escapes HTML
func SanitizeString(input string) string {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Escape HTML entities
	escaped := html.EscapeString(trimmed)

	return escaped
}

// SanitizeEmail sanitizes email input
func SanitizeEmail(email string) string {
	// Convert to lowercase and trim
	email = strings.ToLower(strings.TrimSpace(email))

	// Remove any HTML tags
	email = stripHTML(email)

	// Remove any control characters
	email = removeControlChars(email)

	return email
}

// SanitizePhone sanitizes phone number input
func SanitizePhone(phone string) string {
	// Trim whitespace
	phone = strings.TrimSpace(phone)

	// Remove any HTML tags
	phone = stripHTML(phone)

	// Remove any non-digit, nonplus, non-dash, non-space characters
	var result strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) || r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// SanitizeText sanitizes multi-line text input
func SanitizeText(input string) string {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Escape HTML entities
	escaped := html.EscapeString(trimmed)

	// Remove any control characters except newlines and tabs
	var result strings.Builder
	for _, r := range escaped {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' || r == '\r' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// stripHTML removes HTML tags from string
func stripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(input, "")
}

// removeControlChars removes control characters from string
func removeControlChars(input string) string {
	var result strings.Builder
	for _, r := range input {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ValidateAndSanitizeEmail validates and sanitizes email
func ValidateAndSanitizeEmail(email string) (string, error) {
	sanitized := SanitizeEmail(email)
	if !IsValidEmail(sanitized) {
		return "", fmt.Errorf("invalid email format")
	}
	return sanitized, nil
}
