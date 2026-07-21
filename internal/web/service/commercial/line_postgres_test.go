package commercial

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresLineConstraintsAndConcurrentUpsert(t *testing.T) {
	schema := initPostgresCommercialTest(t, "nova_line_")

	var constraintCount int64
	constraintNames := []string{
		"fk_commercial_line_source_nodes_source",
		"fk_commercial_line_source_nodes_node",
		"fk_commercial_line_group_nodes_group",
		"fk_commercial_line_group_nodes_node",
		"fk_commercial_plan_line_groups_plan",
		"fk_commercial_plan_line_groups_group",
		"fk_commercial_line_nodes_inbound",
	}
	if err := database.GetDB().Raw("SELECT COUNT(*) FROM pg_constraint constraints JOIN pg_class relations ON relations.oid = constraints.conrelid JOIN pg_namespace namespaces ON namespaces.oid = relations.relnamespace WHERE namespaces.nspname = ? AND constraints.conname IN ?", schema, constraintNames).Scan(&constraintCount).Error; err != nil {
		t.Fatal(err)
	}
	if constraintCount != int64(len(constraintNames)) {
		t.Fatalf("line foreign key count=%d want=%d", constraintCount, len(constraintNames))
	}

	group, err := NewLineService().SaveGroup(entity.CommercialLineGroupRequest{Name: "Concurrent", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseLineEntries("vless://same@8.8.8.8:443?type=tcp&security=none#Concurrent")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareLineEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 8
	sourceIDs := make([]string, workers)
	for index := range workers {
		sourceIDs[index] = uuid.NewString()
		source := model.LineSource{ID: sourceIDs[index], Name: fmt.Sprintf("source-%d", index), Kind: lineSourceManual, SecretCiphertext: "protected", Enabled: false, Status: "ready"}
		if err := database.GetDB().Create(&source).Error; err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var wait sync.WaitGroup
	for index := range workers {
		wait.Add(1)
		go func(sourceID string) {
			defer wait.Done()
			<-start
			errorsCh <- database.GetDB().Transaction(func(tx *gorm.DB) error {
				return NewLineService().upsertPreparedLines(tx, sourceID, prepared, []string{group.ID}, time.Now().UTC())
			})
		}(sourceIDs[index])
	}
	close(start)
	wait.Wait()
	close(errorsCh)
	for workerErr := range errorsCh {
		if workerErr != nil {
			t.Fatal(workerErr)
		}
	}
	var nodeCount, sourceMemberships, groupMemberships int64
	if err := database.GetDB().Model(&model.LineNode{}).Where("fingerprint = ?", prepared[0].Fingerprint).Count(&nodeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.LineSourceNode{}).Count(&sourceMemberships).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.LineGroupNode{}).Count(&groupMemberships).Error; err != nil {
		t.Fatal(err)
	}
	if nodeCount != 1 || sourceMemberships != workers || groupMemberships != 1 {
		t.Fatalf("concurrent upsert produced nodes=%d sourceMemberships=%d groupMemberships=%d", nodeCount, sourceMemberships, groupMemberships)
	}
	port := 23456
	firstPortNode := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("c", 64), Remark: "port-one", Protocol: "vless", OutboundTag: lineTagPrefix + "port-one", OutboundCiphertext: "protected", PublicPort: &port, Status: lineStatusOffline, HealthStatus: lineHealthOffline}
	secondPortNode := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("d", 64), Remark: "port-two", Protocol: "vless", OutboundTag: lineTagPrefix + "port-two", OutboundCiphertext: "protected", PublicPort: &port, Status: lineStatusOffline, HealthStatus: lineHealthOffline}
	if err := database.GetDB().Create(&firstPortNode).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&secondPortNode).Error; err == nil {
		t.Fatal("line public port unique constraint accepted a duplicate")
	}
	if err := database.GetDB().Create(&model.LineSourceNode{SourceID: uuid.NewString(), NodeID: uuid.NewString(), LastSeenAt: time.Now().UTC()}).Error; err == nil {
		t.Fatal("line source membership accepted missing foreign keys")
	}
}

