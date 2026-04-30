package otp

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/resend/resend-go/v3"
)

func SendOTPEmail(to, otp string) error {
	devMode := os.Getenv("DEV_MODE") == "true"
	resendAPIKey := os.Getenv("RESEND_API_KEY")

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

	// Check if Resend API key is configured
	if resendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured. Set DEV_MODE=true for development")
	}

	slog.Info("Sending OTP via Resend API", "to", to)

	client := resend.NewClient(resendAPIKey)

	params := &resend.SendEmailRequest{
		From:    "File Vault <onboarding@resend.dev>",
		To:      []string{to},
		Subject: "Your File Vault Verification Code",
		Html: fmt.Sprintf(`
			<html>
			<body style="font-family: Arial, sans-serif; padding: 20px; background: #f5f5f5;">
				<div style="max-width: 500px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px;">
					<h2 style="color: #333; margin-bottom: 20px;">📧 Email Verification</h2>
					<p style="color: #666; font-size: 16px;">Your verification code is:</p>
					<div style="background: #f0f0f0; padding: 20px; border-radius: 8px; text-align: center; margin: 20px 0;">
						<span style="font-size: 36px; font-weight: bold; color: #007bff; letter-spacing: 5px;">%s</span>
					</div>
					<p style="color: #999; font-size: 14px;">This code will expire in 10 minutes.</p>
					<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
					<p style="color: #999; font-size: 12px;">If you didn't request this code, please ignore this email.</p>
				</div>
			</body>
			</html>
		`, otp),
	}

	resp, err := client.Emails.Send(params)
	if err != nil {
		slog.Error("Resend API error", "error", err.Error())
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("OTP email sent successfully via Resend", "to", to, "email_id", resp.Id)
	return nil
}