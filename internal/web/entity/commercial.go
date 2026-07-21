package entity

type SendVerificationCodeRequest struct {
	Email          string `json:"email" validate:"required"`
	Purpose        string `json:"purpose" validate:"required,oneof=register reset"`
	TurnstileToken string `json:"turnstileToken"`
}

type CustomerRegisterRequest struct {
	Email          string `json:"email" validate:"required"`
	Password       string `json:"password" validate:"required"`
	Code           string `json:"code" validate:"omitempty,len=6"`
	InviteCode     string `json:"inviteCode"`
	Locale         string `json:"locale"`
	TurnstileToken string `json:"turnstileToken"`
	AcceptedTerms  bool   `json:"acceptedTerms"`
	TermsVersion   string `json:"termsVersion"`
}

type CustomerLoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CustomerResetPasswordRequest struct {
	Email    string `json:"email" validate:"required"`
	Code     string `json:"code" validate:"required,len=6"`
	Password string `json:"password" validate:"required"`
}

type CustomerChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required"`
}

type CreateCommercialOrderRequest struct {
	PlanPriceID   string `json:"planPriceId" validate:"required"`
	OrderKind     string `json:"orderKind,omitempty" validate:"omitempty,oneof=purchase renewal upgrade"`
	EntitlementID string `json:"entitlementId,omitempty"`
	CouponCode    string `json:"couponCode"`
	UseBalance    bool   `json:"useBalance"`
}

type CreatePaymentRequest struct {
	Provider string `json:"provider" validate:"required,oneof=epay alipay_f2f codepay alipay-demo"`
}

type CreateTicketRequest struct {
	EntitlementID string `json:"entitlementId,omitempty" validate:"omitempty,uuid"`
	Subject       string `json:"subject" validate:"required,max=200"`
	Body          string `json:"body" validate:"required,max=10000"`
}

type ReplyTicketRequest struct {
	Body string `json:"body" validate:"required,max=10000"`
}

type RedeemGiftCardRequest struct {
	Code string `json:"code" validate:"required,max=64"`
}

type CommercialSettingRequest struct {
	Key       string `json:"key" validate:"required,max=120"`
	Value     string `json:"value"`
	Encrypted bool   `json:"encrypted"`
}

type CommercialPaymentSettings struct {
	Provider           string `json:"provider,omitempty" validate:"omitempty,oneof=epay alipay_f2f codepay"`
	EpayEnabled        bool   `json:"epayEnabled"`
	EpayGatewayURL     string `json:"epayGatewayUrl"`
	EpayMerchantID     string `json:"epayMerchantId"`
	EpayMerchantKey    string `json:"epayMerchantKey"`
	EpayPaymentType    string `json:"epayPaymentType"`
	EpayNotifyURL      string `json:"epayNotifyUrl"`
	EpayReturnURL      string `json:"epayReturnUrl"`
	AlipayEnabled      bool   `json:"alipayEnabled"`
	AlipayMode         string `json:"alipayMode" validate:"omitempty,oneof=sandbox production"`
	AlipayAppID        string `json:"alipayAppId"`
	AlipaySellerID     string `json:"alipaySellerId"`
	AlipayNotifyURL    string `json:"alipayNotifyUrl"`
	AlipayPrivateKey   string `json:"alipayPrivateKey"`
	AlipayPublicKey    string `json:"alipayPublicKey"`
	CodepayEnabled     bool   `json:"codepayEnabled"`
	CodepayGatewayURL  string `json:"codepayGatewayUrl"`
	CodepayMerchantID  string `json:"codepayMerchantId"`
	CodepayKey         string `json:"codepayKey"`
	CodepayPaymentType string `json:"codepayPaymentType"`
	CodepayNotifyURL   string `json:"codepayNotifyUrl"`
	CodepayReturnURL   string `json:"codepayReturnUrl"`
}

type CommercialSiteSettings struct {
	SiteName           string `json:"siteName"`
	SiteDescription    string `json:"siteDescription"`
	SiteURL            string `json:"siteUrl"`
	ForceHTTPS         bool   `json:"forceHttps"`
	LogoURL            string `json:"logoUrl"`
	SubscriptionURLs   string `json:"subscriptionUrls"`
	TermsURL           string `json:"termsUrl"`
	TermsTemplate      string `json:"termsTemplate"`
	TermsTitle         string `json:"termsTitle"`
	TermsContent       string `json:"termsContent"`
	RegistrationClosed bool   `json:"registrationClosed"`
	TrialPlanID        string `json:"trialPlanId"`
	Currency           string `json:"currency"`
	CurrencySymbol     string `json:"currencySymbol"`
}