func TestPostgresLineCommercialLifecycle(t *testing.T) {
	initPostgresCommercialTest(t, "nova_lifecycle_")
	db := database.GetDB()

	inboundOne := model.Inbound{Remark: "pg-line-one", Enable: true, Port: 25441, Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none"}`, StreamSettings: `{}`, Tag: "pg-line-one"}
	inboundTwo := model.Inbound{Remark: "pg-line-two", Enable: true, Port: 25442, Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none"}`, StreamSettings: `{}`, Tag: "pg-line-two"}
	if err := db.Create(&inboundOne).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&inboundTwo).Error; err != nil {
		t.Fatal(err)
	}
	oneBinding, _ := json.Marshal([]int{inboundOne.Id})
	twoBinding, _ := json.Marshal([]int{inboundTwo.Id})
	planOne := model.Plan{ID: uuid.NewString(), Slug: "pg-one", Name: "PG One", TrafficBytes: 100 << 30, DeviceLimit: 3, TrafficMultiplierPermille: 1000, ResetCycle: "monthly", Visibility: "public", Renewable: true, Upgradable: true, Active: true, SortOrder: 1, ProvisionInboundIDs: string(oneBinding)}
	planTwo := model.Plan{ID: uuid.NewString(), Slug: "pg-two", Name: "PG Two", TrafficBytes: 200 << 30, DeviceLimit: 5, TrafficMultiplierPermille: 1000, ResetCycle: "monthly", Visibility: "public", Renewable: true, Upgradable: true, Active: true, SortOrder: 2, ProvisionInboundIDs: string(twoBinding)}
	if err := db.Create(&planOne).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&planTwo).Error; err != nil {
		t.Fatal(err)
	}
	priceOne := model.PlanPrice{ID: uuid.NewString(), PlanID: planOne.ID, BillingPeriod: "monthly", Months: 1, AmountFen: 1000, Active: true}
	priceTwo := model.PlanPrice{ID: uuid.NewString(), PlanID: planTwo.ID, BillingPeriod: "monthly", Months: 1, AmountFen: 2000, Active: true}
	if err := db.Create(&priceOne).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&priceTwo).Error; err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "postgres-lifecycle@example.com", PasswordHash: "unused", Status: "active", InviteCode: "PGLIFE01"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}

	orders := NewOrderService()
	purchase, err := orders.Create(customer.ID, priceOne.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	markPostgresOrderPaid(t, orders, purchase, "pg-purchase")
	processPostgresProvisioningConcurrently(t)
	entitlement := requirePostgresEntitlement(t, customer.ID, planOne.ID)
	requirePostgresClientBindings(t, entitlement.InternalClientID, []int{inboundOne.Id})

	previousExpiry := *entitlement.ExpiresAt
	renewal, err := orders.CreateFor(customer.ID, priceOne.ID, "renewal", entitlement.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	markPostgresOrderPaid(t, orders, renewal, "pg-renewal")
	if err := NewWorker().processNextProvisioning(context.Background()); err != nil {
		t.Fatal(err)
	}
	entitlement = requirePostgresEntitlement(t, customer.ID, planOne.ID)
	if entitlement.ExpiresAt == nil || !entitlement.ExpiresAt.After(previousExpiry) {
		t.Fatalf("renewal did not extend expiry: before=%v after=%v", previousExpiry, entitlement.ExpiresAt)
	}

	upgrade, err := orders.CreateFor(customer.ID, priceTwo.ID, "upgrade", entitlement.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	markPostgresOrderPaid(t, orders, upgrade, "pg-upgrade")
	if err := NewWorker().processNextProvisioning(context.Background()); err != nil {
		t.Fatal(err)
	}
	entitlement = requirePostgresEntitlement(t, customer.ID, planTwo.ID)
	requirePostgresClientBindings(t, entitlement.InternalClientID, []int{inboundTwo.Id})

	downgraded, err := NewAdminService().UpsertSubscription(customer.ID, entity.CommercialSubscriptionUpdateRequest{PlanID: planOne.ID, ExpiresAt: time.Now().UTC().AddDate(0, 1, 0).Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	if downgraded.Entitlement.PlanID != planOne.ID {
		t.Fatalf("admin downgrade kept plan %s", downgraded.Entitlement.PlanID)
	}
	requirePostgresClientBindings(t, downgraded.Entitlement.InternalClientID, []int{inboundOne.Id})

	group := model.LineGroup{ID: uuid.NewString(), Name: "PG Dynamic", Active: true}
	inboundThree := model.Inbound{Remark: "pg-line-three", Enable: true, Port: 25443, Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none"}`, StreamSettings: `{}`, Tag: "pg-line-three"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&inboundThree).Error; err != nil {
		t.Fatal(err)
	}
	port := inboundThree.Port
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("e", 64), Remark: "pg-dynamic-node", Protocol: "vless", OutboundTag: lineTagPrefix + "pg-dynamic", OutboundCiphertext: "protected", PublicPort: &port, InboundID: &inboundThree.Id, Status: lineStatusHealthy, HealthStatus: lineHealthHealthy}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineGroupNode{GroupID: group.ID, NodeID: node.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.PlanLineGroup{PlanID: planOne.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatal(err)
	}
	worker := NewWorker()
	if err := worker.reconcileActivePlanClients(planOne.ID); err != nil {
		t.Fatal(err)
	}
	requirePostgresClientBindings(t, downgraded.Entitlement.InternalClientID, []int{inboundOne.Id, inboundThree.Id})
	if err := db.Model(&node).Updates(map[string]any{"status": lineStatusOffline, "health_status": lineHealthOffline}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&inboundThree).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.reconcileActivePlanClients(planOne.ID); err != nil {
		t.Fatal(err)
	}
	requirePostgresClientBindings(t, downgraded.Entitlement.InternalClientID, []int{inboundOne.Id, inboundThree.Id})
	if err := db.Model(&node).Updates(map[string]any{"status": lineStatusHealthy, "health_status": lineHealthHealthy}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&inboundThree).Update("enable", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.reconcileActivePlanClients(planOne.ID); err != nil {
		t.Fatal(err)
	}
	requirePostgresClientBindings(t, downgraded.Entitlement.InternalClientID, []int{inboundOne.Id, inboundThree.Id})
}

func initPostgresCommercialTest(t *testing.T, prefix string) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("XUI_TEST_PG_DSN"))
	if dsn == "" {
		t.Skip("set XUI_TEST_PG_DSN to run PostgreSQL commercial integration tests")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	schema := prefix + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := base.Exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, schema)).Error; err != nil {
		t.Fatal(err)
	}
	scopedDSN, err := postgresSchemaDSN(dsn, schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("XUI_DB_TYPE", "postgres")
	t.Setenv("XUI_DB_DSN", scopedDSN)
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	if err := database.InitDB(""); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = database.CloseDB()
		_ = base.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schema)).Error
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return schema
}

func markPostgresOrderPaid(t *testing.T, orders *OrderService, order *model.Order, trade string) {
	t.Helper()
	payment := &PaymentNotification{OutTradeNo: order.OutTradeNo, ProviderTrade: trade, AmountFen: order.PayableFen, TradeStatus: "TRADE_SUCCESS", RawPayload: "postgres-test"}
	if err := orders.markPaid(context.Background(), "alipay", payment); err != nil {
		t.Fatal(err)
	}
}

func processPostgresProvisioningConcurrently(t *testing.T) {
	t.Helper()
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			results <- NewWorker().processNextProvisioning(context.Background())
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal(err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("concurrent provisioning successes=%d want=1", succeeded)
	}
}

func requirePostgresEntitlement(t *testing.T, customerID, planID string) model.SubscriptionEntitlement {
	t.Helper()
	var entitlement model.SubscriptionEntitlement
	if err := database.GetDB().Where("customer_id = ? AND status = ?", customerID, "active").First(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	if entitlement.PlanID != planID {
		t.Fatalf("entitlement plan=%s want=%s", entitlement.PlanID, planID)
	}
	return entitlement
}

func requirePostgresClientBindings(t *testing.T, email string, want []int) {
	t.Helper()
	var client model.ClientRecord
	if err := database.GetDB().Where("email = ?", email).First(&client).Error; err != nil {
		t.Fatal(err)
	}
	var got []int
	if err := database.GetDB().Model(&model.ClientInbound{}).Where("client_id = ?", client.Id).Order("inbound_id asc").Pluck("inbound_id", &got).Error; err != nil {
		t.Fatal(err)
	}
	want = sortedUniqueInts(want)
	if len(got) != len(want) {
		t.Fatalf("client bindings=%v want=%v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("client bindings=%v want=%v", got, want)
		}
	}
}

func postgresSchemaDSN(dsn, schema string) (string, error) {
	if strings.Contains(dsn, "://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String(), nil
	}
	return dsn + " search_path=" + schema, nil
}
