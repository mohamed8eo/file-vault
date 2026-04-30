package otp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type MailtrapEmail struct {
	From    Address  `json:"from"`
	To      []Address `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
	HTML    string   `json:"html"`
}

type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

func SendOTPEmail(to, otp string) error {
	devMode := os.Getenv("DEV_MODE") == "true"
	smtpUser := os.Getenv("SMTP_USER")
	mailtrapAPIKey := os.Getenv("MAILTRAP_API_KEY")
	mailtrapInboxID := os.Getenv("MAILTRAP_INBOX_ID")

	// Development mode: print OTP to console instead of sending email
	if devMode {
		slog.Info("📧 DEV MODE: OTP Email (use this code)",
			"to", to,
			"otp", otp,
			"expires", "10 minutes")
		fmt.Printf("\n╔════════════════════════════════════════════╗\n")
		fmt.Printf("║  📧 DEV MODE - OTP CODE                    ║\n")
		fmt.Printf("║  To: %-35s║\n", to)
		fmt.Printf("║  OTP: %-35s║\n", otp)
		fmt.Printf("║  Expires: 10 minutes                      ║\n")
		fmt.Printf("╚════════════════════════════════════════════╝\n\n")
		return nil
	}

	// Check if Mailtrap API is configured
	if mailtrapAPIKey == "" {
		return fmt.Errorf("MAILTRAP_API_KEY not configured. Set DEV_MODE=true for development")
	}

	slog.Info("Sending OTP via Mailtrap API", "to", to, "inbox_id", mailtrapInboxID)

	email := MailtrapEmail{
		From: Address{
			Email: "no-reply@filevault.local",
			Name:  "File Vault",
		},
		To: []Address{
			{Email: to},
		},
		Subject: "Your File Vault Verification Code",
		Text:    fmt.Sprintf("Your verification code is: %s. Expires in 10 minutes.", otp),
		HTML: fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif; padding: 20px;">
				<h2>Email Verification</h2>
				<p>Your verification code is:</p>
				<p style="font-size: 32px; font-weight: bold; color: #007bff; background: #f5f5f5; padding: 15px; border-radius: 8px; text-align: center;">%s</p>
				<p>This code will expire in 10 minutes.</p>
				<p style="color: #666; font-size: 12px;">If you didn't request this code, please ignore this email.</p>
			</body>
			</html>
		`, otp),
	}

	jsonData, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	// Use sandbox endpoint if inbox ID is provided
	url := fmt.Sprintf("https://mailtrap.io/api/v1/inboxes/%s/messages", mailtrapInboxID)
	if smtpUser == "sandbox" {
		// Use sandbox mode
		url = fmt.Sprintf("https://mailtrap.io/api/v1/inboxes/%s/messages", mailtrapInboxID)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Token", mailtrapAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Mailtrap API request failed", "error", err.Error())
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Error("Mailtrap API error", "status", resp.Status)
		return fmt.Errorf("mailtrap API error: status %d", resp.Status)
	}

	slog.Info("OTP email sent successfully via Mailtrap API", "to", to)
	return nil
}
