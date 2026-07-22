package email

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	stdhtml "html"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// EmailService sends email notifications via SMTP.
type EmailService struct {
	settingService service.SettingService
	brandProvider  BrandProvider
}

// BrandProvider supplies the commercial site identity without coupling the
// SMTP package to the commercial service package.
type BrandProvider interface {
	GetDefault(key, fallback string) string
}

// SMTPTestResult holds the result of an SMTP connection test.
type SMTPTestResult struct {
	Success bool   `json:"success"`
	Stage   string `json:"stage"`   // "connect" | "auth" | "send"
	Message string `json:"message"` // classified error message
}

// NewEmailService creates a new EmailService.
func NewEmailService(settingService service.SettingService, brandProviders ...BrandProvider) *EmailService {
	var brandProvider BrandProvider
	if len(brandProviders) > 0 {
		brandProvider = brandProviders[0]
	}
	return &EmailService{settingService: settingService, brandProvider: brandProvider}
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

	fromValue, _ := s.settingService.GetSmtpFrom()
	from, err := normalizeSMTPAddress(fromValue)
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
	msg := buildMessage(from, recipients, subject, body, s.siteName())
	heloName := s.helloName()

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
			ch <- result{s.sendWithTLS(addr, auth, from, recipients, msg, host, heloName)}
		case "starttls":
			ch <- result{s.sendWithSMTP(addr, auth, from, recipients, msg, host, heloName, true)}
		case "none":
			ch <- result{s.sendWithSMTP(addr, auth, from, recipients, msg, host, heloName, false)}
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

func (s *EmailService) sendWithSMTP(addr string, auth smtp.Auth, from string, to []string, msg []byte, host, heloName string, startTLS bool) error {
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

	if err = client.Hello(heloName); err != nil {
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

	fromValue, _ := s.settingService.GetSmtpFrom()
	from, fromErr := normalizeSMTPAddress(fromValue)
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
	heloName := s.helloName()

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

	if err = client.Hello(heloName); err != nil {
		return SMTPTestResult{false, "auth", classifySMTPError(err)}
	}

	// STARTTLS upgrade for non-TLS connections
	if encryptionType == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return SMTPTestResult{false, "auth", "pages.settings.smtpErrorStarttls"}
		}
		if err = client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return SMTPTestResult{false, "auth", classifySMTPError(err)}
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

	rawSubject, body := s.testMessage()
	subject, subjectErr := normalizeSMTPSubject(rawSubject)
	if subjectErr != nil {
		return SMTPTestResult{false, "send", classifySMTPError(subjectErr)}
	}
	msg := buildMessage(from, recipients, subject, body, s.siteName())

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

func (s *EmailService) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host, heloName string) error {
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

	if err = client.Hello(heloName); err != nil {
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
	subject, body := s.testMessage()
	return s.Send(subject, body)
}

func (s *EmailService) siteName() string {
	if s.brandProvider == nil {
		return "NOVA"
	}
	return normalizeSMTPDisplayName(s.brandProvider.GetDefault("site.name", "NOVA"))
}

func (s *EmailService) helloName() string {
	if s.brandProvider != nil {
		if value := normalizeSMTPHelloName(s.brandProvider.GetDefault("site.url", "")); value != "" {
			return value
		}
	}
	if domain, err := s.settingService.GetWebDomain(); err == nil {
		if value := normalizeSMTPHelloName(domain); value != "" {
			return value
		}
	}
	return "localhost"
}

func (s *EmailService) testMessage() (string, string) {
	siteName := s.siteName()
	safeSiteName := stdhtml.EscapeString(siteName)
	return fmt.Sprintf("[%s] Test email", siteName), fmt.Sprintf(`<html><body style="font-family:Arial,sans-serif;font-size:14px;color:#17233d">
<h2>Test email from %s</h2>
<p>If you received this, SMTP is configured correctly.</p>
</body></html>`, safeSiteName)
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

func normalizeSMTPDisplayName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 || r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
	if value == "" {
		return "NOVA"
	}
	return value
}

func normalizeSMTPHelloName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return ""
	}
	candidate := value
	if parsed, err := url.Parse(value); err == nil && parsed.Hostname() != "" {
		candidate = parsed.Hostname()
	} else if parsed, err := url.Parse("//" + value); err == nil && parsed.Hostname() != "" {
		candidate = parsed.Hostname()
	}
	candidate = strings.TrimSuffix(strings.ToLower(candidate), ".")
	if ip := net.ParseIP(candidate); ip != nil {
		return "[" + ip.String() + "]"
	}
	if len(candidate) == 0 || len(candidate) > 253 {
		return ""
	}
	for _, label := range strings.Split(candidate, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return ""
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return ""
			}
		}
	}
	return candidate
}

