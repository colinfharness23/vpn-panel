package commercial

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EpayProvider implements the commonly used Epay MAPI/MD5 protocol. The
// gateway is configurable because Epay deployments are self-hosted and do not
// share one fixed API host.
type EpayProvider struct {
	gatewayURL  string
	merchantID  string
	merchantKey string
	paymentType string
	notifyURL   string
	returnURL   string
	siteName    string
	client      *http.Client
}

func NewEpayProvider(config *ConfigStore) (*EpayProvider, error) {
	gatewayURL := strings.TrimRight(config.GetDefault("epay.gateway_url", ""), "/")
	merchantID := strings.TrimSpace(config.GetDefault("epay.pid", ""))
	merchantKey, err := config.Get("epay.key")
	if gatewayURL == "" || merchantID == "" || err != nil || strings.TrimSpace(merchantKey) == "" {
		return nil, errors.New("Epay 网关、商户 ID 或商户密钥尚未配置")
	}
	if _, err := epayEndpoint(gatewayURL, "mapi.php"); err != nil {
		return nil, errors.New("Epay 网关地址无效")
	}
	return &EpayProvider{
		gatewayURL:  gatewayURL,
		merchantID:  merchantID,
		merchantKey: strings.TrimSpace(merchantKey),
		paymentType: strings.ToLower(config.GetDefault("epay.type", "alipay")),
		notifyURL:   strings.TrimSpace(config.GetDefault("epay.notify_url", "")),
		returnURL:   strings.TrimSpace(config.GetDefault("epay.return_url", "")),
		siteName:    config.GetDefault("site.name", "NOVA"),
		client:      &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (p *EpayProvider) Name() string { return "epay" }

func (p *EpayProvider) Precreate(ctx context.Context, request PaymentRequest) (*PaymentIntent, error) {
	if p.notifyURL == "" || p.returnURL == "" {
		return nil, errors.New("Epay 异步通知地址或支付后跳转地址尚未配置")
	}
	endpoint, err := epayEndpoint(p.gatewayURL, "mapi.php")
	if err != nil {
		return nil, err
	}
	values := url.Values{
		"pid":          {p.merchantID},
		"type":         {p.paymentType},
		"out_trade_no": {request.OutTradeNo},
		"notify_url":   {p.notifyURL},
		"return_url":   {p.returnURL},
		"name":         {request.Subject},
		"money":        {formatAmountFen(request.AmountFen)},
		"sitename":     {p.siteName},
	}
	values.Set("sign", epaySign(values, p.merchantKey))
	values.Set("sign_type", "MD5")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Epay 预下单请求失败: HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code       json.RawMessage `json:"code"`
		Message    string          `json:"msg"`
		QRCode     string          `json:"qrcode"`
		PayURL     string          `json:"payurl"`
		URL        string          `json:"url"`
		TradeNo    string          `json:"trade_no"`
		OutTradeNo string          `json:"out_trade_no"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, errors.New("Epay 预下单响应格式无效")
	}
	if code := epayJSONScalar(result.Code); code != "1" && code != "200" {
		return nil, fmt.Errorf("Epay 预下单失败: %s", strings.TrimSpace(result.Message))
	}
	qrCode := firstNonEmpty(result.QRCode, result.PayURL, result.URL)
	if qrCode == "" {
		return nil, errors.New("Epay 未返回可生成二维码的支付链接")
	}
	return &PaymentIntent{
		Provider:      p.Name(),
		OutTradeNo:    request.OutTradeNo,
		ProviderTrade: result.TradeNo,
		QRCode:        qrCode,
		AmountFen:     request.AmountFen,
		ExpiresAt:     request.ExpiresAt,
	}, nil
}

func (p *EpayProvider) VerifyNotification(values map[string][]string) (*PaymentNotification, error) {
	if firstPaymentValue(values, "pid") != p.merchantID {
		return nil, errors.New("Epay 回调商户 ID 不匹配")
	}
	if signType := strings.ToUpper(strings.TrimSpace(firstPaymentValue(values, "sign_type"))); signType != "" && signType != "MD5" {
		return nil, errors.New("Epay 回调签名类型无效")
	}
	expected := epaySign(values, p.merchantKey)
	provided := strings.ToLower(strings.TrimSpace(firstPaymentValue(values, "sign")))
	if len(expected) != len(provided) || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		return nil, errors.New("Epay 回调签名验证失败")
	}
	if firstPaymentValue(values, "trade_status") != "TRADE_SUCCESS" {
		return nil, errors.New("Epay 交易尚未成功")
	}
	outTradeNo := strings.TrimSpace(firstPaymentValue(values, "out_trade_no"))
	providerTrade := strings.TrimSpace(firstPaymentValue(values, "trade_no"))
	if outTradeNo == "" || providerTrade == "" {
		return nil, errors.New("Epay 回调订单号缺失")
	}
	amount, err := parseAmountFen(firstPaymentValue(values, "money"))
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

func (p *EpayProvider) Query(ctx context.Context, outTradeNo string) (*PaymentQuery, error) {
	endpoint, err := epayEndpoint(p.gatewayURL, "api.php")
	if err != nil {
		return nil, err
	}
	parsed, _ := url.Parse(endpoint)
	query := parsed.Query()
	query.Set("act", "order")
	query.Set("pid", p.merchantID)
	query.Set("key", p.merchantKey)
	query.Set("out_trade_no", outTradeNo)
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Epay 查单请求失败: HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code    json.RawMessage `json:"code"`
		Status  json.RawMessage `json:"status"`
		TradeNo string          `json:"trade_no"`
		Money   string          `json:"money"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, errors.New("Epay 查单响应格式无效")
	}
	code := epayJSONScalar(result.Code)
	status := strings.ToUpper(epayJSONScalar(result.Status))
	if (code != "1" && code != "200") || (status != "1" && status != "TRADE_SUCCESS") {
		return &PaymentQuery{OutTradeNo: outTradeNo, TradeStatus: "WAIT_BUYER_PAY"}, nil
	}
	amount, err := parseAmountFen(result.Money)
	if err != nil {
		return nil, err
	}
	return &PaymentQuery{OutTradeNo: outTradeNo, ProviderTrade: result.TradeNo, AmountFen: amount, TradeStatus: "TRADE_SUCCESS"}, nil
}

func epaySign(values map[string][]string, merchantKey string) string {
	keys := make([]string, 0, len(values))
	for key, items := range values {
		if key == "sign" || key == "sign_type" || len(items) == 0 || items[0] == "" {
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

func epayEndpoint(gatewayURL, filename string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(gatewayURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("Epay 网关地址无效")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if slash := strings.LastIndex(path, "/"); strings.HasSuffix(strings.ToLower(path), ".php") {
		path = path[:slash+1] + filename
	} else if path == "" {
		path = "/" + filename
	} else {
		path += "/" + filename
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func epayJSONScalar(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if value, err := strconv.Unquote(trimmed); err == nil {
		return strings.TrimSpace(value)
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
