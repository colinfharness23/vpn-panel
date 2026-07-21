package commercial

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	utilcrypto "github.com/mhsanaei/3x-ui/v3/internal/util/crypto"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var verificationCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

type VerificationMailer interface {
	SendTo(recipients []string, subject, body string) error
}

type AuthService struct {
	db     *gorm.DB
	config *ConfigStore
	mailer VerificationMailer
	secret []byte
}

type SessionIdentity struct {
	Customer model.Customer
	Session  model.CustomerSession
}

func NewAuthService(mailer VerificationMailer) *AuthService {
	settings := service.SettingService{}
	secret, err := settings.GetSecret()
	if err != nil || len(secret) == 0 {
		// Never fall back to a public constant: verification-code and audit
		// digests must remain unpredictable even before panel settings exist.
		secret = make([]byte, 32)
		if _, randomErr := io.ReadFull(rand.Reader, secret); randomErr != nil {
			panic("commercial auth secret is unavailable: " + randomErr.Error())
		}
	}
	return &AuthService{db: database.GetDB(), config: NewConfigStore(), mailer: mailer, secret: secret}
}

func NormalizeGmail(raw string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("请输入有效的 Gmail 邮箱")
	}
	parts := strings.Split(strings.ToLower(address.Address), "@")
	if len(parts) != 2 || parts[1] != "gmail.com" {
		return "", errors.New("仅支持 @gmail.com 邮箱")
	}
	local := parts[0]
	if plus := strings.IndexByte(local, '+'); plus >= 0 {
		local = local[:plus]
	}
	local = strings.ReplaceAll(local, ".", "")
	if local == "" || len(local) > 64 {
		return "", errors.New("请输入有效的 Gmail 邮箱")
	}
	return local + "@gmail.com", nil
}

func normalizeConfiguredEmail(raw string, policy SecurityPolicy) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("请输入有效的邮箱地址")
	}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(address.Address)), "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("请输入有效的邮箱地址")
	}
	local, domain := parts[0], parts[1]
	if domain == "gmail.com" && policy.DisallowGmailAliases && (strings.Contains(local, "+") || strings.Contains(local, ".")) {
		return "", errors.New("不允许使用 Gmail 多别名注册")
	}
	if policy.EmailSuffixWhitelistEnabled {
		allowed := false
		for _, suffix := range policy.AllowedEmailSuffixes {
			if domain == suffix {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", errors.New("该邮箱后缀不在允许注册的白名单中")
		}
	}
	return local + "@" + domain, nil
}

func (s *AuthService) normalizeEmail(raw string) (string, error) {
	return normalizeConfiguredEmail(raw, s.config.SecurityPolicy())
}

func ValidatePassword(password string) error {
	if len(password) < 10 || len(password) > 128 {
		return errors.New("密码长度需为 10 到 128 个字符")
	}
	var lower, upper, digit bool
	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			lower = true
		case char >= 'A' && char <= 'Z':
			upper = true
		case char >= '0' && char <= '9':
			digit = true
		}
	}
	if !lower || !upper || !digit {
		return errors.New("密码必须包含大写字母、小写字母和数字")
	}
	return nil
}

