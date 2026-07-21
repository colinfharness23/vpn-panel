package email

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// EmailService sends email notifications via SMTP.
type EmailService struct {
	settingService service.SettingService
}

// SMTPTestResult holds the result of an SMTP connection test.
type SMTPTestResult struct {
	Success bool   `json:"success"`
	Stage   string `json:"stage"`   // "connect" | "auth" | "send"
	Message string `json:"message"` // classified error message
}

// NewEmailService creates a new EmailService.
func NewEmailService(settingService service.SettingService) *EmailService {
	return &EmailService{settingService: settingService}
}

// Send sends an HTML email to all configured recipients.
func (s *EmailService) Send(subject, body string) error {
	toStr, _ := s.settingService.GetSmtpTo()
	recipients := parseRecipients(toStr)
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients configured")
	}
	return s.SendTo(recipients, subject, body)
}

func (s *EmailService) SendTo(recipients []string, subject, body string) error {
	host, err := s.settingService.GetSmtpHost()
	if err != nil || host == "" {
		return fmt.Errorf("smtp host not configured")
	}
	port, err := s.settingService.GetSmtpPort()
	if err != nil || port <= 0 {
		port = 587
	}
	username, _ := s.settingService.GetSmtpUsername()
	password, _ := s.settingService.GetSmtpPassword()
	encryptionType, _ := s.settingService.GetSmtpEncryptionType()

	from, err := normalizeSMTPAddress(username)
	if err != nil {
		return fmt.Errorf("smtp from not configured")
	}
	recipients, err = normalizeSMTPRecipients(recipients)
	if err != nil || len(recipients) == 0 {
		return fmt.Errorf("no recipients configured")
	}
	subject, err = normalizeSMTPSubject(subject)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	msg := buildMessage(from, recipients, subject, body)

	// Authenticate only when credentials are set. Go's PlainAuth refuses to run
	// over the unencrypted "none" transport, so an open relay must use nil auth.
	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	// Wrap in a channel with timeout to prevent indefinite blocking
	type result struct{ err error }
	ch := make(chan result, 1)
	go func() {
		switch encryptionType {
		case "tls":
			ch <- result{s.sendWithTLS(addr, auth, from, recipients, msg, host)}
		case "starttls":
			ch <- result{s.sendWithSMTP(addr, auth, from, recipients, msg, host, true)}
		case "none":
			ch <- result{s.sendWithSMTP(addr, auth, from, recipients, msg, host, false)}
		default:
			ch <- result{fmt.Errorf("unknown SMTP encryption type: %s", encryptionType)}
		}
	}()

	select {
	case r := <-ch:
		return r.err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("smtp connection timed out after 30s")
	}
}

func (s *EmailService) sendWithSMTP(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string, startTLS bool) error {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Hello("localhost"); err != nil {
		return err
	}
	if startTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		if err = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

// TestConnection tests SMTP connection stage by stage and sends a test email.
func (s *EmailService) TestConnection() SMTPTestResult {
	host, err := s.settingService.GetSmtpHost()
	if err != nil || host == "" {
		return SMTPTestResult{false, "connect", "smtpHostNotConfigured"}
	}
	port, err := s.settingService.GetSmtpPort()
	if err != nil || port <= 0 {
		port = 587
	}
	username, _ := s.settingService.GetSmtpUsername()
	password, _ := s.settingService.GetSmtpPassword()
	toStr, _ := s.settingService.GetSmtpTo()
	encryptionType, _ := s.settingService.GetSmtpEncryptionType()

	from, fromErr := normalizeSMTPAddress(username)
	if fromErr != nil {
		return SMTPTestResult{false, "send", "smtpFromNotConfigured"}
	}
	recipients, recipientsErr := normalizeSMTPRecipients(parseRecipients(toStr))
	if recipientsErr != nil {
		return SMTPTestResult{false, "send", "smtpNoRecipients"}
	}
	if len(recipients) == 0 {
		return SMTPTestResult{false, "send", "smtpNoRecipients"}
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Stage 1: Connect
	var conn net.Conn
	dialer := &net.Dialer{Timeout: 5 * time.Second}

	switch encryptionType {
	case "tls":
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: false,
		}}).DialContext(context.Background(), "tcp", addr)
	default:
		conn, err = dialer.Dial("tcp", addr)
	}

	if err != nil {
		return SMTPTestResult{false, "connect", classifySMTPError(err)}
	}
	defer conn.Close()

	// Stage 2: Handshake + Auth
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return SMTPTestResult{false, "auth", classifySMTPError(err)}
	}
	defer client.Close()

	if err = client.Hello("localhost"); err != nil {
		return SMTPTestResult{false, "auth", classifySMTPError(err)}
	}

	// STARTTLS upgrade for non-TLS connections
	if encryptionType == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return SMTPTestResult{false, "auth", classifySMTPError(err)}
			}
		}
	}

	if username != "" && password != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err = client.Auth(auth); err != nil {
			return SMTPTestResult{false, "auth", classifySMTPError(err)}
		}
	}

	// Stage 3: Send test email
	if err = client.Mail(from); err != nil {
		return SMTPTestResult{false, "send", classifySMTPError(err)}
	}
	for _, r := range recipients {
		if err = client.Rcpt(r); err != nil {
			return SMTPTestResult{false, "send", classifySMTPError(err)}
		}
	}

	msg := buildMessage(from, recipients, "[NOVA] Test email",
		`<html><body style="font-family:monospace;font-size:14px">
<h2>Test email from NOVA</h2>
<p>If you received this, SMTP is configured correctly.</p>
</body></html>`)

	w, err := client.Data()
	if err != nil {
		return SMTPTestResult{false, "send", classifySMTPError(err)}
	}
	if _, err = w.Write(msg); err != nil {
		return SMTPTestResult{false, "send", classifySMTPError(err)}
	}
	if err = w.Close(); err != nil {
		return SMTPTestResult{false, "send", classifySMTPError(err)}
	}

	return SMTPTestResult{true, "send", "smtpTestSuccess"}
}

