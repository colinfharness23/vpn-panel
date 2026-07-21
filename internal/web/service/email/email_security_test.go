package email

import "testing"

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
