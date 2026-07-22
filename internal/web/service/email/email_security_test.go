package email

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	webservice "github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func TestNormalizeSMTPAddressRejectsHeaderInjection(t *testing.T) {
	for _, value := range []string{
		"victim@example.com\r\nBcc: attacker@example.com",
		"victim@example.com\nCc: attacker@example.com",
		"victim@example.com\x00attacker@example.com",
	} {
		if _, err := normalizeSMTPAddress(value); err == nil {
			t.Fatalf("normalizeSMTPAddress(%q) accepted header injection", value)
		}
	}
}

func TestNormalizeSMTPSubjectRejectsHeaderInjection(t *testing.T) {
	if _, err := normalizeSMTPSubject("hello\r\nBcc: attacker@example.com"); err == nil {
		t.Fatal("subject header injection was accepted")
	}
	if got, err := normalizeSMTPSubject("安全通知"); err != nil || got == "" {
		t.Fatalf("valid UTF-8 subject rejected: got=%q err=%v", got, err)
	}
}

func TestNormalizeSMTPRecipientsRejectsInvalidMember(t *testing.T) {
	if _, err := normalizeSMTPRecipients([]string{"valid@example.com", "bad\r\nBcc: attacker@example.com"}); err == nil {
		t.Fatal("recipient list accepted an injected address")
	}
}

func TestBuildMessageBase64EncodesBody(t *testing.T) {
	body := "<p>Hello</p>\r\nBcc: attacker@example.com"
	rawMessage := string(buildMessage("sender@example.com", []string{"user@example.com"}, "subject", body))
	message, err := mail.ReadMessage(strings.NewReader(rawMessage))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/alternative" {
		t.Fatalf("content type = %q params=%v err=%v", mediaType, params, err)
	}
	rawBody, err := io.ReadAll(message.Body)
	if err != nil {
		t.Fatalf("read message body: %v", err)
	}
	if strings.Contains(string(rawBody), "Bcc:") {
		t.Fatal("untrusted body was written as raw SMTP content")
	}
	partReader := multipart.NewReader(strings.NewReader(string(rawBody)), params["boundary"])
	decodedParts := map[string]string{}
	for {
		part, nextErr := partReader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			t.Fatalf("read MIME part: %v", nextErr)
		}
		decoded, decodeErr := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
		if decodeErr != nil {
			t.Fatalf("decode MIME part: %v", decodeErr)
		}
		partType, _, parseErr := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if parseErr != nil {
			t.Fatalf("parse MIME part content type: %v", parseErr)
		}
		decodedParts[partType] = string(decoded)
	}
	if decodedParts["text/html"] != body {
		t.Fatalf("decoded HTML body = %q, want %q", decodedParts["text/html"], body)
	}
	if decodedParts["text/plain"] != "Hello\nBcc: attacker@example.com" {
		t.Fatalf("decoded plain body = %q", decodedParts["text/plain"])
	}
	if message.Header.Get("To") != "<user@example.com>" {
		t.Fatal("single-recipient message does not identify its recipient")
	}
	for _, header := range []string{"Date: ", "Message-ID: ", "Reply-To: <sender@example.com>"} {
		if !strings.Contains(rawMessage, header) {
			t.Fatalf("message is missing deliverability header %q", header)
		}
	}
}

func TestBuildMessageUsesSiteNameAndProtectsMultipleRecipients(t *testing.T) {
	msg := string(buildMessage(
		"sender@example.com",
		[]string{"first@example.com", "second@example.com"},
		"subject",
		"<p>Hello</p>",
		"PHEERO",
	))
	headers := strings.SplitN(msg, "\r\n\r\n", 2)[0]
	if !strings.Contains(headers, `From: "PHEERO" <sender@example.com>`) {
		t.Fatalf("site display name missing from From header: %s", headers)
	}
	if !strings.Contains(headers, "To: undisclosed-recipients:;") {
		t.Fatal("multi-recipient message does not protect recipient addresses")
	}
	for _, recipient := range []string{"first@example.com", "second@example.com"} {
		if strings.Contains(headers, recipient) {
			t.Fatalf("multi-recipient MIME header leaked %s", recipient)
		}
	}
}

type testBrandProvider map[string]string

func (p testBrandProvider) GetDefault(key, fallback string) string {
	if value := p[key]; value != "" {
		return value
	}
	return fallback
}

func TestEmailServiceUsesCurrentCommercialBrand(t *testing.T) {
	provider := testBrandProvider{
		"site.name": "PHEERO <Secure>",
		"site.url":  "https://vpn.pheero.com/",
	}
	emailService := NewEmailService(webservice.SettingService{}, provider)
	subject, body := emailService.testMessage()
	if subject != "[PHEERO <Secure>] Test email" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(body, "PHEERO &lt;Secure&gt;") || strings.Contains(body, "PHEERO <Secure>") {
		t.Fatalf("test body does not safely use current site name: %s", body)
	}
	if got := emailService.helloName(); got != "vpn.pheero.com" {
		t.Fatalf("hello name = %q", got)
	}
}

func TestNormalizeSMTPHelloName(t *testing.T) {
	for input, want := range map[string]string{
		"https://VPN.PHEERO.com/path": "vpn.pheero.com",
		"mail.example.com:587":        "mail.example.com",
		"127.0.0.1":                   "[127.0.0.1]",
		"bad host":                    "",
		"bad.example.com\r\nMAIL":     "",
	} {
		if got := normalizeSMTPHelloName(input); got != want {
			t.Errorf("normalizeSMTPHelloName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSubscriberEscapesUntrustedEventHTML(t *testing.T) {
	subscriber := &Subscriber{}
	_, body := subscriber.formatMessage(eventbus.Event{
		Type:      eventbus.EventLoginAttempt,
		Source:    "portal",
		Timestamp: time.Unix(1, 0),
		Data: &eventbus.LoginEventData{
			Status:   "failed",
			Username: `<img src=x onerror="alert(1)">`,
			IP:       `<script>alert(2)</script>`,
			Reason:   `<a href="https://attacker.invalid">click</a>`,
		},
	})
	for _, raw := range []string{"<img", "<script", "<a href"} {
		if strings.Contains(body, raw) {
			t.Fatalf("untrusted event HTML was not escaped: %q", raw)
		}
	}
	for _, escaped := range []string{"&lt;img", "&lt;script", "&lt;a href"} {
		if !strings.Contains(body, escaped) {
			t.Fatalf("escaped event value missing: %q", escaped)
		}
	}
}
