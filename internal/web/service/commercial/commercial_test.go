package commercial

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	utilcrypto "github.com/mhsanaei/3x-ui/v3/internal/util/crypto"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
)

func currentTermsVersion(auth *AuthService) string {
	_, _, _, version := auth.config.Terms()
	return version
}

func TestValidateCommercialLogo(t *testing.T) {
	validPNG := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9ZfyoAAAAASUVORK5CYII="
	if err := validateCommercialLogo(validPNG); err != nil {
		t.Fatalf("valid PNG rejected: %v", err)
	}
	validPNGBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(validPNG, "data:image/png;base64,"))
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	previousLimitRegression := append(validPNGBytes, make([]byte, 2*1024*1024-len(validPNGBytes))...)
	if err := validateCommercialLogo("data:image/png;base64," + base64.StdEncoding.EncodeToString(previousLimitRegression)); err != nil {
		t.Fatalf("2 MB PNG rejected after raising the limit: %v", err)
	}
	if err := validateCommercialLogo("https://example.com/logo.png"); err != nil {
		t.Fatalf("legacy HTTP logo URL rejected: %v", err)
	}
	if err := validateCommercialLogo("data:image/svg+xml;base64,PHN2Zz48L3N2Zz4="); err == nil {
		t.Fatal("SVG logo should be rejected")
	}
	oversized := "data:image/png;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(maxCommercialLogoBytes+1))
	if err := validateCommercialLogo(oversized); err == nil {
		t.Fatal("oversized logo should be rejected")
	}
}

func TestNormalizeGmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "canonical", input: "User.Name+offer@GMAIL.com", want: "username@gmail.com"},
		{name: "display name", input: "Member <first.last@gmail.com>", want: "firstlast@gmail.com"},
		{name: "googlemail rejected", input: "member@googlemail.com", wantErr: true},
		{name: "other domain rejected", input: "member@example.com", wantErr: true},
		{name: "invalid rejected", input: "not-an-email", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeGmail(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeGmail(%q) succeeded, want error", test.input)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("NormalizeGmail(%q) = %q, %v; want %q", test.input, got, err, test.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	for _, password := range []string{"short", "alllowercase1", "ALLUPPERCASE1", "MissingNumber"} {
		if ValidatePassword(password) == nil {
			t.Fatalf("ValidatePassword(%q) succeeded, want error", password)
		}
	}
	if err := ValidatePassword("SecurePass2026"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}

func TestChangePasswordRotatesCustomerSessions(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	hash, err := utilcrypto.HashPasswordAsBcrypt("CurrentPass2026")
	if err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "change-password@gmail.com", PasswordHash: hash, Status: "active", InviteCode: "CHANGEPW1"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	auth := NewAuthService(nil)
	currentToken, currentSession, err := auth.CreateSession(customer.ID, "203.0.113.41", "current")
	if err != nil {
		t.Fatal(err)
	}
	otherToken, _, err := auth.CreateSession(customer.ID, "203.0.113.42", "other")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := auth.ChangePassword(customer.ID, "WrongPass2026", "UpdatedPass2026", "203.0.113.41", "current"); err == nil {
		t.Fatal("wrong current password was accepted")
	}
	if _, err := auth.Authenticate(currentToken); err != nil {
		t.Fatalf("failed change revoked current session: %v", err)
	}
	newToken, newSession, err := auth.ChangePassword(customer.ID, "CurrentPass2026", "UpdatedPass2026", "203.0.113.41", "current")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if newSession.ID == currentSession.ID {
		t.Fatal("password change did not rotate the current session")
	}
	if _, err := auth.Authenticate(currentToken); err == nil {
		t.Fatal("old current session remained active")
	}
	if _, err := auth.Authenticate(otherToken); err == nil {
		t.Fatal("other session remained active")
	}
	if _, err := auth.Authenticate(newToken); err != nil {
		t.Fatalf("rotated session is not active: %v", err)
	}
	if _, err := auth.Login(customer.Email, "CurrentPass2026"); err == nil {
		t.Fatal("old password remained valid")
	}
	if _, err := auth.Login(customer.Email, "UpdatedPass2026"); err != nil {
		t.Fatalf("new password login failed: %v", err)
	}
	activeSessions, err := auth.Sessions(customer.ID)
	if err != nil || len(activeSessions) != 1 || activeSessions[0].ID != newSession.ID {
		t.Fatalf("active sessions=%+v err=%v", activeSessions, err)
	}
}

func TestChangePasswordKeepsAdminPortalCredentialsInSync(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	var admin model.User
	if err := db.Order("id asc").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt("AdminCurrent2026")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&admin).Update("password", hash).Error; err != nil {
		t.Fatal(err)
	}
	var customer model.Customer
	if err := db.Where("admin_user_id = ?", admin.Id).First(&customer).Error; err != nil {
		t.Fatal(err)
	}
	auth := NewAuthService(nil)
	oldToken, _, err := auth.CreateSession(customer.ID, "203.0.113.43", "admin")
	if err != nil {
		t.Fatal(err)
	}
	newToken, _, err := auth.ChangePassword(customer.ID, "AdminCurrent2026", "AdminUpdated2026", "203.0.113.43", "admin")
	if err != nil {
		t.Fatalf("change linked admin password: %v", err)
	}
	if _, err := auth.Authenticate(oldToken); err == nil {
		t.Fatal("linked admin old portal session remained active")
	}
	if _, err := auth.Authenticate(newToken); err != nil {
		t.Fatalf("linked admin rotated session failed: %v", err)
	}
	if _, err := auth.Login(admin.Username, "AdminCurrent2026"); err == nil {
		t.Fatal("linked admin old password remained valid")
	}
	if _, err := auth.Login(admin.Username, "AdminUpdated2026"); err != nil {
		t.Fatalf("linked admin new password login failed: %v", err)
	}
	if err := db.First(&admin, admin.Id).Error; err != nil || !utilcrypto.CheckPasswordHash(admin.Password, "AdminUpdated2026") {
		t.Fatalf("panel admin password was not synchronized: err=%v", err)
	}
}

func TestRegistrationCanBeClosedFromSiteSettings(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	auth := NewAuthService(nil)
	if err := auth.config.Set("registration.closed", "true", false); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.SendCode(context.Background(), "closed@gmail.com", "register", "203.0.113.30", ""); err == nil || !strings.Contains(err.Error(), "停止新用户注册") {
		t.Fatalf("registration code was not blocked: %v", err)
	}
	if _, err := auth.Register(context.Background(), "closed@gmail.com", "SecurePass2026", "123456", "", "zh-CN", "203.0.113.30", "", true, currentTermsVersion(auth)); err == nil || !strings.Contains(err.Error(), "停止新用户注册") {
		t.Fatalf("registration was not blocked: %v", err)
	}
}

