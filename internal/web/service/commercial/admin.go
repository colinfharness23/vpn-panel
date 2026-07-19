package commercial

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	utilcrypto "github.com/mhsanaei/3x-ui/v3/internal/util/crypto"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminOverview struct {
	Customers          int64            `json:"customers"`
	ActiveEntitlements int64            `json:"activeEntitlements"`
	PendingOrders      int64            `json:"pendingOrders"`
	OpenTickets        int64            `json:"openTickets"`
	ManualJobs         int64            `json:"manualJobs"`
	RevenueFen         int64            `json:"revenueFen"`
	OrderStatus        map[string]int64 `json:"orderStatus"`
}

type PaginatedCustomers struct {
	Items []AdminCustomerRow `json:"items"`
	Total int64              `json:"total"`
}

type AdminSubscriptionSummary struct {
	Entitlement model.SubscriptionEntitlement `json:"entitlement"`
	Plan        model.Plan                    `json:"plan"`
}

type AdminCustomerRow struct {
	model.Customer
	Subscription *AdminSubscriptionSummary `json:"subscription,omitempty"`
	SystemAdmin  bool                      `json:"systemAdmin"`
}

type PaginatedOrders struct {
	Items []model.Order `json:"items"`
	Total int64         `json:"total"`
}

type AdminUserRole struct {
	UserID   int    `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

var commercialSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,78}[a-z0-9]$`)
var commercialCurrencyPattern = regexp.MustCompile(`^[A-Z]{3,8}$`)

type AdminService struct {
	db       *gorm.DB
	config   *ConfigStore
	clients  service.ClientService
	inbounds service.InboundService
	xray     service.XrayService
}

func NewAdminService() *AdminService {
	return &AdminService{db: database.GetDB(), config: NewConfigStore()}
}

func (s *AdminService) Overview() (*AdminOverview, error) {
	result := &AdminOverview{OrderStatus: map[string]int64{}}
	queries := []struct {
		model any
		where string
		args  []any
		out   *int64
	}{
		{&model.Customer{}, "", nil, &result.Customers},
		{&model.SubscriptionEntitlement{}, "status = ?", []any{"active"}, &result.ActiveEntitlements},
		{&model.Order{}, "status = ?", []any{OrderPending}, &result.PendingOrders},
		{&model.Ticket{}, "status IN ?", []any{[]string{"open", "pending"}}, &result.OpenTickets},
		{&model.ProvisioningJob{}, "status = ?", []any{"manual"}, &result.ManualJobs},
	}
	for _, query := range queries {
		db := s.db.Model(query.model)
		if query.where != "" {
			db = db.Where(query.where, query.args...)
		}
		if err := db.Count(query.out).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Model(&model.Order{}).Select("COALESCE(SUM(paid_fen), 0)").Where("status IN ?", []string{OrderPaid, OrderProvisioning, OrderCompleted}).Scan(&result.RevenueFen).Error; err != nil {
		return nil, err
	}
	var statuses []struct {
		Status string
		Count  int64
	}
	if err := s.db.Model(&model.Order{}).Select("status, COUNT(*) AS count").Group("status").Scan(&statuses).Error; err != nil {
		return nil, err
	}
	for _, row := range statuses {
		result.OrderStatus[row.Status] = row.Count
	}
	return result, nil
}

func (s *AdminService) Customers(search, status string, page, pageSize int) (*PaginatedCustomers, error) {
	page, pageSize = normalizePage(page, pageSize)
	query := s.db.Model(&model.Customer{})
	if search != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		query = query.Where("LOWER(email) LIKE ? OR LOWER(display_name) LIKE ? OR LOWER(invite_code) LIKE ?", pattern, pattern, pattern)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	result := &PaginatedCustomers{}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	var customers []model.Customer
	if err := query.Order("created_at asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&customers).Error; err != nil {
		return nil, err
	}
	result.Items = make([]AdminCustomerRow, 0, len(customers))
	if len(customers) == 0 {
		return result, nil
	}
	customerIDs := make([]string, 0, len(customers))
	for _, customer := range customers {
		customerIDs = append(customerIDs, customer.ID)
	}
	var entitlements []model.SubscriptionEntitlement
	if err := s.db.Where("customer_id IN ? AND status = ?", customerIDs, "active").Order("created_at desc").Find(&entitlements).Error; err != nil {
		return nil, err
	}
	entitlementByCustomer := make(map[string]model.SubscriptionEntitlement, len(entitlements))
	planIDs := make([]string, 0, len(entitlements))
	for _, entitlement := range entitlements {
		if _, exists := entitlementByCustomer[entitlement.CustomerID]; exists {
			continue
		}
		entitlementByCustomer[entitlement.CustomerID] = entitlement
		planIDs = append(planIDs, entitlement.PlanID)
	}
	var plans []model.Plan
	if len(planIDs) > 0 {
		if err := s.db.Where("id IN ?", planIDs).Find(&plans).Error; err != nil {
			return nil, err
		}
	}
	planByID := make(map[string]model.Plan, len(plans))
	for _, plan := range plans {
		planByID[plan.ID] = plan
	}
	for _, customer := range customers {
		row := AdminCustomerRow{Customer: customer, SystemAdmin: customer.AdminUserID != nil}
		if entitlement, exists := entitlementByCustomer[customer.ID]; exists {
			row.Subscription = &AdminSubscriptionSummary{Entitlement: entitlement, Plan: planByID[entitlement.PlanID]}
		}
		result.Items = append(result.Items, row)
	}
	return result, nil
}

func (s *AdminService) CreateCustomer(request entity.CommercialCustomerCreateRequest) (*model.Customer, error) {
	policy := s.config.SecurityPolicy()
	// Alias blocking protects public self-registration from disposable Gmail
	// variants. A trusted administrator may create an exact address manually,
	// while the configured domain whitelist still remains authoritative.
	policy.DisallowGmailAliases = false
	email, err := normalizeConfiguredEmail(request.Email, policy)
	if err != nil {
		return nil, err
	}
	if err := ValidatePassword(request.Password); err != nil {
		return nil, err
	}
	hash, err := utilcrypto.HashPasswordAsBcrypt(request.Password)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "active"
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}
	locale := strings.TrimSpace(request.Locale)
	if locale == "" {
		locale = "zh-CN"
	}
	now := time.Now().UTC()
	customer := &model.Customer{ID: uuid.NewString(), Email: email, PasswordHash: hash, DisplayName: displayName, Locale: locale, Status: status, InviteCode: strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", "")), EmailVerifiedAt: &now}
	if err := s.db.Create(customer).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("该邮箱已存在")
		}
		return nil, err
	}
	return customer, nil
}