func (s *AuthService) SendCode(ctx context.Context, rawEmail, purpose, ip, turnstileToken string) (string, error) {
	policy := s.config.SecurityPolicy()
	email, err := normalizeConfiguredEmail(rawEmail, policy)
	if err != nil {
		return "", err
	}
	if purpose != "register" && purpose != "reset" {
		return "", errors.New("不支持的验证码用途")
	}
	if purpose == "register" && strings.EqualFold(s.config.GetDefault("registration.closed", "false"), "true") {
		return "", errors.New("本站当前已停止新用户注册")
	}
	if purpose == "register" && !policy.EmailVerification {
		return "", errors.New("当前注册流程未启用邮箱验证码")
	}
	if err := s.verifyTurnstile(ctx, turnstileToken, ip); err != nil {
		return "", err
	}
	if err := s.checkCodeRate(email, ip); err != nil {
		return "", err
	}
	if purpose == "register" {
		var count int64
		if err := s.db.Model(&model.Customer{}).Where("email = ?", email).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			return "", errors.New("该邮箱已注册")
		}
	}
	code, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	record := model.EmailVerification{ID: uuid.NewString(), Email: email, Purpose: purpose, CodeHash: s.codeHash(email, purpose, code), IPHash: s.auditHash(ip), ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.EmailVerification{}).Where("email = ? AND purpose = ? AND used_at IS NULL", email, purpose).Update("used_at", now).Error; err != nil {
			return err
		}
		return tx.Create(&record).Error
	}); err != nil {
		return "", err
	}
	if s.mailer != nil {
		siteName := s.config.GetDefault("site.name", "NOVA")
		subject := fmt.Sprintf("[%s] 邮箱验证码", siteName)
		body := fmt.Sprintf(`<html><body style="font-family:Arial,sans-serif;color:#17233d"><h2>%s</h2><p>你的验证码是：</p><p style="font-size:32px;font-weight:700;letter-spacing:8px">%s</p><p>验证码 10 分钟内有效，请勿转发给他人。</p></body></html>`, html.EscapeString(siteName), code)
		if err := s.mailer.SendTo([]string{email}, subject, body); err != nil {
			if !isDemoMode() {
				return "", err
			}
		}
	} else if !isDemoMode() {
		return "", errors.New("SMTP 尚未配置")
	}
	if isDemoMode() {
		return code, nil
	}
	return "", nil
}

func (s *AuthService) Register(ctx context.Context, rawEmail, password, code, inviteCode, locale, ip, turnstileToken string, acceptedTerms bool, termsVersion string) (*model.Customer, error) {
	if strings.EqualFold(s.config.GetDefault("registration.closed", "false"), "true") {
		return nil, errors.New("本站当前已停止新用户注册")
	}
	_, _, _, currentTermsVersion := s.config.Terms()
	if !acceptedTerms {
		return nil, errors.New("请先阅读并同意使用条款")
	}
	if strings.TrimSpace(termsVersion) == "" || termsVersion != currentTermsVersion {
		return nil, errors.New("使用条款已更新，请重新阅读并确认")
	}
	policy := s.config.SecurityPolicy()
	invitationPolicy := s.config.InvitationPolicy()
	inviteCode = strings.ToUpper(strings.TrimSpace(inviteCode))
	if invitationPolicy.ForcedInvitation && inviteCode == "" {
		return nil, errors.New("当前站点仅允许通过有效邀请码注册")
	}
	if policy.RegistrationCaptchaEnabled && !policy.EmailVerification {
		if err := s.verifyTurnstile(ctx, turnstileToken, ip); err != nil {
			return nil, err
		}
	}
	email, err := normalizeConfiguredEmail(rawEmail, policy)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt(password)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(locale) == "" {
		locale = "zh-CN"
	}
	now := time.Now().UTC()
	ipHash := s.auditHash(ip)
	if policy.IPRegistrationLimitEnabled {
		var registrationCount int64
		if err := s.db.Model(&model.Customer{}).Where("registration_ip_hash = ? AND created_at > ?", ipHash, now.Add(-24*time.Hour)).Count(&registrationCount).Error; err != nil {
			return nil, err
		}
		if registrationCount >= defaultIPRegistrationLimit {
			return nil, errors.New("当前网络今日注册账号数量已达到上限")
		}
	}
	customer := &model.Customer{ID: uuid.NewString(), Email: email, PasswordHash: hash, DisplayName: strings.Split(email, "@")[0], Locale: locale, Status: "active", InviteCode: strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", "")), RegistrationIPHash: ipHash, TermsAcceptedAt: &now, TermsVersion: currentTermsVersion}
	if policy.EmailVerification {
		customer.EmailVerifiedAt = &now
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if policy.EmailVerification {
			if err := s.consumeCode(tx, email, "register", code, now); err != nil {
				return err
			}
		}
		if inviteCode != "" {
			var inviter model.Customer
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("invite_code = ? AND status = ?", inviteCode, "active").First(&inviter).Error; err != nil {
				return errors.New("邀请码无效")
			}
			if !invitationPolicy.InviteCodesNeverExpire {
				var used int64
				if err := tx.Model(&model.Customer{}).Where("invited_by_id = ?", inviter.ID).Count(&used).Error; err != nil {
					return err
				}
				if used > 0 {
					return errors.New("邀请码已被使用，请联系邀请人获取新邀请码")
				}
			}
			customer.InvitedByID = &inviter.ID
		}
		return tx.Create(customer).Error
	})
	if err != nil {
		return nil, err
	}
	if trialPlanID := strings.TrimSpace(s.config.GetDefault("registration.trial_plan_id", "")); trialPlanID != "" {
		var price model.PlanPrice
		priceErr := s.db.Where("plan_id = ? AND active = ? AND months > 0", trialPlanID, true).Order("months asc, amount_fen asc").First(&price).Error
		if priceErr == nil {
			expiresAt := time.Now().UTC().AddDate(0, price.Months, 0).Format(time.RFC3339)
			_, grantErr := NewAdminService().UpsertSubscription(customer.ID, entity.CommercialSubscriptionUpdateRequest{PlanID: trialPlanID, ExpiresAt: expiresAt})
			if grantErr != nil {
				logger.Warning("commercial registration trial provisioning failed:", grantErr)
			}
		} else {
			logger.Warning("commercial registration trial price unavailable:", priceErr)
		}
	}
	return customer, nil
}

