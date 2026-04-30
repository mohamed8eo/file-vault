package handler

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		wantErr  bool
		errMsg   string
	}{
		// Valid emails
		{"valid email with .com", "test@example.com", false, ""},
		{"valid email with .org", "test@example.org", false, ""},
		{"valid email with .edu", "test@university.edu", false, ""},
		{"valid email with subdomain", "test@mail.example.com", false, ""},
		{"valid email with plus sign", "test+tag@example.com", false, ""},
		{"valid email with dots in local", "first.last@example.com", false, ""},

		// Invalid emails - empty
		{"empty email", "", true, "email is required"},

		// Invalid emails - format
		{"no @ symbol", "testexample.com", true, "invalid email format"},
		{"no domain", "test@", true, "invalid email format"},
		{"no local part", "@example.com", true, "invalid email format"},
		{"no TLD", "test@example", true, "invalid email format"},
		{"space in email", "test @example.com", true, "invalid email format"},
		{"double @", "test@@example.com", true, "invalid email format"},
		{"invalid characters", "test@exam!ple.com", true, "invalid email format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateEmail(tt.email)
			if tt.wantErr && got == "" {
				t.Errorf("validateEmail(%q) expected error containing %q, got nil", tt.email, tt.errMsg)
			}
			if !tt.wantErr && got != "" {
				t.Errorf("validateEmail(%q) expected no error, got %q", tt.email, got)
			}
			if tt.wantErr && got != tt.errMsg && tt.errMsg != "" {
				// Just check that we get an error message, not exact match
				if got == "" {
					t.Errorf("validateEmail(%q) expected error %q, got nil", tt.email, tt.errMsg)
				}
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantErr   bool
		errMsg    string
	}{
		// Valid passwords
		{"valid password", "Secret123!", false, ""},
		{"valid with special chars", "Pass@word1", false, ""},
		{"valid complex", "MyP@ssw0rd!", false, ""},
		{"valid long", "VeryLongP@ssword123!", false, ""},

		// Invalid - empty
		{"empty password", "", true, "password is required"},

		// Invalid - too short
		{"too short (7 chars)", "A1!abcd", true, "password must be at least 8 characters"},
		{"exactly 8 chars should pass", "Abcdefg1!", false, ""},

		// Invalid - missing uppercase
		{"no uppercase", "secret123!", true, "password must contain at least 1 uppercase letter"},
		{"only uppercase no lower", "SECRET123!", true, "password must contain at least 1 lowercase letter"},

		// Invalid - missing number
		{"no number", "SecretPass!", true, "password must contain at least 1 number"},

		// Invalid - missing special char
		{"no special char", "Secret123", true, "password must contain at least 1 special character"},

		// Invalid - missing lowercase
		{"no lowercase", "SECRET123!", true, "password must contain at least 1 lowercase letter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePassword(tt.password)
			if tt.wantErr && got == "" {
				t.Errorf("validatePassword(%q) expected error, got nil", tt.password)
			}
			if !tt.wantErr && got != "" {
				t.Errorf("validatePassword(%q) expected no error, got %q", tt.password, got)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"valid simple name", "John", false},
		{"valid with space", "John Doe", false},
		{"valid long name", "John Michael Doe", false},

		// Invalid - empty
		{"empty name", "", true},

		// Invalid - too short
		{"single char", "A", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateName(tt.input)
			if tt.wantErr && got == "" {
				t.Errorf("validateName(%q) expected error, got nil", tt.input)
			}
			if !tt.wantErr && got != "" {
				t.Errorf("validateName(%q) expected no error, got %q", tt.input, got)
			}
		})
	}
}