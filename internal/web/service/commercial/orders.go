package commercial

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	OrderPending            = "pending"
	OrderPaid               = "paid"
	OrderProvisioning       = "provisioning"
	OrderCompleted          = "completed"
	OrderCancelled          = "cancelled"
	OrderExpired            = "expired"
	OrderProvisioningFailed = "provisioning_failed"
)

type PlanCatalogItem struct {
	Plan         model.Plan        `json:"plan"`
	Prices       []model.PlanPrice `json:"prices"`
	LineGroupIDs []string          `json:"lineGroupIds,omitempty"`
}

type OrderService struct {
	db     *gorm.DB
	config *ConfigStore
}

func NewOrderService() *OrderService {
	return &OrderService{db: database.GetDB(), config: NewConfigStore()}
}

func (s *OrderService) Catalog(publicOnly bool) ([]PlanCatalogItem, error) {
	query := s.db.Model(&model.Plan{})
	if publicOnly {
		query = query.Where("active = ? AND visibility = ?", true, "public")
	}
	var plans []model.Plan
	if err := query.Order("sort_order asc, created_at asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	items := make([]PlanCatalogItem, 0, len(plans))
	for _, plan := range plans {
		if publicOnly {
			var manualInboundIDs []int
			if plan.ProvisionInboundIDs != "" {
				if err := json.Unmarshal([]byte(plan.ProvisionInboundIDs), &manualInboundIDs); err != nil {
					return nil, errors.New("套餐入站配置无效")
				}
			}
			healthy, err := planHasHealthyLineDB(s.db, plan.ID, manualInboundIDs)
			if err != nil {
				return nil, err
			}
			if !healthy {
				continue
			}
		}
		var prices []model.PlanPrice
		priceQuery := s.db.Where("plan_id = ?", plan.ID)
		if publicOnly {
			priceQuery = priceQuery.Where("active = ?", true)
		}
		if err := priceQuery.Order("months asc").Find(&prices).Error; err != nil {
			return nil, err
		}
		item := PlanCatalogItem{Plan: plan, Prices: prices}
		if !publicOnly {
			if err := s.db.Model(&model.PlanLineGroup{}).Where("plan_id = ?", plan.ID).Order("group_id asc").Pluck("group_id", &item.LineGroupIDs).Error; err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *OrderService) Create(customerID, priceID, couponCode string, useBalance bool) (*model.Order, error) {
	return s.CreateFor(customerID, priceID, "purchase", "", couponCode, useBalance)
}

func (s *OrderService) CreateFor(customerID, priceID, orderKind, entitlementID, couponCode string, useBalance bool) (*model.Order, error) {
	orderKind = strings.ToLower(strings.TrimSpace(orderKind))
	if orderKind == "" {
		orderKind = "purchase"
	}
	if orderKind != "purchase" && orderKind != "renewal" && orderKind != "upgrade" {
		return nil, errors.New("订单类型无效")
	}
	policy := s.config.SubscriptionPolicy()
	if orderKind != "purchase" && !policy.AllowUserChange {
		return nil, errors.New("当前未开放用户自助续费或升级，请联系管理员处理")
	}
	order := &model.Order{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var price model.PlanPrice
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ? AND active = ?", priceID, true).First(&price).Error; err != nil {
			return errors.New("套餐价格不可用")
		}
		var plan model.Plan
		// Lock the plan row so capacity checks are serialized on PostgreSQL.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND active = ?", price.PlanID, true).First(&plan).Error; err != nil {
			return errors.New("套餐不可用")
		}
		if plan.Visibility != "public" {
			return errors.New("该套餐不能直接购买")
		}
		var manualInboundIDs []int
		if plan.ProvisionInboundIDs != "" {
			if err := json.Unmarshal([]byte(plan.ProvisionInboundIDs), &manualInboundIDs); err != nil {
				return errors.New("套餐入站配置无效")
			}
		}
		hasLine, err := planHasHealthyLineDB(tx, plan.ID, manualInboundIDs)
		if err != nil {
			return err
		}
		if !hasLine {
			return errors.New("该套餐暂无可用线路")
		}
		now := time.Now().UTC()
		var targetEntitlement model.SubscriptionEntitlement
		var resultExpiresAt *time.Time
		upgradeOffsetFen := int64(0)
		if orderKind == "purchase" {
			var active int64
			if err := tx.Model(&model.SubscriptionEntitlement{}).
				Where("customer_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", customerID, "active", now).
				Count(&active).Error; err != nil {
				return err
			}
			if active > 0 {
				return errors.New("已有生效订阅，请选择续费或升级")
			}
		} else {
			if entitlementID == "" {
				return errors.New("请选择需要续费或升级的订阅")
			}
			if price.Months <= 0 {
				return errors.New("续费和升级必须选择有效的计费周期")
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND customer_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", entitlementID, customerID, "active", now).
				First(&targetEntitlement).Error; err != nil {
				return errors.New("当前订阅不可续费或升级")
			}
			var currentPlan model.Plan
			if err := tx.Where("id = ?", targetEntitlement.PlanID).First(&currentPlan).Error; err != nil {
				return errors.New("当前套餐不存在")
			}
			if orderKind == "renewal" {
				if !currentPlan.Renewable {
					return errors.New("当前套餐未开放续费")
				}
				if plan.ID != currentPlan.ID {
					return errors.New("续费必须选择当前套餐")
				}
			} else {
				if !currentPlan.Upgradable {
					return errors.New("当前套餐未开放升级")
				}
				if plan.ID == currentPlan.ID {
					return errors.New("升级必须选择其他套餐")
				}
				if plan.SortOrder <= currentPlan.SortOrder && plan.TrafficBytes <= currentPlan.TrafficBytes && plan.DeviceLimit <= currentPlan.DeviceLimit {
					return errors.New("只能升级到规格更高的套餐")
				}
			}
			var pending int64
			if err := tx.Model(&model.Order{}).
				Where("entitlement_id = ? AND status IN ?", targetEntitlement.ID, []string{OrderPending, OrderPaid, OrderProvisioning, OrderProvisioningFailed}).
				Count(&pending).Error; err != nil {
				return err
			}
			if pending > 0 {
				return errors.New("该订阅已有待处理的续费或升级订单")
			}
			base := now
			if orderKind == "upgrade" && policy.OffsetEnabled {
				upgradeOffsetFen = calculateUpgradeOffset(tx, currentPlan.ID, targetEntitlement.ExpiresAt, price.AmountFen, now)
			} else if targetEntitlement.ExpiresAt != nil && targetEntitlement.ExpiresAt.After(base) {
				base = *targetEntitlement.ExpiresAt
			}
			result := base.AddDate(0, price.Months, 0)
			resultExpiresAt = &result
		}
		if plan.Capacity > 0 && orderKind != "renewal" {
			var active int64
			if err := tx.Model(&model.SubscriptionEntitlement{}).
				Where("plan_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", plan.ID, "active", now).
				Count(&active).Error; err != nil {
				return err
			}
			if active >= int64(plan.Capacity) {
				return errors.New("该套餐名额已满")
			}
		}
		normalizedCoupon := strings.ToUpper(strings.TrimSpace(couponCode))
		discount, couponID, err := s.applyCoupon(tx, normalizedCoupon, price.AmountFen)
		if err != nil {
			return err
		}
		remainingAfterCoupon := max(int64(0), price.AmountFen-discount)
		if upgradeOffsetFen > remainingAfterCoupon {
			upgradeOffsetFen = remainingAfterCoupon
		}
		discount += upgradeOffsetFen
		payable := max(0, price.AmountFen-discount)
		balancePaid := int64(0)
		if useBalance && payable > 0 {
			var customer model.Customer
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", customerID, "active").First(&customer).Error; err != nil {
				return errors.New("客户账户不可用")
			}
			balancePaid = min(customer.BalanceFen, payable)
			if balancePaid > 0 {
				if err := tx.Model(&customer).UpdateColumn("balance_fen", gorm.Expr("balance_fen - ?", balancePaid)).Error; err != nil {
					return err
				}
				payable -= balancePaid
			}
		}
		order = &model.Order{ID: uuid.NewString(), OutTradeNo: newOutTradeNo(now), CustomerID: customerID, PlanID: plan.ID, PlanPriceID: price.ID, OrderKind: orderKind, EntitlementID: targetEntitlement.ID, ResultExpiresAt: resultExpiresAt, Status: OrderPending, OriginalFen: price.AmountFen, DiscountFen: discount, BalancePaidFen: balancePaid, PayableFen: payable, Currency: "CNY", CouponCode: normalizedCoupon, ExpiresAt: now.Add(15 * time.Minute)}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if couponID != "" {
			reservation := model.CouponRedemption{ID: uuid.NewString(), CouponID: couponID, OrderID: order.ID, CustomerID: customerID, Status: "reserved"}
			if err := tx.Create(&reservation).Error; err != nil {
				return err
			}
		}
		if payable == 0 {
			payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: "BALANCE-" + order.ID, AmountFen: 0, TradeStatus: "TRADE_SUCCESS", RawPayload: "account-balance"}
			return s.finalizePaid(tx, order, "balance", payment)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func calculateUpgradeOffset(tx *gorm.DB, currentPlanID string, expiresAt *time.Time, newPriceFen int64, now time.Time) int64 {
	if expiresAt == nil || !expiresAt.After(now) || newPriceFen <= 0 {
		return 0
	}
	var prices []model.PlanPrice
	if err := tx.Where("plan_id = ? AND active = ? AND months > ?", currentPlanID, true, 0).Find(&prices).Error; err != nil {
		return 0
	}
	monthlyFen := int64(0)
	for _, price := range prices {
		candidate := price.AmountFen / int64(price.Months)
		if candidate > 0 && (monthlyFen == 0 || candidate < monthlyFen) {
			monthlyFen = candidate
		}
	}
	if monthlyFen == 0 {
		return 0
	}
	const secondsPerBillingMonth = int64(30 * 24 * 60 * 60)
	remainingSeconds := int64(expiresAt.Sub(now).Seconds())
	credit := monthlyFen * remainingSeconds / secondsPerBillingMonth
	if credit < 0 {
		return 0
	}
	return min(credit, newPriceFen)
}

func (s *OrderService) List(customerID string) ([]model.Order, error) {
	var orders []model.Order
	err := s.db.Where("customer_id = ?", customerID).Order("created_at desc").Find(&orders).Error
	return orders, err
}

func (s *OrderService) Cancel(customerID, orderID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND customer_id = ? AND status = ?", orderID, customerID, OrderPending).First(&order).Error; err != nil {
			return errors.New("仅待支付订单可以取消")
		}
		var started int64
		if err := tx.Model(&model.PaymentTransaction{}).Where("order_id = ? AND status = ?", order.ID, "pending").Count(&started).Error; err != nil {
			return err
		}
		if started > 0 {
			return errors.New("支付二维码已生成，请等待订单自动过期")
		}
		if err := s.releaseCouponReservation(tx, order.ID); err != nil {
			return err
		}
		return s.releaseOrderFunds(tx, &order, OrderCancelled)
	})
}

