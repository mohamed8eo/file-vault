package handler

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func validateEmail(email string) string {
	if email == "" {
		return "email is required"
	}
	if !emailRegex.MatchString(email) {
		return "invalid email format"
	}
	return ""
}

func validatePassword(password string) string {
	if password == "" {
		return "password is required"
	}
	if len(password) < 8 {
		return "password must be at least 8 characters"
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.Contains("!@#$%^&*()_+-=[]{}|;':\",./<>?", string(char)):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return "password must contain at least 1 uppercase letter"
	}
	if !hasNumber {
		return "password must contain at least 1 number"
	}
	if !hasSpecial {
		return "password must contain at least 1 special character"
	}
	if !hasLower {
		return "password must contain at least 1 lowercase letter"
	}

	return ""
}

func validateName(name string) string {
	if name == "" {
		return "name is required"
	}
	if len(name) < 2 {
		return "name must be at least 2 characters"
	}
	if len(name) > 100 {
		return "name must not exceed 100 characters"
	}
	return ""
}