package utils

import (
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword creates a bcrypt hash of a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidatePasswordStrength checks password strength
func ValidatePasswordStrength(password string) (bool, string) {
	if len(password) < 8 {
		return false, "Password must be at least 8 characters long"
	}
	if len(password) > 128 {
		return false, "Password must be less than 128 characters"
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasUpper {
		return false, "Password must contain at least one uppercase letter"
	}
	if !hasLower {
		return false, "Password must contain at least one lowercase letter"
	}
	if !hasDigit {
		return false, "Password must contain at least one digit"
	}

	return true, ""
}

// ValidateUsername validates username
func ValidateUsername(username string) (bool, string) {
	if len(username) < 3 {
		return false, "Username must be at least 3 characters"
	}
	if len(username) > 50 {
		return false, "Username must be less than 50 characters"
	}

	// Only allow alphanumeric and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	if !matched {
		return false, "Username can only contain letters, numbers, and underscores"
	}

	return true, ""
}

// SanitizeInput sanitizes user input
func SanitizeInput(input string) string {
	// Remove leading/trailing whitespace
	input = strings.TrimSpace(input)

	// Limit length
	if len(input) > 1000 {
		input = input[:1000]
	}

	return input
}