func (s *OrderService) Precreate(ctx context.Context, customerID, orderID, providerName string) (*PaymentIntent, error) {
	var order model.Order
	if err := s.db.Where("id = ? AND customer_id = ?", orderID, customerID).First(&order).Error; err != nil {
		return nil, errors.New("订单不存在")
	}
	if order.Status != OrderPending {
		return nil, errors.New("订单状态不允许支付")
	}
	if time.Now().UTC().After(order.ExpiresAt) {
		_ = s.expireOrder(order.ID)
		return nil, errors.New("订单已过期")
	}
	var plan model.Plan
	if err := s.db.Where("id = ?", order.PlanID).First(&plan).Error; err != nil {
		return nil, err
	}
	providerName = normalizePaymentProviderName(providerName)
	if isDemoMode() {
		if providerName != "alipay-demo" {
			return nil, errors.New("演示环境仅支持演示支付")
		}
	} else if !PaymentProviderEnabled(s.config, providerName) {
		return nil, errors.New("所选支付方式未启用")
	}
	provider, err := PaymentProviderByName(s.config, providerName)
	if err != nil {
		return nil, err
	}
	var existing model.PaymentTransaction
	if err := s.db.Where("order_id = ? AND provider = ? AND status = ?", order.ID, provider.Name(), "pending").Order("created_at desc").First(&existing).Error; err == nil && existing.RawPayload != "" {
		var intent PaymentIntent
		if json.Unmarshal([]byte(existing.RawPayload), &intent) == nil && intent.QRCode != "" && time.Now().UTC().Before(intent.ExpiresAt) {
			return &intent, nil
		}
	}
	intent, err := provider.Precreate(ctx, PaymentRequest{OutTradeNo: order.OutTradeNo, Subject: plan.Name, AmountFen: order.PayableFen, ExpiresAt: order.ExpiresAt})
	if err != nil {
		return nil, err
	}
	transactionID := uuid.NewString()
	intentJSON, _ := json.Marshal(intent)
	transaction := model.PaymentTransaction{ID: transactionID, OrderID: order.ID, Provider: provider.Name(), ProviderTradeNo: "pending:" + transactionID, AmountFen: order.PayableFen, Status: "pending", RawPayload: string(intentJSON)}
	if err := s.db.Create(&transaction).Error; err != nil {
		return nil, err
	}
	return intent, nil
}

