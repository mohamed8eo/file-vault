package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr bool
	}{
		{"valid password", "Secret123!", false},
		{"empty password", "", true},
		{"short password", "Ab1!", false}, // argon2id still hashes it
		{"long password", "VeryLongSecurePassword123!@#", false},
		{"password with spaces", "Secret 123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash) // hash should be different
			}
		})
	}
}

func TestCheckHash(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		hash      string
		wantMatch bool
		wantErr   bool
	}{
		{
			name:      "correct password",
			password:  "Secret123!",
			hash:      "",
			wantMatch: false,
			wantErr:   true, // will fail because we need to generate hash first
		},
		{
			name:      "empty hash",
			password:  "Secret123!",
			hash:      "",
			wantMatch: false,
			wantErr:   true,
		},
		{
			name:      "empty password",
			password:  "",
			hash:      "someshash",
			wantMatch: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that need pre-generated hash
			if tt.hash == "" {
				t.Skip("Need to generate hash first")
			}
			match, err := CheckHash(tt.hash, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMatch, match)
			}
		})
	}
}

func TestCheckHashWithGeneratedHash(t *testing.T) {
	password := "Secret123!"
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test correct password
	match, err := CheckHash(hash, password)
	assert.NoError(t, err)
	assert.True(t, match)

	// Test incorrect password - returns error "passwords are not match"
	match, err = CheckHash(hash, "WrongPassword123!")
	assert.Error(t, err) // Returns error when passwords don't match
	assert.Contains(t, err.Error(), "passwords are not match")
	assert.False(t, match)
}

func TestMakeToken(t *testing.T) {
	tests := []struct {
		name      string
		issue     string
		secret    string
		userID    string
		expiresIn time.Duration
		wantErr   bool
	}{
		{
			name:      "valid access token",
			issue:     "access-token",
			secret:    "test-secret-key-12345678901234567890",
			userID:    "550e8400-e29b-41d4-a716-446655440000",
			expiresIn: 15 * time.Minute,
			wantErr:   false,
		},
		{
			name:      "valid refresh token",
			issue:     "refresh-token",
			secret:    "test-secret-key-12345678901234567890",
			userID:    "550e8400-e29b-41d4-a716-446655440000",
			expiresIn: 30 * 24 * time.Hour,
			wantErr:   false,
		},
		{
			name:      "empty secret still works (not validated)",
			issue:     "access-token",
			secret:    "",
			userID:    "550e8400-e29b-41d4-a716-446655440000",
			expiresIn: 15 * time.Minute,
			wantErr:   false, // JWT doesn't validate empty secret at creation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := MakeToken(tt.issue, tt.secret, tt.userID, tt.expiresIn)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestMakeAndValidateToken(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresIn := 15 * time.Minute

	// Create token
	token, err := MakeToken("access-token", secret, userID, expiresIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	retrievedID, err := ValidateJWT(secret, token)
	assert.NoError(t, err)
	assert.Equal(t, userID, retrievedID.String())
}

func TestMakeAndValidateRefreshToken(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"
	userID := "550e8400-e29b-41d4-a716-446655440000"
	expiresIn := 30 * 24 * time.Hour

	// Create refresh token
	token, err := MakeToken("refresh-token", secret, userID, expiresIn)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate refresh token
	retrievedID, err := ValidateJWT(secret, token)
	assert.NoError(t, err)
	assert.Equal(t, userID, retrievedID.String())
}

func TestValidateJWTWithWrongSecret(t *testing.T) {
	token, err := MakeToken("access-token", "secret1", "550e8400-e29b-41d4-a716-446655440000", 15*time.Minute)
	assert.NoError(t, err)

	// Validate with wrong secret
	_, err = ValidateJWT("secret2", token)
	assert.Error(t, err)
}

func TestValidateJWTWithInvalidToken(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"

	// Invalid token
	_, err := ValidateJWT(secret, "invalid-token")
	assert.Error(t, err)

	// Empty token
	_, err = ValidateJWT(secret, "")
	assert.Error(t, err)
}

func TestValidateJWTWithExpiredToken(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Create token that expires immediately
	token, err := MakeToken("access-token", secret, userID, -1*time.Minute)
	assert.NoError(t, err)

	// Validate - should fail due to expiration
	_, err = ValidateJWT(secret, token)
	assert.Error(t, err)
}

func TestValidateJWTWithWrongIssuer(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"

	// Create token with invalid issuer
	token, err := MakeToken("invalid-issuer", secret, "550e8400-e29b-41d4-a716-446655440000", 15*time.Minute)
	assert.NoError(t, err)

	// Validate - should fail due to wrong issuer
	_, err = ValidateJWT(secret, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token issuer")
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "valid bearer token",
			header:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:   false,
		},
		{
			name:      "empty header",
			header:    "",
			wantToken: "",
			wantErr:   true,
		},
		{
			name:      "no bearer prefix",
			header:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "",
			wantErr:   true,
		},
		{
			name:      "bearer without space",
			header:    "Bearer",
			wantToken: "",
			wantErr:   true,
		},
		{
			name:      "lowercase bearer",
			header:    "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GetBearerToken(tt.header)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func TestTokenTypeConstants(t *testing.T) {
	assert.Equal(t, TokenType("access-token"), AccessToken)
	assert.Equal(t, TokenType("refresh-token"), RefreshToken)
}