func TestRegistrationRequiresCurrentTermsAcceptance(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	auth := NewAuthService(nil)
	if err := auth.config.Set("security.email_verification", "false", false); err != nil {
		t.Fatal(err)
	}
	version := currentTermsVersion(auth)
	if _, err := auth.Register(context.Background(), "terms-missing@gmail.com", "SecurePass2026", "", "", "zh-CN", "203.0.113.31", "", false, version); err == nil || !strings.Contains(err.Error(), "使用条款") {
		t.Fatalf("registration without terms acceptance was allowed: %v", err)
	}
	if _, err := auth.Register(context.Background(), "terms-stale@gmail.com", "SecurePass2026", "", "", "zh-CN", "203.0.113.32", "", true, "stale-version"); err == nil || !strings.Contains(err.Error(), "已更新") {
		t.Fatalf("registration with stale terms was allowed: %v", err)
	}
	customer, err := auth.Register(context.Background(), "terms-ok@gmail.com", "SecurePass2026", "", "", "zh-CN", "203.0.113.33", "", true, version)
	if err != nil {
		t.Fatalf("registration with current terms failed: %v", err)
	}
	if customer.TermsAcceptedAt == nil || customer.TermsVersion != version {
		t.Fatalf("terms acceptance was not recorded: %+v", customer)
	}
}

func TestSiteSettingsRoundTripAndValidation(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	var plan model.Plan
	if err := database.GetDB().Where("active = ?", true).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	request := entity.CommercialSiteSettings{
		SiteName:           "Airport",
		SiteDescription:    "Fast and reliable",
		SiteURL:            "https://airport.example",
		ForceHTTPS:         true,
		LogoURL:            "https://airport.example/logo.png",
		SubscriptionURLs:   "https://sub-a.example;https://sub-b.example",
		TermsURL:           "https://airport.example/terms",
		TermsTemplate:      "concise",
		TermsTitle:         "Airport Terms",
		TermsContent:       "Use the service responsibly.",
		RegistrationClosed: true,
		TrialPlanID:        plan.ID,
		Currency:           "cny",
		CurrencySymbol:     "¥",
	}
	if err := admin.SaveSiteSettings(request); err != nil {
		t.Fatalf("SaveSiteSettings: %v", err)
	}
	response, err := admin.SiteSettings()
	if err != nil {
		t.Fatal(err)
	}
	if response.Settings.SiteName != "Airport" || response.Settings.Currency != "CNY" || !response.Settings.ForceHTTPS || !response.Settings.RegistrationClosed || response.Settings.TermsContent != request.TermsContent {
		t.Fatalf("site settings did not round trip: %+v", response.Settings)
	}
	public := admin.config.Public()
	if public["logoUrl"] != request.LogoURL || public["currencySymbol"] != request.CurrencySymbol || public["registrationClosed"] != "true" || public["termsTitle"] != request.TermsTitle || public["termsContent"] != request.TermsContent || public["termsVersion"] == "" {
		t.Fatalf("public site settings not updated: %+v", public)
	}
	request.SiteURL = "ftp://invalid.example"
	if err := admin.SaveSiteSettings(request); err == nil {
		t.Fatal("invalid site URL was accepted")
	}
}

func TestSecuritySettingsRoundTripAndEmailPolicy(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	request := entity.CommercialSecuritySettings{
		EmailVerification:           true,
		DisallowGmailAliases:        true,
		SafeMode:                    false,
		EmailSuffixWhitelistEnabled: true,
		AllowedEmailSuffixes:        "gmail.com\nqq.com",
		RegistrationCaptchaEnabled:  false,
		IPRegistrationLimitEnabled:  true,
		PasswordAttemptLimitEnabled: true,
		MaxPasswordAttempts:         5,
		PasswordLockDurationMinutes: 60,
	}
	if err := admin.SaveSecuritySettings(request); err != nil {
		t.Fatalf("SaveSecuritySettings: %v", err)
	}
	got := admin.SecuritySettings()
	if !got.DisallowGmailAliases || !got.EmailSuffixWhitelistEnabled || got.AllowedEmailSuffixes != "gmail.com\nqq.com" || got.MaxPasswordAttempts != 5 {
		t.Fatalf("security settings did not round trip: %+v", got)
	}
	policy := admin.config.SecurityPolicy()
	if _, err := normalizeConfiguredEmail("first.last+offer@gmail.com", policy); err == nil || !strings.Contains(err.Error(), "Gmail 多别名") {
		t.Fatalf("Gmail alias was not rejected: %v", err)
	}
	if email, err := normalizeConfiguredEmail("member@qq.com", policy); err != nil || email != "member@qq.com" {
		t.Fatalf("whitelisted suffix rejected: email=%q err=%v", email, err)
	}
	created, err := admin.CreateCustomer(entity.CommercialCustomerCreateRequest{Email: "Manual.Member@QQ.com", Password: "ManualMember2026", Status: "active"})
	if err != nil || created.Email != "manual.member@qq.com" {
		t.Fatalf("admin customer did not use configured email whitelist: customer=%+v err=%v", created, err)
	}
	if _, err := admin.CreateCustomer(entity.CommercialCustomerCreateRequest{Email: "blocked@outlook.com", Password: "BlockedMember2026", Status: "active"}); err == nil || !strings.Contains(err.Error(), "白名单") {
		t.Fatalf("admin customer outside configured whitelist was accepted: %v", err)
	}
	request.AllowedEmailSuffixes = "bad/@suffix"
	if err := admin.SaveSecuritySettings(request); err == nil {
		t.Fatal("invalid email suffix was accepted")
	}
}