func (s *OrderService) HandleNotification(ctx context.Context, values map[string][]string) error {
	provider, err := ActivePaymentProvider(s.config)
	if err != nil {
		return err
	}
	return s.handleNotificationWithProvider(ctx, provider, values)
}

func (s *OrderService) HandleNotificationForProvider(ctx context.Context, providerName string, values map[string][]string) error {
	provider, err := PaymentProviderByName(s.config, providerName)
	if err != nil {
		return err
	}
	return s.handleNotificationWithProvider(ctx, provider, values)
}

func (s *OrderService) handleNotificationWithProvider(ctx context.Context, provider PaymentProvider, values map[string][]string) error {
	notification, err := provider.VerifyNotification(values)
	if err != nil {
		return err
	}
	return s.markPaid(ctx, provider.Name(), notification)
}

func (s *OrderService) DemoPay(ctx context.Context, customerID, orderID string) error {
	if !isDemoMode() {
		return errors.New("演示支付未启用")
	}
	var order model.Order
	if err := s.db.Where("id = ? AND customer_id = ?", orderID, customerID).First(&order).Error; err != nil {
		return errors.New("订单不存在")
	}
	return s.markPaid(ctx, "alipay-demo", &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: "DEMO-" + uuid.NewString(), AmountFen: order.PayableFen, TradeStatus: "TRADE_SUCCESS", RawPayload: "demo"})
}

