package otp

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateOTP(t *testing.T) {
	tests := []struct {
		name    string
		wantLen int
		wantErr bool
	}{
		{"generates 6 digit OTP", 6, false},
		{"multiple generations", 6, false},
		{"third generation", 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := GenerateOTP()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, otp, tt.wantLen)

				for _, char := range otp {
					assert.True(t, char >= '0' && char <= '9', "OTP should contain only digits")
				}

				_, err := strconv.Atoi(otp)
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateOTPDigitsOnly(t *testing.T) {
	for i := 0; i < 100; i++ {
		otp, err := GenerateOTP()
		assert.NoError(t, err)
		assert.Len(t, otp, 6)

		for _, char := range otp {
			assert.True(t, char >= '0' && char <= '9', "OTP should contain only digits, got: %c", char)
		}
	}
}

func TestGenerateOTPRange(t *testing.T) {
	seen := make(map[string]bool)
	duplicates := 0

	for i := 0; i < 10000; i++ {
		otp, err := GenerateOTP()
		assert.NoError(t, err)

		num, err := strconv.Atoi(otp)
		assert.NoError(t, err)
		assert.True(t, num >= 0 && num <= 999999, "OTP should be between 000000 and 999999")

		if seen[otp] {
			duplicates++
		}
		seen[otp] = true
	}

	t.Logf("Generated 10000 OTPs, found %d duplicates", duplicates)
	assert.Less(t, duplicates, 200, "Should have reasonable number of duplicates")
}