func TestSubscriptionSettingsRoundTripAndSelfServiceGate(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	request := entity.CommercialSubscriptionSettings{
		AllowUserChange:        false,
		MonthlyResetMode:       MonthlyResetNever,
		OffsetEnabled:          true,
		PurchaseEvent:          SubscriptionEventNone,
		RenewalEvent:           SubscriptionEventResetTraffic,
		ChangeEvent:            SubscriptionEventNone,
		ShowSubscriptionInfo:   true,
		ShowProtocolInNodeName: true,
	}
	if err := admin.SaveSubscriptionSettings(request); err != nil {
		t.Fatalf("SaveSubscriptionSettings: %v", err)
	}
	got := admin.SubscriptionSettings()
	if got != request {
		t.Fatalf("subscription settings did not round trip: got=%+v want=%+v", got, request)
	}
	if admin.config.Public()["allowUserSubscriptionChange"] != "false" {
		t.Fatalf("public self-service flag was not updated: %+v", admin.config.Public())
	}
	if err := admin.SaveSubscriptionSettings(entity.CommercialSubscriptionSettings{MonthlyResetMode: "invalid"}); err == nil {
		t.Fatal("invalid monthly reset mode was accepted")
	}

	db := database.GetDB()
	var price model.PlanPrice
	if err := db.Where("active = ? AND months > ?", true, 0).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	var plan model.Plan
	if err := db.First(&plan, "id = ?", price.PlanID).Error; err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "subscription-policy@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "SUBPOLICY1"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().UTC().AddDate(0, 1, 0)
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: customer.ID, PlanID: plan.ID, OrderID: uuid.NewString(), InternalClientID: uuid.NewString(), SubscriptionID: uuid.NewString(), Status: "active", StartsAt: time.Now().UTC(), ExpiresAt: &expiresAt}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewOrderService().CreateFor(customer.ID, price.ID, "renewal", entitlement.ID, "", false); err == nil || !strings.Contains(err.Error(), "未开放用户自助") {
		t.Fatalf("self-service gate error=%v", err)
	}
}