func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string) error {
	// Dial with explicit timeout
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := (&tls.Dialer{NetDialer: dialer, Config: &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}}).DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Hello("localhost"); err != nil {
		return err
	}
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	for _, r := range to {
		if err = client.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

// SendTest sends a test email and returns any error with detail.
func (s *EmailService) SendTest() error {
	return s.Send(
		"[NOVA] Test email",
		`<html><body style="font-family:monospace;font-size:14px">
<h2>Test email from NOVA</h2>
<p>If you received this, SMTP is configured correctly.</p>
</body></html>`,
	)
}

// classifySMTPError maps raw SMTP errors to human-readable messages.
func classifySMTPError(err error) string {
	msg := err.Error()
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msg, "535") || strings.Contains(msgLower, "authentication"):
		return "pages.settings.smtpErrorAuth"
	case strings.Contains(msg, "534") || strings.Contains(msgLower, "starttls"):
		return "pages.settings.smtpErrorStarttls"
	case strings.Contains(msg, "465") || strings.Contains(msgLower, "tls"):
		return "pages.settings.smtpErrorTls"
	case strings.Contains(msgLower, "connection refused") || strings.Contains(msgLower, "dial"):
		return "pages.settings.smtpErrorRefused"
	case strings.Contains(msgLower, "timeout"):
		return "pages.settings.smtpErrorTimeout"
	case strings.Contains(msg, "550") || strings.Contains(msgLower, "relay"):
		return "pages.settings.smtpErrorRelay"
	case strings.Contains(msgLower, "eof"):
		return "pages.settings.smtpErrorEof"
	default:
		return fmt.Sprintf("pages.settings.smtpErrorUnknown: %s", msg)
	}
}

func parseRecipients(toStr string) []string {
	if toStr == "" {
		return nil
	}
	var out []string
	for s := range strings.SplitSeq(toStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func normalizeSMTPAddress(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("invalid email address")
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address == "" || strings.ContainsAny(address.Address, "\r\n\x00") {
		return "", fmt.Errorf("invalid email address")
	}
	return address.Address, nil
}

func normalizeSMTPRecipients(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("no recipients")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		address, err := normalizeSMTPAddress(value)
		if err != nil {
			return nil, err
		}
		result = append(result, address)
	}
	return result, nil
}

func normalizeSMTPSubject(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("invalid email subject")
	}
	return mime.QEncoding.Encode("UTF-8", value), nil
}

func buildMessage(from string, _ []string, subject, body string) []byte {
	headers := map[string]string{
		"From": from,
		// Envelope recipients are supplied to SMTP RCPT after strict address
		// parsing. Keeping user-controlled addresses out of the MIME headers
		// removes the remaining header/content injection data path.
		"To":                        "undisclosed-recipients:;",
		"Subject":                   subject,
		"MIME-Version":              "1.0",
		"Content-Type":              "text/html; charset=utf-8",
		"Content-Transfer-Encoding": "base64",
	}
	var msg strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	for len(encodedBody) > 76 {
		msg.WriteString(encodedBody[:76])
		msg.WriteString("\r\n")
		encodedBody = encodedBody[76:]
	}
	msg.WriteString(encodedBody)
	return []byte(msg.String())
}
