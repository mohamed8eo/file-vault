package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/otp"
)

type VerifyOTPInput struct {
	Email string `json:"email"`
	Otp   string `json:"otp"`
}

func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request, email string, userID uuid.UUID) bool {
	generatedOtp, err := otp.GenerateOTP()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	hashedOTP, err := argon2id.CreateHash(generatedOtp, argon2id.DefaultParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	if err = h.queries.SaveOTP(r.Context(), db.SaveOTPParams{
		ID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		Otp:          pgtype.Text{String: hashedOTP, Valid: true},
		OtpExpiresAt: pgtype.Timestamp{Time: time.Now().Add(10 * time.Minute), Valid: true},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}

	if err = otp.SendOTPEmail(email, generatedOtp); err != nil {
		// Log the error but don't crash - return better error message
		http.Error(w, fmt.Sprintf("failed to send OTP email: %v", err), http.StatusInternalServerError)
		return false
	}

	return true
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var input VerifyOTPInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), input.Email)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if user.VerifiedAt.Valid {
		http.Error(w, "email already verified", http.StatusBadRequest)
		return
	}

	if !user.Otp.Valid {
		http.Error(w, "no OTP requested", http.StatusBadRequest)
		return
	}

	if time.Now().After(user.OtpExpiresAt.Time) {
		http.Error(w, "OTP expired", http.StatusUnauthorized)
		return
	}

	match, err := argon2id.ComparePasswordAndHash(input.Otp, user.Otp.String)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError) // was 401
		return
	}
	if !match {
		http.Error(w, "incorrect otp", http.StatusUnauthorized) // was 400
		return
	}

	// mark verified, clear OTP
	if err := h.queries.MarkUserVerified(r.Context(), user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.queries.MarkUserVerified(r.Context(), user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// generate tokens now that user is verified
	accessToken, refreshToken, err := h.generateTokens(user.ID.String())
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	if _, err := h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(refreshTokenExpiry),
			Valid: true,
		},
	}); err != nil {
		http.Error(w, "failed to create refresh token", http.StatusInternalServerError)
		return
	}

	type response struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		AccessToken string `json:"access_token"`
	}

	h.setRefreshCookie(w, refreshToken, time.Now().Add(refreshTokenExpiry))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Name:        user.Name,
		Email:       user.Email,
		AccessToken: accessToken,
	})
}