func (s *AuthService) Login(rawEmail, password string) (*model.Customer, error) {
	var customer model.Customer
	rawLogin := strings.TrimSpace(rawEmail)
	email, emailErr := s.normalizeEmail(rawLogin)
	lookupErr := gorm.ErrRecordNotFound
	if emailErr == nil {
		lookupErr = s.db.Where("email = ?", email).First(&customer).Error
	}
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	passwordMatches := false
	if lookupErr != nil {
		var admin model.User
		if err := s.db.Where("username = ?", rawLogin).First(&admin).Error; err != nil {
			return nil, errors.New("邮箱、管理员账号或密码错误")
		}
		if err := s.db.Where("admin_user_id = ?", admin.Id).First(&customer).Error; err != nil {
			return nil, errors.New("管理员前台账户尚未初始化，请重启面板后重试")
		}
		passwordMatches = utilcrypto.CheckPasswordHash(admin.Password, password)
	} else if customer.AdminUserID != nil {
		var admin model.User
		if err := s.db.Where("id = ?", *customer.AdminUserID).First(&admin).Error; err != nil {
			return nil, errors.New("邮箱、管理员账号或密码错误")
		}
		passwordMatches = utilcrypto.CheckPasswordHash(admin.Password, password)
	} else {
		passwordMatches = utilcrypto.CheckPasswordHash(customer.PasswordHash, password)
	}
	policy := s.config.SecurityPolicy()
	if policy.PasswordAttemptLimitEnabled && customer.LoginLockedUntil != nil && customer.LoginLockedUntil.After(time.Now().UTC()) {
		return nil, errors.New("密码尝试次数过多，请稍后再试")
	}
	if !passwordMatches {
		if policy.PasswordAttemptLimitEnabled {
			if err := s.registerPasswordFailure(&customer, policy); err != nil {
				return nil, err
			}
		}
		return nil, errors.New("邮箱或密码错误")
	}
	if customer.Status != "active" {
		return nil, errors.New("账户当前不可用，请联系客服")
	}
	now := time.Now().UTC()
	updates := map[string]any{"last_login_at": now, "failed_login_attempts": 0, "login_locked_until": nil}
	if err := s.db.Model(&customer).Updates(updates).Error; err != nil {
		return nil, err
	}
	customer.LastLoginAt = &now
	customer.FailedLoginAttempts = 0
	customer.LoginLockedUntil = nil
	return &customer, nil
}

func (s *AuthService) registerPasswordFailure(customer *model.Customer, policy SecurityPolicy) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var locked model.Customer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", customer.ID).First(&locked).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if locked.LoginLockedUntil != nil && !locked.LoginLockedUntil.After(now) {
			locked.FailedLoginAttempts = 0
			locked.LoginLockedUntil = nil
		}
		attempts := locked.FailedLoginAttempts + 1
		updates := map[string]any{"failed_login_attempts": attempts}
		if attempts >= policy.MaxPasswordAttempts {
			lockedUntil := now.Add(time.Duration(policy.PasswordLockDurationMinutes) * time.Minute)
			updates["login_locked_until"] = lockedUntil
			customer.LoginLockedUntil = &lockedUntil
		}
		customer.FailedLoginAttempts = attempts
		return tx.Model(&locked).Updates(updates).Error
	})
}