type CommercialSiteLogoRequest struct {
	LogoURL string `json:"logoUrl"`
}

type CommercialTrialPlanOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CommercialSiteSettingsResponse struct {
	Settings CommercialSiteSettings      `json:"settings"`
	Plans    []CommercialTrialPlanOption `json:"plans"`
}

type CommercialSecuritySettings struct {
	EmailVerification           bool   `json:"emailVerification"`
	DisallowGmailAliases        bool   `json:"disallowGmailAliases"`
	SafeMode                    bool   `json:"safeMode"`
	EmailSuffixWhitelistEnabled bool   `json:"emailSuffixWhitelistEnabled"`
	AllowedEmailSuffixes        string `json:"allowedEmailSuffixes"`
	RegistrationCaptchaEnabled  bool   `json:"registrationCaptchaEnabled"`
	IPRegistrationLimitEnabled  bool   `json:"ipRegistrationLimitEnabled"`
	PasswordAttemptLimitEnabled bool   `json:"passwordAttemptLimitEnabled"`
	MaxPasswordAttempts         int    `json:"maxPasswordAttempts"`
	PasswordLockDurationMinutes int    `json:"passwordLockDurationMinutes"`
}

type CommercialSubscriptionSettings struct {
	AllowUserChange        bool   `json:"allowUserChange"`
	MonthlyResetMode       string `json:"monthlyResetMode"`
	OffsetEnabled          bool   `json:"offsetEnabled"`
	PurchaseEvent          string `json:"purchaseEvent"`
	RenewalEvent           string `json:"renewalEvent"`
	ChangeEvent            string `json:"changeEvent"`
	ShowSubscriptionInfo   bool   `json:"showSubscriptionInfo"`
	ShowProtocolInNodeName bool   `json:"showProtocolInNodeName"`
}

type CommercialInvitationSettings struct {
	ForcedInvitation           bool `json:"forcedInvitation"`
	CommissionPercent          int  `json:"commissionPercent"`
	MaxInviteCodes             int  `json:"maxInviteCodes"`
	InviteCodesNeverExpire     bool `json:"inviteCodesNeverExpire"`
	CommissionFirstPaymentOnly bool `json:"commissionFirstPaymentOnly"`
	CommissionAutoConfirm      bool `json:"commissionAutoConfirm"`
	MultiLevelEnabled          bool `json:"multiLevelEnabled"`
}

type CommercialEmailTemplateRequest struct {
	Name     string `json:"name" validate:"required,max=120"`
	Subject  string `json:"subject" validate:"required,max=200"`
	BodyHTML string `json:"bodyHtml" validate:"required,max=200000"`
	Active   bool   `json:"active"`
}

type CommercialEmailSendRequest struct {
	Audience    string   `json:"audience" validate:"required,oneof=selected active subscribed"`
	CustomerIDs []string `json:"customerIds" validate:"max=5000"`
	TemplateKey string   `json:"templateKey" validate:"max=64"`
	Subject     string   `json:"subject" validate:"required,max=200"`
	BodyHTML    string   `json:"bodyHtml" validate:"required,max=200000"`
}

type CommercialPlanRequest struct {
	ID                        string            `json:"id"`
	Slug                      string            `json:"slug" validate:"required,max=80"`
	Name                      string            `json:"name" validate:"required,max=120"`
	Description               string            `json:"description"`
	TrafficBytes              int64             `json:"trafficBytes" validate:"gte=0"`
	DeviceLimit               int               `json:"deviceLimit" validate:"gte=0,lte=1000"`
	TrafficMultiplierPermille int               `json:"trafficMultiplierPermille" validate:"omitempty,gte=100,lte=100000"`
	UploadLimitMbps           int               `json:"uploadLimitMbps" validate:"gte=0,lte=100000"`
	DownloadLimitMbps         int               `json:"downloadLimitMbps" validate:"gte=0,lte=100000"`
	ResidentialRelayEnabled   bool              `json:"residentialRelayEnabled"`
	ResidentialRelayLimit     int               `json:"residentialRelayLimit" validate:"gte=0,lte=20"`
	ResetCycle                string            `json:"resetCycle" validate:"required,oneof=never daily weekly monthly quarterly"`
	NodeGroup                 string            `json:"nodeGroup"`
	Capacity                  int               `json:"capacity" validate:"gte=0"`
	Visibility                string            `json:"visibility" validate:"required,oneof=public hidden invite"`
	Renewable                 bool              `json:"renewable"`
	Upgradable                bool              `json:"upgradable"`
	Active                    bool              `json:"active"`
	SortOrder                 int               `json:"sortOrder"`
	ProvisionInboundIDs       []int             `json:"provisionInboundIds"`
	LineGroupIDs              []string          `json:"lineGroupIds" validate:"max=100"`
	DisplayBenefits           map[string]string `json:"displayBenefits"`
}

type CommercialLineGroupRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name" validate:"required,max=120"`
	Description string `json:"description" validate:"max=500"`
	Active      bool   `json:"active"`
}

type CommercialLineSourceRequest struct {
	ID              string   `json:"id"`
	Name            string   `json:"name" validate:"required,max=120"`
	URL             string   `json:"url" validate:"max=4096"`
	RefreshInterval int      `json:"refreshInterval" validate:"gte=300,lte=86400"`
	Enabled         bool     `json:"enabled"`
	GroupIDs        []string `json:"groupIds" validate:"required,min=1,max=100"`
	PlanIDs         []string `json:"planIds" validate:"max=100"`
}

type CommercialLineImportRequest struct {
	Name  string `json:"name" validate:"max=120"`
	Links string `json:"links" validate:"required,max=8388608"`
}

type CommercialLineNodeGroupsRequest struct {
	NodeIDs  []string `json:"nodeIds" validate:"required,min=1,max=500"`
	GroupIDs []string `json:"groupIds" validate:"max=100"`
}

type CommercialPlanPriceRequest struct {
	ID            string `json:"id"`
	PlanID        string `json:"planId" validate:"required"`
	BillingPeriod string `json:"billingPeriod" validate:"required"`
	Months        int    `json:"months" validate:"gte=0,lte=120"`
	AmountFen     int64  `json:"amountFen" validate:"gte=0"`
	Active        bool   `json:"active"`
}

type CommercialCustomerUpdateRequest struct {
	Status     string `json:"status"`
	BalanceFen *int64 `json:"balanceFen"`
}

type CommercialCustomerCreateRequest struct {
	Email       string `json:"email" validate:"required"`
	Password    string `json:"password" validate:"required"`
	DisplayName string `json:"displayName" validate:"max=80"`
	Locale      string `json:"locale"`
	Status      string `json:"status" validate:"omitempty,oneof=active suspended"`
}

type CommercialCustomerDeleteRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,max=500"`
}

type CommercialSubscriptionUpdateRequest struct {
	PlanID                    string `json:"planId" validate:"required"`
	ExpiresAt                 string `json:"expiresAt"`
	TrafficQuota              int64  `json:"trafficQuota" validate:"gte=0"`
	DeviceLimit               int    `json:"deviceLimit" validate:"gte=0,lte=1000"`
	TrafficMultiplierPermille *int   `json:"trafficMultiplierPermille" validate:"omitempty,gte=100,lte=100000"`
	UploadLimitMbps           *int   `json:"uploadLimitMbps" validate:"omitempty,gte=0,lte=100000"`
	DownloadLimitMbps         *int   `json:"downloadLimitMbps" validate:"omitempty,gte=0,lte=100000"`
	ResetTraffic              bool   `json:"resetTraffic"`
}

type ResidentialRelayRequest struct {
	InboundID int    `json:"inboundId" validate:"required,gt=0"`
	Name      string `json:"name" validate:"max=80"`
	Host      string `json:"host" validate:"required,max=253"`
	Port      int    `json:"port" validate:"required,gte=1,lte=65535"`
	Username  string `json:"username" validate:"max=256"`
	Password  string `json:"password" validate:"max=1024"`
}

type CommercialTicketReplyRequest struct {
	Body   string `json:"body" validate:"required,max=10000"`
	Status string `json:"status"`
}

type CommercialGiftCardBatchRequest struct {
	ValueFen  int64  `json:"valueFen" validate:"gt=0" example:"1000"`
	Count     int    `json:"count" validate:"gte=1,lte=100"`
	ExpiresAt string `json:"expiresAt"`
}

type CommercialRoleRequest struct {
	Role string `json:"role" validate:"required"`
}