func (s *OrderService) markPaid(_ context.Context, providerName string, payment *PaymentNotification) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("out_trade_no = ?", payment.OutTradeNo).First(&order).Error; err != nil {
			return errors.New("支付订单不存在")
		}
		if order.PayableFen != payment.AmountFen {
			return errors.New("支付金额与订单不一致")
		}
		if order.Status == OrderPaid || order.Status == OrderProvisioning || order.Status == OrderCompleted {
			return nil
		}
		if order.Status != OrderPending && order.Status != OrderExpired {
			return errors.New("订单状态不允许确认支付")
		}
		if order.Status == OrderExpired && time.Now().UTC().After(order.ExpiresAt.Add(24*time.Hour)) {
			return errors.New("订单支付补偿窗口已结束，请人工核查异常到账")
		}
		if payment.TradeStatus != "TRADE_SUCCESS" && payment.TradeStatus != "TRADE_FINISHED" {
			return errors.New("交易状态未成功")
		}
		return s.finalizePaid(tx, &order, providerName, payment)
	})
}

func (s *OrderService) finalizePaid(tx *gorm.DB, order *model.Order, providerName string, payment *PaymentNotification) error {
	now := time.Now().UTC()
	transaction := model.PaymentTransaction{ID: uuid.NewString(), OrderID: order.ID, Provider: providerName, ProviderTradeNo: payment.ProviderTrade, AmountFen: payment.AmountFen, Status: "paid", RawPayload: payment.RawPayload}
	if err := tx.Where("provider = ? AND provider_trade_no = ?", providerName, payment.ProviderTrade).FirstOrCreate(&transaction).Error; err != nil {
		return err
	}
	if err := tx.Model(order).Updates(map[string]any{"status": OrderPaid, "paid_fen": payment.AmountFen, "paid_at": now, "failure_reason": ""}).Error; err != nil {
		return err
	}
	order.Status = OrderPaid
	order.PaidFen = payment.AmountFen
	order.PaidAt = &now
	if err := s.consumeCouponReservation(tx, order); err != nil {
		return err
	}
	job := model.ProvisioningJob{ID: uuid.NewString(), OrderID: order.ID, CustomerID: order.CustomerID, Status: "pending", NextRunAt: now}
	if err := tx.Where("order_id = ?", order.ID).FirstOrCreate(&job).Error; err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"orderId": order.ID, "customerId": order.CustomerID})
	event := model.OutboxEvent{ID: uuid.NewString(), AggregateType: "order", AggregateID: order.ID, EventType: "order.paid", Payload: string(payload), Status: "pending", NextRunAt: now}
	if err := tx.Where("aggregate_id = ? AND event_type = ?", order.ID, "order.paid").FirstOrCreate(&event).Error; err != nil {
		return err
	}
	return s.createCommission(tx, order)
}