func (s *AuthService) ResetPassword(rawEmail, code, password string) error {
	email, err := s.normalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt(password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.consumeCode(tx, email, "reset", code, now); err != nil {
			return err
		}
		var customer model.Customer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&customer).Error; err != nil {
			return errors.New("账户不存在")
		}
		if customer.AdminUserID != nil {
			return errors.New("管理员前台账户请在面板中修改密码")
		}
		result := tx.Model(&customer).Updates(map[string]any{"password_hash": hash, "login_epoch": gorm.Expr("login_epoch + 1")})
		if result.Error != nil {
			return result.Error
		}
		return tx.Model(&model.CustomerSession{}).Where("customer_id = ? AND revoked_at IS NULL", customer.ID).Update("revoked_at", now).Error
	})
}

func (s *AuthService) ChangePassword(customerID, currentPassword, newPassword, ip, userAgent string) (string, *model.CustomerSession, error) {
	if strings.TrimSpace(currentPassword) == "" {
		return "", nil, errors.New("请输入当前密码")
	}
	if currentPassword == newPassword {
		return "", nil, errors.New("新密码不能与当前密码相同")
	}
	if err := ValidatePassword(newPassword); err != nil {
		return "", nil, err
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt(newPassword)
	if err != nil {
		return "", nil, err
	}
	token, nextSession, err := s.newSession(customerID, ip, userAgent)
	if err != nil {
		return "", nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var customer model.Customer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", customerID, "active").First(&customer).Error; err != nil {
			return errors.New("账户当前不可用，请重新登录")
		}
		passwordHash := customer.PasswordHash
		var admin *model.User
		if customer.AdminUserID != nil {
			var row model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *customer.AdminUserID).First(&row).Error; err != nil {
				return errors.New("管理员账户不可用，请在面板中检查账户")
			}
			passwordHash = row.Password
			admin = &row
		}
		if !utilcrypto.CheckPasswordHash(passwordHash, currentPassword) {
			return errors.New("当前密码错误")
		}
		if admin != nil {
			if err := tx.Model(admin).Updates(map[string]any{"password": hash, "login_epoch": gorm.Expr("login_epoch + 1")}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&customer).Updates(map[string]any{
			"password_hash":         hash,
			"login_epoch":           gorm.Expr("login_epoch + 1"),
			"failed_login_attempts": 0,
			"login_locked_until":    nil,
		}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&model.CustomerSession{}).Where("customer_id = ? AND revoked_at IS NULL", customer.ID).Update("revoked_at", now).Error; err != nil {
			return err
		}
		return tx.Create(nextSession).Error
	})
	if err != nil {
		return "", nil, err
	}
	return token, nextSession, nil
}

func (s *AuthService) CreateSession(customerID, ip, userAgent string) (string, *model.CustomerSession, error) {
	token, row, err := s.newSession(customerID, ip, userAgent)
	if err != nil {
		return "", nil, err
	}
	if err := s.db.Create(row).Error; err != nil {
		return "", nil, err
	}
	return token, row, nil
}

func (s *AuthService) newSession(customerID, ip, userAgent string) (string, *model.CustomerSession, error) {
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	now := time.Now().UTC()
	row := &model.CustomerSession{ID: uuid.NewString(), CustomerID: customerID, TokenHash: hashToken(token), IPHash: s.auditHash(ip), UserAgentHash: s.auditHash(userAgent), LastSeenAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now}
	return token, row, nil
}

