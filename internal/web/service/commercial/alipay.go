package commercial

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type AlipayProvider struct {
	appID      string
	sellerID   string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	gateway    string
	notifyURL  string
	client     *http.Client
}

func NewAlipayProvider(config *ConfigStore) (*AlipayProvider, error) {
	appID, err := config.Get("alipay.app_id")
	if err != nil || appID == "" {
		return nil, errors.New("支付宝 App ID 尚未配置")
	}
	privatePEM, err := config.Get("alipay.private_key")
	if err != nil {
		return nil, errors.New("支付宝应用私钥尚未配置")
	}
	publicPEM, err := config.Get("alipay.public_key")
	if err != nil {
		return nil, errors.New("支付宝公钥尚未配置")
	}
	privateKey, err := parseRSAPrivateKey(privatePEM)
	if err != nil {
		return nil, err
	}
	publicKey, err := parseRSAPublicKey(publicPEM)
	if err != nil {
		return nil, err
	}
	mode := strings.ToLower(config.GetDefault("alipay.mode", "sandbox"))
	gateway := "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	if mode == "production" {
		gateway = "https://openapi.alipay.com/gateway.do"
	}
	return &AlipayProvider{appID: appID, sellerID: config.GetDefault("alipay.seller_id", ""), privateKey: privateKey, publicKey: publicKey, gateway: gateway, notifyURL: config.GetDefault("alipay.notify_url", ""), client: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (p *AlipayProvider) Name() string { return "alipay" }

func (p *AlipayProvider) Precreate(ctx context.Context, request PaymentRequest) (*PaymentIntent, error) {
	bizContent, err := json.Marshal(map[string]any{
		"out_trade_no":    request.OutTradeNo,
		"total_amount":    formatAmountFen(request.AmountFen),
		"subject":         request.Subject,
		"timeout_express": fmt.Sprintf("%dm", max(1, int(time.Until(request.ExpiresAt).Minutes()))),
	})
	if err != nil {
		return nil, err
	}
	values := p.baseValues("alipay.trade.precreate", string(bizContent))
	if p.notifyURL != "" {
		values.Set("notify_url", p.notifyURL)
	}
	var response struct {
		Payload struct {
			Code    string `json:"code"`
			Message string `json:"msg"`
			SubCode string `json:"sub_code"`
			SubMsg  string `json:"sub_msg"`
			QR      string `json:"qr_code"`
		} `json:"alipay_trade_precreate_response"`
		Sign string `json:"sign"`
	}
	if err := p.call(ctx, values, &response); err != nil {
		return nil, err
	}
	if response.Payload.Code != "10000" || response.Payload.QR == "" {
		return nil, fmt.Errorf("支付宝预下单失败: %s %s", response.Payload.SubCode, response.Payload.SubMsg)
	}
	return &PaymentIntent{Provider: p.Name(), OutTradeNo: request.OutTradeNo, QRCode: response.Payload.QR, AmountFen: request.AmountFen, ExpiresAt: request.ExpiresAt}, nil
}

func (p *AlipayProvider) VerifyNotification(values map[string][]string) (*PaymentNotification, error) {
	if firstPaymentValue(values, "app_id") != p.appID {
		return nil, errors.New("支付宝回调 App ID 不匹配")
	}
	if p.sellerID != "" && firstPaymentValue(values, "seller_id") != p.sellerID {
		return nil, errors.New("支付宝回调商户不匹配")
	}
	signature, err := base64.StdEncoding.DecodeString(firstPaymentValue(values, "sign"))
	if err != nil {
		return nil, errors.New("支付宝回调签名格式无效")
	}
	canonical := canonicalPaymentValues(values, true)
	digest := sha256.Sum256([]byte(canonical))
	if err := rsa.VerifyPKCS1v15(p.publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return nil, errors.New("支付宝回调签名验证失败")
	}
	status := firstPaymentValue(values, "trade_status")
	if status != "TRADE_SUCCESS" && status != "TRADE_FINISHED" {
		return nil, errors.New("支付宝交易尚未成功")
	}
	amount, err := parseAmountFen(firstPaymentValue(values, "total_amount"))
	if err != nil {
		return nil, err
	}
	return &PaymentNotification{OutTradeNo: firstPaymentValue(values, "out_trade_no"), ProviderTrade: firstPaymentValue(values, "trade_no"), AmountFen: amount, TradeStatus: status, RawPayload: canonicalPaymentValues(values, false), NotificationID: firstPaymentValue(values, "notify_id")}, nil
}

func (p *AlipayProvider) Query(ctx context.Context, outTradeNo string) (*PaymentQuery, error) {
	bizContent, _ := json.Marshal(map[string]string{"out_trade_no": outTradeNo})
	values := p.baseValues("alipay.trade.query", string(bizContent))
	var response struct {
		Payload struct {
			Code        string `json:"code"`
			SubCode     string `json:"sub_code"`
			OutTradeNo  string `json:"out_trade_no"`
			TradeNo     string `json:"trade_no"`
			TradeStatus string `json:"trade_status"`
			TotalAmount string `json:"total_amount"`
		} `json:"alipay_trade_query_response"`
	}
	if err := p.call(ctx, values, &response); err != nil {
		return nil, err
	}
	if response.Payload.Code != "10000" {
		if response.Payload.SubCode == "ACQ.TRADE_NOT_EXIST" {
			return &PaymentQuery{OutTradeNo: outTradeNo, TradeStatus: "NOT_EXIST"}, nil
		}
		return nil, fmt.Errorf("支付宝查单失败: %s", response.Payload.SubCode)
	}
	amount, err := parseAmountFen(response.Payload.TotalAmount)
	if err != nil {
		return nil, err
	}
	return &PaymentQuery{OutTradeNo: response.Payload.OutTradeNo, ProviderTrade: response.Payload.TradeNo, AmountFen: amount, TradeStatus: response.Payload.TradeStatus}, nil
}

func (p *AlipayProvider) baseValues(method, bizContent string) url.Values {
	values := url.Values{
		"app_id":      {p.appID},
		"method":      {method},
		"format":      {"JSON"},
		"charset":     {"utf-8"},
		"sign_type":   {"RSA2"},
		"timestamp":   {time.Now().In(time.FixedZone("CST", 8*60*60)).Format("2006-01-02 15:04:05")},
		"version":     {"1.0"},
		"biz_content": {bizContent},
	}
	signature, _ := p.sign(canonicalURLValues(values))
	values.Set("sign", signature)
	return values
}

func (p *AlipayProvider) call(ctx context.Context, values url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.gateway, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("支付宝网关返回 HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(target)
}

func (p *AlipayProvider) sign(payload string) (string, error) {
	digest := sha256.Sum256([]byte(payload))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func canonicalURLValues(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "sign" && values.Get(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	return strings.Join(parts, "&")
}

func formatAmountFen(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	return fmt.Sprintf("%s%d.%02d", sign, amount/100, amount%100)
}

func canonicalPaymentValues(values map[string][]string, excludeSignature bool) string {
	keys := make([]string, 0, len(values))
	for key, items := range values {
		if len(items) == 0 || items[0] == "" {
			continue
		}
		if excludeSignature && (key == "sign" || key == "sign_type") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key][0])
	}
	return strings.Join(parts, "&")
}

func parseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(raw, "PRIVATE KEY")))
	if block == nil {
		return nil, errors.New("支付宝应用私钥格式无效")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("支付宝应用私钥格式无效")
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(raw, "PUBLIC KEY")))
	if block == nil {
		return nil, errors.New("支付宝公钥格式无效")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("支付宝公钥格式无效")
}

func normalizePEM(raw, kind string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.Contains(trimmed, "BEGIN") {
		return trimmed
	}
	return "-----BEGIN " + kind + "-----\n" + trimmed + "\n-----END " + kind + "-----"
}
