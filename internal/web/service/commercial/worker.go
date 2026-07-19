package commercial

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	emailservice "github.com/mhsanaei/3x-ui/v3/internal/web/service/email"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Worker struct {
	db       *gorm.DB
	orders   *OrderService
	config   *ConfigStore
	clients  service.ClientService
	inbounds service.InboundService
	mailer   customerEmailSender
}

type customerEmailSender interface {
	SendTo(recipients []string, subject, body string) error
}

func NewWorker() *Worker {
	return &Worker{
		db: database.GetDB(), orders: NewOrderService(), config: NewConfigStore(),
		mailer: emailservice.NewEmailService(service.SettingService{}),
	}
}

func (w *Worker) Run(ctx context.Context) {
	fast := time.NewTicker(3 * time.Second)
	slow := time.NewTicker(time.Minute)
	defer fast.Stop()
	defer slow.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-fast.C:
			if err := w.processNextProvisioning(ctx); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warning("commercial provisioning:", err)
			}
			if err := w.processNextOutbox(); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Warning("commercial outbox:", err)
			}
		case <-slow.C:
			w.recoverStaleJobs()
			if err := w.reconcilePayments(ctx); err != nil {
				logger.Warning("commercial payment reconciliation:", err)
			}
			w.expireOrders()
			if err := w.confirmMatureCommissions(); err != nil {
				logger.Warning("commercial commission confirmation:", err)
			}
		}
	}
}

func (w *Worker) confirmMatureCommissions() error {
	policy := w.config.InvitationPolicy()
	if !policy.CommissionAutoConfirm {
		return nil
	}
	var rows []model.InvitationCommission
	err := w.db.Table("commercial_invitation_commissions AS commissions").
		Select("commissions.*").
		Joins("JOIN commercial_orders AS orders ON orders.id = commissions.order_id").
		Where("commissions.status = ? AND orders.completed_at <= ? AND orders.status = ?", "pending", time.Now().UTC().Add(-commissionAutoConfirmDelay), OrderCompleted).
		Order("commissions.created_at asc").
		Limit(200).
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := settleCommission(w.db, row.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (w *Worker) recoverStaleJobs() {
	now := time.Now().UTC()
	cutoff := now.Add(-15 * time.Minute)
	_ = w.db.Model(&model.ProvisioningJob{}).
		Where("status = ? AND locked_at < ?", "running", cutoff).
		Updates(map[string]any{"status": "retry", "locked_at": nil, "next_run_at": now, "last_error": "recovered stale worker lock"}).Error
	_ = w.db.Model(&model.OutboxEvent{}).
		Where("status = ? AND updated_at < ?", "running", cutoff).
		Updates(map[string]any{"status": "retry", "next_run_at": now, "last_error": "recovered stale outbox lock"}).Error
}

func (w *Worker) processNextProvisioning(ctx context.Context) error {
	var job model.ProvisioningJob
	now := time.Now().UTC()
	err := w.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status IN ? AND next_run_at <= ?", []string{"pending", "retry"}, now).Order("next_run_at asc")
		if err := query.First(&job).Error; err != nil {
			return err
		}
		return tx.Model(&job).Updates(map[string]any{"status": "running", "locked_at": now, "attempts": gorm.Expr("attempts + 1")}).Error
	})
	if err != nil {
		return err
	}
	err = w.provision(ctx, &job)
	if err == nil {
		now = time.Now().UTC()
		return w.db.Model(&model.ProvisioningJob{}).Where("id = ? AND status = ?", job.ID, "running").Updates(map[string]any{"status": "completed", "locked_at": nil, "last_error": "", "updated_at": now}).Error
	}
	attempts := job.Attempts + 1
	if attempts >= 10 {
		_ = w.db.Model(&model.Order{}).Where("id = ? AND status IN ?", job.OrderID, []string{OrderPaid, OrderProvisioning}).Updates(map[string]any{"status": OrderProvisioningFailed, "failure_reason": err.Error()}).Error
		return w.db.Model(&model.ProvisioningJob{}).Where("id = ? AND status = ?", job.ID, "running").Updates(map[string]any{"status": "manual", "locked_at": nil, "last_error": err.Error()}).Error
	}
	delay := time.Duration(math.Pow(2, float64(min(attempts, 8)))) * time.Minute
	return w.db.Model(&model.ProvisioningJob{}).Where("id = ? AND status = ?", job.ID, "running").Updates(map[string]any{"status": "retry", "locked_at": nil, "last_error": err.Error(), "next_run_at": time.Now().UTC().Add(delay)}).Error
}

