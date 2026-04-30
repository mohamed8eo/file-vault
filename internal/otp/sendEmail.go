package otp

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/gomail.v2"
)

func SendOTPEmail(to, otp string) error {
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	devMode := os.Getenv("DEV_MODE") == "true"

	// Development mode: print OTP to console instead of sending email
	if devMode {
		slog.Info("📧 DEV MODE: OTP Email (use this code)",
			"to", to,
			"otp", otp,
			"expires", "10 minutes")
		fmt.Printf("\n╔════════════════════════════════════════════╗\n")
		fmt.Printf("║  📧 DEV MODE - OTP CODE                    ║\n")
		fmt.Printf("║  To: %s                          ║\n", to)
		fmt.Printf("║  OTP: %s                          ║\n", otp)
		fmt.Printf("║  Expires: 10 minutes                      ║\n")
		fmt.Printf("╚════════════════════════════════════════════╝\n\n")
		return nil
	}

	slog.Info("SMTP Configuration", "host", smtpHost, "port", smtpPort, "user", smtpUser)

	if smtpUser == "" || smtpPass == "" {
		return fmt.Errorf("SMTP credentials not configured (SMTP_USER or SMTP_PASS missing)")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@filevault.local")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your File Vault Verification Code")
	m.SetBody("text/html", fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif;">
			<h2>Email Verification</h2>
			<p>Your verification code is:</p>
			<p style="font-size: 24px; font-weight: bold; color: #007bff;">%s</p>
			<p>This code will expire in 10 minutes.</p>
			<p>If you didn't request this code, please ignore this email.</p>
		</body>
		</html>
	`, otp))

	// Default to Mailtrap if not specified
	if smtpHost == "" {
		smtpHost = "sandbox.smtp.mailtrap.io"
	}
	if smtpPort == "" {
		smtpPort = "2525"
	}

	// Parse port
	var port int
	fmt.Sscanf(smtpPort, "%d", &port)
	if port == 0 {
		port = 2525
	}

	slog.Info("Attempting to send OTP email", "to", to, "host", smtpHost, "port", port)

	d := gomail.NewDialer(smtpHost, port, smtpUser, smtpPass)
	if err := d.DialAndSend(m); err != nil {
		slog.Error("Failed to send OTP email", "error", err.Error(), "to", to)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("OTP email sent successfully", "to", to)
	return nil
}