func TestInvitationSettingsRoundTripAndForcedRegistration(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	admin := NewAdminService()
	request := entity.CommercialInvitationSettings{
		ForcedInvitation:           true,
		CommissionPercent:          10,
		MaxInviteCodes:             5,
		InviteCodesNeverExpire:     false,
		CommissionFirstPaymentOnly: true,
		CommissionAutoConfirm:      true,
		MultiLevelEnabled:          false,
	}
	if err := admin.SaveInvitationSettings(request); err != nil {
		t.Fatalf("SaveInvitationSettings: %v", err)
	}
	got := admin.InvitationSettings()
	if got.ForcedInvitation != request.ForcedInvitation || got.CommissionPercent != 10 || got.MaxInviteCodes != 5 || got.InviteCodesNeverExpire {
		t.Fatalf("invitation settings did not round trip: %+v", got)
	}
	if admin.config.Public()["forcedInvitation"] != "true" {
		t.Fatalf("public forced-invitation flag was not updated: %+v", admin.config.Public())
	}
	invalid := request
	invalid.MultiLevelEnabled = true
	invalid.CommissionPercent = 34
	if err := admin.SaveInvitationSettings(invalid); err == nil {
		t.Fatal("invalid three-level commission total was accepted")
	}

	db := database.GetDB()
	inviter := model.Customer{ID: uuid.NewString(), Email: "inviter@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "INVITE01"}
	if err := db.Create(&inviter).Error; err != nil {
		t.Fatal(err)
	}
	if err := admin.config.Set("security.email_verification", "false", false); err != nil {
		t.Fatal(err)
	}
	auth := NewAuthService(nil)
	if _, err := auth.Register(context.Background(), "blocked@gmail.com", "SecurePass2026", "", "", "zh-CN", "203.0.113.40", "", true, currentTermsVersion(auth)); err == nil || !strings.Contains(err.Error(), "有效邀请码") {
		t.Fatalf("forced invitation did not block direct registration: %v", err)
	}
	if _, err := auth.Register(context.Background(), "invited@gmail.com", "SecurePass2026", "", inviter.InviteCode, "zh-CN", "203.0.113.41", "", true, currentTermsVersion(auth)); err != nil {
		t.Fatalf("valid invitation registration failed: %v", err)
	}
	if _, err := auth.Register(context.Background(), "reused@gmail.com", "SecurePass2026", "", inviter.InviteCode, "zh-CN", "203.0.113.42", "", true, currentTermsVersion(auth)); err == nil || !strings.Contains(err.Error(), "已被使用") {
		t.Fatalf("single-use invitation code was reused: %v", err)
	}
}

func TestPortalInvitationOverviewAndPublicTelegramLink(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	inviter := model.Customer{ID: uuid.NewString(), Email: "portal-inviter@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "PORTALINVITE"}
	if err := db.Create(&inviter).Error; err != nil {
		t.Fatal(err)
	}
	invitee := model.Customer{ID: uuid.NewString(), Email: "portal-invitee@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "PORTALJOINED", InvitedByID: &inviter.ID}
	if err := db.Create(&invitee).Error; err != nil {
		t.Fatal(err)
	}
	for index, row := range []struct {
		status string
		amount int64
	}{{"pending", 125}, {"confirmed", 250}, {"settled", 500}} {
		commission := model.InvitationCommission{ID: uuid.NewString(), InviterID: inviter.ID, InviteeID: invitee.ID, OrderID: uuid.NewString(), AmountFen: row.amount, Status: row.status, CreatedAt: time.Now().UTC().Add(time.Duration(index) * time.Second)}
		if err := db.Create(&commission).Error; err != nil {
			t.Fatal(err)
		}
	}

	portal := NewPortalService()
	got, err := portal.invitationOverview(&inviter)
	if err != nil {
		t.Fatal(err)
	}
	if got.InviteCode != inviter.InviteCode || got.DirectInviteCount != 1 || got.PendingFen != 125 || got.ConfirmedFen != 250 || got.SettledFen != 500 {
		t.Fatalf("unexpected portal invitation overview: %+v", got)
	}

	if err := portal.config.settingService.SetTgGroupLink("https://t.me/airport_support"); err != nil {
		t.Fatal(err)
	}
	public := portal.config.Public()
	if public["telegramGroupLink"] != "https://t.me/airport_support" {
		t.Fatalf("public Telegram group link missing: %+v", public)
	}
	for _, sensitive := range []string{"tgBotToken", "tgBotChatId", "tgWebhookURL"} {
		if _, exists := public[sensitive]; exists {
			t.Fatalf("sensitive Telegram setting %q leaked into public config", sensitive)
		}
	}
}

func TestInvitationCommissionPolicyDistributionFirstPaymentAndAutoConfirm(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	db := database.GetDB()
	admin := NewAdminService()
	settings := entity.CommercialInvitationSettings{
		CommissionPercent:          10,
		MaxInviteCodes:             5,
		InviteCodesNeverExpire:     true,
		CommissionFirstPaymentOnly: false,
		CommissionAutoConfirm:      false,
		MultiLevelEnabled:          true,
	}
	if err := admin.SaveInvitationSettings(settings); err != nil {
		t.Fatal(err)
	}
	level3 := model.Customer{ID: uuid.NewString(), Email: "level3@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "LEVEL003"}
	level2 := model.Customer{ID: uuid.NewString(), Email: "level2@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "LEVEL002", InvitedByID: &level3.ID}
	level1 := model.Customer{ID: uuid.NewString(), Email: "level1@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "LEVEL001", InvitedByID: &level2.ID}
	buyer := model.Customer{ID: uuid.NewString(), Email: "commission-buyer@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "BUYCOMM1", InvitedByID: &level1.ID}
	for _, customer := range []*model.Customer{&level3, &level2, &level1, &buyer} {
		if err := db.Create(customer).Error; err != nil {
			t.Fatal(err)
		}
	}
	var price model.PlanPrice
	if err := db.Where("active = ?", true).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	orders := NewOrderService()
	order, err := orders.Create(buyer.ID, price.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: "commission-trade-1", AmountFen: order.PayableFen, TradeStatus: "TRADE_SUCCESS"}
	if err := orders.markPaid(context.Background(), "alipay", payment); err != nil {
		t.Fatal(err)
	}
	var commission model.InvitationCommission
	if err := db.Where("order_id = ?", order.ID).First(&commission).Error; err != nil {
		t.Fatal(err)
	}
	shareAmount := max(0, order.OriginalFen-order.DiscountFen) * 10 / 100
	if commission.AmountFen != shareAmount*3 || len(commissionShares(&commission)) != 3 {
		t.Fatalf("unexpected three-level distribution: commission=%+v shares=%+v", commission, commissionShares(&commission))
	}
	if err := admin.SettleCommission(commission.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{level1.ID, level2.ID, level3.ID} {
		var customer model.Customer
		if err := db.First(&customer, "id = ?", id).Error; err != nil || customer.BalanceFen != shareAmount {
			t.Fatalf("commission balance for %s = %d, err=%v", id, customer.BalanceFen, err)
		}
	}

	settings.MultiLevelEnabled = false
	settings.CommissionFirstPaymentOnly = true
	settings.CommissionAutoConfirm = true
	if err := admin.SaveInvitationSettings(settings); err != nil {
		t.Fatal(err)
	}
	second, err := orders.Create(buyer.ID, price.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.markPaid(context.Background(), "alipay", &PaymentNotification{OutTradeNo: second.OutTradeNo, ProviderTrade: "commission-trade-2", AmountFen: second.PayableFen, TradeStatus: "TRADE_SUCCESS"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.InvitationCommission{}).Where("invitee_id = ?", buyer.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("first-payment-only commission count=%d err=%v", count, err)
	}

	autoBuyer := model.Customer{ID: uuid.NewString(), Email: "auto-commission@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "AUTOCOM1", InvitedByID: &level1.ID}
	if err := db.Create(&autoBuyer).Error; err != nil {
		t.Fatal(err)
	}
	autoOrder, err := orders.Create(autoBuyer.ID, price.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := orders.markPaid(context.Background(), "alipay", &PaymentNotification{OutTradeNo: autoOrder.OutTradeNo, ProviderTrade: "commission-trade-3", AmountFen: autoOrder.PayableFen, TradeStatus: "TRADE_SUCCESS"}); err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now().UTC().Add(-4 * 24 * time.Hour)
	if err := db.Model(&model.Order{}).Where("id = ?", autoOrder.ID).Updates(map[string]any{"status": OrderCompleted, "completed_at": completedAt}).Error; err != nil {
		t.Fatal(err)
	}
	before := level1.BalanceFen
	if err := db.First(&level1, "id = ?", level1.ID).Error; err != nil {
		t.Fatal(err)
	}
	before = level1.BalanceFen
	// Legacy installations may still contain the old withdrawal flag. It must
	// never prevent commission credit now that rewards are account credit only.
	if err := admin.config.Set("invitation.withdrawal_closed", "false", false); err != nil {
		t.Fatal(err)
	}
	if err := NewWorker().confirmMatureCommissions(); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&level1, "id = ?", level1.ID).Error; err != nil || level1.BalanceFen != before+max(0, autoOrder.OriginalFen-autoOrder.DiscountFen)*10/100 {
		t.Fatalf("auto-confirmed balance=%d before=%d err=%v", level1.BalanceFen, before, err)
	}
}

func TestPasswordAttemptLimitLocksCustomer(t *testing.T) {
	initCommercialTestDB(t)
	auth := NewAuthService(nil)
	if err := auth.config.SetMany(map[string]string{
		"security.password_attempt_limit": "true",
		"security.max_password_attempts":  "2",
		"security.password_lock_minutes":  "60",
	}); err != nil {
		t.Fatal(err)
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt("SecurePass2026")
	if err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "locked@gmail.com", PasswordHash: hash, Status: "active", InviteCode: "LOCKED01"}
	if err := database.GetDB().Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := auth.Login(customer.Email, "WrongPass2026"); err == nil {
			t.Fatalf("wrong password attempt %d succeeded", attempt+1)
		}
	}
	if _, err := auth.Login(customer.Email, "SecurePass2026"); err == nil || !strings.Contains(err.Error(), "尝试次数过多") {
		t.Fatalf("locked account accepted correct password: %v", err)
	}
}

func TestPaymentAmounts(t *testing.T) {
	tests := map[string]int64{"0": 0, "1": 100, "1.2": 120, "1.23": 123, "999999.99": 99999999}
	for raw, want := range tests {
		got, err := parseAmountFen(raw)
		if err != nil || got != want {
			t.Fatalf("parseAmountFen(%q) = %d, %v; want %d", raw, got, err, want)
		}
		if formatted := formatAmountFen(got); formatted != formatAmountFen(want) {
			t.Fatalf("formatAmountFen(%d) = %q", got, formatted)
		}
	}
	for _, raw := range []string{"", "1.234", "-1", "1.x", "1.2.3"} {
		if _, err := parseAmountFen(raw); err == nil {
			t.Fatalf("parseAmountFen(%q) succeeded, want error", raw)
		}
	}
}

func TestAlipayNotificationSignatureAndTampering(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	provider := &AlipayProvider{appID: "app-1", sellerID: "seller-1", publicKey: &key.PublicKey}
	values := map[string][]string{
		"app_id":       {"app-1"},
		"seller_id":    {"seller-1"},
		"out_trade_no": {"NV202607180001"},
		"trade_no":     {"2026071800001"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"20.00"},
		"notify_id":    {"notify-1"},
		"sign_type":    {"RSA2"},
	}
	digest := sha256.Sum256([]byte(canonicalPaymentValues(values, true)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	values["sign"] = []string{base64.StdEncoding.EncodeToString(signature)}
	result, err := provider.VerifyNotification(values)
	if err != nil || result.AmountFen != 2000 {
		t.Fatalf("valid notification rejected: result=%+v err=%v", result, err)
	}
	values["total_amount"] = []string{"0.01"}
	if _, err := provider.VerifyNotification(values); err == nil || !strings.Contains(err.Error(), "签名") {
		t.Fatalf("tampered notification was not rejected by signature: %v", err)
	}
}

func TestVerificationCodeReplayExpiryAndRateLimit(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	auth := NewAuthService(nil)
	code, err := auth.SendCode(context.Background(), "firstlast@gmail.com", "register", "203.0.113.10", "")
	if err != nil {
		t.Fatalf("SendCode: %v", err)
	}
	if _, err := auth.Register(context.Background(), "firstlast@gmail.com", "SecurePass2026", code, "", "zh-CN", "203.0.113.10", "", true, currentTermsVersion(auth)); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := auth.Register(context.Background(), "firstlast@gmail.com", "SecurePass2026", code, "", "zh-CN", "203.0.113.10", "", true, currentTermsVersion(auth)); err == nil {
		t.Fatal("verification code replay succeeded")
	}
	if _, err := auth.SendCode(context.Background(), "ratelimit@gmail.com", "register", "203.0.113.11", ""); err != nil {
		t.Fatalf("first rate-limit SendCode: %v", err)
	}
	if _, err := auth.SendCode(context.Background(), "ratelimit@gmail.com", "register", "203.0.113.11", ""); err == nil {
		t.Fatal("second immediate verification request was not rate limited")
	}
	expiredCode, err := auth.SendCode(context.Background(), "expired@gmail.com", "register", "203.0.113.12", "")
	if err != nil {
		t.Fatalf("expired SendCode: %v", err)
	}
	if err := database.GetDB().Model(&model.EmailVerification{}).Where("email = ? AND purpose = ?", "expired@gmail.com", "register").Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Register(context.Background(), "expired@gmail.com", "SecurePass2026", expiredCode, "", "zh-CN", "203.0.113.12", "", true, currentTermsVersion(auth)); err == nil {
		t.Fatal("expired verification code succeeded")
	}
}

func TestPaidOrderIsProvisionedOnlyOnce(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	db := database.GetDB()
	customer := model.Customer{ID: uuid.NewString(), Email: "buyer@gmail.com", PasswordHash: "unused", DisplayName: "buyer", Locale: "zh-CN", Status: "active", InviteCode: "BUYER001"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	var price model.PlanPrice
	if err := db.Where("active = ?", true).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	orders := NewOrderService()
	order, err := orders.Create(customer.ID, price.ID, "", false)
	if err != nil {
		t.Fatalf("Create order: %v", err)
	}
	payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: "trade-1", AmountFen: order.PayableFen, TradeStatus: "TRADE_SUCCESS", RawPayload: "signed"}
	if err := orders.markPaid(context.Background(), "alipay", payment); err != nil {
		t.Fatalf("first markPaid: %v", err)
	}
	if err := orders.markPaid(context.Background(), "alipay", payment); err != nil {
		t.Fatalf("duplicate markPaid: %v", err)
	}
	for table, target := range map[string]any{"commercial_payment_transactions": &model.PaymentTransaction{}, "commercial_provisioning_jobs": &model.ProvisioningJob{}} {
		var count int64
		if err := db.Model(target).Where("order_id = ?", order.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s rows = %d, want 1", table, count)
		}
	}
	var eventCount int64
	if err := db.Model(&model.OutboxEvent{}).Where("aggregate_id = ? AND event_type = ?", order.ID, "order.paid").Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("commercial_outbox_events rows = %d, want 1", eventCount)
	}
	second, err := orders.Create(customer.ID, price.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	bad := &PaymentNotification{OutTradeNo: second.OutTradeNo, ProviderTrade: "trade-2", AmountFen: second.PayableFen - 1, TradeStatus: "TRADE_SUCCESS"}
	if err := orders.markPaid(context.Background(), "alipay", bad); err == nil {
		t.Fatal("amount-tampered payment succeeded")
	}
}

func TestOrderGuardsVisibilityCapacityAndCountsCouponAfterPayment(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	db := database.GetDB()
	customer := model.Customer{ID: uuid.NewString(), Email: "guarded@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "GUARD001"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	var price model.PlanPrice
	if err := db.Where("active = ?", true).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	var plan model.Plan
	if err := db.Where("id = ?", price.PlanID).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	coupon := model.Coupon{ID: uuid.NewString(), Code: "PAYONLY", Kind: "fixed", Value: 100, Active: true, MaxRedemptions: 1}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}
	orders := NewOrderService()
	order, err := orders.Create(customer.ID, price.ID, coupon.Code, false)
	if err != nil {
		t.Fatalf("Create with coupon: %v", err)
	}
	if err := db.First(&coupon, "id = ?", coupon.ID).Error; err != nil {
		t.Fatal(err)
	}
	if coupon.RedeemedCount != 0 {
		t.Fatalf("pending order consumed coupon: redeemed_count=%d", coupon.RedeemedCount)
	}
	var reservation model.CouponRedemption
	if err := db.Where("order_id = ?", order.ID).First(&reservation).Error; err != nil {
		t.Fatalf("coupon reservation missing: %v", err)
	}
	if reservation.Status != "reserved" {
		t.Fatalf("coupon reservation status=%q, want reserved", reservation.Status)
	}
	if _, err := orders.Create(customer.ID, price.ID, coupon.Code, false); err == nil || !strings.Contains(err.Error(), "不可用") {
		t.Fatalf("second pending order bypassed coupon limit: %v", err)
	}
	payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: "coupon-trade", AmountFen: order.PayableFen, TradeStatus: "TRADE_SUCCESS"}
	if err := orders.markPaid(context.Background(), "alipay", payment); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&coupon, "id = ?", coupon.ID).Error; err != nil {
		t.Fatal(err)
	}
	if coupon.RedeemedCount != 1 {
		t.Fatalf("paid order coupon count=%d, want 1", coupon.RedeemedCount)
	}
	if err := db.First(&reservation, "id = ?", reservation.ID).Error; err != nil || reservation.Status != "consumed" {
		t.Fatalf("consumed coupon reservation=%+v, err=%v", reservation, err)
	}
	if err := db.Model(&plan).Update("visibility", "hidden").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := orders.Create(customer.ID, price.ID, "", false); err == nil || !strings.Contains(err.Error(), "不能直接购买") {
		t.Fatalf("hidden plan purchase error=%v", err)
	}
	if err := db.Model(&plan).Updates(map[string]any{"visibility": "public", "capacity": 1}).Error; err != nil {
		t.Fatal(err)
	}
	capacityCustomer := model.Customer{ID: uuid.NewString(), Email: "capacity@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "CAPACITY1"}
	if err := db.Create(&capacityCustomer).Error; err != nil {
		t.Fatal(err)
	}
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: capacityCustomer.ID, PlanID: plan.ID, OrderID: uuid.NewString(), InternalClientID: uuid.NewString(), SubscriptionID: uuid.NewString(), Status: "active", TrafficQuota: plan.TrafficBytes, StartsAt: time.Now().UTC()}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := orders.Create(customer.ID, price.ID, "", false); err == nil || !strings.Contains(err.Error(), "名额已满") {
		t.Fatalf("full plan purchase error=%v", err)
	}
}

func TestRenewalAndUpgradeOrderPolicies(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	db := database.GetDB()
	var plans []model.Plan
	if err := db.Where("active = ?", true).Order("sort_order asc").Find(&plans).Error; err != nil || len(plans) < 2 {
		t.Fatalf("load plans: count=%d err=%v", len(plans), err)
	}
	currentPlan, higherPlan := plans[0], plans[1]
	var currentPrice, higherPrice model.PlanPrice
	if err := db.Where("plan_id = ? AND active = ?", currentPlan.ID, true).Order("months asc").First(&currentPrice).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("plan_id = ? AND active = ?", higherPlan.ID, true).Order("months asc").First(&higherPrice).Error; err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "renew-upgrade@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "RENEWUP1", BalanceFen: 250}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().UTC().AddDate(0, 0, 10).Truncate(time.Second)
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: customer.ID, PlanID: currentPlan.ID, OrderID: uuid.NewString(), InternalClientID: uuid.NewString(), SubscriptionID: uuid.NewString(), Status: "active", TrafficQuota: currentPlan.TrafficBytes, DeviceLimit: currentPlan.DeviceLimit, NodeGroup: currentPlan.NodeGroup, StartsAt: time.Now().UTC(), ExpiresAt: &expiresAt}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}

	orders := NewOrderService()
	if _, err := orders.Create(customer.ID, currentPrice.ID, "", false); err == nil || !strings.Contains(err.Error(), "续费或升级") {
		t.Fatalf("purchase with active entitlement error=%v", err)
	}
	renewal, err := orders.CreateFor(customer.ID, currentPrice.ID, "renewal", entitlement.ID, "", true)
	if err != nil {
		t.Fatalf("create renewal: %v", err)
	}
	wantRenewalExpiry := expiresAt.AddDate(0, currentPrice.Months, 0)
	if renewal.OrderKind != "renewal" || renewal.EntitlementID != entitlement.ID || renewal.ResultExpiresAt == nil || !renewal.ResultExpiresAt.Equal(wantRenewalExpiry) {
		t.Fatalf("renewal order=%+v want expiry=%v", renewal, wantRenewalExpiry)
	}
	if renewal.BalancePaidFen != 250 || renewal.PayableFen != max(int64(0), currentPrice.AmountFen-250) {
		t.Fatalf("renewal did not apply account credit: %+v", renewal)
	}
	if _, err := orders.CreateFor(customer.ID, currentPrice.ID, "renewal", entitlement.ID, "", false); err == nil || !strings.Contains(err.Error(), "已有待处理") {
		t.Fatalf("duplicate renewal error=%v", err)
	}
	if err := orders.Cancel(customer.ID, renewal.ID); err != nil {
		t.Fatalf("cancel renewal: %v", err)
	}
	if err := db.First(&customer, "id = ?", customer.ID).Error; err != nil || customer.BalanceFen != 250 {
		t.Fatalf("renewal cancellation did not restore account credit: balance=%d err=%v", customer.BalanceFen, err)
	}
	if err := orders.config.Set("subscription.offset_enabled", "true", false); err != nil {
		t.Fatal(err)
	}

	upgrade, err := orders.CreateFor(customer.ID, higherPrice.ID, "upgrade", entitlement.ID, "", false)
	if err != nil {
		t.Fatalf("create upgrade: %v", err)
	}
	if upgrade.OrderKind != "upgrade" || upgrade.PlanID != higherPlan.ID || upgrade.EntitlementID != entitlement.ID {
		t.Fatalf("upgrade order=%+v", upgrade)
	}
	if upgrade.DiscountFen <= 0 || upgrade.ResultExpiresAt == nil || !upgrade.ResultExpiresAt.Before(expiresAt.AddDate(0, higherPrice.Months, 0)) {
		t.Fatalf("upgrade offset was not applied: %+v", upgrade)
	}
	if err := orders.Cancel(customer.ID, upgrade.ID); err != nil {
		t.Fatalf("cancel upgrade: %v", err)
	}
	if err := db.Model(&model.Plan{}).Where("id = ?", currentPlan.ID).Update("upgradable", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := orders.CreateFor(customer.ID, higherPrice.ID, "upgrade", entitlement.ID, "", false); err == nil || !strings.Contains(err.Error(), "未开放升级") {
		t.Fatalf("disabled upgrade policy error=%v", err)
	}
}

func TestOrderBalancePaymentAndCancellationRestore(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_DEMO", "true")
	initCommercialTestDB(t)
	db := database.GetDB()
	var price model.PlanPrice
	if err := db.Where("active = ?", true).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	orders := NewOrderService()

	full := model.Customer{ID: uuid.NewString(), Email: "balance-full@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "BALFULL1", BalanceFen: price.AmountFen + 500}
	if err := db.Create(&full).Error; err != nil {
		t.Fatal(err)
	}
	paidOrder, err := orders.Create(full.ID, price.ID, "", true)
	if err != nil {
		t.Fatalf("Create fully balance-paid order: %v", err)
	}
	if paidOrder.Status != OrderPaid || paidOrder.PayableFen != 0 || paidOrder.BalancePaidFen != price.AmountFen {
		t.Fatalf("balance-paid order = %+v", paidOrder)
	}
	if err := db.First(&full, "id = ?", full.ID).Error; err != nil {
		t.Fatal(err)
	}
	if full.BalanceFen != 500 {
		t.Fatalf("remaining balance = %d, want 500", full.BalanceFen)
	}
	var jobs int64
	if err := db.Model(&model.ProvisioningJob{}).Where("order_id = ?", paidOrder.ID).Count(&jobs).Error; err != nil || jobs != 1 {
		t.Fatalf("provisioning jobs = %d, err=%v", jobs, err)
	}

	partial := model.Customer{ID: uuid.NewString(), Email: "balance-partial@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "BALPART1", BalanceFen: 200}
	if err := db.Create(&partial).Error; err != nil {
		t.Fatal(err)
	}
	pendingOrder, err := orders.Create(partial.ID, price.ID, "", true)
	if err != nil {
		t.Fatalf("Create partially balance-paid order: %v", err)
	}
	if pendingOrder.Status != OrderPending || pendingOrder.BalancePaidFen != 200 || pendingOrder.PayableFen != price.AmountFen-200 {
		t.Fatalf("partial order = %+v", pendingOrder)
	}
	if err := orders.Cancel(partial.ID, pendingOrder.ID); err != nil {
		t.Fatalf("Cancel partial order: %v", err)
	}
	if err := db.First(&partial, "id = ?", partial.ID).Error; err != nil {
		t.Fatal(err)
	}
	if partial.BalanceFen != 200 {
		t.Fatalf("restored balance = %d, want 200", partial.BalanceFen)
	}
	if err := db.First(&pendingOrder, "id = ?", pendingOrder.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pendingOrder.Status != OrderCancelled || pendingOrder.BalancePaidFen != 0 || pendingOrder.PayableFen != price.AmountFen {
		t.Fatalf("cancelled order = %+v", pendingOrder)
	}

	held := model.Customer{ID: uuid.NewString(), Email: "balance-held@gmail.com", PasswordHash: "unused", Status: "active", InviteCode: "BALHELD1", BalanceFen: 200}
	if err := db.Create(&held).Error; err != nil {
		t.Fatal(err)
	}
	heldOrder, err := orders.Create(held.ID, price.ID, "", true)
	if err != nil {
		t.Fatalf("Create held balance order: %v", err)
	}
	if err := orders.expireOrder(heldOrder.ID); err != nil {
		t.Fatalf("Expire held balance order: %v", err)
	}
	if err := db.First(&held, "id = ?", held.ID).Error; err != nil || held.BalanceFen != 0 {
		t.Fatalf("balance must stay held during reconciliation: balance=%d err=%v", held.BalanceFen, err)
	}
	oldExpiry := time.Now().UTC().Add(-25 * time.Hour)
	if err := db.Model(&model.Order{}).Where("id = ?", heldOrder.ID).Update("expires_at", oldExpiry).Error; err != nil {
		t.Fatal(err)
	}
	if err := orders.releaseExpiredOrder(heldOrder.ID, time.Now().UTC().Add(-24*time.Hour)); err != nil {
		t.Fatalf("Release expired order: %v", err)
	}
	if err := db.First(&held, "id = ?", held.ID).Error; err != nil || held.BalanceFen != 200 {
		t.Fatalf("balance restoration after reconciliation: balance=%d err=%v", held.BalanceFen, err)
	}
}

func TestLocalizedValueFallback(t *testing.T) {
	raw := `{"zh-CN":"简体中文","en-US":"English","ar-EG":"العربية"}`
	if got := localizedValue(raw, "ar-EG"); got != "العربية" {
		t.Fatalf("requested locale = %q", got)
	}
	if got := localizedValue(raw, "ja-JP"); got != "English" {
		t.Fatalf("English fallback = %q", got)
	}
	if got := localizedValue(`{"zh-CN":"简体中文"}`, "ja-JP"); got != "简体中文" {
		t.Fatalf("Chinese fallback = %q", got)
	}
}

func TestFirstAdminCustomerCanUsePanelCredentials(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	var admin model.User
	if err := db.Order("id asc").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt("AdminPortal2026")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&admin).Update("password", hash).Error; err != nil {
		t.Fatal(err)
	}
	customer, err := NewAuthService(nil).Login(admin.Username, "AdminPortal2026")
	if err != nil {
		t.Fatalf("admin portal login: %v", err)
	}
	if customer.AdminUserID == nil || *customer.AdminUserID != admin.Id {
		t.Fatalf("admin customer link=%v want %d", customer.AdminUserID, admin.Id)
	}
	rows, err := NewAdminService().Customers("", "", 1, 100)
	if err != nil || len(rows.Items) == 0 || !rows.Items[0].SystemAdmin {
		t.Fatalf("first customer is not protected admin: rows=%+v err=%v", rows, err)
	}
}

func TestAdminCreateAndCompletelyDeleteCustomer(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	adminService := NewAdminService()
	customer, err := adminService.CreateCustomer(entity.CommercialCustomerCreateRequest{Email: "manual.customer@gmail.com", Password: "ManualPass2026", DisplayName: "Manual", Locale: "zh-CN", Status: "active"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	var price model.PlanPrice
	if err := db.Where("active = ?", true).First(&price).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	order := model.Order{ID: uuid.NewString(), OutTradeNo: "DELETE-" + uuid.NewString(), CustomerID: customer.ID, PlanID: price.PlanID, PlanPriceID: price.ID, OrderKind: "purchase", Status: OrderCompleted, OriginalFen: price.AmountFen, PayableFen: price.AmountFen, PaidFen: price.AmountFen, Currency: "CNY", ExpiresAt: now, CompletedAt: &now}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: customer.ID, PlanID: price.PlanID, OrderID: order.ID, InternalClientID: "missing-" + uuid.NewString(), SubscriptionID: uuid.NewString(), Status: "active", TrafficQuota: 1, StartsAt: now}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	payment := model.PaymentTransaction{ID: uuid.NewString(), OrderID: order.ID, Provider: "alipay", ProviderTradeNo: uuid.NewString(), AmountFen: price.AmountFen, Status: "paid"}
	job := model.ProvisioningJob{ID: uuid.NewString(), OrderID: order.ID, CustomerID: customer.ID, Status: "completed", NextRunAt: now}
	ticket := model.Ticket{ID: uuid.NewString(), CustomerID: customer.ID, Subject: "delete me", Status: "open"}
	message := model.TicketMessage{ID: uuid.NewString(), TicketID: ticket.ID, SenderType: "customer", SenderID: customer.ID, Body: "body"}
	session := model.CustomerSession{ID: uuid.NewString(), CustomerID: customer.ID, TokenHash: uuid.NewString(), LastSeenAt: now, ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	verification := model.EmailVerification{ID: uuid.NewString(), Email: customer.Email, Purpose: "reset", CodeHash: "hash", ExpiresAt: now.Add(time.Hour)}
	for _, row := range []any{&payment, &job, &ticket, &message, &session, &verification} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	result, err := adminService.DeleteCustomers([]string{customer.ID})
	if err != nil || len(result.Deleted) != 1 || len(result.Failed) != 0 {
		t.Fatalf("delete customer result=%+v err=%v", result, err)
	}
	for name, row := range map[string]any{
		"customer":     &model.Customer{},
		"order":        &model.Order{},
		"payment":      &model.PaymentTransaction{},
		"entitlement":  &model.SubscriptionEntitlement{},
		"job":          &model.ProvisioningJob{},
		"ticket":       &model.Ticket{},
		"message":      &model.TicketMessage{},
		"session":      &model.CustomerSession{},
		"verification": &model.EmailVerification{},
	} {
		var count int64
		query := db.Model(row)
		switch name {
		case "payment", "job":
			query = query.Where("order_id = ?", order.ID)
		case "message":
			query = query.Where("ticket_id = ?", ticket.ID)
		case "verification":
			query = query.Where("email = ?", customer.Email)
		default:
			if name == "customer" {
				query = query.Where("id = ?", customer.ID)
			} else {
				query = query.Where("customer_id = ?", customer.ID)
			}
		}
		if err := query.Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%s rows=%d err=%v", name, count, err)
		}
	}
	var adminCustomer model.Customer
	if err := db.Where("admin_user_id IS NOT NULL").First(&adminCustomer).Error; err != nil {
		t.Fatal(err)
	}
	adminUserID := *adminCustomer.AdminUserID
	deletedAdminCustomer, err := adminService.DeleteCustomers([]string{adminCustomer.ID})
	if err != nil || len(deletedAdminCustomer.Deleted) != 1 || len(deletedAdminCustomer.Failed) != 0 {
		t.Fatalf("admin-linked customer was not deleted: result=%+v err=%v", deletedAdminCustomer, err)
	}
	var remainingAdmin model.User
	if err := db.Where("id = ?", adminUserID).First(&remainingAdmin).Error; err != nil {
		t.Fatalf("panel administrator was deleted with customer: %v", err)
	}
	var deletionMarker model.CommercialSetting
	if err := db.Where("key = ?", database.AdminCustomerDeletionMarkerKey(adminUserID)).First(&deletionMarker).Error; err != nil {
		t.Fatalf("admin customer deletion marker was not persisted: %v", err)
	}
	if deletionMarker.Value != "true" {
		t.Fatalf("admin customer deletion marker=%q want true", deletionMarker.Value)
	}
}

func TestCustomerEmailTemplatesPersist(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	templates, err := admin.EmailTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != len(defaultCustomerEmailTemplates) {
		t.Fatalf("templates=%d want %d", len(templates), len(defaultCustomerEmailTemplates))
	}
	updated, err := admin.SaveEmailTemplate("announcement", entity.CommercialEmailTemplateRequest{
		Name: "重要公告", Subject: "[{{site_name}}] 重要通知", BodyHTML: "<p>{{display_name}}：测试通知</p>", Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "重要公告" || updated.Subject != "[{{site_name}}] 重要通知" {
		t.Fatalf("updated template=%+v", updated)
	}
	reloaded, err := admin.EmailTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded[0].Name != "重要公告" {
		t.Fatalf("template update was not persisted: %+v", reloaded[0])
	}
}

type recordingCustomerMailer struct {
	recipients []string
	subject    string
	body       string
}

func (m *recordingCustomerMailer) SendTo(recipients []string, subject, body string) error {
	m.recipients = append([]string(nil), recipients...)
	m.subject = subject
	m.body = body
	return nil
}

func TestQueueAndDeliverSelectedCustomerEmail(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	customer, err := admin.CreateCustomer(entity.CommercialCustomerCreateRequest{
		Email: "mail.target@gmail.com", Password: "MailTarget2026", DisplayName: "目标用户", Locale: "zh-CN", Status: "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := admin.QueueCustomerEmail(entity.CommercialEmailSendRequest{
		Audience: "selected", CustomerIDs: []string{customer.ID}, TemplateKey: "announcement",
		Subject: "[{{site_name}}] 给 {{display_name}}", BodyHTML: "<p>{{display_name}} / {{email}}</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Queued != 1 || result.CampaignID == "" {
		t.Fatalf("queue result=%+v", result)
	}

	mailer := &recordingCustomerMailer{}
	worker := NewWorker()
	worker.mailer = mailer
	if err := worker.processNextOutbox(); err != nil {
		t.Fatal(err)
	}
	if len(mailer.recipients) != 1 || mailer.recipients[0] != customer.Email {
		t.Fatalf("recipients=%v", mailer.recipients)
	}
	if !strings.Contains(mailer.subject, "目标用户") || !strings.Contains(mailer.body, customer.Email) {
		t.Fatalf("subject=%q body=%q", mailer.subject, mailer.body)
	}
	var event model.OutboxEvent
	if err := database.GetDB().Where("aggregate_id = ? AND event_type = ?", result.CampaignID, "email.customer").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Status != "processed" || event.ProcessedAt == nil || event.Attempts != 1 {
		t.Fatalf("event=%+v", event)
	}
}

func TestSubscribedAudienceOnlyQueuesActiveEntitlements(t *testing.T) {
	initCommercialTestDB(t)
	admin := NewAdminService()
	db := database.GetDB()
	active, err := admin.CreateCustomer(entity.CommercialCustomerCreateRequest{Email: "subscribed.target@gmail.com", Password: "Subscribed2026", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := admin.CreateCustomer(entity.CommercialCustomerCreateRequest{Email: "plain.target@gmail.com", Password: "PlainTarget2026", Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	var plan model.Plan
	if err := db.Where("active = ?", true).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	entitlement := model.SubscriptionEntitlement{
		ID: uuid.NewString(), CustomerID: active.ID, PlanID: plan.ID, OrderID: uuid.NewString(),
		InternalClientID: uuid.NewString(), SubscriptionID: uuid.NewString(), Status: "active",
		TrafficQuota: 1, StartsAt: now, CreatedAt: now,
	}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	result, err := admin.QueueCustomerEmail(entity.CommercialEmailSendRequest{
		Audience: "subscribed", Subject: "订阅通知", BodyHTML: "<p>{{email}}</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Queued != 1 {
		t.Fatalf("queued=%d want 1 (plain customer %s must be excluded)", result.Queued, plain.ID)
	}
	var event model.OutboxEvent
	if err := db.Where("aggregate_id = ?", result.CampaignID).First(&event).Error; err != nil {
		t.Fatal(err)
	}
	var payload customerEmailPayload
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.CustomerID != active.ID {
		t.Fatalf("queued customer=%s want %s", payload.CustomerID, active.ID)
	}
}

func initCommercialTestDB(t *testing.T) {
	t.Helper()
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	if err := database.InitDB(filepath.Join(t.TempDir(), "commercial-test.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := database.GetDB().Model(&model.Plan{}).Where("active = ?", false).Update("active", true).Error; err != nil {
		t.Fatalf("activate test plans: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := database.GetDB().DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}
