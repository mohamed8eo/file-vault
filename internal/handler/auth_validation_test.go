package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignUpHandler_JSONParsing(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty body",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name:           "partial body - missing password",
			body:           `{"name": "Test", "email": "test@test.com"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password is required",
		},
		{
			name:           "partial body - missing email",
			body:           `{"name": "Test", "password": "Secret123!"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyMap map[string]string
			json.Unmarshal([]byte(tt.body), &bodyMap)

			result := validateSignUpInput(bodyMap["name"], bodyMap["email"], bodyMap["password"])

			if tt.expectedStatus == http.StatusBadRequest {
				assert.Equal(t, tt.expectedError, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestLoginHandler_JSONParsing(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		expectedError string
	}{
		{
			name:          "empty body",
			body:          `{}`,
			expectedError: "email is required",
		},
		{
			name:          "partial - only email",
			body:          `{"email": "test@test.com"}`,
			expectedError: "password is required",
		},
		{
			name:          "partial - only password",
			body:          `{"password": "test123"}`,
			expectedError: "email is required",
		},
		{
			name:          "valid body",
			body:          `{"email": "test@test.com", "password": "test123"}`,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyMap map[string]string
			json.Unmarshal([]byte(tt.body), &bodyMap)

			email := bodyMap["email"]
			password := bodyMap["password"]

			err := validateLoginInput(email, password)
			if tt.expectedError == "" {
				assert.Empty(t, err)
			} else {
				assert.Equal(t, tt.expectedError, err)
			}
		})
	}
}

func TestEmailValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		email  string
		valid  bool
	}{
		{"standard .com", "user@example.com", true},
		{"standard .org", "user@example.org", true},
		{"standard .net", "user@example.net", true},
		{"standard .edu", "user@university.edu", true},
		{"subdomain", "user@sub.example.com", true},
		{"multiple subdomains", "user@a.b.c.example.com", true},
		{"plus addressing", "user+tag@example.com", true},
		{"dot in local part", "first.last@example.com", true},
		{"dash in domain", "user@my-domain.com", true},
		{"no @", "userexample.com", false},
		{"just @", "@", false},
		{"@ only", "@example.com", false},
		{"no TLD", "user@example", false},
		{"space in local", "user name@example.com", false},
		{"double @", "user@@example.com", false},
		{"leading dot", ".user@example.com", true},
		{"trailing dot in local", "user.@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if tt.valid && err != "" {
				t.Errorf("validateEmail(%q) expected valid, got error: %s", tt.email, err)
			}
			if !tt.valid && err == "" {
				t.Errorf("validateEmail(%q) expected invalid, got valid", tt.email)
			}
		})
	}
}

func TestPasswordValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"simple valid", "Password1!", true},
		{"complex valid", "MyP@ssw0rd#123", true},
		{"all special chars", "Ab1!@#$%^&*()", true},
		{"long password", "VeryLongPassword123!@#", true},
		{"7 chars", "Pas1!12", false},
		{"8 chars valid", "Password1!", true},
		{"3 chars", "A1!", false},
		{"no uppercase", "password1!", false},
		{"no lowercase", "PASSWORD1!", false},
		{"no number", "Password!", false},
		{"no special", "Password12", false},
		{"only letters", "Password", false},
		{"only numbers", "12345678", false},
		{"only special", "!@#$%^&*", false},
		{"spaces allowed", "Pass word1!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.valid && err != "" {
				t.Errorf("validatePassword(%q) expected valid, got error: %s", tt.password, err)
			}
			if !tt.valid && err == "" {
				t.Errorf("validatePassword(%q) expected invalid, got valid", tt.password)
			}
		})
	}
}

func TestNameValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		valid  bool
	}{
		{"single word", "John", true},
		{"two words", "John Doe", true},
		{"many words", "John Michael Doe Smith", true},
		{"with numbers", "John123", true},
		{"with hyphens", "Jean-Pierre", true},
		{"with apostrophe", "O'Brien", true},
		{"empty", "", false},
		{"single char", "A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if tt.valid && err != "" {
				t.Errorf("validateName(%q) expected valid, got error: %s", tt.input, err)
			}
			if !tt.valid && err == "" {
				t.Errorf("validateName(%q) expected invalid, got valid", tt.input)
			}
		})
	}
}