package service

import (
	"context"
	"fmt"
	"net/smtp"
	util "github.com/1AyushGarg1/EmailWorker/utils"
)

// EmailService defines the interface for sending emails.
type EmailService interface {
	SendOTP(ctx context.Context, recipientEmail, otp string) error
	SendMailToStudent(ctx context.Context, recipientEmail, userName string, testPaperTitle string, marksObtained int, feedbackURL string) error
	GeneralMailSend(ctx context.Context,recipientEmail string, subject string, body string) error
}

// MockEmailService is a mock implementation for testing and development.
// It logs the OTP to the console instead of sending a real email.
type MockEmailService struct{}

// NewMockEmailService creates a new MockEmailService.
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

// SendOTP logs the OTP to the console.
func (s *MockEmailService) SendOTP(ctx context.Context, recipientEmail, otp string) error {
	log := util.GetLoggerUsingCtx(ctx)
	// In a real application, this would use an email client (e.g., SMTP, SendGrid) to send an email.
	log.Infof("SIMULATING EMAIL: OTP for user %s is %s", recipientEmail, otp)
	return nil
}

func (s *MockEmailService) SendMailToStudent(ctx context.Context, recipientEmail string, userName string, testPaperTitle string, marksObtained int, feedbackURL string) error {
	log := util.GetLoggerUsingCtx(ctx)
	// In a real application, this would use an email client (e.g., SMTP, SendGrid) to send an email.
	log.Infof("SIMULATING EMAIL: TO %s -> Hey %s, Your Mock-Test Paper %s has been evaluated. You scored %d marks. For detailed feedback visit %s", recipientEmail, userName, testPaperTitle, marksObtained, feedbackURL)
	return nil
}

func (s *MockEmailService) GeneralMailSend(ctx context.Context,recipientEmail string, subject string, body string) error {
	log := util.GetLoggerUsingCtx(ctx)
	log.Info("SIMULATING EMAIL: To %s .-> Subject is ->%s .-> body is: ->%s",recipientEmail,subject,body)
	return nil;
}

// SMTPEmailService implements EmailService using a standard SMTP server.
type SMTPEmailService struct {
	host     string
	port     string
	user     string
	password string
	sender   string
}

// NewSMTPEmailService creates a new SMTPEmailService.
func NewSMTPEmailService(host, port, user, password, sender string) *SMTPEmailService {
	return &SMTPEmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		sender:   sender,
	}
}

// SendOTP sends an OTP to the user via an SMTP server.
func (s *SMTPEmailService) SendOTP(ctx context.Context, recipientEmail, otp string) error {
	log := util.GetLoggerUsingCtx(ctx)

	if s.host == "" || s.user == "" || s.password == "" || s.sender == "" {
		log.Error("SMTP service is not configured. Please set SMTP_HOST, SMTP_USER, SMTP_PASSWORD, and SMTP_SENDER environment variables.")
		// Fallback to logging for development convenience if not configured.
		log.Infof("FALLBACK: OTP for user %s is %s", recipientEmail, otp)
		return nil // Don't return an error, just log it.
	}

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	subject := "Subject: Your One-Time Password for ByteStream\r\n"
	body := fmt.Sprintf("Your OTP is: %s\r\nThis code will expire in 5 minutes.", otp)
	msg := []byte(subject + "\r\n" + body)

	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	log.Infof("Sending real OTP email to %s via %s", recipientEmail, s.host)
	err := smtp.SendMail(addr, auth, s.sender, []string{recipientEmail}, msg)
	if err != nil {
		log.Errorf("Failed to send OTP email: %v", err)
		return err
	}

	log.Info("Successfully sent OTP email.")
	return nil
}

func (s *SMTPEmailService) SendMailToStudent(ctx context.Context, recipientEmail string, userName string,testPaperTitle string,marksObtained int,feedbackURL string) error {
	log := util.GetLoggerUsingCtx(ctx)
	if s.host == "" || s.user == "" || s.password == "" || s.sender == "" {
		log.Error("SMTP service is not configured. Please set SMTP_HOST, SMTP_USER, SMTP_PASSWORD, and SMTP_SENDER environment variables.")
		// Fallback to logging for development convenience if not configured.
		log.Infof("FALLBACK: Failed to send Evaluation email to %s", recipientEmail)
		return nil // Don't return an error, just log it.
	}
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	subject := "Subject: Your Test Paper " + testPaperTitle + " has been evaluated\r\n"
	body := fmt.Sprintf("Hey %s, Your Test Paper %s has been evaluated. You scored %d marks. For detailed feedback visit %s", userName, testPaperTitle, marksObtained, feedbackURL)
	msg := []byte(subject + "\r\n" + body)

	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	log.Infof("Sending real Evaluation email to %s via %s", recipientEmail, s.host)
	err := smtp.SendMail(addr, auth, s.sender, []string{recipientEmail}, msg)
	if err != nil {
		log.Errorf("Failed to send Evaluation email to %s: %v", recipientEmail, err)
		return err
	}

	log.Infof("Successfully sent Evaluation email to Student %s.",recipientEmail)
	return nil
}

func (s *SMTPEmailService) GeneralMailSend(ctx context.Context,recipientEmail string, subject string, body string) error {
	log := util.GetLoggerUsingCtx(ctx)
	if s.host == "" || s.user == "" || s.password == "" || s.sender == "" {
		log.Error("SMTP service is not configured. Please set SMTP_HOST, SMTP_USER, SMTP_PASSWORD, and SMTP_SENDER environment variables.")
		// Fallback to logging for development convenience if not configured.
		log.Infof("FALLBACK: Failed to send Evaluation email to %s", recipientEmail)
		return nil // Don't return an error, just log it.
	}
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	sub := "Subject:" + subject + "\r\n"
	msg := []byte(sub + "\r\n" + body)

	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	log.Infof("Sending General email to %s via %s", recipientEmail, s.host)
	err := smtp.SendMail(addr, auth, s.sender, []string{recipientEmail}, msg)
	if err != nil {
		log.Errorf("Failed to send Evaluation email to %s: %v", recipientEmail, err)
		return err
	}
	log.Infof("Successfully sent General email to %s.",recipientEmail)
	return nil
}