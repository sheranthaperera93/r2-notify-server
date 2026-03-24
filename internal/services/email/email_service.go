package emailService

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EmailService struct {
	apiKey string
	from   string
	client *http.Client
}

func NewEmailService(apiKey, from string) *EmailService {
	return &EmailService{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (e *EmailService) send(to, subject, html string) error {
	payload := resendPayload{
		From:    e.from,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API error: status %d", resp.StatusCode)
	}
	return nil
}

func (e *EmailService) SendVerification(to, verifyURL string) error {
	html := fmt.Sprintf(`
		<h2>Verify your R2-Notify account</h2>
		<p>Click the link below to verify your email address. This link expires in 24 hours.</p>
		<p><a href="%s">Verify Email</a></p>
		<p>If you didn't create an account, you can safely ignore this email.</p>
	`, verifyURL)
	return e.send(to, "Verify your R2-Notify email", html)
}

func (e *EmailService) SendPasswordReset(to, resetURL string) error {
	html := fmt.Sprintf(`
		<h2>Reset your R2-Notify password</h2>
		<p>Click the link below to reset your password. This link expires in 1 hour.</p>
		<p><a href="%s">Reset Password</a></p>
		<p>If you didn't request a password reset, you can safely ignore this email.</p>
	`, resetURL)
	return e.send(to, "Reset your R2-Notify password", html)
}
