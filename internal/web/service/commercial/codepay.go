package commercial

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
)

// CodepayProvider implements the commonly deployed CodePay/码支付 redirect
// protocol. Deployments use different hosts, so the complete create-order
// endpoint remains administrator-configurable.
type CodepayProvider struct {
	gatewayURL  string
	merchantID  string
	merchantKey string
	paymentType string
	notifyURL   string
	returnURL   string
}

func NewCodepayProvider(config *ConfigStore) (*CodepayProvider, error) {
	gatewayURL := strings.TrimSpace(config.GetDefault("codepay.gateway_url", ""))
	merchantID := strings.TrimSpace(config.GetDefault("codepay.id", ""))
	merchantKey, err := config.Get("codepay.key")
	if gatewayURL == "" || merchantID == "" || err != nil || strings.TrimSpace(merchantKey) == "" {
		return nil, errors.New("码支付网关、商户 ID 或通信密钥尚未配置")
	}
	parsed, err := url.ParseRequestURI(gatewayURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("码支付网关地址无效")
	}
	paymentType := strings.TrimSpace(config.GetDefault("codepay.type", "1"))
	if paymentType != "1" && paymentType != "2" && paymentType != "3" {
		return nil, errors.New("码支付通道配置无效")
	}
	return &CodepayProvider{
		gatewayURL:  gatewayURL,
		merchantID:  merchantID,
		merchantKey: strings.TrimSpace(merchantKey),
		paymentType: paymentType,
		notifyURL:   strings.TrimSpace(config.GetDefault("codepay.notify_url", "")),
		returnURL:   strings.TrimSpace(config.GetDefault("codepay.return_url", "")),
	}, nil
}

func (p *CodepayProvider) Name() string { return "codepay" }

func (p *CodepayProvider) Precreate(_ context.Context, request PaymentRequest) (*PaymentIntent, error) {
	if p.notifyURL == "" || p.returnURL == "" {
		return nil, errors.New("码支付异步通知地址或支付后跳转地址尚未配置")
	}
	parsed, err := url.Parse(p.gatewayURL)
	if err != nil {
		return nil, errors.New("码支付网关地址无效")
	}
	values := parsed.Query()
	values.Set("id", p.merchantID)
	values.Set("pay_id", request.OutTradeNo)
	values.Set("type", p.paymentType)
	values.Set("price", formatAmountFen(request.AmountFen))
	values.Set("param", request.Subject)
	values.Set("notify_url", p.notifyURL)
	values.Set("return_url", p.returnURL)
	values.Set("sign", codepaySign(values, p.merchantKey))
	parsed.RawQuery = values.Encode()
	return &PaymentIntent{
		Provider:   p.Name(),
		OutTradeNo: request.OutTradeNo,
		QRCode:     parsed.String(),
		AmountFen:  request.AmountFen,
		ExpiresAt:  request.ExpiresAt,
	}, nil
}

func (p *CodepayProvider) VerifyNotification(values map[string][]string) (*PaymentNotification, error) {
	if id := strings.TrimSpace(firstPaymentValue(values, "id")); id != "" && id != p.merchantID {
		return nil, errors.New("码支付回调商户 ID 不匹配")
	}
	expected := codepaySign(values, p.merchantKey)
	provided := strings.ToLower(strings.TrimSpace(firstPaymentValue(values, "sign")))
	if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		return nil, errors.New("码支付回调签名验证失败")
	}
	status := strings.ToUpper(strings.TrimSpace(firstNonEmpty(firstPaymentValue(values, "trade_status"), firstPaymentValue(values, "status"))))
	if status != "" && status != "TRADE_SUCCESS" && status != "SUCCESS" && status != "1" {
		return nil, errors.New("码支付交易尚未成功")
	}
	outTradeNo := strings.TrimSpace(firstNonEmpty(firstPaymentValue(values, "pay_id"), firstPaymentValue(values, "out_trade_no")))
	providerTrade := strings.TrimSpace(firstNonEmpty(firstPaymentValue(values, "pay_no"), firstPaymentValue(values, "trade_no"), firstPaymentValue(values, "transaction_id")))
	if outTradeNo == "" {
		return nil, errors.New("码支付回调订单号缺失")
	}
	if providerTrade == "" {
		providerTrade = outTradeNo
	}
	amount, err := parseAmountFen(firstNonEmpty(firstPaymentValue(values, "money"), firstPaymentValue(values, "price"), firstPaymentValue(values, "total_amount")))
	if err != nil {
		return nil, err
	}
	return &PaymentNotification{
		OutTradeNo:     outTradeNo,
		ProviderTrade:  providerTrade,
		AmountFen:      amount,
		TradeStatus:    "TRADE_SUCCESS",
		RawPayload:     encodePaymentValues(values),
		NotificationID: providerTrade,
	}, nil
}

// The legacy CodePay protocol has no portable query endpoint across hosted
// deployments. Missed notifications remain visible for administrator-side
// reconciliation instead of guessing a vendor-specific endpoint.
func (p *CodepayProvider) Query(_ context.Context, outTradeNo string) (*PaymentQuery, error) {
	return &PaymentQuery{OutTradeNo: outTradeNo, TradeStatus: "WAIT_BUYER_PAY"}, nil
}

func codepaySign(values map[string][]string, merchantKey string) string {
	keys := make([]string, 0, len(values))
	for key, items := range values {
		if key == "sign" || key == "token" || len(items) == 0 || items[0] == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key][0])
	}
	digest := md5.Sum([]byte(strings.Join(parts, "&") + merchantKey))
	return hex.EncodeToString(digest[:])
}
