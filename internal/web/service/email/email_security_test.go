package email

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
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
	msg := string(buildMessage("sender@example.com", []string{"user@example.com"}, "subject", body))
	parts := strings.SplitN(msg, "\r\n\r\n", 2)
	if len(parts) != 2 {
		t.Fatal("message does not contain a MIME header/body boundary")
	}
	if !strings.Contains(parts[0], "Content-Transfer-Encoding: base64") {
		t.Fatal("message body is not declared as base64")
	}
	if strings.Contains(parts[1], "Bcc:") {
		t.Fatal("untrusted body was written as raw SMTP content")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(parts[1], "\r\n", ""))
	if err != nil {
		t.Fatalf("decode message body: %v", err)
	}
	if string(decoded) != body {
		t.Fatalf("decoded body = %q, want %q", decoded, body)
	}
	if strings.Contains(parts[0], "user@example.com") {
		t.Fatal("envelope recipient leaked into MIME headers")
	}
	if !strings.Contains(parts[0], "To: undisclosed-recipients:;") {
		t.Fatal("message does not use a constant undisclosed-recipient header")
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
