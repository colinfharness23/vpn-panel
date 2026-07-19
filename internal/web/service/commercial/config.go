package commercial

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"gorm.io/gorm"
)

type ConfigStore struct {
	db             *gorm.DB
	settingService service.SettingService
}

const (
	defaultTermsTitle = "服务使用条款"
	defaultTermsText  = `欢迎使用本站服务。注册或购买前，请完整阅读以下条款：

1. 账户与安全
用户应提供真实、有效的注册信息，并妥善保管账户、密码及订阅链接。因主动分享或保管不当造成的损失由用户自行承担。

2. 服务使用范围
本站提供网络连接订阅及相关技术支持。用户不得利用服务从事违法活动、网络攻击、垃圾信息发送、欺诈、侵犯他人权益或其他可能影响平台及节点安全的行为。

3. 套餐、流量与设备
套餐流量、有效期、设备限制及重置周期以购买页面显示为准。超出套餐限制或套餐到期后，相关服务可能自动暂停。

4. 支付与款项确认
数字订阅在支付确认后自动开通。订单支付成功后不可撤销或退回款项，请在付款前确认套餐、周期及设备限制。

5. 服务调整与账号处置
为保障节点及其他用户的正常使用，本站可对异常流量、滥用、攻击或违反本条款的账户采取限制、暂停或终止服务等措施。

6. 隐私与数据
本站仅处理提供账户、订阅、支付、安全审计及客户支持所必要的信息，不记录用户具体浏览内容。

继续注册即表示您已阅读、理解并同意遵守本条款。`
)

func NewConfigStore() *ConfigStore {
	return &ConfigStore{db: database.GetDB()}
}

func (s *ConfigStore) Terms() (title, content, externalURL, version string) {
	title = strings.TrimSpace(s.GetDefault("site.terms_title", defaultTermsTitle))
	content = strings.TrimSpace(s.GetDefault("site.terms_content", defaultTermsText))
	externalURL = strings.TrimSpace(s.GetDefault("site.terms_url", ""))
	sum := sha256.Sum256([]byte(title + "\x00" + content + "\x00" + externalURL))
	version = base64.RawURLEncoding.EncodeToString(sum[:12])
	return
}

func (s *ConfigStore) Get(key string) (string, error) {
	var row model.CommercialSetting
	if err := s.db.Where("key = ?", key).First(&row).Error; err != nil {
		return "", err
	}
	if !row.Encrypted {
		return row.Value, nil
	}
	return s.decrypt(row.Value)
}

func (s *ConfigStore) GetDefault(key, fallback string) string {
	value, err := s.Get(key)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *ConfigStore) Set(key, value string, encrypted bool) error {
	stored := value
	var err error
	if encrypted && value != "" {
		stored, err = s.encrypt(value)
		if err != nil {
			return err
		}
	}
	row := model.CommercialSetting{Key: key, Value: stored, Encrypted: encrypted}
	return s.db.Save(&row).Error
}

func (s *ConfigStore) SetMany(values map[string]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			row := model.CommercialSetting{Key: key, Value: value, Encrypted: false}
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SetManyProtected stores a related settings group atomically. Empty protected
// values are skipped so an administrator can update non-secret fields without
// having to paste the existing credential again.
func (s *ConfigStore) SetManyProtected(values map[string]string, protectedKeys map[string]bool) error {
	prepared := make(map[string]model.CommercialSetting, len(values))
	for key, value := range values {
		encrypted := protectedKeys[key]
		if encrypted && strings.TrimSpace(value) == "" {
			continue
		}
		stored := value
		if encrypted {
			var err error
			stored, err = s.encrypt(value)
			if err != nil {
				return err
			}
		}
		prepared[key] = model.CommercialSetting{Key: key, Value: stored, Encrypted: encrypted}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range prepared {
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ConfigStore) Public() map[string]string {
	policy := s.SecurityPolicy()
	subscriptionPolicy := s.SubscriptionPolicy()
	invitationPolicy := s.InvitationPolicy()
	telegramGroupLink, _ := s.settingService.GetTgGroupLink()
	termsTitle, termsContent, termsURL, termsVersion := s.Terms()
	turnstileSiteKey := ""
	if policy.RegistrationCaptchaEnabled {
		turnstileSiteKey = s.GetDefault("turnstile.site_key", "")
	}
	return map[string]string{
		"siteName":                    s.GetDefault("site.name", "NOVA"),
		"siteTagline":                 s.GetDefault("site.tagline", "稳定连接，清晰可控"),
		"siteUrl":                     s.GetDefault("site.url", ""),
		"forceHttps":                  s.GetDefault("site.force_https", "false"),
		"logoUrl":                     s.GetDefault("site.logo_url", ""),
		"supportUrl":                  s.GetDefault("site.support_url", ""),
		"termsUrl":                    termsURL,
		"termsTitle":                  termsTitle,
		"termsContent":                termsContent,
		"termsVersion":                termsVersion,
		"privacyUrl":                  s.GetDefault("site.privacy_url", ""),
		"telegramGroupLink":           strings.TrimSpace(telegramGroupLink),
		"registrationClosed":          s.GetDefault("registration.closed", "false"),
		"currency":                    s.GetDefault("currency.code", "CNY"),
		"currencySymbol":              s.GetDefault("currency.symbol", "¥"),
		"emailVerification":           strconv.FormatBool(policy.EmailVerification),
		"emailSuffixWhitelist":        strconv.FormatBool(policy.EmailSuffixWhitelistEnabled),
		"allowedEmailSuffixes":        strings.Join(policy.AllowedEmailSuffixes, ","),
		"turnstileSiteKey":            turnstileSiteKey,
		"allowUserSubscriptionChange": strconv.FormatBool(subscriptionPolicy.AllowUserChange),
		"forcedInvitation":            strconv.FormatBool(invitationPolicy.ForcedInvitation),
	}
}

func (s *ConfigStore) key() ([]byte, error) {
	secret, err := s.settingService.GetSecret()
	if err != nil || len(secret) == 0 {
		return nil, errors.New("panel secret is unavailable")
	}
	sum := sha256.Sum256(append([]byte("commercial-settings-v1:"), secret...))
	return sum[:], nil
}

func (s *ConfigStore) encrypt(value string) (string, error) {
	key, err := s.key()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), []byte("commercial-settings-v1"))
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s *ConfigStore) decrypt(value string) (string, error) {
	key, err := s.key()
	if err != nil {
		return "", err
	}
	data, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("encrypted setting is truncated")
	}
	nonce := data[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, data[gcm.NonceSize():], []byte("commercial-settings-v1"))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