func (w *Worker) provision(_ context.Context, job *model.ProvisioningJob) error {
	var order model.Order
	if err := w.db.Where("id = ?", job.OrderID).First(&order).Error; err != nil {
		return err
	}
	if order.Status == OrderCompleted {
		return nil
	}
	if order.Status != OrderPaid && order.Status != OrderProvisioning {
		return fmt.Errorf("order %s is not paid", order.ID)
	}
	var plan model.Plan
	if err := w.db.Where("id = ?", order.PlanID).First(&plan).Error; err != nil {
		return err
	}
	var price model.PlanPrice
	if err := w.db.Where("id = ?", order.PlanPriceID).First(&price).Error; err != nil {
		return err
	}
	if order.OrderKind == "renewal" || order.OrderKind == "upgrade" {
		return w.provisionExistingSubscription(&order, &plan, &price)
	}
	var existing model.SubscriptionEntitlement
	if err := w.db.Where("order_id = ?", job.OrderID).First(&existing).Error; err == nil {
		return w.completeOrder(job.OrderID)
	}
	inboundIDs, err := w.provisionInboundIDs(&plan)
	if err != nil {
		return err
	}
	if len(inboundIDs) == 0 {
		return errors.New("套餐尚未绑定可用入站")
	}
	result := w.db.Model(&model.Order{}).Where("id = ? AND status IN ?", order.ID, []string{OrderPaid, OrderProvisioning}).Update("status", OrderProvisioning)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("订单状态已变更，停止开通")
	}
	internalID := internalClientID(order.CustomerID, order.ID)
	subscriptionID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("subscription:"+order.ID)).String()
	clientID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("client:"+order.ID)).String()
	expiry := time.Now().UTC().AddDate(0, price.Months, 0)
	policy := w.config.SubscriptionPolicy()
	client := model.Client{ID: clientID, Email: internalID, SubID: subscriptionID, Enable: true, TotalGB: plan.TrafficBytes, ExpiryTime: expiry.UnixMilli(), LimitIP: plan.DeviceLimit, Group: plan.NodeGroup, Reset: resetDaysForPolicy(plan.ResetCycle, policy.MonthlyResetMode), Comment: "commercial:" + order.ID}
	var record model.ClientRecord
	lookupErr := w.db.Where("email = ?", internalID).First(&record).Error
	if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		if _, err := w.clients.Create(&w.inbounds, &service.ClientCreatePayload{Client: client, InboundIds: inboundIDs}); err != nil {
			return err
		}
	} else if lookupErr != nil {
		return lookupErr
	} else {
		if _, err := w.clients.UpdateByEmail(&w.inbounds, internalID, client, inboundIDs...); err != nil {
			return err
		}
		if _, err := w.clients.AttachByEmail(&w.inbounds, internalID, inboundIDs); err != nil {
			return err
		}
	}
	starts := time.Now().UTC()
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: order.CustomerID, PlanID: plan.ID, OrderID: order.ID, InternalClientID: internalID, SubscriptionID: subscriptionID, Status: "active", TrafficQuota: plan.TrafficBytes, DeviceLimit: plan.DeviceLimit, NodeGroup: plan.NodeGroup, StartsAt: starts, ExpiresAt: &expiry}
	if policy.EventFor(order.OrderKind) == SubscriptionEventResetTraffic {
		if _, err := w.clients.ResetTrafficByEmail(&w.inbounds, internalID); err != nil {
			return err
		}
		entitlement.LastResetAt = &starts
	}
	return w.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", order.ID).FirstOrCreate(&entitlement).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		result := tx.Model(&model.Order{}).Where("id = ? AND status IN ?", order.ID, []string{OrderPaid, OrderProvisioning}).Updates(map[string]any{"status": OrderCompleted, "completed_at": now, "failure_reason": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("订单状态已变更，停止开通")
		}
		payload, _ := json.Marshal(map[string]string{"orderId": order.ID, "entitlementId": entitlement.ID})
		event := model.OutboxEvent{ID: uuid.NewString(), AggregateType: "subscription", AggregateID: entitlement.ID, EventType: "subscription.provisioned", Payload: string(payload), Status: "pending", NextRunAt: now}
		return tx.Where("aggregate_id = ? AND event_type = ?", entitlement.ID, "subscription.provisioned").FirstOrCreate(&event).Error
	})
}

