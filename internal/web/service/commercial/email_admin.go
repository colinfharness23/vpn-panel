package commercial

import (
	"encoding/json"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
)

type CustomerEmailQueueResult struct {
	CampaignID string `json:"campaignId"`
	Queued     int    `json:"queued"`
}

type customerEmailPayload struct {
	CampaignID  string `json:"campaignId"`
	CustomerID  string `json:"customerId"`
	Recipient   string `json:"recipient"`
	TemplateKey string `json:"templateKey,omitempty"`
	Subject     string `json:"subject"`
	BodyHTML    string `json:"bodyHtml"`
}

var defaultCustomerEmailTemplates = []model.EmailTemplate{
	{
		Key: "announcement", Name: "运营公告", SortOrder: 10, Active: true, System: true,
		Subject:  "[{{site_name}}] 服务公告",
		BodyHTML: `<p>{{display_name}}，您好：</p><p>这里填写需要通知用户的公告内容。</p><p>感谢您使用 {{site_name}}。</p>`,
	},
	{
		Key: "subscription_activated", Name: "订阅开通成功", SortOrder: 20, Active: true, System: true,
		Subject:  "[{{site_name}}] 您的订阅已开通",
		BodyHTML: `<p>{{display_name}}，您好：</p><p>您的订阅已经成功开通，请登录网站查看订阅链接和使用教程。</p><p>{{site_name}}</p>`,
	},
	{
		Key: "subscription_expiring", Name: "订阅即将到期", SortOrder: 30, Active: true, System: true,
		Subject:  "[{{site_name}}] 订阅即将到期提醒",
		BodyHTML: `<p>{{display_name}}，您好：</p><p>您的订阅即将到期，请及时登录网站续费，以免影响使用。</p><p>{{site_name}}</p>`,
	},
	{
		Key: "traffic_warning", Name: "流量不足提醒", SortOrder: 40, Active: true, System: true,
		Subject:  "[{{site_name}}] 流量不足提醒",
		BodyHTML: `<p>{{display_name}}，您好：</p><p>您的订阅流量即将用尽，请登录网站查看用量或购买新的套餐。</p><p>{{site_name}}</p>`,
	},
	{
		Key: "ticket_reply", Name: "工单回复通知", SortOrder: 50, Active: true, System: true,
		Subject:  "[{{site_name}}] 您的工单有新回复",
		BodyHTML: `<p>{{display_name}}，您好：</p><p>您的工单已有新的客服回复，请登录网站查看。</p><p>{{site_name}}</p>`,
	},
}

func (s *AdminService) ensureCustomerEmailTemplates() error {
	rows := make([]model.EmailTemplate, len(defaultCustomerEmailTemplates))
	copy(rows, defaultCustomerEmailTemplates)
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (s *AdminService) EmailTemplates() ([]model.EmailTemplate, error) {
	if err := s.ensureCustomerEmailTemplates(); err != nil {
		return nil, err
	}
	var rows []model.EmailTemplate
	if err := s.db.Order("sort_order asc, key asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *AdminService) SaveEmailTemplate(key string, request entity.CommercialEmailTemplateRequest) (*model.EmailTemplate, error) {
	key = strings.TrimSpace(key)
	request.Name = strings.TrimSpace(request.Name)
	request.Subject = strings.TrimSpace(request.Subject)
	request.BodyHTML = strings.TrimSpace(request.BodyHTML)
	if key == "" || request.Name == "" || request.Subject == "" || request.BodyHTML == "" {
		return nil, errors.New("模板名称、主题和正文不能为空")
	}
	if strings.ContainsAny(request.Subject, "\r\n") {
		return nil, errors.New("邮件主题不能包含换行符")
	}
	if err := s.ensureCustomerEmailTemplates(); err != nil {
		return nil, err
	}
	result := s.db.Model(&model.EmailTemplate{}).Where("key = ?", key).Updates(map[string]any{
		"name": request.Name, "subject": request.Subject, "body_html": request.BodyHTML, "active": request.Active,
	})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row model.EmailTemplate
	if err := s.db.Where("key = ?", key).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AdminService) QueueCustomerEmail(request entity.CommercialEmailSendRequest) (*CustomerEmailQueueResult, error) {
	request.Subject = strings.TrimSpace(request.Subject)
	request.BodyHTML = strings.TrimSpace(request.BodyHTML)
	request.TemplateKey = strings.TrimSpace(request.TemplateKey)
	if request.Subject == "" || request.BodyHTML == "" {
		return nil, errors.New("邮件主题和正文不能为空")
	}
	if strings.ContainsAny(request.Subject, "\r\n") {
		return nil, errors.New("邮件主题不能包含换行符")
	}

	query := s.db.Model(&model.Customer{}).Where("email <> ''")
	switch request.Audience {
	case "selected":
		if len(request.CustomerIDs) == 0 {
			return nil, errors.New("请选择至少一个用户")
		}
		query = query.Where("id IN ?", request.CustomerIDs)
	case "active":
		query = query.Where("status = ?", "active")
	case "subscribed":
		now := time.Now().UTC()
		query = query.Where("status = ?", "active").Where(
			"EXISTS (SELECT 1 FROM commercial_subscription_entitlements AS entitlements WHERE entitlements.customer_id = commercial_customers.id AND entitlements.status = ? AND (entitlements.expires_at IS NULL OR entitlements.expires_at > ?))",
			"active", now,
		)
	default:
		return nil, errors.New("收件用户范围无效")
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("当前范围内没有可发送邮件的用户")
	}
	if count > 5000 {
		return nil, errors.New("单次最多向 5000 个用户加入发送队列")
	}
	var customers []model.Customer
	if err := query.Order("commercial_customers.created_at asc").Limit(5000).Find(&customers).Error; err != nil {
		return nil, err
	}

	campaignID := uuid.NewString()
	siteName := s.config.GetDefault("site.name", "NOVA")
	now := time.Now().UTC()
	events := make([]model.OutboxEvent, 0, len(customers))
	for _, customer := range customers {
		payload := customerEmailPayload{
			CampaignID:  campaignID,
			CustomerID:  customer.ID,
			Recipient:   customer.Email,
			TemplateKey: request.TemplateKey,
			Subject:     renderCustomerEmail(request.Subject, customer, siteName, false),
			BodyHTML:    renderCustomerEmail(request.BodyHTML, customer, siteName, true),
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		events = append(events, model.OutboxEvent{
			ID: uuid.NewString(), AggregateType: "email_campaign", AggregateID: campaignID,
			EventType: "email.customer", Payload: string(encoded), Status: "pending", NextRunAt: now, CreatedAt: now,
		})
	}
	if err := s.db.CreateInBatches(events, 100).Error; err != nil {
		return nil, err
	}
	return &CustomerEmailQueueResult{CampaignID: campaignID, Queued: len(events)}, nil
}

func renderCustomerEmail(content string, customer model.Customer, siteName string, escapeValues bool) string {
	displayName := strings.TrimSpace(customer.DisplayName)
	if displayName == "" {
		displayName = strings.Split(customer.Email, "@")[0]
	}
	emailAddress := customer.Email
	if escapeValues {
		displayName = html.EscapeString(displayName)
		emailAddress = html.EscapeString(emailAddress)
		siteName = html.EscapeString(siteName)
	}
	return strings.NewReplacer(
		"{{display_name}}", displayName,
		"{{email}}", emailAddress,
		"{{site_name}}", siteName,
	).Replace(content)
}
