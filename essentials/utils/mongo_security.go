package utils

import (
	"regexp"
	"strings"
)

// MongoInjectionPatterns detects potential MongoDB injection patterns
// Using (?i) for case-insensitive matching
var MongoInjectionPatterns = []*regexp.Regexp{
	// Dollar sign operators
	regexp.MustCompile(`(?i)\$[a-zA-Z]`),
	// Object literal injection
	regexp.MustCompile(`(?i)\{.*\$.*\}`),
	// NE operator injection
	regexp.MustCompile(`(?i)ne\s*[:=]`),
	// GT/GTE/LT/LTE injection
	regexp.MustCompile(`(?i)[gl]e\s*[:=]`),
	// In operator injection
	regexp.MustCompile(`(?i)in\s*[:=]`),
	// Where clause injection
	regexp.MustCompile(`(?i)where\s*\(|\$\s*where`),
	// JavaScript execution
	regexp.MustCompile(`(?i)eval\s*\(`),
	// Script injection
	regexp.MustCompile(`(?i)<script>`),
}

// SanitizeMongoQuery sanitizes user input for MongoDB queries
// Returns a safe string for use in queries
func SanitizeMongoQuery(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check for injection patterns and escape them
	for _, pattern := range MongoInjectionPatterns {
		if pattern.MatchString(input) {
			// Replace potentially dangerous characters
			input = strings.ReplaceAll(input, "$", "&#36;")
			input = strings.ReplaceAll(input, "{", "&#123;")
			input = strings.ReplaceAll(input, "}", "&#125;")
		}
	}

	return input
}

// SanitizeMongoValue sanitizes a value before inserting into MongoDB
func SanitizeMongoValue(input string) string {
	// Apply general sanitization
	sanitized := SanitizeMongoQuery(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Limit length to prevent DoS
	if len(sanitized) > 10000 {
		sanitized = sanitized[:10000]
	}

	return sanitized
}

// BuildSafeQuery builds a MongoDB query with proper sanitization
func BuildSafeQuery(allowedFields map[string]bool, userInput map[string]interface{}) map[string]interface{} {
	safeQuery := make(map[string]interface{})

	for key, value := range userInput {
		if allowedFields[key] {
			// If value is a string, sanitize it
			if strVal, ok := value.(string); ok {
				safeQuery[key] = SanitizeMongoValue(strVal)
			} else {
				safeQuery[key] = value
			}
		}
	}

	return safeQuery
}

// ValidateFieldAccess ensures only allowed fields are accessed
func ValidateFieldAccess(field string, allowedFields map[string]bool) error {
	if !allowedFields[field] {
		return ErrUnauthorizedField
	}
	return nil
}

// ErrUnauthorizedField is returned when a user tries to access an unauthorized field
var ErrUnauthorizedField = &ValidationError{Message: "Unauthorized field access"}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidMongoFieldName checks if a field name is valid for MongoDB
func IsValidMongoFieldName(fieldName string) bool {
	// Field names cannot start with dollar sign
	if strings.HasPrefix(fieldName, "$") {
		return false
	}

	// Field names cannot contain null bytes
	if strings.Contains(fieldName, "\x00") {
		return false
	}

	return true
}

// SanitizeForSearch sanitizes input for search operations
func SanitizeForSearch(input string) string {
	// Escape regex special characters
	specialChars := []string{
		"\\", ".", "*", "+", "?", "[", "]", "^", "$", "|", "(", ")", "{", "}", "=",
	}

	for _, char := range specialChars {
		input = strings.ReplaceAll(input, char, "\\"+char)
	}

	return input
}
