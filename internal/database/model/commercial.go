package model

import "time"

type Customer struct {
	ID                  string     `json:"id" gorm:"primaryKey;size:36"`
	AdminUserID         *int       `json:"-" gorm:"uniqueIndex"`
	Email               string     `json:"email" gorm:"uniqueIndex;size:320;not null"`
	PasswordHash        string     `json:"-" gorm:"size:128;not null"`
	DisplayName         string     `json:"displayName" gorm:"size:80"`
	Locale              string     `json:"locale" gorm:"size:16;default:zh-CN"`
	Status              string     `json:"status" gorm:"size:24;index;default:active"`
	BalanceFen          int64      `json:"balanceFen" gorm:"default:0"`
	InviteCode          string     `json:"inviteCode" gorm:"uniqueIndex;size:20"`
	InvitedByID         *string    `json:"invitedById,omitempty" gorm:"size:36;index"`
	EmailVerifiedAt     *time.Time `json:"emailVerifiedAt,omitempty"`
	TermsAcceptedAt     *time.Time `json:"termsAcceptedAt,omitempty"`
	TermsVersion        string     `json:"termsVersion,omitempty" gorm:"size:64"`
	LastLoginAt         *time.Time `json:"lastLoginAt,omitempty"`
	LoginEpoch          int64      `json:"-" gorm:"default:0"`
	RegistrationIPHash  string     `json:"-" gorm:"size:64;index"`
	FailedLoginAttempts int        `json:"-" gorm:"default:0"`
	LoginLockedUntil    *time.Time `json:"-" gorm:"index"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

func (Customer) TableName() string { return "commercial_customers" }

type CustomerSession struct {
	ID            string     `json:"id" gorm:"primaryKey;size:36"`
	CustomerID    string     `json:"customerId" gorm:"size:36;index;not null"`
	TokenHash     string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	IPHash        string     `json:"ipHash" gorm:"size:64"`
	UserAgentHash string     `json:"userAgentHash" gorm:"size:64"`
	LastSeenAt    time.Time  `json:"lastSeenAt"`
	ExpiresAt     time.Time  `json:"expiresAt" gorm:"index"`
	RevokedAt     *time.Time `json:"revokedAt,omitempty" gorm:"index"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func (CustomerSession) TableName() string { return "commercial_customer_sessions" }

type EmailVerification struct {
	ID        string     `json:"id" gorm:"primaryKey;size:36"`
	Email     string     `json:"email" gorm:"size:320;index;not null"`
	Purpose   string     `json:"purpose" gorm:"size:24;index;not null"`
	CodeHash  string     `json:"-" gorm:"size:64;not null"`
	IPHash    string     `json:"-" gorm:"size:64;index"`
	Attempts  int        `json:"attempts" gorm:"default:0"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"index"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt" gorm:"index"`
}

func (EmailVerification) TableName() string { return "commercial_email_verifications" }

type Plan struct {
	ID                        string            `json:"id" gorm:"primaryKey;size:36"`
	Slug                      string            `json:"slug" gorm:"uniqueIndex;size:80;not null"`
	Name                      string            `json:"name" gorm:"size:120;not null"`
	Description               string            `json:"description" gorm:"type:text"`
	TrafficBytes              int64             `json:"trafficBytes" gorm:"not null"`
	DeviceLimit               int               `json:"deviceLimit" gorm:"default:5"`
	TrafficMultiplierPermille int               `json:"trafficMultiplierPermille" gorm:"default:1000"`
	UploadLimitMbps           int               `json:"uploadLimitMbps" gorm:"default:0"`
	DownloadLimitMbps         int               `json:"downloadLimitMbps" gorm:"default:0"`
	ResidentialRelayEnabled   bool              `json:"residentialRelayEnabled" gorm:"default:false"`
	ResidentialRelayLimit     int               `json:"residentialRelayLimit" gorm:"default:0"`
	ResetCycle                string            `json:"resetCycle" gorm:"size:24;default:monthly"`
	NodeGroup                 string            `json:"nodeGroup" gorm:"size:80;index"`
	Capacity                  int               `json:"capacity" gorm:"default:0"`
	Visibility                string            `json:"visibility" gorm:"size:24;index;default:public"`
	Renewable                 bool              `json:"renewable" gorm:"default:true"`
	Upgradable                bool              `json:"upgradable" gorm:"default:true"`
	Active                    bool              `json:"active" gorm:"index;default:true"`
	SortOrder                 int               `json:"sortOrder" gorm:"default:0"`
	ProvisionInboundIDs       string            `json:"provisionInboundIds" gorm:"type:text"`
	DisplayBenefits           map[string]string `json:"displayBenefits" gorm:"serializer:json;type:text"`
	CreatedAt                 time.Time         `json:"createdAt"`
	UpdatedAt                 time.Time         `json:"updatedAt"`
}

func (Plan) TableName() string { return "commercial_plans" }

type PlanPrice struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	PlanID        string    `json:"planId" gorm:"size:36;index;not null"`
	BillingPeriod string    `json:"billingPeriod" gorm:"size:24;index;not null"`
	Months        int       `json:"months" gorm:"not null"`
	AmountFen     int64     `json:"amountFen" gorm:"not null"`
	Active        bool      `json:"active" gorm:"index;default:true"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (PlanPrice) TableName() string { return "commercial_plan_prices" }

type Order struct {
	ID              string     `json:"id" gorm:"primaryKey;size:36"`
	OutTradeNo      string     `json:"outTradeNo" gorm:"uniqueIndex;size:64;not null"`
	CustomerID      string     `json:"customerId" gorm:"size:36;index;not null"`
	PlanID          string     `json:"planId" gorm:"size:36;index;not null"`
	PlanPriceID     string     `json:"planPriceId" gorm:"size:36;index;not null"`
	OrderKind       string     `json:"orderKind" gorm:"size:24;index;not null;default:purchase"`
	EntitlementID   string     `json:"entitlementId,omitempty" gorm:"size:36;index"`
	ResultExpiresAt *time.Time `json:"resultExpiresAt,omitempty"`
	Status          string     `json:"status" gorm:"size:24;index;not null"`
	OriginalFen     int64      `json:"originalFen" gorm:"not null"`
	DiscountFen     int64      `json:"discountFen" gorm:"default:0"`
	BalancePaidFen  int64      `json:"balancePaidFen" gorm:"default:0"`
	PayableFen      int64      `json:"payableFen" gorm:"not null"`
	PaidFen         int64      `json:"paidFen" gorm:"default:0"`
	Currency        string     `json:"currency" gorm:"size:8;default:CNY"`
	CouponCode      string     `json:"couponCode" gorm:"size:48"`
	FailureReason   string     `json:"failureReason" gorm:"type:text"`
	ExpiresAt       time.Time  `json:"expiresAt" gorm:"index"`
	PaidAt          *time.Time `json:"paidAt,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (Order) TableName() string { return "commercial_orders" }

type PaymentTransaction struct {
	ID              string    `json:"id" gorm:"primaryKey;size:36"`
	OrderID         string    `json:"orderId" gorm:"size:36;index;not null"`
	Provider        string    `json:"provider" gorm:"size:32;index;uniqueIndex:idx_commercial_provider_trade;not null"`
	ProviderTradeNo string    `json:"providerTradeNo" gorm:"size:96;uniqueIndex:idx_commercial_provider_trade"`
	AmountFen       int64     `json:"amountFen" gorm:"not null"`
	Status          string    `json:"status" gorm:"size:24;index;not null"`
	RawPayload      string    `json:"-" gorm:"type:text"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (PaymentTransaction) TableName() string { return "commercial_payment_transactions" }

type SubscriptionEntitlement struct {
	ID                        string     `json:"id" gorm:"primaryKey;size:36"`
	CustomerID                string     `json:"customerId" gorm:"size:36;index;not null"`
	PlanID                    string     `json:"planId" gorm:"size:36;index;not null"`
	OrderID                   string     `json:"orderId" gorm:"size:36;uniqueIndex;not null"`
	InternalClientID          string     `json:"internalClientId" gorm:"size:80;uniqueIndex;not null"`
	SubscriptionID            string     `json:"-" gorm:"size:80;uniqueIndex;not null"`
	Status                    string     `json:"status" gorm:"size:24;index;not null"`
	TrafficQuota              int64      `json:"trafficQuota" gorm:"not null"`
	TrafficUsed               int64      `json:"trafficUsed" gorm:"default:0"`
	DeviceLimit               int        `json:"deviceLimit" gorm:"default:5"`
	TrafficMultiplierPermille int        `json:"trafficMultiplierPermille" gorm:"default:1000"`
	UploadLimitMbps           int        `json:"uploadLimitMbps" gorm:"default:0"`
	DownloadLimitMbps         int        `json:"downloadLimitMbps" gorm:"default:0"`
	ResidentialRelayEnabled   bool       `json:"residentialRelayEnabled" gorm:"default:false"`
	ResidentialRelayLimit     int        `json:"residentialRelayLimit" gorm:"default:0"`
	NodeGroup                 string     `json:"nodeGroup" gorm:"size:80;index"`
	StartsAt                  time.Time  `json:"startsAt"`
	ExpiresAt                 *time.Time `json:"expiresAt,omitempty" gorm:"index"`
	LastResetAt               *time.Time `json:"lastResetAt,omitempty"`
	CreatedAt                 time.Time  `json:"createdAt"`
	UpdatedAt                 time.Time  `json:"updatedAt"`
}

func (SubscriptionEntitlement) TableName() string { return "commercial_subscription_entitlements" }

type ResidentialRelay struct {
	ID                 string    `json:"id" gorm:"primaryKey;size:36"`
	CustomerID         string    `json:"customerId" gorm:"size:36;index;not null"`
	EntitlementID      string    `json:"entitlementId" gorm:"size:36;uniqueIndex:idx_commercial_relay_line;not null"`
	InboundID          int       `json:"inboundId" gorm:"uniqueIndex:idx_commercial_relay_line;index;not null"`
	Name               string    `json:"name" gorm:"size:80;not null"`
	OutboundTag        string    `json:"-" gorm:"size:96;uniqueIndex;not null"`
	SOCKSHost          string    `json:"host" gorm:"size:253;not null"`
	SOCKSPort          int       `json:"port" gorm:"not null"`
	UsernameCiphertext string    `json:"-" gorm:"type:text"`
	PasswordCiphertext string    `json:"-" gorm:"type:text"`
	Status             string    `json:"status" gorm:"size:24;index;default:pending"`
	LastError          string    `json:"-" gorm:"type:text"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (ResidentialRelay) TableName() string { return "commercial_residential_relays" }

type LineSource struct {
	ID                   string     `json:"id" gorm:"primaryKey;size:36"`
	Name                 string     `json:"name" gorm:"size:120;not null"`
	Kind                 string     `json:"kind" gorm:"size:16;index;not null"`
	SecretCiphertext     string     `json:"-" gorm:"type:text"`
	URLHost              string     `json:"urlHost,omitempty" gorm:"size:253"`
	RefreshInterval      int        `json:"refreshInterval" gorm:"default:1800;not null"`
	Enabled              bool       `json:"enabled" gorm:"index;default:true"`
	Status               string     `json:"status" gorm:"size:24;index;default:pending"`
	LastError            string     `json:"lastError,omitempty" gorm:"type:text"`
	ConsecutiveSuccesses int        `json:"consecutiveSuccesses" gorm:"default:0"`
	LastSuccessAt        *time.Time `json:"lastSuccessAt,omitempty"`
	NextRefreshAt        *time.Time `json:"nextRefreshAt,omitempty" gorm:"index"`
	LockedAt             *time.Time `json:"-" gorm:"index"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (LineSource) TableName() string { return "commercial_line_sources" }

type LineNode struct {
	ID                 string     `json:"id" gorm:"primaryKey;size:36"`
	Fingerprint        string     `json:"fingerprint" gorm:"uniqueIndex;size:64;not null"`
	Remark             string     `json:"remark" gorm:"size:160;not null"`
	PublicName         string     `json:"publicName" gorm:"size:160;not null;default:''"`
	PublicNameCustom   bool       `json:"-" gorm:"not null;default:false"`
	Protocol           string     `json:"protocol" gorm:"size:24;index;not null"`
	OutboundTag        string     `json:"outboundTag" gorm:"uniqueIndex;size:96;not null"`
	OutboundCiphertext string     `json:"-" gorm:"type:text;not null"`
	TLSAutoPinned      bool       `json:"-" gorm:"not null;default:false"`
	PublicPort         *int       `json:"publicPort,omitempty" gorm:"uniqueIndex"`
	InboundID          *int       `json:"inboundId,omitempty" gorm:"uniqueIndex;index"`
	Status             string     `json:"status" gorm:"size:24;index:idx_commercial_line_health,priority:1;not null"`
	HealthStatus       string     `json:"healthStatus" gorm:"size:24;index:idx_commercial_line_health,priority:2;not null"`
	LatencyMS          int64      `json:"latencyMs" gorm:"default:0"`
	ConsecutiveFails   int        `json:"consecutiveFails" gorm:"default:0"`
	ConsecutivePasses  int        `json:"consecutivePasses" gorm:"default:0"`
	ProvisionAttempts  int        `json:"provisionAttempts" gorm:"default:0"`
	ProvisionVersion   int        `json:"-" gorm:"not null;default:0"`
	ProvisionLockedAt  *time.Time `json:"-" gorm:"index"`
	NextProvisionAt    *time.Time `json:"-" gorm:"index"`
	LastProbeAt        *time.Time `json:"lastProbeAt,omitempty"`
	LastSeenAt         *time.Time `json:"lastSeenAt,omitempty"`
	MissingSince       *time.Time `json:"missingSince,omitempty" gorm:"index"`
	LastError          string     `json:"lastError,omitempty" gorm:"type:text"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (LineNode) TableName() string { return "commercial_line_nodes" }

type LineSourceNode struct {
	SourceID     string     `json:"sourceId" gorm:"primaryKey;size:36"`
	NodeID       string     `json:"nodeId" gorm:"primaryKey;size:36;index"`
	LastSeenAt   time.Time  `json:"lastSeenAt"`
	MissingSince *time.Time `json:"missingSince,omitempty" gorm:"index"`
}

func (LineSourceNode) TableName() string { return "commercial_line_source_nodes" }

type LineGroup struct {
	ID          string    `json:"id" gorm:"primaryKey;size:36"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:120;not null"`
	Description string    `json:"description" gorm:"size:500"`
	Active      bool      `json:"active" gorm:"index;default:true"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (LineGroup) TableName() string { return "commercial_line_groups" }

type LineSourceGroup struct {
	SourceID string `json:"sourceId" gorm:"primaryKey;size:36"`
	GroupID  string `json:"groupId" gorm:"primaryKey;size:36;index"`
}

func (LineSourceGroup) TableName() string { return "commercial_line_source_groups" }

type LineGroupNode struct {
	GroupID string `json:"groupId" gorm:"primaryKey;size:36"`
	NodeID  string `json:"nodeId" gorm:"primaryKey;size:36;index"`
}

func (LineGroupNode) TableName() string { return "commercial_line_group_nodes" }

type PlanLineGroup struct {
	PlanID  string `json:"planId" gorm:"primaryKey;size:36"`
	GroupID string `json:"groupId" gorm:"primaryKey;size:36;index"`
}

func (PlanLineGroup) TableName() string { return "commercial_plan_line_groups" }

type ProvisioningJob struct {
	ID         string     `json:"id" gorm:"primaryKey;size:36"`
	OrderID    string     `json:"orderId" gorm:"size:36;uniqueIndex;not null"`
	CustomerID string     `json:"customerId" gorm:"size:36;index;not null"`
	Status     string     `json:"status" gorm:"size:24;index;not null"`
	Attempts   int        `json:"attempts" gorm:"default:0"`
	NextRunAt  time.Time  `json:"nextRunAt" gorm:"index"`
	LockedAt   *time.Time `json:"lockedAt,omitempty"`
	LastError  string     `json:"lastError" gorm:"type:text"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (ProvisioningJob) TableName() string { return "commercial_provisioning_jobs" }

type OutboxEvent struct {
	ID            string     `json:"id" gorm:"primaryKey;size:36"`
	AggregateType string     `json:"aggregateType" gorm:"size:48;index;not null"`
	AggregateID   string     `json:"aggregateId" gorm:"size:64;index;not null"`
	EventType     string     `json:"eventType" gorm:"size:80;index;not null"`
	Payload       string     `json:"payload" gorm:"type:text;not null"`
	Status        string     `json:"status" gorm:"size:24;index;not null"`
	Attempts      int        `json:"attempts" gorm:"default:0"`
	NextRunAt     time.Time  `json:"nextRunAt" gorm:"index"`
	LastError     string     `json:"lastError" gorm:"type:text"`
	ProcessedAt   *time.Time `json:"processedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (OutboxEvent) TableName() string { return "commercial_outbox_events" }

type EmailTemplate struct {
	Key       string    `json:"key" gorm:"primaryKey;size:64"`
	Name      string    `json:"name" gorm:"size:120;not null"`
	Subject   string    `json:"subject" gorm:"size:200;not null"`
	BodyHTML  string    `json:"bodyHtml" gorm:"type:text;not null"`
	Active    bool      `json:"active" gorm:"default:true"`
	System    bool      `json:"system" gorm:"default:true"`
	SortOrder int       `json:"sortOrder" gorm:"default:0"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (EmailTemplate) TableName() string { return "commercial_email_templates" }

type Notice struct {
	ID          string     `json:"id" gorm:"primaryKey;size:36"`
	Slug        string     `json:"slug" gorm:"uniqueIndex;size:96;not null"`
	Level       string     `json:"level" gorm:"size:24;default:info"`
	TitleI18n   string     `json:"titleI18n" gorm:"type:text;not null"`
	ContentI18n string     `json:"contentI18n" gorm:"type:text;not null"`
	Published   bool       `json:"published" gorm:"index;default:false"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func (Notice) TableName() string { return "commercial_notices" }

type KnowledgeArticle struct {
	ID          string    `json:"id" gorm:"primaryKey;size:36"`
	Slug        string    `json:"slug" gorm:"uniqueIndex;size:96;not null"`
	Category    string    `json:"category" gorm:"size:48;index"`
	TitleI18n   string    `json:"titleI18n" gorm:"type:text;not null"`
	ContentI18n string    `json:"contentI18n" gorm:"type:text;not null"`
	Published   bool      `json:"published" gorm:"index;default:false"`
	SortOrder   int       `json:"sortOrder" gorm:"default:0"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (KnowledgeArticle) TableName() string { return "commercial_knowledge_articles" }

type ClientApplication struct {
	ID                 string    `json:"id" gorm:"primaryKey;size:36"`
	Slug               string    `json:"slug" gorm:"uniqueIndex;size:80;not null"`
	Name               string    `json:"name" gorm:"size:120;not null"`
	Platform           string    `json:"platform" gorm:"size:32;index"`
	OfficialURL        string    `json:"officialUrl,omitempty" gorm:"size:1024;not null"`
	SourceURL          string    `json:"sourceUrl,omitempty" gorm:"size:1024"`
	Description        string    `json:"description" gorm:"type:text"`
	PackageFileName    string    `json:"packageFileName,omitempty" gorm:"size:255"`
	PackageStoredName  string    `json:"-" gorm:"size:255"`
	PackageSize        int64     `json:"packageSize,omitempty" gorm:"default:0"`
	PackageSHA256      string    `json:"packageSha256,omitempty" gorm:"size:64"`
	PackageContentType string    `json:"packageContentType,omitempty" gorm:"size:120"`
	DownloadURL        string    `json:"downloadUrl,omitempty" gorm:"-"`
	Active             bool      `json:"active" gorm:"index;default:true"`
	SortOrder          int       `json:"sortOrder" gorm:"default:0"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (ClientApplication) TableName() string { return "commercial_client_applications" }

type Ticket struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	CustomerID    string    `json:"customerId" gorm:"size:36;index;not null"`
	EntitlementID string    `json:"entitlementId,omitempty" gorm:"size:36;index"`
	PlanID        string    `json:"planId,omitempty" gorm:"size:36;index"`
	PlanName      string    `json:"planName,omitempty" gorm:"size:160"`
	Subject       string    `json:"subject" gorm:"size:200;not null"`
	Status        string    `json:"status" gorm:"size:24;index;not null"`
	Priority      string    `json:"priority" gorm:"size:24;index;default:normal"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (Ticket) TableName() string { return "commercial_tickets" }

type TicketMessage struct {
	ID         string    `json:"id" gorm:"primaryKey;size:36"`
	TicketID   string    `json:"ticketId" gorm:"size:36;index;not null"`
	SenderType string    `json:"senderType" gorm:"size:24;not null"`
	SenderID   string    `json:"senderId" gorm:"size:36;index"`
	Body       string    `json:"body" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (TicketMessage) TableName() string { return "commercial_ticket_messages" }

type Coupon struct {
	ID             string     `json:"id" gorm:"primaryKey;size:36"`
	Code           string     `json:"code" gorm:"uniqueIndex;size:48;not null"`
	Kind           string     `json:"kind" gorm:"size:24;not null"`
	Value          int64      `json:"value" gorm:"not null"`
	MinimumFen     int64      `json:"minimumFen" gorm:"default:0"`
	MaxRedemptions int        `json:"maxRedemptions" gorm:"default:0"`
	RedeemedCount  int        `json:"redeemedCount" gorm:"default:0"`
	StartsAt       *time.Time `json:"startsAt,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	Active         bool       `json:"active" gorm:"index;default:true"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (Coupon) TableName() string { return "commercial_coupons" }

// CouponRedemption reserves a limited-use coupon when an order is created.
// A reservation becomes consumed only after payment succeeds, or released
// after cancellation/the payment reconciliation window. OrderID is unique so
// repeated payment callbacks cannot increment a coupon twice.
type CouponRedemption struct {
	ID         string    `json:"id" gorm:"primaryKey;size:36"`
	CouponID   string    `json:"couponId" gorm:"size:36;index;not null"`
	OrderID    string    `json:"orderId" gorm:"size:36;uniqueIndex;not null"`
	CustomerID string    `json:"customerId" gorm:"size:36;index;not null"`
	Status     string    `json:"status" gorm:"size:24;index;not null"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (CouponRedemption) TableName() string { return "commercial_coupon_redemptions" }

type GiftCard struct {
	ID          string     `json:"id" gorm:"primaryKey;size:36"`
	CodeHash    string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	DisplayCode string     `json:"displayCode" gorm:"size:24"`
	ValueFen    int64      `json:"valueFen" gorm:"not null"`
	Status      string     `json:"status" gorm:"size:24;index;not null"`
	RedeemedBy  *string    `json:"redeemedBy,omitempty" gorm:"size:36;index"`
	RedeemedAt  *time.Time `json:"redeemedAt,omitempty"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (GiftCard) TableName() string { return "commercial_gift_cards" }

type InvitationCommission struct {
	ID           string     `json:"id" gorm:"primaryKey;size:36"`
	InviterID    string     `json:"inviterId" gorm:"size:36;index;not null"`
	InviteeID    string     `json:"inviteeId" gorm:"size:36;index;not null"`
	OrderID      string     `json:"orderId" gorm:"size:36;uniqueIndex;not null"`
	AmountFen    int64      `json:"amountFen" gorm:"not null"`
	Distribution string     `json:"-" gorm:"type:text"`
	Status       string     `json:"status" gorm:"size:24;index;not null"`
	SettledAt    *time.Time `json:"settledAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func (InvitationCommission) TableName() string { return "commercial_invitation_commissions" }

type CommercialSetting struct {
	Key       string    `json:"key" gorm:"primaryKey;size:120"`
	Value     string    `json:"value" gorm:"type:text"`
	Encrypted bool      `json:"encrypted" gorm:"default:false"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (CommercialSetting) TableName() string { return "commercial_settings" }

type AdminRoleBinding struct {
	UserID    int       `json:"userId" gorm:"primaryKey"`
	Role      string    `json:"role" gorm:"size:32;index;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AdminRoleBinding) TableName() string { return "commercial_admin_role_bindings" }

type CommercialAuditLog struct {
	ID            string    `json:"id" gorm:"primaryKey;size:36"`
	ActorUserID   int       `json:"actorUserId" gorm:"index"`
	ActorRole     string    `json:"actorRole" gorm:"size:32;index"`
	Action        string    `json:"action" gorm:"size:96;index;not null"`
	TargetType    string    `json:"targetType" gorm:"size:64;index"`
	TargetID      string    `json:"targetId" gorm:"size:64;index"`
	Metadata      string    `json:"metadata" gorm:"type:text"`
	IPHash        string    `json:"ipHash" gorm:"size:64"`
	CorrelationID string    `json:"correlationId" gorm:"size:64;index"`
	CreatedAt     time.Time `json:"createdAt" gorm:"index"`
}

func (CommercialAuditLog) TableName() string { return "commercial_audit_logs" }