func buildMessage(from string, recipients []string, subject, body string, displayNames ...string) []byte {
	displayName := ""
	if len(displayNames) > 0 {
		displayName = normalizeSMTPDisplayName(displayNames[0])
	}
	fromHeader := (&mail.Address{Name: displayName, Address: from}).String()
	toHeader := "undisclosed-recipients:;"
	if len(recipients) == 1 {
		toHeader = (&mail.Address{Address: recipients[0]}).String()
	}
	contentType, multipartBody := buildAlternativeBody(body)
	headers := [][2]string{
		{"From", fromHeader},
		{"Reply-To", fromHeader},
		{"To", toHeader},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"Message-ID", newMessageID(from)},
		{"Subject", subject},
		{"MIME-Version", "1.0"},
		{"Content-Type", contentType},
		{"X-Mailer", displayName},
	}
	var msg strings.Builder
	for _, header := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", header[0], header[1])
	}
	msg.WriteString("\r\n")
	msg.WriteString(multipartBody)
	return []byte(msg.String())
}

func buildAlternativeBody(htmlBody string) (string, string) {
	boundary := newMIMEBoundary()
	var body strings.Builder
	writeMIMEPart := func(contentType, value string) {
		fmt.Fprintf(&body, "--%s\r\n", boundary)
		fmt.Fprintf(&body, "Content-Type: %s; charset=utf-8\r\n", contentType)
		body.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		writeBase64Lines(&body, []byte(value))
		body.WriteString("\r\n")
	}
	writeMIMEPart("text/plain", htmlToPlainText(htmlBody))
	writeMIMEPart("text/html", htmlBody)
	fmt.Fprintf(&body, "--%s--\r\n", boundary)
	return mime.FormatMediaType("multipart/alternative", map[string]string{"boundary": boundary}), body.String()
}

func writeBase64Lines(target *strings.Builder, value []byte) {
	encodedBody := base64.StdEncoding.EncodeToString(value)
	for len(encodedBody) > 76 {
		target.WriteString(encodedBody[:76])
		target.WriteString("\r\n")
		encodedBody = encodedBody[76:]
	}
	target.WriteString(encodedBody)
}

func newMIMEBoundary() string {
	var random [18]byte
	if _, err := rand.Read(random[:]); err == nil {
		return fmt.Sprintf("=_panel_%x", random)
	}
	return fmt.Sprintf("=_panel_%d", time.Now().UnixNano())
}

func htmlToPlainText(markup string) string {
	document, err := xhtml.Parse(strings.NewReader(markup))
	if err != nil {
		return strings.TrimSpace(stdhtml.UnescapeString(markup))
	}
	var output strings.Builder
	var lastByte byte
	appendBreak := func() {
		if output.Len() > 0 && lastByte != '\n' {
			output.WriteByte('\n')
			lastByte = '\n'
		}
	}
	blockElements := map[string]bool{
		"address": true, "article": true, "blockquote": true, "div": true,
		"footer": true, "h1": true, "h2": true, "h3": true, "h4": true,
		"h5": true, "h6": true, "header": true, "li": true, "p": true,
		"section": true, "table": true, "tr": true,
	}
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && (node.Data == "script" || node.Data == "style") {
			return
		}
		if node.Type == xhtml.ElementNode && (node.Data == "br" || blockElements[node.Data]) {
			appendBreak()
		}
		if node.Type == xhtml.TextNode {
			text := strings.Join(strings.Fields(node.Data), " ")
			if text != "" {
				if output.Len() > 0 && lastByte != '\n' && lastByte != ' ' {
					output.WriteByte(' ')
				}
				output.WriteString(text)
				lastByte = text[len(text)-1]
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if node.Type == xhtml.ElementNode && blockElements[node.Data] {
			appendBreak()
		}
	}
	visit(document)
	lines := strings.Split(output.String(), "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}

func newMessageID(from string) string {
	domain := "localhost"
	if at := strings.LastIndexByte(from, '@'); at >= 0 && at+1 < len(from) {
		if candidate := normalizeSMTPHelloName(from[at+1:]); candidate != "" {
			domain = candidate
		}
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), domain)
	}
	return fmt.Sprintf("<%d.%x@%s>", time.Now().UnixNano(), random, domain)
}
