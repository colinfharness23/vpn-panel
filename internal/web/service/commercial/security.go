package commercial

import (
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
)

const (
	defaultMaxPasswordAttempts         = 5
	defaultPasswordLockDurationMinutes = 60
	defaultIPRegistrationLimit         = 3
)

type SecurityPolicy struct {
	EmailVerification           bool
	DisallowGmailAliases        bool
	SafeMode                    bool
	EmailSuffixWhitelistEnabled bool
	AllowedEmailSuffixes        []string
	RegistrationCaptchaEnabled  bool
	IPRegistrationLimitEnabled  bool
	PasswordAttemptLimitEnabled bool
	MaxPasswordAttempts         int
	PasswordLockDurationMinutes int
}

func configBool(store *ConfigStore, key string, fallback bool) bool {
	return strings.EqualFold(store.GetDefault(key, strconv.FormatBool(fallback)), "true")
}

func configInt(store *ConfigStore, key string, fallback int) int {
	value, err := strconv.Atoi(store.GetDefault(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func normalizeEmailSuffixes(raw string) ([]string, string, error) {
	seen := map[string]bool{}
	values := make([]string, 0)
	for _, candidate := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';' || r == ' '
	}) {
		suffix := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(candidate), "@"))
		if suffix == "" || strings.Contains(suffix, "@") || strings.Contains(suffix, "/") || strings.HasPrefix(suffix, ".") || strings.HasSuffix(suffix, ".") {
			return nil, "", errors.New("邮箱后缀格式无效")
		}
		if _, err := mail.ParseAddress("user@" + suffix); err != nil {
			return nil, "", errors.New("邮箱后缀格式无效")
		}
		if !seen[suffix] {
			seen[suffix] = true
			values = append(values, suffix)
		}
	}
	return values, strings.Join(values, "\n"), nil
}

func (s *ConfigStore) SecurityPolicy() SecurityPolicy {
	suffixes, _, _ := normalizeEmailSuffixes(s.GetDefault("security.allowed_email_suffixes", "gmail.com"))
	if len(suffixes) == 0 {
		suffixes = []string{"gmail.com"}
	}
	maxAttempts := configInt(s, "security.max_password_attempts", defaultMaxPasswordAttempts)
	if maxAttempts < 1 || maxAttempts > 20 {
		maxAttempts = defaultMaxPasswordAttempts
	}
	lockMinutes := configInt(s, "security.password_lock_minutes", defaultPasswordLockDurationMinutes)
	if lockMinutes < 1 || lockMinutes > 1440 {
		lockMinutes = defaultPasswordLockDurationMinutes
	}
	return SecurityPolicy{
		EmailVerification:           configBool(s, "security.email_verification", true),
		DisallowGmailAliases:        configBool(s, "security.disallow_gmail_aliases", true),
		SafeMode:                    configBool(s, "security.safe_mode", false),
		EmailSuffixWhitelistEnabled: configBool(s, "security.email_suffix_whitelist", true),
		AllowedEmailSuffixes:        suffixes,
		RegistrationCaptchaEnabled:  configBool(s, "security.registration_captcha", false),
		IPRegistrationLimitEnabled:  configBool(s, "security.ip_registration_limit", false),
		PasswordAttemptLimitEnabled: configBool(s, "security.password_attempt_limit", true),
		MaxPasswordAttempts:         maxAttempts,
		PasswordLockDurationMinutes: lockMinutes,
	}
}

func (s *AdminService) SecuritySettings() entity.CommercialSecuritySettings {
	policy := s.config.SecurityPolicy()
	return entity.CommercialSecuritySettings{
		EmailVerification:           policy.EmailVerification,
		DisallowGmailAliases:        policy.DisallowGmailAliases,
		SafeMode:                    policy.SafeMode,
		EmailSuffixWhitelistEnabled: policy.EmailSuffixWhitelistEnabled,
		AllowedEmailSuffixes:        strings.Join(policy.AllowedEmailSuffixes, "\n"),
		RegistrationCaptchaEnabled:  policy.RegistrationCaptchaEnabled,
		IPRegistrationLimitEnabled:  policy.IPRegistrationLimitEnabled,
		PasswordAttemptLimitEnabled: policy.PasswordAttemptLimitEnabled,
		MaxPasswordAttempts:         policy.MaxPasswordAttempts,
		PasswordLockDurationMinutes: policy.PasswordLockDurationMinutes,
	}
}

func (s *AdminService) SaveSecuritySettings(request entity.CommercialSecuritySettings) error {
	suffixes, normalizedSuffixes, err := normalizeEmailSuffixes(request.AllowedEmailSuffixes)
	if err != nil {
		return err
	}
	if request.EmailSuffixWhitelistEnabled && len(suffixes) == 0 {
		return errors.New("启用邮箱后缀白名单时至少需要填写一个邮箱后缀")
	}
	if request.MaxPasswordAttempts < 1 || request.MaxPasswordAttempts > 20 {
		return errors.New("密码尝试次数必须在 1 到 20 之间")
	}
	if request.PasswordLockDurationMinutes < 1 || request.PasswordLockDurationMinutes > 1440 {
		return errors.New("锁定时长必须在 1 到 1440 分钟之间")
	}
	if request.SafeMode {
		siteURL := strings.TrimSpace(s.config.GetDefault("site.url", ""))
		parsed, parseErr := url.Parse(siteURL)
		if parseErr != nil || parsed.Hostname() == "" {
			return errors.New("启用安全模式前请先配置有效的站点网址")
		}
	}
	if request.RegistrationCaptchaEnabled {
		siteKey := strings.TrimSpace(s.config.GetDefault("turnstile.site_key", ""))
		secret, secretErr := s.config.Get("turnstile.secret")
		if siteKey == "" || secretErr != nil || strings.TrimSpace(secret) == "" {
			return errors.New("启用注册验证码前请先在商业设置中配置 Turnstile Site Key 和 Secret")
		}
	}
	return s.config.SetMany(map[string]string{
		"security.email_verification":     strconv.FormatBool(request.EmailVerification),
		"security.disallow_gmail_aliases": strconv.FormatBool(request.DisallowGmailAliases),
		"security.safe_mode":              strconv.FormatBool(request.SafeMode),
		"security.email_suffix_whitelist": strconv.FormatBool(request.EmailSuffixWhitelistEnabled),
		"security.allowed_email_suffixes": normalizedSuffixes,
		"security.registration_captcha":   strconv.FormatBool(request.RegistrationCaptchaEnabled),
		"security.ip_registration_limit":  strconv.FormatBool(request.IPRegistrationLimitEnabled),
		"security.password_attempt_limit": strconv.FormatBool(request.PasswordAttemptLimitEnabled),
		"security.max_password_attempts":  strconv.Itoa(request.MaxPasswordAttempts),
		"security.password_lock_minutes":  strconv.Itoa(request.PasswordLockDurationMinutes),
	})
}
