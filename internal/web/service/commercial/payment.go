package commercial

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PaymentRequest struct {
	OutTradeNo string
	Subject    string
	AmountFen  int64
	ExpiresAt  time.Time
}

type PaymentIntent struct {
	Provider      string    `json:"provider"`
	OutTradeNo    string    `json:"outTradeNo"`
	ProviderTrade string    `json:"providerTradeNo,omitempty"`
	QRCode        string    `json:"qrCode"`
	AmountFen     int64     `json:"amountFen"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

type PaymentNotification struct {
	OutTradeNo     string
	ProviderTrade  string
	AmountFen      int64
	TradeStatus    string
	RawPayload     string
	NotificationID string
}

type PaymentQuery struct {
	OutTradeNo    string
	ProviderTrade string
	AmountFen     int64
	TradeStatus   string
}

type PaymentProvider interface {
	Name() string
	Precreate(ctx context.Context, request PaymentRequest) (*PaymentIntent, error)
	VerifyNotification(values map[string][]string) (*PaymentNotification, error)
	Query(ctx context.Context, outTradeNo string) (*PaymentQuery, error)
}

type PaymentMethod struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var paymentMethodDefinitions = []PaymentMethod{
	{Code: "epay", Name: "Epay"},
	{Code: "alipay_f2f", Name: "AlipayF2F"},
	{Code: "codepay", Name: "码支付"},
}

type DemoPaymentProvider struct{}

func (DemoPaymentProvider) Name() string { return "alipay-demo" }

func (DemoPaymentProvider) Precreate(_ context.Context, request PaymentRequest) (*PaymentIntent, error) {
	return &PaymentIntent{Provider: "alipay-demo", OutTradeNo: request.OutTradeNo, QRCode: "https://open.alipay.com/", AmountFen: request.AmountFen, ExpiresAt: request.ExpiresAt}, nil
}

func (DemoPaymentProvider) VerifyNotification(values map[string][]string) (*PaymentNotification, error) {
	outTradeNo := firstPaymentValue(values, "out_trade_no")
	amount, err := parseAmountFen(firstPaymentValue(values, "total_amount"))
	if err != nil {
		return nil, err
	}
	return &PaymentNotification{OutTradeNo: outTradeNo, ProviderTrade: firstPaymentValue(values, "trade_no"), AmountFen: amount, TradeStatus: firstPaymentValue(values, "trade_status"), RawPayload: encodePaymentValues(values), NotificationID: firstPaymentValue(values, "notify_id")}, nil
}

func (DemoPaymentProvider) Query(_ context.Context, outTradeNo string) (*PaymentQuery, error) {
	return &PaymentQuery{OutTradeNo: outTradeNo, TradeStatus: "WAIT_BUYER_PAY"}, nil
}

func ActivePaymentProvider(config *ConfigStore) (PaymentProvider, error) {
	if isDemoMode() {
		return DemoPaymentProvider{}, nil
	}
	for _, method := range EnabledPaymentMethods(config) {
		provider, err := PaymentProviderByName(config, method.Code)
		if err == nil {
			return provider, nil
		}
	}
	return nil, errors.New("当前没有已启用且配置完整的支付接口")
}

func PaymentProviderByName(config *ConfigStore, name string) (PaymentProvider, error) {
	switch normalizePaymentProviderName(name) {
	case "balance", "alipay-demo":
		return DemoPaymentProvider{}, nil
	case "alipay_f2f":
		return NewAlipayProvider(config)
	case "epay":
		return NewEpayProvider(config)
	case "codepay":
		return NewCodepayProvider(config)
	default:
		return nil, fmt.Errorf("不支持的支付接口: %s", name)
	}
}

func EnabledPaymentMethods(config *ConfigStore) []PaymentMethod {
	if isDemoMode() {
		return []PaymentMethod{{Code: "alipay-demo", Name: "演示支付"}}
	}
	methods := make([]PaymentMethod, 0, len(paymentMethodDefinitions))
	for _, method := range paymentMethodDefinitions {
		if !PaymentProviderEnabled(config, method.Code) {
			continue
		}
		if _, err := PaymentProviderByName(config, method.Code); err == nil {
			methods = append(methods, method)
		}
	}
	return methods
}

func PaymentProviderEnabled(config *ConfigStore, name string) bool {
	name = normalizePaymentProviderName(name)
	if value, err := config.Get("payment." + name + ".enabled"); err == nil {
		enabled, parseErr := strconv.ParseBool(strings.TrimSpace(value))
		return parseErr == nil && enabled
	}
	// Compatibility with installations configured before each interface had
	// its own switch: only the formerly active interface remains enabled.
	return normalizePaymentProviderName(config.GetDefault("payment.provider", "alipay_f2f")) == name
}

func normalizePaymentProviderName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "alipay", "alipay_f2f":
		return "alipay_f2f"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

func firstPaymentValue(values map[string][]string, key string) string {
	items := values[key]
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func encodePaymentValues(values map[string][]string) string {
	pairs := make([]string, 0, len(values))
	for key, items := range values {
		if len(items) > 0 {
			pairs = append(pairs, key+"="+items[0])
		}
	}
	return strings.Join(pairs, "&")
}

func parseAmountFen(value string) (int64, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts) == 0 || parts[0] == "" {
		return 0, errors.New("invalid payment amount")
	}
	var whole int64
	for _, char := range parts[0] {
		if char < '0' || char > '9' {
			return 0, errors.New("invalid payment amount")
		}
		whole = whole*10 + int64(char-'0')
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 2 {
		return 0, errors.New("invalid payment amount")
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	var cents int64
	for _, char := range fraction {
		if char < '0' || char > '9' {
			return 0, errors.New("invalid payment amount")
		}
		cents = cents*10 + int64(char-'0')
	}
	if whole > (1<<63-1-cents)/100 {
		return 0, fmt.Errorf("payment amount is too large")
	}
	return whole*100 + cents, nil
}
