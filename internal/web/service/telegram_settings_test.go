package service

import "testing"

func TestValidateTelegramWebhookURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid root callback", input: "https://vpn.example.com/telegram/webhook", want: "https://vpn.example.com/telegram/webhook"},
		{name: "valid base path callback", input: " https://vpn.example.com/panel/telegram/webhook ", want: "https://vpn.example.com/panel/telegram/webhook"},
		{name: "requires https", input: "http://vpn.example.com/telegram/webhook", wantErr: true},
		{name: "requires receiver path", input: "https://vpn.example.com/hook", wantErr: true},
		{name: "rejects query", input: "https://vpn.example.com/telegram/webhook?token=leak", wantErr: true},
		{name: "required", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateTelegramWebhookURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got URL %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateTelegramGroupLink(t *testing.T) {
	t.Parallel()
	for _, valid := range []string{"https://t.me/example", "https://t.me/+invite", "https://telegram.me/example"} {
		if _, err := validateTelegramGroupLink(valid); err != nil {
			t.Errorf("expected %q to be accepted: %v", valid, err)
		}
	}
	for _, invalid := range []string{"http://t.me/example", "https://example.com/group", "tg://resolve?domain=example", "https://t.me/"} {
		if _, err := validateTelegramGroupLink(invalid); err == nil {
			t.Errorf("expected %q to be rejected", invalid)
		}
	}
}
