package email

import (
	"encoding/base64"
	"strings"
	"testing"
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
}
