package commercial

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestEpayPrecreateAndQuery(t *testing.T) {
	const merchantKey = "merchant-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mapi.php":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got, want := r.Form.Get("sign"), epaySign(r.Form, merchantKey); got != want {
				t.Fatalf("invalid precreate sign: got %q want %q", got, want)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "qrcode": "alipays://platformapi/startapp", "trade_no": "EPAY-1"})
		case "/api.php":
			if r.URL.Query().Get("act") != "order" || r.URL.Query().Get("key") != merchantKey {
				t.Fatalf("unexpected query parameters: %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 1, "status": 1, "money": "12.34", "trade_no": "EPAY-1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &EpayProvider{
		gatewayURL:  server.URL,
		merchantID:  "1000",
		merchantKey: merchantKey,
		paymentType: "alipay",
		notifyURL:   "https://example.com/api/v1/guest/payments/epay/notify",
		returnURL:   "https://example.com/portal/",
		siteName:    "NOVA",
		client:      server.Client(),
	}
	intent, err := provider.Precreate(context.Background(), PaymentRequest{OutTradeNo: "ORDER-1", Subject: "专业套餐", AmountFen: 1234, ExpiresAt: time.Now().Add(15 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Provider != "epay" || intent.QRCode != "alipays://platformapi/startapp" || intent.ProviderTrade != "EPAY-1" {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	query, err := provider.Query(context.Background(), "ORDER-1")
	if err != nil {
		t.Fatal(err)
	}
	if query.TradeStatus != "TRADE_SUCCESS" || query.AmountFen != 1234 || query.ProviderTrade != "EPAY-1" {
		t.Fatalf("unexpected query: %#v", query)
	}
}

func TestEpayNotificationSignatureAndTamperGuard(t *testing.T) {
	provider := &EpayProvider{merchantID: "1000", merchantKey: "merchant-secret"}
	values := url.Values{
		"pid": {"1000"}, "trade_no": {"EPAY-2"}, "out_trade_no": {"ORDER-2"},
		"type": {"alipay"}, "name": {"专业套餐"}, "money": {"20.00"},
		"trade_status": {"TRADE_SUCCESS"}, "sign_type": {"MD5"},
	}
	values.Set("sign", epaySign(values, provider.merchantKey))
	notification, err := provider.VerifyNotification(values)
	if err != nil {
		t.Fatal(err)
	}
	if notification.AmountFen != 2000 || notification.OutTradeNo != "ORDER-2" {
		t.Fatalf("unexpected notification: %#v", notification)
	}
	values.Set("money", "2.00")
	if _, err := provider.VerifyNotification(values); err == nil {
		t.Fatal("tampered amount must fail signature verification")
	}
}

func TestEpayEndpointAcceptsRootAndMAPIURL(t *testing.T) {
	for input, want := range map[string]string{
		"https://pay.example.com":                    "https://pay.example.com/mapi.php",
		"https://pay.example.com/mapi.php":           "https://pay.example.com/mapi.php",
		"https://pay.example.com/payment/submit.php": "https://pay.example.com/payment/mapi.php",
	} {
		got, err := epayEndpoint(input, "mapi.php")
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got != want {
			t.Fatalf("%s: got %s want %s", input, got, want)
		}
	}
}