func (s *OrderService) releaseOrderFunds(tx *gorm.DB, order *model.Order, status string) error {
	if order.BalancePaidFen > 0 {
		if err := tx.Model(&model.Customer{}).Where("id = ?", order.CustomerID).UpdateColumn("balance_fen", gorm.Expr("balance_fen + ?", order.BalancePaidFen)).Error; err != nil {
			return err
		}
	}
	fullPayable := max(0, order.OriginalFen-order.DiscountFen)
	return tx.Model(order).Updates(map[string]any{"status": status, "balance_paid_fen": 0, "payable_fen": fullPayable}).Error
}

func (s *OrderService) expireOrder(orderID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", orderID, OrderPending).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		// Keep account balance and coupon reservations held while the worker
		// actively reconciles late or missed Alipay callbacks for 24 hours.
		// Releasing them immediately could let a late callback provision an
		// order whose balance portion was already returned to the customer.
		return tx.Model(&order).Update("status", OrderExpired).Error
	})
}

func (s *OrderService) releaseExpiredOrder(orderID string, before time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ? AND expires_at < ?", orderID, OrderExpired, before).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if err := s.releaseCouponReservation(tx, order.ID); err != nil {
			return err
		}
		return s.releaseOrderFunds(tx, &order, OrderExpired)
	})
}

func (s *OrderService) applyCoupon(tx *gorm.DB, code string, amount int64) (int64, string, error) {
	if code == "" {
		return 0, "", nil
	}
	var coupon model.Coupon
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ? AND active = ?", code, true).First(&coupon).Error; err != nil {
		return 0, "", errors.New("优惠券无效")
	}
	now := time.Now().UTC()
	var reserved int64
	if err := tx.Model(&model.CouponRedemption{}).Where("coupon_id = ? AND status = ?", coupon.ID, "reserved").Count(&reserved).Error; err != nil {
		return 0, "", err
	}
	if coupon.StartsAt != nil && now.Before(*coupon.StartsAt) || coupon.ExpiresAt != nil && now.After(*coupon.ExpiresAt) || coupon.MaxRedemptions > 0 && int64(coupon.RedeemedCount)+reserved >= int64(coupon.MaxRedemptions) || amount < coupon.MinimumFen {
		return 0, "", errors.New("优惠券当前不可用")
	}
	discount := coupon.Value
	if coupon.Kind == "percent" {
		discount = amount * coupon.Value / 10000
	}
	if discount < 0 {
		discount = 0
	}
	if discount > amount {
		discount = amount
	}
	return discount, coupon.ID, nil
}

