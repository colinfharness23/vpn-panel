package commercial

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LocalizedContent struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Category string `json:"category,omitempty"`
	Level    string `json:"level,omitempty"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

type GuestBootstrap struct {
	Site           map[string]string         `json:"site"`
	Plans          []PlanCatalogItem         `json:"plans"`
	PaymentMethods []PaymentMethod           `json:"paymentMethods"`
	Notices        []LocalizedContent        `json:"notices"`
	Articles       []LocalizedContent        `json:"articles"`
	Applications   []model.ClientApplication `json:"applications"`
}

type SubscriptionLinks struct {
	Raw   string `json:"raw"`
	Clash string `json:"clash"`
	JSON  string `json:"json"`
}

type SubscriptionOverview struct {
	Entitlement model.SubscriptionEntitlement `json:"entitlement"`
	Plan        model.Plan                    `json:"plan"`
	UsedBytes   int64                         `json:"usedBytes"`
	Links       SubscriptionLinks             `json:"links"`
}

// InvitationOverview contains only the signed-in customer's own referral
// totals and the public parts of the invitation policy. Administrative
// configuration and other customers' identities are intentionally excluded.
type InvitationOverview struct {
	Enabled                    bool   `json:"enabled"`
	InviteCode                 string `json:"inviteCode"`
	DirectInviteCount          int64  `json:"directInviteCount"`
	CommissionPercent          int    `json:"commissionPercent"`
	PendingFen                 int64  `json:"pendingFen"`
	ConfirmedFen               int64  `json:"confirmedFen"`
	SettledFen                 int64  `json:"settledFen"`
	CommissionFirstPaymentOnly bool   `json:"commissionFirstPaymentOnly"`
	InviteCodesNeverExpire     bool   `json:"inviteCodesNeverExpire"`
}

type Dashboard struct {
	Customer     model.Customer        `json:"customer"`
	Subscription *SubscriptionOverview `json:"subscription,omitempty"`
	Invitation   InvitationOverview    `json:"invitation"`
	Notices      []LocalizedContent    `json:"notices"`
	Orders       []model.Order         `json:"orders"`
}

type PortalService struct {
	db       *gorm.DB
	config   *ConfigStore
	settings service.SettingService
	clients  service.ClientService
	inbounds service.InboundService
}

func NewPortalService() *PortalService {
	return &PortalService{db: database.GetDB(), config: NewConfigStore()}
}

func (s *PortalService) Bootstrap(locale string) (*GuestBootstrap, error) {
	plans, err := NewOrderService().Catalog(true)
	if err != nil {
		return nil, err
	}
	notices, err := s.notices(locale)
	if err != nil {
		return nil, err
	}
	articles, err := s.articles(locale)
	if err != nil {
		return nil, err
	}
	var applications []model.ClientApplication
	if err := s.db.Where("active = ?", true).Order("sort_order asc").Find(&applications).Error; err != nil {
		return nil, err
	}
	return &GuestBootstrap{Site: s.config.Public(), Plans: plans, PaymentMethods: EnabledPaymentMethods(s.config), Notices: notices, Articles: articles, Applications: applications}, nil
}

func (s *PortalService) Dashboard(customer *model.Customer, requestOrigin string) (*Dashboard, error) {
	notices, err := s.notices(customer.Locale)
	if err != nil {
		return nil, err
	}
	orders, err := NewOrderService().List(customer.ID)
	if err != nil {
		return nil, err
	}
	invitation, err := s.invitationOverview(customer)
	if err != nil {
		return nil, err
	}
	result := &Dashboard{Customer: *customer, Invitation: invitation, Notices: notices, Orders: orders}
	var entitlement model.SubscriptionEntitlement
	err = s.db.Where("customer_id = ? AND status = ?", customer.ID, "active").Order("created_at desc").First(&entitlement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	var plan model.Plan
	if err := s.db.Where("id = ?", entitlement.PlanID).First(&plan).Error; err != nil {
		return nil, err
	}
	var traffic xray.ClientTraffic
	used := int64(0)
	if err := s.db.Where("email = ?", entitlement.InternalClientID).First(&traffic).Error; err == nil {
		used = traffic.Up + traffic.Down
	}
	var global struct {
		Up   int64
		Down int64
	}
	if err := s.db.Model(&model.ClientGlobalTraffic{}).Select("MAX(up) AS up, MAX(down) AS down").Where("email = ?", entitlement.InternalClientID).Scan(&global).Error; err == nil && global.Up+global.Down > used {
		used = global.Up + global.Down
	}
	result.Subscription = &SubscriptionOverview{Entitlement: entitlement, Plan: plan, UsedBytes: used, Links: s.subscriptionLinks(entitlement.SubscriptionID, requestOrigin)}
	return result, nil
}

func (s *PortalService) invitationOverview(customer *model.Customer) (InvitationOverview, error) {
	policy := s.config.InvitationPolicy()
	result := InvitationOverview{
		Enabled:                    policy.CommissionPercent > 0,
		InviteCode:                 customer.InviteCode,
		CommissionPercent:          policy.CommissionPercent,
		CommissionFirstPaymentOnly: policy.CommissionFirstPaymentOnly,
		InviteCodesNeverExpire:     policy.InviteCodesNeverExpire,
	}
	if err := s.db.Model(&model.Customer{}).Where("invited_by_id = ?", customer.ID).Count(&result.DirectInviteCount).Error; err != nil {
		return result, err
	}

	var rows []model.InvitationCommission
	distributionMatch := `%"customerId":"` + customer.ID + `"%`
	if err := s.db.Where("inviter_id = ? OR distribution LIKE ?", customer.ID, distributionMatch).Order("created_at desc").Find(&rows).Error; err != nil {
		return result, err
	}
	for _, row := range rows {
		for _, share := range commissionShares(&row) {
			if share.CustomerID != customer.ID {
				continue
			}
			switch row.Status {
			case "pending":
				result.PendingFen += share.AmountFen
			case "confirmed":
				result.ConfirmedFen += share.AmountFen
			case "settled":
				result.SettledFen += share.AmountFen
			}
		}
	}
	return result, nil
}

func (s *PortalService) RotateSubscription(customerID, requestOrigin string) (*SubscriptionLinks, error) {
	var entitlement model.SubscriptionEntitlement
	if err := s.db.Where("customer_id = ? AND status = ?", customerID, "active").Order("created_at desc").First(&entitlement).Error; err != nil {
		return nil, errors.New("当前没有可用订阅")
	}
	var record model.ClientRecord
	if err := s.db.Where("email = ?", entitlement.InternalClientID).First(&record).Error; err != nil {
		return nil, err
	}
	var inboundIDs []int
	if err := s.db.Model(&model.ClientInbound{}).Where("client_id = ?", record.Id).Pluck("inbound_id", &inboundIDs).Error; err != nil {
		return nil, err
	}
	newID := uuid.NewString()
	client := record.ToClient()
	client.SubID = newID
	if _, err := s.clients.UpdateByEmail(&s.inbounds, record.Email, *client, inboundIDs...); err != nil {
		return nil, err
	}
	if err := s.db.Model(&entitlement).Update("subscription_id", newID).Error; err != nil {
		return nil, err
	}
	links := s.subscriptionLinks(newID, requestOrigin)
	return &links, nil
}

func (s *PortalService) CreateTicket(customerID, subject, body string) (*model.Ticket, error) {
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if subject == "" || len(subject) > 200 || body == "" || len(body) > 10000 {
		return nil, errors.New("请填写有效的工单主题和内容")
	}
	ticket := &model.Ticket{ID: uuid.NewString(), CustomerID: customerID, Subject: subject, Status: "open", Priority: "normal"}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		message := model.TicketMessage{ID: uuid.NewString(), TicketID: ticket.ID, SenderType: "customer", SenderID: customerID, Body: body}
		return tx.Create(&message).Error
	})
	return ticket, err
}

func (s *PortalService) Tickets(customerID string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	err := s.db.Where("customer_id = ?", customerID).Order("updated_at desc").Find(&tickets).Error
	return tickets, err
}

func (s *PortalService) TicketMessages(customerID, ticketID string) ([]model.TicketMessage, error) {
	var ticket model.Ticket
	if err := s.db.Where("id = ? AND customer_id = ?", ticketID, customerID).First(&ticket).Error; err != nil {
		return nil, errors.New("工单不存在")
	}
	var messages []model.TicketMessage
	err := s.db.Where("ticket_id = ?", ticketID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (s *PortalService) ReplyTicket(customerID, ticketID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 10000 {
		return errors.New("回复内容无效")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var ticket model.Ticket
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND customer_id = ?", ticketID, customerID).First(&ticket).Error; err != nil {
			return errors.New("工单不存在")
		}
		if ticket.Status == "closed" {
			return errors.New("已关闭的工单不能继续回复")
		}
		message := model.TicketMessage{ID: uuid.NewString(), TicketID: ticket.ID, SenderType: "customer", SenderID: customerID, Body: body}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		return tx.Model(&ticket).Updates(map[string]any{"status": "open", "updated_at": time.Now().UTC()}).Error
	})
}

func (s *PortalService) RedeemGiftCard(customerID, code string) (int64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, errors.New("请输入礼品卡兑换码")
	}
	var value int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var card model.GiftCard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code_hash = ?", hashToken(code)).First(&card).Error; err != nil {
			return errors.New("礼品卡无效")
		}
		now := time.Now().UTC()
		if card.Status != "active" || card.ExpiresAt != nil && now.After(*card.ExpiresAt) {
			return errors.New("礼品卡已使用或已过期")
		}
		if err := tx.Model(&model.Customer{}).Where("id = ?", customerID).UpdateColumn("balance_fen", gorm.Expr("balance_fen + ?", card.ValueFen)).Error; err != nil {
			return err
		}
		if err := tx.Model(&card).Updates(map[string]any{"status": "redeemed", "redeemed_by": customerID, "redeemed_at": now}).Error; err != nil {
			return err
		}
		value = card.ValueFen
		return nil
	})
	return value, err
}

func (s *PortalService) notices(locale string) ([]LocalizedContent, error) {
	var rows []model.Notice
	if err := s.db.Where("published = ?", true).Order("published_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]LocalizedContent, 0, len(rows))
	for _, row := range rows {
		items = append(items, LocalizedContent{ID: row.ID, Slug: row.Slug, Level: row.Level, Title: localizedValue(row.TitleI18n, locale), Content: localizedValue(row.ContentI18n, locale)})
	}
	return items, nil
}

func (s *PortalService) articles(locale string) ([]LocalizedContent, error) {
	var rows []model.KnowledgeArticle
	if err := s.db.Where("published = ?", true).Order("sort_order asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]LocalizedContent, 0, len(rows))
	for _, row := range rows {
		items = append(items, LocalizedContent{ID: row.ID, Slug: row.Slug, Category: row.Category, Title: localizedValue(row.TitleI18n, locale), Content: localizedValue(row.ContentI18n, locale)})
	}
	return items, nil
}

func (s *PortalService) subscriptionLinks(subscriptionID, requestOrigin string) SubscriptionLinks {
	base := s.subscriptionBase(subscriptionID, requestOrigin)
	rawURI, _ := s.settings.GetSubURI()
	if strings.TrimSpace(rawURI) == "" {
		rawURI = base
	}
	jsonURI, _ := s.settings.GetSubJsonURI()
	if strings.TrimSpace(jsonURI) == "" {
		jsonURI = base
	}
	clashURI, _ := s.settings.GetSubClashURI()
	if strings.TrimSpace(clashURI) == "" {
		clashURI = base
	}
	rawPath, _ := s.settings.GetSubPath()
	jsonPath, _ := s.settings.GetSubJsonPath()
	clashPath, _ := s.settings.GetSubClashPath()
	return SubscriptionLinks{Raw: joinSubscriptionURL(rawURI, rawPath, subscriptionID), JSON: joinSubscriptionURL(jsonURI, jsonPath, subscriptionID), Clash: joinSubscriptionURL(clashURI, clashPath, subscriptionID)}
}

func (s *PortalService) subscriptionBase(subscriptionID, requestOrigin string) string {
	raw, err := s.config.Get("subscription.base_url")
	if err != nil || strings.TrimSpace(raw) == "" {
		raw = s.config.GetDefault("site.url", requestOrigin)
	}
	candidates := strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == '\n' || r == '\r' })
	clean := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if value := strings.TrimRight(strings.TrimSpace(candidate), "/"); value != "" {
			clean = append(clean, value)
		}
	}
	if len(clean) == 0 {
		return strings.TrimRight(requestOrigin, "/")
	}
	if len(clean) == 1 {
		return clean[0]
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(subscriptionID))
	return clean[int(hasher.Sum32())%len(clean)]
}

func localizedValue(raw, locale string) string {
	values := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return raw
	}
	for _, key := range []string{locale, "en-US", "zh-CN"} {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func joinSubscriptionURL(base, path, id string) string {
	base = strings.TrimRight(base, "/")
	path = "/" + strings.Trim(path, "/") + "/"
	return base + path + id
}