func (w *Worker) provisionExistingSubscription(order *model.Order, plan *model.Plan, _ *model.PlanPrice) error {
	if order.EntitlementID == "" || order.ResultExpiresAt == nil {
		return errors.New("续费或升级订单缺少目标订阅信息")
	}
	var entitlement model.SubscriptionEntitlement
	if err := w.db.Where("id = ? AND customer_id = ? AND status = ?", order.EntitlementID, order.CustomerID, "active").First(&entitlement).Error; err != nil {
		return errors.New("续费或升级的订阅不存在")
	}
	inboundIDs, err := w.provisionInboundIDs(plan)
	if err != nil {
		return err
	}
	if len(inboundIDs) == 0 {
		return errors.New("套餐尚未绑定可用入站")
	}
	result := w.db.Model(&model.Order{}).
		Where("id = ? AND status IN ?", order.ID, []string{OrderPaid, OrderProvisioning}).
		Update("status", OrderProvisioning)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("订单状态已变更，停止处理")
	}
	var record model.ClientRecord
	if err := w.db.Where("email = ?", entitlement.InternalClientID).First(&record).Error; err != nil {
		return errors.New("订阅对应的 3X-UI 客户端不存在")
	}
	currentInboundIDs, err := w.clients.GetInboundIdsForEmail(w.db, entitlement.InternalClientID)
	if err != nil {
		return err
	}
	client := record.ToClient()
	client.Email = entitlement.InternalClientID
	client.SubID = entitlement.SubscriptionID
	client.Enable = true
	client.TotalGB = plan.TrafficBytes
	client.ExpiryTime = order.ResultExpiresAt.UnixMilli()
	client.LimitIP = plan.DeviceLimit
	client.Group = plan.NodeGroup
	policy := w.config.SubscriptionPolicy()
	client.Reset = resetDaysForPolicy(plan.ResetCycle, policy.MonthlyResetMode)
	if _, err := w.clients.UpdateByEmail(&w.inbounds, entitlement.InternalClientID, *client, currentInboundIDs...); err != nil {
		return err
	}
	if _, err := w.clients.AttachByEmail(&w.inbounds, entitlement.InternalClientID, inboundIDs); err != nil {
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
		if _, err := w.clients.DetachByEmailMany(&w.inbounds, entitlement.InternalClientID, obsolete); err != nil {
			return err
		}
	}
	resetTraffic := policy.EventFor(order.OrderKind) == SubscriptionEventResetTraffic
	if resetTraffic {
		if _, err := w.clients.ResetTrafficByEmail(&w.inbounds, entitlement.InternalClientID); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	return w.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"plan_id":       plan.ID,
			"traffic_quota": plan.TrafficBytes,
			"device_limit":  plan.DeviceLimit,
			"node_group":    plan.NodeGroup,
			"expires_at":    *order.ResultExpiresAt,
		}
		if resetTraffic {
			updates["traffic_used"] = 0
			updates["last_reset_at"] = now
		}
		updated := tx.Model(&model.SubscriptionEntitlement{}).
			Where("id = ? AND customer_id = ? AND status = ?", entitlement.ID, order.CustomerID, "active").
			Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return errors.New("订阅状态已变更，停止处理")
		}
		completed := tx.Model(&model.Order{}).
			Where("id = ? AND status IN ?", order.ID, []string{OrderPaid, OrderProvisioning}).
			Updates(map[string]any{"status": OrderCompleted, "completed_at": now, "failure_reason": ""})
		if completed.Error != nil {
			return completed.Error
		}
		if completed.RowsAffected != 1 {
			return errors.New("订单状态已变更，停止处理")
		}
		payload, _ := json.Marshal(map[string]string{"orderId": order.ID, "entitlementId": entitlement.ID, "orderKind": order.OrderKind})
		event := model.OutboxEvent{ID: uuid.NewString(), AggregateType: "subscription", AggregateID: order.ID, EventType: "subscription." + order.OrderKind, Payload: string(payload), Status: "pending", NextRunAt: now}
		return tx.Where("aggregate_id = ? AND event_type = ?", order.ID, event.EventType).FirstOrCreate(&event).Error
	})
}