func (s *AuthService) Authenticate(token string) (*SessionIdentity, error) {
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var session model.CustomerSession
	if err := s.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hashToken(token), time.Now().UTC()).First(&session).Error; err != nil {
		return nil, err
	}
	var customer model.Customer
	if err := s.db.Where("id = ? AND status = ?", session.CustomerID, "active").First(&customer).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if now.Sub(session.LastSeenAt) > 5*time.Minute {
		_ = s.db.Model(&session).Update("last_seen_at", now).Error
		session.LastSeenAt = now
	}
	return &SessionIdentity{Customer: customer, Session: session}, nil
}

func (s *AuthService) Sessions(customerID string) ([]model.CustomerSession, error) {
	var sessions []model.CustomerSession
	err := s.db.Where("customer_id = ? AND revoked_at IS NULL AND expires_at > ?", customerID, time.Now().UTC()).Order("last_seen_at desc").Find(&sessions).Error
	return sessions, err
}

func (s *AuthService) RevokeSession(customerID, sessionID string) error {
	now := time.Now().UTC()
	return s.db.Model(&model.CustomerSession{}).Where("id = ? AND customer_id = ?", sessionID, customerID).Update("revoked_at", now).Error
}

func (s *AuthService) consumeCode(tx *gorm.DB, email, purpose, code string, now time.Time) error {
	if !verificationCodePattern.MatchString(code) {
		return errors.New("验证码无效")
	}
	var record model.EmailVerification
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ? AND purpose = ? AND used_at IS NULL", email, purpose).Order("created_at desc").First(&record).Error; err != nil {
		return errors.New("验证码无效或已使用")
	}
	if record.ExpiresAt.Before(now) {
		return errors.New("验证码已过期")
	}
	if record.Attempts >= 5 {
		return errors.New("验证码尝试次数过多")
	}
	if !hmac.Equal([]byte(record.CodeHash), []byte(s.codeHash(email, purpose, code))) {
		if err := tx.Model(&record).UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error; err != nil {
			return err
		}
		return errors.New("验证码错误")
	}
	return tx.Model(&record).Updates(map[string]any{"used_at": now, "attempts": gorm.Expr("attempts + 1")}).Error
}

func (s *AuthService) checkCodeRate(email, ip string) error {
	now := time.Now().UTC()
	var latest model.EmailVerification
	if err := s.db.Where("email = ?", email).Order("created_at desc").First(&latest).Error; err == nil && now.Sub(latest.CreatedAt) < time.Minute {
		return errors.New("验证码发送过于频繁，请稍后再试")
	}
	var emailCount int64
	if err := s.db.Model(&model.EmailVerification{}).Where("email = ? AND created_at > ?", email, now.Add(-time.Hour)).Count(&emailCount).Error; err != nil {
		return err
	}
	if emailCount >= 5 {
		return errors.New("该邮箱请求次数过多，请一小时后再试")
	}
	var ipCount int64
	if err := s.db.Model(&model.EmailVerification{}).Where("ip_hash = ? AND created_at > ?", s.auditHash(ip), now.Add(-time.Hour)).Count(&ipCount).Error; err != nil {
		return err
	}
	if ipCount >= 20 {
		return errors.New("当前网络请求次数过多，请稍后再试")
	}
	return nil
}

func (s *AuthService) verifyTurnstile(ctx context.Context, token, remoteIP string) error {
	if !s.config.SecurityPolicy().RegistrationCaptchaEnabled {
		return nil
	}
	secret, err := s.config.Get("turnstile.secret")
	if errors.Is(err, gorm.ErrRecordNotFound) || strings.TrimSpace(secret) == "" {
		return errors.New("人机验证尚未配置")
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("请完成人机验证")
	}
	form := url.Values{"secret": {secret}, "response": {token}, "remoteip": {remoteIP}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("人机验证暂时不可用")
	}
	defer resp.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil || !result.Success {
		return errors.New("人机验证失败")
	}
	return nil
}

func (s *AuthService) codeHash(email, purpose, code string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(email))
	mac.Write([]byte{0})
	mac.Write([]byte(purpose))
	mac.Write([]byte{0})
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *AuthService) auditHash(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomDigits(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", err
	}
	for index := range buffer {
		buffer[index] = '0' + buffer[index]%10
	}
	return string(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isDemoMode() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("XUI_COMMERCIAL_DEMO")))
	return value == "1" || value == "true" || value == "yes"
}