func (s *OrderService) consumeCouponReservation(tx *gorm.DB, order *model.Order) error {
	if order.CouponCode == "" {
		return nil
	}
	var redemption model.CouponRedemption
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_id = ?", order.ID).First(&redemption).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Compatibility for orders created before coupon reservations existed.
		var coupon model.Coupon
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", order.CouponCode).First(&coupon).Error; err != nil {
			return errors.New("优惠券记录不存在")
		}
		redemption = model.CouponRedemption{ID: uuid.NewString(), CouponID: coupon.ID, OrderID: order.ID, CustomerID: order.CustomerID, Status: "consumed"}
		if err := tx.Create(&redemption).Error; err != nil {
			return err
		}
		return tx.Model(&coupon).UpdateColumn("redeemed_count", gorm.Expr("redeemed_count + 1")).Error
	}
	if err != nil {
		return err
	}
	if redemption.Status == "consumed" {
		return nil
	}
	if redemption.Status != "reserved" {
		return errors.New("优惠券预留已释放，无法确认支付")
	}
	result := tx.Model(&redemption).Where("status = ?", "reserved").Update("status", "consumed")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("优惠券消费状态冲突")
	}
	return tx.Model(&model.Coupon{}).Where("id = ?", redemption.CouponID).UpdateColumn("redeemed_count", gorm.Expr("redeemed_count + 1")).Error
}

func (s *OrderService) releaseCouponReservation(tx *gorm.DB, orderID string) error {
	return tx.Model(&model.CouponRedemption{}).Where("order_id = ? AND status = ?", orderID, "reserved").Update("status", "released").Error
}

func (s *OrderService) createCommission(tx *gorm.DB, order *model.Order) error {
	var customer model.Customer
	if err := tx.Where("id = ?", order.CustomerID).First(&customer).Error; err != nil || customer.InvitedByID == nil {
		return nil
	}
	policy := s.config.InvitationPolicy()
	if policy.CommissionPercent <= 0 {
		return nil
	}
	if policy.CommissionFirstPaymentOnly {
		var previousPayments int64
		if err := tx.Model(&model.Order{}).
			Where("customer_id = ? AND id <> ? AND status IN ?", customer.ID, order.ID, []string{OrderPaid, OrderProvisioning, OrderCompleted}).
			Count(&previousPayments).Error; err != nil {
			return err
		}
		if previousPayments > 0 {
			return nil
		}
	}
	shareAmount := max(0, order.OriginalFen-order.DiscountFen) * int64(policy.CommissionPercent) / 100
	if shareAmount <= 0 {
		return nil
	}
	maxLevels := 1
	if policy.MultiLevelEnabled {
		maxLevels = 3
	}
	shares := make([]commissionShare, 0, maxLevels)
	nextInviterID := customer.InvitedByID
	seen := map[string]bool{customer.ID: true}
	for level := 1; level <= maxLevels && nextInviterID != nil && *nextInviterID != ""; level++ {
		if seen[*nextInviterID] {
			break
		}
		var inviter model.Customer
		if err := tx.Where("id = ? AND status = ?", *nextInviterID, "active").First(&inviter).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return err
		}
		seen[inviter.ID] = true
		shares = append(shares, commissionShare{CustomerID: inviter.ID, Level: level, AmountFen: shareAmount})
		nextInviterID = inviter.InvitedByID
	}
	if len(shares) == 0 {
		return nil
	}
	distribution, err := json.Marshal(shares)
	if err != nil {
		return err
	}
	totalAmount := int64(0)
	for _, share := range shares {
		totalAmount += share.AmountFen
	}
	row := model.InvitationCommission{ID: uuid.NewString(), InviterID: shares[0].CustomerID, InviteeID: customer.ID, OrderID: order.ID, AmountFen: totalAmount, Distribution: string(distribution), Status: "pending"}
	return tx.Where("order_id = ?", order.ID).FirstOrCreate(&row).Error
}

func newOutTradeNo(now time.Time) string {
	return fmt.Sprintf("NV%s%s", now.Format("20060102150405"), strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:10], "-", "")))
}
