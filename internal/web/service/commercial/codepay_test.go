package commercial

import (
	"context"
	"net/url"
	"testing"
	"time"
)

func TestCodepayPrecreateBuildsSignedPaymentURL(t *testing.T) {
	provider := &CodepayProvider{
		gatewayURL:  "https://codepay.example/creat_order/",
		merchantID:  "10041",
		merchantKey: "codepay-secret",
		paymentType: "1",
		notifyURL:   "https://airport.example/api/v1/guest/payments/codepay/notify",
		returnURL:   "https://airport.example/portal/",
	}
	intent, err := provider.Precreate(context.Background(), PaymentRequest{OutTradeNo: "ORDER-CODE-1", Subject: "专业套餐", AmountFen: 2088, ExpiresAt: time.Now().Add(15 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(intent.QRCode)
	if err != nil {
		t.Fatal(err)
	}
	values := parsed.Query()
	if values.Get("id") != "10041" || values.Get("pay_id") != "ORDER-CODE-1" || values.Get("price") != "20.88" || values.Get("type") != "1" {
		t.Fatalf("unexpected payment URL values: %s", parsed.RawQuery)
	}
	if got, want := values.Get("sign"), codepaySign(values, provider.merchantKey); got != want {
		t.Fatalf("invalid payment URL signature: got %q want %q", got, want)
	}
}

func TestCodepayNotificationSignatureAndAmountTamperGuard(t *testing.T) {
	provider := &CodepayProvider{merchantID: "10041", merchantKey: "codepay-secret"}
	values := url.Values{
		"id": {"10041"}, "pay_id": {"ORDER-CODE-2"}, "pay_no": {"CODEPAY-2"},
		"type": {"1"}, "money": {"45.00"}, "status": {"1"},
	}
	values.Set("sign", codepaySign(values, provider.merchantKey))
	notification, err := provider.VerifyNotification(values)
	if err != nil {
		t.Fatal(err)
	}
	if notification.AmountFen != 4500 || notification.OutTradeNo != "ORDER-CODE-2" || notification.ProviderTrade != "CODEPAY-2" {
		t.Fatalf("unexpected notification: %#v", notification)
	}
	values.Set("money", "4.50")
	if _, err := provider.VerifyNotification(values); err == nil {
		t.Fatal("tampered amount must fail signature verification")
	}
}

func TestPaymentProviderEnablementIsIndependent(t *testing.T) {
	initCommercialTestDB(t)
	config := NewConfigStore()
	if err := config.Set("payment.epay.enabled", "true", false); err != nil {
		t.Fatal(err)
	}
	if err := config.Set("payment.alipay_f2f.enabled", "false", false); err != nil {
		t.Fatal(err)
	}
	if err := config.Set("payment.codepay.enabled", "true", false); err != nil {
		t.Fatal(err)
	}
	if !PaymentProviderEnabled(config, "epay") || PaymentProviderEnabled(config, "alipay_f2f") || !PaymentProviderEnabled(config, "codepay") {
		t.Fatal("payment provider switches were not evaluated independently")
	}
}

func TestEnabledPaymentMethodsReturnsEveryConfiguredEnabledProvider(t *testing.T) {
	initCommercialTestDB(t)
	config := NewConfigStore()
	settings := map[string]string{
		"payment.epay.enabled":       "true",
		"payment.alipay_f2f.enabled": "false",
		"payment.codepay.enabled":    "true",
		"epay.gateway_url":           "https://epay.example.com",
		"epay.pid":                   "1000",
		"epay.key":                   "epay-secret",
		"codepay.gateway_url":        "https://codepay.example.com/creat_order/",
		"codepay.id":                 "10041",
		"codepay.key":                "codepay-secret",
	}
	for key, value := range settings {
		if err := config.Set(key, value, false); err != nil {
			t.Fatalf("set %s: %v", key, err)
		}
	}

	methods := EnabledPaymentMethods(config)
	if len(methods) != 2 || methods[0].Code != "epay" || methods[1].Code != "codepay" {
		t.Fatalf("expected Epay and CodePay to be exposed, got %#v", methods)
	}
}