func (w *Worker) provisionInboundIDs(plan *model.Plan) ([]int, error) {
	var ids []int
	if plan.ProvisionInboundIDs != "" {
		if err := json.Unmarshal([]byte(plan.ProvisionInboundIDs), &ids); err != nil {
			return nil, errors.New("套餐入站配置无效")
		}
	}
	if len(ids) > 0 {
		return ids, nil
	}
	if err := w.db.Model(&model.Inbound{}).Where("enable = ?", true).Order("id asc").Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (w *Worker) completeOrder(orderID string) error {
	now := time.Now().UTC()
	return w.db.Model(&model.Order{}).Where("id = ? AND status IN ?", orderID, []string{OrderPaid, OrderProvisioning}).Updates(map[string]any{"status": OrderCompleted, "completed_at": now, "failure_reason": ""}).Error
}

func (w *Worker) processNextOutbox() error {
	var event model.OutboxEvent
	now := time.Now().UTC()
	err := w.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status IN ? AND next_run_at <= ?", []string{"pending", "retry"}, now).Order("created_at asc").First(&event).Error; err != nil {
			return err
		}
		return tx.Model(&event).Updates(map[string]any{"status": "running", "attempts": gorm.Expr("attempts + 1"), "last_error": ""}).Error
	})
	if err != nil {
		return err
	}
	event.Attempts++
	if err := w.deliverOutboxEvent(&event); err != nil {
		status := "retry"
		if event.Attempts >= 6 {
			status = "failed"
		}
		delayMinutes := math.Min(60, math.Pow(2, float64(event.Attempts-1)))
		nextRunAt := time.Now().UTC().Add(time.Duration(delayMinutes) * time.Minute)
		_ = w.db.Model(&model.OutboxEvent{}).Where("id = ? AND status = ?", event.ID, "running").Updates(map[string]any{
			"status": status, "next_run_at": nextRunAt, "last_error": err.Error(),
		}).Error
		return err
	}
	processedAt := time.Now().UTC()
	return w.db.Model(&model.OutboxEvent{}).Where("id = ? AND status = ?", event.ID, "running").Updates(map[string]any{
		"status": "processed", "processed_at": processedAt, "last_error": "",
	}).Error
}

func (w *Worker) deliverOutboxEvent(event *model.OutboxEvent) error {
	if event.EventType != "email.customer" {
		return nil
	}
	var payload customerEmailPayload
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Recipient) == "" || strings.TrimSpace(payload.Subject) == "" || strings.TrimSpace(payload.BodyHTML) == "" {
		return errors.New("customer email payload is incomplete")
	}
	if w.mailer == nil {
		return errors.New("customer email sender is unavailable")
	}
	return w.mailer.SendTo([]string{payload.Recipient}, payload.Subject, payload.BodyHTML)
}

func (w *Worker) reconcilePayments(ctx context.Context) error {
	var transactions []model.PaymentTransaction
	if err := w.db.Where("status = ? AND created_at > ?", "pending", time.Now().UTC().Add(-24*time.Hour)).Order("created_at asc").Limit(200).Find(&transactions).Error; err != nil {
		return err
	}
	for _, transaction := range transactions {
		var order model.Order
		if err := w.db.Where("id = ? AND status IN ?", transaction.OrderID, []string{OrderPending, OrderExpired}).First(&order).Error; err != nil {
			continue
		}
		provider, err := PaymentProviderByName(w.orders.config, transaction.Provider)
		if err != nil {
			continue
		}
		query, err := provider.Query(ctx, order.OutTradeNo)
		if err != nil || query == nil {
			continue
		}
		if query.TradeStatus == "TRADE_SUCCESS" || query.TradeStatus == "TRADE_FINISHED" {
			payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: query.ProviderTrade, AmountFen: query.AmountFen, TradeStatus: query.TradeStatus, RawPayload: "active-query"}
			_ = w.orders.markPaid(ctx, transaction.Provider, payment)
		}
	}
	return nil
}

func (w *Worker) expireOrders() {
	now := time.Now().UTC()
	var orders []model.Order
	if err := w.db.Select("id").Where("status = ? AND expires_at < ?", OrderPending, now).Limit(200).Find(&orders).Error; err != nil {
		return
	}
	for _, order := range orders {
		_ = w.orders.expireOrder(order.ID)
	}
	orders = nil
	cutoff := now.Add(-24 * time.Hour)
	if err := w.db.Select("id").Where("status = ? AND expires_at < ?", OrderExpired, cutoff).Limit(200).Find(&orders).Error; err != nil {
		return
	}
	for _, order := range orders {
		_ = w.orders.releaseExpiredOrder(order.ID, cutoff)
	}
}

func internalClientID(customerID, orderID string) string {
	sum := sha256.Sum256([]byte(customerID + ":" + orderID))
	return "c_" + hex.EncodeToString(sum[:10])
}

func resetDays(cycle string) int {
	switch cycle {
	case "daily":
		return 1
	case "weekly":
		return 7
	case "monthly":
		return 30
	case "quarterly":
		return 90
	default:
		return 0
	}
}
