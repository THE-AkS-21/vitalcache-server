package services

import (
	"log/slog"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

// SendEmail sends an email with an attachment.
func (s *EmailService) SendEmailWithAttachment(to, subject, body, attachmentPath string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	m.Attach(attachmentPath)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

	if err := d.DialAndSend(m); err != nil {
		slog.Error("Failed to send email", "error", err)
		return err
	}

	slog.Info("Email sent successfully", "to", to)
	return nil
}
