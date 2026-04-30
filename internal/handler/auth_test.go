package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock database queries for testing
type mockQueries struct {
	createUserFunc       func(email string) error
	getUserByEmailFunc   func(email string) (string, error)
	createRefreshTokenFunc func(token string, userID string) error
}

func TestSignUpValidation(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		expectedError  string
	}{
		// Missing fields
		{
			name:           "missing name",
			body:           map[string]string{"email": "test@test.com", "password": "Secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name:           "missing email",
			body:           map[string]string{"name": "Test", "password": "Secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email is required",
		},
		{
			name:           "missing password",
			body:           map[string]string{"name": "Test", "email": "test@test.com"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password is required",
		},
		{
			name:           "empty body",
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},

		// Invalid email format
		{
			name:           "invalid email - no @",
			body:           map[string]string{"name": "Test", "email": "testtest.com", "password": "Secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email format",
		},
		{
			name:           "invalid email - no TLD",
			body:           map[string]string{"name": "Test", "email": "test@test", "password": "Secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email format",
		},
		{
			name:           "invalid email - no domain",
			body:           map[string]string{"name": "Test", "email": "test@", "password": "Secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid email format",
		},

		// Invalid password - too short
		{
			name:           "password too short",
			body:           map[string]string{"name": "Test", "email": "test@test.com", "password": "Ab1!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
		// Invalid password - no uppercase
		{
			name:           "password no uppercase",
			body:           map[string]string{"name": "Test", "email": "test@test.com", "password": "secret123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must contain at least 1 uppercase letter",
		},
		// Invalid password - no lowercase
		{
			name:           "password no lowercase",
			body:           map[string]string{"name": "Test", "email": "test@test.com", "password": "SECRET123!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must contain at least 1 lowercase letter",
		},
		// Invalid password - no number
		{
			name:           "password no number",
			body:           map[string]string{"name": "Test", "email": "test@test.com", "password": "SecretPass!"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must contain at least 1 number",
		},
		// Invalid password - no special char
		{
			name:           "password no special char",
			body:           map[string]string{"name": "Test", "email": "test@test.com", "password": "Secret123"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must contain at least 1 special character",
		},

		// Valid input - would fail on actual DB but passes validation
		// (We can't test successful signup without mocking the DB)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually call SignUp without DB, so we test the validation functions directly
			result := validateSignUpInput(tt.body["name"], tt.body["email"], tt.body["password"])

			if tt.expectedStatus == http.StatusBadRequest {
				assert.Equal(t, tt.expectedError, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func validateSignUpInput(name, email, password string) string {
	if errMsg := validateName(name); errMsg != "" {
		return errMsg
	}
	if errMsg := validateEmail(email); errMsg != "" {
		return errMsg
	}
	if errMsg := validatePassword(password); errMsg != "" {
		return errMsg
	}
	return ""
}

func TestValidateSignUpInput(t *testing.T) {
	tests := []struct {
		name     string
		nameVal  string
		emailVal string
		passVal  string
		wantErr  string
	}{
		// Valid inputs
		{"valid input", "John Doe", "test@example.com", "Secret123!", ""},

		// Name validation
		{"empty name", "", "test@example.com", "Secret123!", "name is required"},
		{"short name", "A", "test@example.com", "Secret123!", "name must be at least 2 characters"},

		// Email validation
		{"valid email .com", "Test", "test@example.com", "Secret123!", ""},
		{"valid email .org", "Test", "test@example.org", "Secret123!", ""},
		{"valid email .edu", "Test", "test@university.edu", "Secret123!", ""},
		{"invalid email no @", "Test", "testexample.com", "Secret123!", "invalid email format"},
		{"invalid email no TLD", "Test", "test@test", "Secret123!", "invalid email format"},
		{"invalid email empty", "Test", "", "Secret123!", "email is required"},

		// Password validation
		{"valid strong password", "Test", "test@example.com", "MyP@ss1!", ""},
		{"password too short", "Test", "test@example.com", "A1!abc", "password must be at least 8 characters"},
		{"password no uppercase", "Test", "test@example.com", "secret123!", "password must contain at least 1 uppercase letter"},
		{"password no lowercase", "Test", "test@example.com", "SECRET123!", "password must contain at least 1 lowercase letter"},
		{"password no number", "Test", "test@example.com", "SecretPass!", "password must contain at least 1 number"},
		{"password no special", "Test", "test@example.com", "Secret1234", "password must contain at least 1 special character"},
		{"password empty", "Test", "test@example.com", "", "password is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateSignUpInput(tt.nameVal, tt.emailVal, tt.passVal)
			if tt.wantErr == "" {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.wantErr, got)
			}
		})
	}
}

func TestLoginInputValidation(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  string
	}{
		// Valid
		{"valid login", "test@example.com", "password123", ""},

		// Empty
		{"empty email", "", "password123", "email is required"},
		{"empty password", "test@example.com", "", "password is required"},
		{"both empty", "", "", "email is required"},

		// Invalid email format
		{"invalid email no @", "testexample.com", "password123", "invalid email format"},
		{"invalid email no TLD", "test@test", "password123", "invalid email format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateLoginInput(tt.email, tt.password)
			if tt.wantErr == "" {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.wantErr, got)
			}
		})
	}
}

func validateLoginInput(email, password string) string {
	if errMsg := validateEmail(email); errMsg != "" {
		return errMsg
	}
	if password == "" {
		return "password is required"
	}
	return ""
}