func (s *AdminService) UpdateCustomer(id, status string, balanceFen *int64) error {
	updates := map[string]any{}
	if status != "" {
		if status != "active" && status != "suspended" && status != "closed" {
			return errors.New("客户状态无效")
		}
		updates["status"] = status
	}
	if balanceFen != nil {
		if *balanceFen < 0 {
			return errors.New("客户余额不能为负数")
		}
		updates["balance_fen"] = *balanceFen
	}
	if len(updates) == 0 {
		return errors.New("没有可更新的字段")
	}
	return s.db.Model(&model.Customer{}).Where("id = ?", id).Updates(updates).Error
}

func (s *AdminService) UpsertSubscription(customerID string, request entity.CommercialSubscriptionUpdateRequest) (*AdminSubscriptionSummary, error) {
	var customer model.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		return nil, errors.New("客户不存在")
	}
	var plan model.Plan
	if err := s.db.Where("id = ?", request.PlanID).First(&plan).Error; err != nil {
		return nil, errors.New("套餐不存在")
	}
	inboundIDs, err := NewWorker().provisionInboundIDs(&plan)
	if err != nil || len(inboundIDs) == 0 {
		return nil, errors.New("套餐尚未绑定可用入站")
	}
	var expiresAt *time.Time
	if strings.TrimSpace(request.ExpiresAt) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, request.ExpiresAt)
		if parseErr != nil {
			return nil, errors.New("到期时间格式无效")
		}
		parsed = parsed.UTC()
		if !parsed.After(time.Now().UTC()) {
			return nil, errors.New("到期时间必须晚于当前时间")
		}
		expiresAt = &parsed
	}
	trafficQuota := request.TrafficQuota
	if trafficQuota == 0 {
		trafficQuota = plan.TrafficBytes
	}
	deviceLimit := request.DeviceLimit
	if deviceLimit == 0 {
		deviceLimit = plan.DeviceLimit
	}
	var entitlement model.SubscriptionEntitlement
	lookupErr := s.db.Where("customer_id = ? AND status = ?", customerID, "active").Order("created_at desc").First(&entitlement).Error
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	creating := errors.Is(lookupErr, gorm.ErrRecordNotFound)
	if creating {
		grantID := "admin-" + uuid.NewString()
		entitlement = model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: customerID, PlanID: plan.ID, OrderID: grantID, InternalClientID: internalClientID(customerID, grantID), SubscriptionID: uuid.NewString(), Status: "active", StartsAt: time.Now().UTC()}
	}
	entitlement.PlanID = plan.ID
	entitlement.TrafficQuota = trafficQuota
	entitlement.DeviceLimit = deviceLimit
	entitlement.NodeGroup = plan.NodeGroup
	entitlement.ExpiresAt = expiresAt
	if err := s.applyAdminSubscriptionClient(&entitlement, &plan, inboundIDs, request.ResetTraffic); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if creating {
		if err := s.db.Create(&entitlement).Error; err != nil {
			needRestart, _ := s.clients.DeleteByEmail(&s.inbounds, entitlement.InternalClientID, false)
			if needRestart {
				s.xray.SetToNeedRestart()
			}
			return nil, err
		}
	} else {
		updates := map[string]any{"plan_id": plan.ID, "traffic_quota": trafficQuota, "device_limit": deviceLimit, "node_group": plan.NodeGroup, "expires_at": expiresAt}
		if request.ResetTraffic {
			updates["traffic_used"] = 0
			updates["last_reset_at"] = now
			entitlement.TrafficUsed = 0
			entitlement.LastResetAt = &now
		}
		if err := s.db.Model(&model.SubscriptionEntitlement{}).Where("id = ? AND customer_id = ?", entitlement.ID, customerID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.First(&entitlement, "id = ?", entitlement.ID).Error; err != nil {
		return nil, err
	}
	return &AdminSubscriptionSummary{Entitlement: entitlement, Plan: plan}, nil
}

func (s *AdminService) applyAdminSubscriptionClient(entitlement *model.SubscriptionEntitlement, plan *model.Plan, inboundIDs []int, resetTraffic bool) error {
	expiryMillis := int64(0)
	if entitlement.ExpiresAt != nil {
		expiryMillis = entitlement.ExpiresAt.UnixMilli()
	}
	var record model.ClientRecord
	lookupErr := s.db.Where("email = ?", entitlement.InternalClientID).First(&record).Error
	if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		client := model.Client{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("admin-client:"+entitlement.ID)).String(), Email: entitlement.InternalClientID, SubID: entitlement.SubscriptionID, Enable: true, TotalGB: entitlement.TrafficQuota, ExpiryTime: expiryMillis, LimitIP: entitlement.DeviceLimit, Group: plan.NodeGroup, Reset: resetDaysForPolicy(plan.ResetCycle, s.config.SubscriptionPolicy().MonthlyResetMode), Comment: "commercial:admin:" + entitlement.CustomerID}
		needRestart, err := s.clients.Create(&s.inbounds, &service.ClientCreatePayload{Client: client, InboundIds: inboundIDs})
		if needRestart {
			s.xray.SetToNeedRestart()
		}
		if err != nil {
			return err
		}
	} else if lookupErr != nil {
		return lookupErr
	} else {
		currentInboundIDs, err := s.clients.GetInboundIdsForEmail(s.db, entitlement.InternalClientID)
		if err != nil {
			return err
		}
		client := record.ToClient()
		client.Email = entitlement.InternalClientID
		client.SubID = entitlement.SubscriptionID
		client.Enable = true
		client.TotalGB = entitlement.TrafficQuota
		client.ExpiryTime = expiryMillis
		client.LimitIP = entitlement.DeviceLimit
		client.Group = plan.NodeGroup
		client.Reset = resetDaysForPolicy(plan.ResetCycle, s.config.SubscriptionPolicy().MonthlyResetMode)
		needRestart, err := s.clients.UpdateByEmail(&s.inbounds, entitlement.InternalClientID, *client, currentInboundIDs...)
		if needRestart {
			s.xray.SetToNeedRestart()
		}
		if err != nil {
			return err
		}
		needRestart, err = s.clients.AttachByEmail(&s.inbounds, entitlement.InternalClientID, inboundIDs)
		if needRestart {
			s.xray.SetToNeedRestart()
		}
		if err != nil {
			return err
		}
		targetSet := make(map[int]struct{}, len(inboundIDs))
		for _, inboundID := range inboundIDs {
			targetSet[inboundID] = struct{}{}
		}
		obsolete := make([]int, 0)
		for _, inboundID := range currentInboundIDs {
			if _, keep := targetSet[inboundID]; !keep {
				obsolete = append(obsolete, inboundID)
			}
		}
		if len(obsolete) > 0 {
			needRestart, err = s.clients.DetachByEmailMany(&s.inbounds, entitlement.InternalClientID, obsolete)
			if needRestart {
				s.xray.SetToNeedRestart()
			}
			if err != nil {
				return err
			}
		}
	}
	if resetTraffic {
		needRestart, err := s.clients.ResetTrafficByEmail(&s.inbounds, entitlement.InternalClientID)
		if needRestart {
			s.xray.SetToNeedRestart()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *AdminService) DeleteSubscription(customerID string) error {
	var entitlement model.SubscriptionEntitlement
	if err := s.db.Where("customer_id = ? AND status = ?", customerID, "active").First(&entitlement).Error; err != nil {
		return errors.New("客户当前没有可删除的订阅")
	}
	needRestart, err := s.clients.DeleteByEmail(&s.inbounds, entitlement.InternalClientID, false)
	if needRestart {
		s.xray.SetToNeedRestart()
	}
	if err != nil && !isMissingCommercialClient(err) {
		return fmt.Errorf("清理 3X-UI 客户端失败: %w", err)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("aggregate_id = ?", entitlement.ID).Delete(&model.OutboxEvent{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND customer_id = ?", entitlement.ID, customerID).Delete(&model.SubscriptionEntitlement{}).Error
	})
}

type CustomerDeleteResult struct {
	Deleted []string          `json:"deleted"`
	Failed  map[string]string `json:"failed"`
}

func (s *AdminService) DeleteCustomers(ids []string) (*CustomerDeleteResult, error) {
	if len(ids) == 0 || len(ids) > 500 {
		return nil, errors.New("每次必须选择 1 到 500 个客户")
	}
	result := &CustomerDeleteResult{Deleted: []string{}, Failed: map[string]string{}}
	seen := map[string]struct{}{}
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if err := s.deleteCustomer(id); err != nil {
			result.Failed[id] = err.Error()
			continue
		}
		result.Deleted = append(result.Deleted, id)
	}
	return result, nil
}

func (s *AdminService) deleteCustomer(customerID string) error {
	var customer model.Customer
	if err := s.db.Where("id = ?", customerID).First(&customer).Error; err != nil {
		return errors.New("客户不存在")
	}
	var entitlements []model.SubscriptionEntitlement
	if err := s.db.Where("customer_id = ?", customerID).Find(&entitlements).Error; err != nil {
		return err
	}
	for _, entitlement := range entitlements {
		needRestart, err := s.clients.DeleteByEmail(&s.inbounds, entitlement.InternalClientID, false)
		if needRestart {
			s.xray.SetToNeedRestart()
		}
		if err != nil && !isMissingCommercialClient(err) {
			return fmt.Errorf("清理 3X-UI 客户端失败: %w", err)
		}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if customer.AdminUserID != nil {
			marker := model.CommercialSetting{
				Key:       database.AdminCustomerDeletionMarkerKey(*customer.AdminUserID),
				Value:     "true",
				Encrypted: false,
				UpdatedAt: time.Now().UTC(),
			}
			if err := tx.Save(&marker).Error; err != nil {
				return err
			}
		}
		var orders []model.Order
		if err := tx.Where("customer_id = ?", customerID).Find(&orders).Error; err != nil {
			return err
		}
		orderIDs := make([]string, 0, len(orders))
		for _, order := range orders {
			orderIDs = append(orderIDs, order.ID)
		}
		var tickets []model.Ticket
		if err := tx.Where("customer_id = ?", customerID).Find(&tickets).Error; err != nil {
			return err
		}
		ticketIDs := make([]string, 0, len(tickets))
		for _, ticket := range tickets {
			ticketIDs = append(ticketIDs, ticket.ID)
		}
		entitlementIDs := make([]string, 0, len(entitlements))
		for _, entitlement := range entitlements {
			entitlementIDs = append(entitlementIDs, entitlement.ID)
		}
		if len(ticketIDs) > 0 {
			if err := tx.Where("ticket_id IN ?", ticketIDs).Delete(&model.TicketMessage{}).Error; err != nil {
				return err
			}
		}
		if len(orderIDs) > 0 {
			for _, target := range []any{&model.PaymentTransaction{}, &model.ProvisioningJob{}, &model.CouponRedemption{}} {
				if err := tx.Where("order_id IN ?", orderIDs).Delete(target).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Where("customer_id = ?", customerID).Delete(&model.CustomerSession{}).Error; err != nil {
			return err
		}
		if err := tx.Where("email = ?", customer.Email).Delete(&model.EmailVerification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("customer_id = ?", customerID).Delete(&model.CouponRedemption{}).Error; err != nil {
			return err
		}
		distributionMatch := "%\"customerId\":\"" + customerID + "\"%"
		if err := tx.Where("inviter_id = ? OR invitee_id = ? OR distribution LIKE ?", customerID, customerID, distributionMatch).Delete(&model.InvitationCommission{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GiftCard{}).Where("redeemed_by = ?", customerID).Update("redeemed_by", nil).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Customer{}).Where("invited_by_id = ?", customerID).Update("invited_by_id", nil).Error; err != nil {
			return err
		}
		aggregateIDs := append(append([]string{}, orderIDs...), entitlementIDs...)
		if len(aggregateIDs) > 0 {
			if err := tx.Where("aggregate_id IN ?", aggregateIDs).Delete(&model.OutboxEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("target_id IN ?", aggregateIDs).Delete(&model.CommercialAuditLog{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("target_id = ?", customerID).Delete(&model.CommercialAuditLog{}).Error; err != nil {
			return err
		}
		for _, deletion := range []struct {
			query string
			args  []any
			model any
		}{
			{"customer_id = ?", []any{customerID}, &model.Ticket{}},
			{"customer_id = ?", []any{customerID}, &model.SubscriptionEntitlement{}},
			{"customer_id = ?", []any{customerID}, &model.ProvisioningJob{}},
			{"customer_id = ?", []any{customerID}, &model.Order{}},
		} {
			if err := tx.Where(deletion.query, deletion.args...).Delete(deletion.model).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ?", customerID).Delete(&model.Customer{}).Error
	})
}

func isMissingCommercialClient(err error) bool {
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") || strings.Contains(message, "不存在")
}

func (s *AdminService) Orders(search, status string, page, pageSize int) (*PaginatedOrders, error) {
	page, pageSize = normalizePage(page, pageSize)
	query := s.db.Model(&model.Order{})
	if search != "" {
		pattern := "%" + strings.TrimSpace(search) + "%"
		query = query.Where("out_trade_no LIKE ? OR customer_id LIKE ?", pattern, pattern)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	result := &PaginatedOrders{}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&result.Items).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *AdminService) RetryProvisioning(orderID string) error {
	now := time.Now().UTC()
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Order{}).Where("id = ? AND status IN ?", orderID, []string{OrderPaid, OrderProvisioning, OrderProvisioningFailed}).Updates(map[string]any{"status": OrderProvisioning, "failure_reason": ""}).Error; err != nil {
			return err
		}
		return tx.Model(&model.ProvisioningJob{}).Where("order_id = ?", orderID).Updates(map[string]any{"status": "pending", "attempts": 0, "next_run_at": now, "last_error": "", "locked_at": nil}).Error
	})
}

func (s *AdminService) SavePlan(request entity.CommercialPlanRequest) (*model.Plan, error) {
	request.Slug = strings.TrimSpace(request.Slug)
	request.Name = strings.TrimSpace(request.Name)
	if !commercialSlugPattern.MatchString(request.Slug) || request.Name == "" || len(request.Name) > 120 {
		return nil, errors.New("套餐名称或唯一标识无效")
	}
	if request.TrafficBytes < 0 || request.DeviceLimit < 0 || request.DeviceLimit > 1000 || request.Capacity < 0 {
		return nil, errors.New("套餐流量、设备上限或容量无效")
	}
	if !map[string]bool{"never": true, "daily": true, "weekly": true, "monthly": true, "quarterly": true}[request.ResetCycle] {
		return nil, errors.New("流量重置周期无效")
	}
	if !map[string]bool{"public": true, "hidden": true, "invite": true}[request.Visibility] {
		return nil, errors.New("套餐可见性无效")
	}
	uniqueInboundIDs := make([]int, 0, len(request.ProvisionInboundIDs))
	seen := map[int]bool{}
	for _, id := range request.ProvisionInboundIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		uniqueInboundIDs = append(uniqueInboundIDs, id)
	}
	if request.Active && len(uniqueInboundIDs) == 0 {
		return nil, errors.New("上架套餐前必须绑定至少一个已启用的 3X-UI 入站")
	}
	if request.Active && request.ID != "" {
		var activePrices int64
		if err := s.db.Model(&model.PlanPrice{}).Where("plan_id = ? AND active = ?", request.ID, true).Count(&activePrices).Error; err != nil {
			return nil, err
		}
		if activePrices == 0 {
			return nil, errors.New("上架套餐前必须启用至少一个价格")
		}
	}
	if len(uniqueInboundIDs) > 0 {
		var count int64
		if err := s.db.Model(&model.Inbound{}).Where("id IN ? AND enable = ?", uniqueInboundIDs, true).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != int64(len(uniqueInboundIDs)) {
			return nil, errors.New("套餐包含不存在或未启用的 3X-UI 入站")
		}
	}
	inboundIDs, _ := json.Marshal(uniqueInboundIDs)
	row := &model.Plan{ID: request.ID, Slug: strings.TrimSpace(request.Slug), Name: strings.TrimSpace(request.Name), Description: request.Description, TrafficBytes: request.TrafficBytes, DeviceLimit: request.DeviceLimit, ResetCycle: request.ResetCycle, NodeGroup: request.NodeGroup, Capacity: request.Capacity, Visibility: request.Visibility, Renewable: request.Renewable, Upgradable: request.Upgradable, Active: request.Active, SortOrder: request.SortOrder, ProvisionInboundIDs: string(inboundIDs)}
	if row.ID == "" {
		row.ID = uuid.NewString()
		if err := s.db.Create(row).Error; err != nil {
			return nil, err
		}
		return row, nil
	}
	result := s.db.Model(&model.Plan{}).Where("id = ?", row.ID).Updates(map[string]any{"slug": row.Slug, "name": row.Name, "description": row.Description, "traffic_bytes": row.TrafficBytes, "device_limit": row.DeviceLimit, "reset_cycle": row.ResetCycle, "node_group": row.NodeGroup, "capacity": row.Capacity, "visibility": row.Visibility, "renewable": row.Renewable, "upgradable": row.Upgradable, "active": row.Active, "sort_order": row.SortOrder, "provision_inbound_ids": row.ProvisionInboundIDs})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("套餐不存在")
	}
	if err := s.db.First(row, "id = ?", row.ID).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AdminService) SavePlanPrice(request entity.CommercialPlanPriceRequest) (*model.PlanPrice, error) {
	periods := map[string]bool{"monthly": true, "quarterly": true, "half_yearly": true, "yearly": true, "multi_year": true, "one_time": true}
	if !periods[request.BillingPeriod] || request.AmountFen < 0 || request.Months < 0 || request.Months > 120 {
		return nil, errors.New("计费周期或价格无效")
	}
	if request.BillingPeriod == "one_time" {
		request.Months = 0
	} else if request.Months == 0 {
		return nil, errors.New("非一次性套餐必须设置有效月数")
	}
	var planCount int64
	if err := s.db.Model(&model.Plan{}).Where("id = ?", request.PlanID).Count(&planCount).Error; err != nil || planCount != 1 {
		return nil, errors.New("所属套餐不存在")
	}
	if request.Active {
		duplicate := s.db.Model(&model.PlanPrice{}).Where("plan_id = ? AND billing_period = ? AND active = ?", request.PlanID, request.BillingPeriod, true)
		if request.ID != "" {
			duplicate = duplicate.Where("id <> ?", request.ID)
		}
		var duplicateCount int64
		if err := duplicate.Count(&duplicateCount).Error; err != nil {
			return nil, err
		}
		if duplicateCount > 0 {
			return nil, errors.New("同一套餐不能重复添加相同计费周期")
		}
	}
	row := &model.PlanPrice{ID: request.ID, PlanID: request.PlanID, BillingPeriod: request.BillingPeriod, Months: request.Months, AmountFen: request.AmountFen, Active: request.Active}
	if row.ID == "" {
		row.ID = uuid.NewString()
		if err := s.db.Create(row).Error; err != nil {
			return nil, err
		}
		return row, nil
	}
	result := s.db.Model(&model.PlanPrice{}).Where("id = ?", row.ID).Updates(map[string]any{"plan_id": row.PlanID, "billing_period": row.BillingPeriod, "months": row.Months, "amount_fen": row.AmountFen, "active": row.Active})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("套餐价格不存在")
	}
	if err := s.db.First(row, "id = ?", row.ID).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AdminService) Notices() ([]model.Notice, error) {
	var rows []model.Notice
	err := s.db.Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) SaveNotice(row *model.Notice) error {
	if !commercialSlugPattern.MatchString(strings.TrimSpace(row.Slug)) || !validLocalizedContent(row.TitleI18n, row.ContentI18n) {
		return errors.New("公告标识或多语言内容无效")
	}
	row.Slug = strings.TrimSpace(row.Slug)
	if row.Published && row.PublishedAt == nil {
		now := time.Now().UTC()
		row.PublishedAt = &now
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
		return s.db.Create(row).Error
	}
	result := s.db.Model(&model.Notice{}).Where("id = ?", row.ID).Updates(map[string]any{"slug": row.Slug, "level": row.Level, "title_i18n": row.TitleI18n, "content_i18n": row.ContentI18n, "published": row.Published, "published_at": row.PublishedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("公告不存在")
	}
	return nil
}

func (s *AdminService) Articles() ([]model.KnowledgeArticle, error) {
	var rows []model.KnowledgeArticle
	err := s.db.Order("sort_order asc, created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) SaveArticle(row *model.KnowledgeArticle) error {
	if !commercialSlugPattern.MatchString(strings.TrimSpace(row.Slug)) || strings.TrimSpace(row.Category) == "" || !validLocalizedContent(row.TitleI18n, row.ContentI18n) {
		return errors.New("教程标识、分类或多语言内容无效")
	}
	row.Slug = strings.TrimSpace(row.Slug)
	if row.ID == "" {
		row.ID = uuid.NewString()
		return s.db.Create(row).Error
	}
	result := s.db.Model(&model.KnowledgeArticle{}).Where("id = ?", row.ID).Updates(map[string]any{"slug": row.Slug, "category": row.Category, "title_i18n": row.TitleI18n, "content_i18n": row.ContentI18n, "published": row.Published, "sort_order": row.SortOrder})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("教程不存在")
	}
	return nil
}

func (s *AdminService) Applications() ([]model.ClientApplication, error) {
	var rows []model.ClientApplication
	err := s.db.Order("sort_order asc, created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) SaveApplication(row *model.ClientApplication) error {
	if !commercialSlugPattern.MatchString(strings.TrimSpace(row.Slug)) || strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Platform) == "" || !validHTTPSURL(row.OfficialURL) || row.SourceURL != "" && !validHTTPSURL(row.SourceURL) {
		return errors.New("客户端名称、平台或官方地址无效（仅允许 HTTPS）")
	}
	row.Slug = strings.TrimSpace(row.Slug)
	if row.ID == "" {
		row.ID = uuid.NewString()
		return s.db.Create(row).Error
	}
	result := s.db.Model(&model.ClientApplication{}).Where("id = ?", row.ID).Updates(map[string]any{"slug": row.Slug, "name": row.Name, "platform": row.Platform, "official_url": row.OfficialURL, "source_url": row.SourceURL, "description": row.Description, "active": row.Active, "sort_order": row.SortOrder})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("客户端入口不存在")
	}
	return nil
}

func (s *AdminService) Tickets(status string) ([]model.Ticket, error) {
	query := s.db.Model(&model.Ticket{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var rows []model.Ticket
	err := query.Order("updated_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) TicketMessages(ticketID string) ([]model.TicketMessage, error) {
	var rows []model.TicketMessage
	err := s.db.Where("ticket_id = ?", ticketID).Order("created_at asc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) ReplyTicket(ticketID string, userID int, body, status string) error {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 10000 {
		return errors.New("回复内容无效")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		message := model.TicketMessage{ID: uuid.NewString(), TicketID: ticketID, SenderType: "admin", SenderID: strconv.Itoa(userID), Body: body}
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		if status == "" {
			status = "pending"
		}
		return tx.Model(&model.Ticket{}).Where("id = ?", ticketID).Update("status", status).Error
	})
}

func (s *AdminService) Coupons() ([]model.Coupon, error) {
	var rows []model.Coupon
	err := s.db.Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) SaveCoupon(row *model.Coupon) error {
	row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
	if !regexp.MustCompile(`^[A-Z0-9_-]{3,48}$`).MatchString(row.Code) || !map[string]bool{"fixed": true, "percent": true}[row.Kind] || row.Value <= 0 || row.MinimumFen < 0 || row.MaxRedemptions < 0 {
		return errors.New("优惠券参数无效")
	}
	if row.Kind == "percent" && row.Value > 10000 {
		return errors.New("折扣比例不能超过 10000（100%）")
	}
	if row.StartsAt != nil && row.ExpiresAt != nil && !row.ExpiresAt.After(*row.StartsAt) {
		return errors.New("优惠券到期时间必须晚于开始时间")
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
		row.RedeemedCount = 0
		return s.db.Create(row).Error
	}
	// RedeemedCount is payment-derived state. Never trust a value posted by an
	// administrator client, otherwise coupon usage limits can be reset.
	result := s.db.Model(&model.Coupon{}).Where("id = ?", row.ID).Updates(map[string]any{"code": row.Code, "kind": row.Kind, "value": row.Value, "minimum_fen": row.MinimumFen, "max_redemptions": row.MaxRedemptions, "starts_at": row.StartsAt, "expires_at": row.ExpiresAt, "active": row.Active})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("优惠券不存在")
	}
	return nil
}

func (s *AdminService) CreateGiftCards(valueFen int64, count int, expiresAt *time.Time) ([]string, error) {
	if valueFen <= 0 || count < 1 || count > 100 {
		return nil, errors.New("礼品卡金额或数量无效")
	}
	codes := make([]string, 0, count)
	rows := make([]model.GiftCard, 0, count)
	for range count {
		bytes := make([]byte, 10)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		code := strings.ToUpper(hex.EncodeToString(bytes))
		codes = append(codes, code)
		rows = append(rows, model.GiftCard{ID: uuid.NewString(), CodeHash: hashToken(code), DisplayCode: code[:4] + "••••" + code[len(code)-4:], ValueFen: valueFen, Status: "active", ExpiresAt: expiresAt})
	}
	return codes, s.db.Create(&rows).Error
}

func (s *AdminService) GiftCards() ([]model.GiftCard, error) {
	var rows []model.GiftCard
	err := s.db.Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) Commissions() ([]model.InvitationCommission, error) {
	var rows []model.InvitationCommission
	err := s.db.Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (s *AdminService) SettleCommission(id string) error {
	return settleCommission(s.db, id)
}

func (s *AdminService) Settings() map[string]any {
	publicKeys := []string{
		"payment.provider",
		"epay.gateway_url", "epay.pid", "epay.type", "epay.notify_url", "epay.return_url",
		"alipay.mode", "alipay.app_id", "alipay.seller_id", "alipay.notify_url",
		"codepay.gateway_url", "codepay.id", "codepay.type", "codepay.notify_url", "codepay.return_url",
	}
	secretKeys := []string{"epay.key", "alipay.private_key", "alipay.public_key", "codepay.key"}
	result := map[string]any{}
	for _, key := range publicKeys {
		result[key] = s.config.GetDefault(key, "")
	}
	if result["payment.provider"] == "" {
		result["payment.provider"] = "alipay_f2f"
	}
	if result["epay.type"] == "" {
		result["epay.type"] = "alipay"
	}
	if result["alipay.mode"] == "" {
		result["alipay.mode"] = "sandbox"
	}
	if result["codepay.type"] == "" {
		result["codepay.type"] = "1"
	}
	result["payment.epay.enabled"] = PaymentProviderEnabled(s.config, "epay")
	result["payment.alipay_f2f.enabled"] = PaymentProviderEnabled(s.config, "alipay_f2f")
	result["payment.codepay.enabled"] = PaymentProviderEnabled(s.config, "codepay")
	for _, key := range secretKeys {
		value, err := s.config.Get(key)
		result[key+".configured"] = err == nil && value != ""
	}
	return result
}

func (s *AdminService) SavePaymentSettings(request entity.CommercialPaymentSettings) error {
	request.EpayGatewayURL = strings.TrimRight(strings.TrimSpace(request.EpayGatewayURL), "/")
	request.EpayMerchantID = strings.TrimSpace(request.EpayMerchantID)
	request.EpayMerchantKey = strings.TrimSpace(request.EpayMerchantKey)
	request.EpayPaymentType = strings.ToLower(strings.TrimSpace(request.EpayPaymentType))
	request.EpayNotifyURL = strings.TrimSpace(request.EpayNotifyURL)
	request.EpayReturnURL = strings.TrimSpace(request.EpayReturnURL)
	request.AlipayMode = strings.ToLower(strings.TrimSpace(request.AlipayMode))
	request.AlipayAppID = strings.TrimSpace(request.AlipayAppID)
	request.AlipaySellerID = strings.TrimSpace(request.AlipaySellerID)
	request.AlipayNotifyURL = strings.TrimSpace(request.AlipayNotifyURL)
	request.AlipayPrivateKey = strings.TrimSpace(request.AlipayPrivateKey)
	request.AlipayPublicKey = strings.TrimSpace(request.AlipayPublicKey)
	request.CodepayGatewayURL = strings.TrimSpace(request.CodepayGatewayURL)
	request.CodepayMerchantID = strings.TrimSpace(request.CodepayMerchantID)
	request.CodepayKey = strings.TrimSpace(request.CodepayKey)
	request.CodepayPaymentType = strings.TrimSpace(request.CodepayPaymentType)
	request.CodepayNotifyURL = strings.TrimSpace(request.CodepayNotifyURL)
	request.CodepayReturnURL = strings.TrimSpace(request.CodepayReturnURL)

	if !request.EpayEnabled && !request.AlipayEnabled && !request.CodepayEnabled {
		return errors.New("请至少启用一个支付接口")
	}
	if request.EpayPaymentType == "" {
		request.EpayPaymentType = "alipay"
	}
	if request.EpayPaymentType != "alipay" && request.EpayPaymentType != "wxpay" && request.EpayPaymentType != "qqpay" {
		return errors.New("Epay 支付通道必须是 alipay、wxpay 或 qqpay")
	}
	if request.AlipayMode == "" {
		request.AlipayMode = "sandbox"
	}
	if request.AlipayMode != "sandbox" && request.AlipayMode != "production" {
		return errors.New("AlipayF2F 模式必须是 sandbox 或 production")
	}
	if request.CodepayPaymentType == "" {
		request.CodepayPaymentType = "1"
	}
	if request.CodepayPaymentType != "1" && request.CodepayPaymentType != "2" && request.CodepayPaymentType != "3" {
		return errors.New("码支付通道必须是支付宝、QQ 钱包或微信支付")
	}
	for label, value := range map[string]string{
		"Epay 网关地址":        request.EpayGatewayURL,
		"Epay 异步通知地址":      request.EpayNotifyURL,
		"Epay 支付后跳转地址":     request.EpayReturnURL,
		"AlipayF2F 异步通知地址": request.AlipayNotifyURL,
		"码支付网关地址":          request.CodepayGatewayURL,
		"码支付异步通知地址":        request.CodepayNotifyURL,
		"码支付支付后跳转地址":       request.CodepayReturnURL,
	} {
		if err := validateCommercialHTTPURL(label, value); err != nil {
			return err
		}
	}
	hasSecret := func(key, supplied string) bool {
		if supplied != "" {
			return true
		}
		stored, err := s.config.Get(key)
		return err == nil && strings.TrimSpace(stored) != ""
	}
	if request.EpayEnabled {
		if request.EpayGatewayURL == "" || request.EpayMerchantID == "" || request.EpayNotifyURL == "" || request.EpayReturnURL == "" {
			return errors.New("启用 Epay 前必须填写网关、商户 ID、异步通知地址和支付后跳转地址")
		}
		if !hasSecret("epay.key", request.EpayMerchantKey) {
			return errors.New("启用 Epay 前必须配置商户密钥")
		}
	}
	if request.AlipayEnabled {
		if request.AlipayAppID == "" || request.AlipayNotifyURL == "" {
			return errors.New("启用 AlipayF2F 前必须填写 App ID 和异步通知地址")
		}
		if !hasSecret("alipay.private_key", request.AlipayPrivateKey) || !hasSecret("alipay.public_key", request.AlipayPublicKey) {
			return errors.New("启用 AlipayF2F 前必须配置应用私钥和支付宝公钥")
		}
	}
	if request.CodepayEnabled {
		if request.CodepayGatewayURL == "" || request.CodepayMerchantID == "" || request.CodepayNotifyURL == "" || request.CodepayReturnURL == "" {
			return errors.New("启用码支付前必须填写网关、商户 ID、异步通知地址和支付后跳转地址")
		}
		if !hasSecret("codepay.key", request.CodepayKey) {
			return errors.New("启用码支付前必须配置通信密钥")
		}
	}

	legacyProvider := "codepay"
	if request.AlipayEnabled {
		legacyProvider = "alipay_f2f"
	}
	if request.EpayEnabled {
		legacyProvider = "epay"
	}

	values := map[string]string{
		"payment.provider":           legacyProvider,
		"payment.epay.enabled":       strconv.FormatBool(request.EpayEnabled),
		"payment.alipay_f2f.enabled": strconv.FormatBool(request.AlipayEnabled),
		"payment.codepay.enabled":    strconv.FormatBool(request.CodepayEnabled),
		"epay.gateway_url":           request.EpayGatewayURL,
		"epay.pid":                   request.EpayMerchantID,
		"epay.key":                   request.EpayMerchantKey,
		"epay.type":                  request.EpayPaymentType,
		"epay.notify_url":            request.EpayNotifyURL,
		"epay.return_url":            request.EpayReturnURL,
		"alipay.mode":                request.AlipayMode,
		"alipay.app_id":              request.AlipayAppID,
		"alipay.seller_id":           request.AlipaySellerID,
		"alipay.notify_url":          request.AlipayNotifyURL,
		"alipay.private_key":         request.AlipayPrivateKey,
		"alipay.public_key":          request.AlipayPublicKey,
		"codepay.gateway_url":        request.CodepayGatewayURL,
		"codepay.id":                 request.CodepayMerchantID,
		"codepay.key":                request.CodepayKey,
		"codepay.type":               request.CodepayPaymentType,
		"codepay.notify_url":         request.CodepayNotifyURL,
		"codepay.return_url":         request.CodepayReturnURL,
	}
	return s.config.SetManyProtected(values, map[string]bool{
		"epay.key": true, "alipay.private_key": true, "alipay.public_key": true, "codepay.key": true,
	})
}

func (s *AdminService) SiteSettings() (*entity.CommercialSiteSettingsResponse, error) {
	termsTitle, termsContent, _, _ := s.config.Terms()
	settings := entity.CommercialSiteSettings{
		SiteName:           s.config.GetDefault("site.name", "NOVA"),
		SiteDescription:    s.config.GetDefault("site.tagline", "稳定连接，清晰可控"),
		SiteURL:            s.config.GetDefault("site.url", ""),
		ForceHTTPS:         strings.EqualFold(s.config.GetDefault("site.force_https", "false"), "true"),
		LogoURL:            s.config.GetDefault("site.logo_url", ""),
		SubscriptionURLs:   s.config.GetDefault("subscription.base_url", ""),
		TermsURL:           s.config.GetDefault("site.terms_url", ""),
		TermsTemplate:      s.config.GetDefault("site.terms_template", "standard"),
		TermsTitle:         termsTitle,
		TermsContent:       termsContent,
		RegistrationClosed: strings.EqualFold(s.config.GetDefault("registration.closed", "false"), "true"),
		TrialPlanID:        s.config.GetDefault("registration.trial_plan_id", ""),
		Currency:           s.config.GetDefault("currency.code", "CNY"),
		CurrencySymbol:     s.config.GetDefault("currency.symbol", "¥"),
	}
	var plans []model.Plan
	if err := s.db.Where("active = ?", true).Order("sort_order asc, created_at asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	options := make([]entity.CommercialTrialPlanOption, 0, len(plans))
	for _, plan := range plans {
		options = append(options, entity.CommercialTrialPlanOption{ID: plan.ID, Name: plan.Name})
	}
	return &entity.CommercialSiteSettingsResponse{Settings: settings, Plans: options}, nil
}

func validateCommercialHTTPURL(label, raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s必须是有效的 HTTP 或 HTTPS 地址", label)
	}
	return nil
}

const maxCommercialLogoBytes = 5 * 1024 * 1024

func validateCommercialLogo(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "data:") {
		return validateCommercialHTTPURL("LOGO", value)
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 || !strings.HasSuffix(parts[0], ";base64") {
		return errors.New("LOGO 图片数据格式无效")
	}
	mediaType := strings.TrimSuffix(strings.TrimPrefix(parts[0], "data:"), ";base64")
	allowed := map[string]bool{
		"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true,
	}
	if !allowed[mediaType] {
		return errors.New("LOGO 仅支持 PNG、JPG、WebP 或 GIF 图片")
	}
	if len(parts[1]) > base64.StdEncoding.EncodedLen(maxCommercialLogoBytes)+4 {
		return errors.New("LOGO 图片不能超过 5 MB")
	}
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil || len(data) == 0 {
		return errors.New("LOGO 图片数据格式无效")
	}
	if len(data) > maxCommercialLogoBytes {
		return errors.New("LOGO 图片不能超过 5 MB")
	}
	if detected := http.DetectContentType(data); detected != mediaType {
		return errors.New("LOGO 图片类型与文件内容不匹配")
	}
	return nil
}

func (s *AdminService) SaveSiteSettings(request entity.CommercialSiteSettings) error {
	request.SiteName = strings.TrimSpace(request.SiteName)
	request.SiteDescription = strings.TrimSpace(request.SiteDescription)
	request.SiteURL = strings.TrimSpace(request.SiteURL)
	request.LogoURL = strings.TrimSpace(request.LogoURL)
	request.SubscriptionURLs = strings.TrimSpace(request.SubscriptionURLs)
	request.TermsURL = strings.TrimSpace(request.TermsURL)
	request.TermsTemplate = strings.TrimSpace(request.TermsTemplate)
	request.TermsTitle = strings.TrimSpace(request.TermsTitle)
	request.TermsContent = strings.TrimSpace(request.TermsContent)
	request.TrialPlanID = strings.TrimSpace(request.TrialPlanID)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.CurrencySymbol = strings.TrimSpace(request.CurrencySymbol)
	if request.SiteName == "" || len([]rune(request.SiteName)) > 120 {
		return errors.New("站点名称不能为空且不能超过 120 个字符")
	}
	if len([]rune(request.SiteDescription)) > 500 {
		return errors.New("站点描述不能超过 500 个字符")
	}
	if request.TermsTitle == "" || len([]rune(request.TermsTitle)) > 120 {
		return errors.New("使用条款标题不能为空且不能超过 120 个字符")
	}
	if request.TermsContent == "" || len([]rune(request.TermsContent)) > 50000 {
		return errors.New("使用条款正文不能为空且不能超过 50000 个字符")
	}
	if request.TermsTemplate == "" {
		request.TermsTemplate = "custom"
	}
	for label, value := range map[string]string{"站点网址": request.SiteURL, "用户条款 URL": request.TermsURL} {
		if err := validateCommercialHTTPURL(label, value); err != nil {
			return err
		}
	}
	if err := validateCommercialLogo(request.LogoURL); err != nil {
		return err
	}
	if request.SubscriptionURLs != "" {
		for _, candidate := range strings.FieldsFunc(request.SubscriptionURLs, func(r rune) bool { return r == ';' || r == '\n' || r == '\r' }) {
			if err := validateCommercialHTTPURL("订阅 URL", candidate); err != nil {
				return err
			}
		}
	}
	if !commercialCurrencyPattern.MatchString(request.Currency) {
		return errors.New("货币单位必须由 3 到 8 个大写英文字母组成")
	}
	if request.CurrencySymbol == "" || len([]rune(request.CurrencySymbol)) > 8 {
		return errors.New("货币符号不能为空且不能超过 8 个字符")
	}
	if request.TrialPlanID != "" {
		var planCount int64
		if err := s.db.Model(&model.Plan{}).Where("id = ? AND active = ?", request.TrialPlanID, true).Count(&planCount).Error; err != nil {
			return err
		}
		if planCount != 1 {
			return errors.New("注册试用套餐不存在或已下架")
		}
		var priceCount int64
		if err := s.db.Model(&model.PlanPrice{}).Where("plan_id = ? AND active = ? AND months > 0", request.TrialPlanID, true).Count(&priceCount).Error; err != nil {
			return err
		}
		if priceCount == 0 {
			return errors.New("注册试用套餐必须至少配置一个有期限的有效价格")
		}
	}
	return s.config.SetMany(map[string]string{
		"site.name":                  request.SiteName,
		"site.tagline":               request.SiteDescription,
		"site.url":                   request.SiteURL,
		"site.force_https":           strconv.FormatBool(request.ForceHTTPS),
		"site.logo_url":              request.LogoURL,
		"subscription.base_url":      request.SubscriptionURLs,
		"site.terms_url":             request.TermsURL,
		"site.terms_template":        request.TermsTemplate,
		"site.terms_title":           request.TermsTitle,
		"site.terms_content":         request.TermsContent,
		"registration.closed":        strconv.FormatBool(request.RegistrationClosed),
		"registration.trial_plan_id": request.TrialPlanID,
		"currency.code":              request.Currency,
		"currency.symbol":            request.CurrencySymbol,
	})
}

func (s *AdminService) SaveSiteLogo(raw string) error {
	logoURL := strings.TrimSpace(raw)
	if err := validateCommercialLogo(logoURL); err != nil {
		return err
	}
	return s.config.Set("site.logo_url", logoURL, false)
}

func (s *AdminService) SetSetting(request entity.CommercialSettingRequest) error {
	allowed := map[string]bool{"site.name": true, "site.tagline": true, "site.support_url": true, "site.terms_url": true, "site.privacy_url": true, "subscription.base_url": true, "alipay.mode": true, "alipay.app_id": true, "alipay.seller_id": true, "alipay.notify_url": true, "alipay.private_key": true, "alipay.public_key": true, "turnstile.site_key": true, "turnstile.secret": true}
	if !allowed[request.Key] {
		return errors.New("不支持的商业设置")
	}
	encrypted := request.Encrypted || strings.Contains(request.Key, "private_key") || strings.Contains(request.Key, "public_key") || strings.HasSuffix(request.Key, ".secret")
	return s.config.Set(request.Key, request.Value, encrypted)
}

func (s *AdminService) Audit(userID int, role, action, targetType, targetID, metadata, ipHash string) {
	row := model.CommercialAuditLog{ID: uuid.NewString(), ActorUserID: userID, ActorRole: role, Action: action, TargetType: targetType, TargetID: targetID, Metadata: metadata, IPHash: ipHash, CorrelationID: uuid.NewString()}
	_ = s.db.Create(&row).Error
}

func (s *AdminService) AuditLogs(limit int) ([]model.CommercialAuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []model.CommercialAuditLog
	err := s.db.Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *AdminService) Role(userID int) string {
	var row model.AdminRoleBinding
	if err := s.db.Where("user_id = ?", userID).First(&row).Error; err != nil {
		return "read_only_auditor"
	}
	return row.Role
}

func (s *AdminService) SetRole(userID int, role string) error {
	allowed := map[string]bool{"owner": true, "administrator": true, "finance": true, "support": true, "node_operator": true, "read_only_auditor": true}
	if !allowed[role] {
		return errors.New("角色无效")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var userCount int64
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Count(&userCount).Error; err != nil || userCount != 1 {
			return errors.New("管理员不存在")
		}
		var current model.AdminRoleBinding
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&current).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && current.Role == "owner" && role != "owner" {
			var owners int64
			if err := tx.Model(&model.AdminRoleBinding{}).Where("role = ?", "owner").Count(&owners).Error; err != nil {
				return err
			}
			if owners <= 1 {
				return errors.New("不能移除系统中最后一个所有者")
			}
		}
		row := model.AdminRoleBinding{UserID: userID, Role: role}
		return tx.Save(&row).Error
	})
}

func (s *AdminService) AdminUsers() ([]AdminUserRole, error) {
	var users []model.User
	if err := s.db.Select("id", "username").Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	rows := make([]AdminUserRole, 0, len(users))
	for _, user := range users {
		rows = append(rows, AdminUserRole{UserID: user.Id, Username: user.Username, Role: s.Role(user.Id)})
	}
	return rows, nil
}

func validLocalizedContent(titleRaw, contentRaw string) bool {
	var titles, contents map[string]string
	if json.Unmarshal([]byte(titleRaw), &titles) != nil || json.Unmarshal([]byte(contentRaw), &contents) != nil {
		return false
	}
	return strings.TrimSpace(titles["zh-CN"]) != "" && strings.TrimSpace(titles["en-US"]) != "" && strings.TrimSpace(contents["zh-CN"]) != "" && strings.TrimSpace(contents["en-US"]) != ""
}

func validHTTPSURL(raw string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 30
	}
	return page, pageSize
}
