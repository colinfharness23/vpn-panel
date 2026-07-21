package commercial

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
)

const lineTestLinks = `vmess://eyJ2IjoiMiIsInBzIjoidGVzdCIsImFkZCI6IjEuMi4zLjQiLCJwb3J0Ijo0NDMsImlkIjoidXVpZCIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiIiLCJob3N0IjoiZXguY29tIiwicGF0aCI6Ii8iLCJ0bHMiOiJ0bHMifQ==
vless://uuid@1.2.3.5:443?type=ws&security=tls&path=/&host=ex.com#VLESS
trojan://secret@1.2.3.6:443?type=tcp&security=tls&sni=example.com#Trojan
ss://YWVzLTI1Ni1nY206c2VjcmV0cGFzcw==@1.2.3.7:8388#SS
hysteria2://auth-secret@1.2.3.8:443?sni=example.com#HY2
wireguard://private-key@1.2.3.9:51820?publickey=public-key&address=10.0.0.2%2F32#WG
vless://uuid@1.2.3.5:443?type=ws&security=tls&path=/&host=ex.com#Renamed
invalid://value`

func TestLineImportSixProtocolsDeduplicatesAndEncrypts(t *testing.T) {
	initCommercialTestDB(t)
	service := NewLineService()
	preview, err := service.PreviewImport(lineTestLinks)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ValidCount != 6 || preview.DuplicateCount != 1 || preview.InvalidCount != 1 {
		t.Fatalf("preview counts = valid %d duplicate %d invalid %d", preview.ValidCount, preview.DuplicateCount, preview.InvalidCount)
	}
	protocols := map[string]bool{}
	for _, entry := range preview.Entries {
		if entry.Valid && !entry.Duplicate {
			protocols[entry.Protocol] = true
		}
	}
	for _, protocol := range []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria", "wireguard"} {
		if !protocols[protocol] {
			t.Fatalf("protocol %s missing from preview: %#v", protocol, protocols)
		}
	}
	source, err := service.CommitImport(entity.CommercialLineImportRequest{Name: "six protocols", Links: lineTestLinks})
	if err != nil {
		t.Fatal(err)
	}
	if source.NodeCount != 6 {
		t.Fatalf("source counted incorrectly: %+v", source)
	}
	encodedView, _ := json.Marshal(source)
	if strings.Contains(string(encodedView), "secretCiphertext") || strings.Contains(string(encodedView), "auth-secret") {
		t.Fatalf("source API view leaked encrypted or raw secrets: %s", encodedView)
	}
	var storedSource model.LineSource
	if err := database.GetDB().First(&storedSource, "id = ?", source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedSource.SecretCiphertext == "" || strings.Contains(storedSource.SecretCiphertext, "auth-secret") {
		t.Fatal("manual source was not encrypted")
	}
	if storedSource.Enabled {
		t.Fatal("manual source must not be scheduled for URL refresh")
	}
	var nodes []model.LineNode
	if err := database.GetDB().Order("protocol asc").Find(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 6 {
		t.Fatalf("stored nodes = %d, want 6", len(nodes))
	}
	for _, node := range nodes {
		if node.OutboundCiphertext == "" || strings.Contains(node.OutboundCiphertext, "secret") {
			t.Fatalf("node %s outbound was not encrypted", node.ID)
		}
	}
}

func TestLineAssignmentAndPlanInboundUnion(t *testing.T) {
	initCommercialTestDB(t)
	service := NewLineService()
	if _, err := service.CommitImport(entity.CommercialLineImportRequest{Name: "assign", Links: strings.Split(lineTestLinks, "\n")[1]}); err != nil {
		t.Fatal(err)
	}
	group, err := service.SaveGroup(entity.CommercialLineGroupRequest{Name: "Premium", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	var node model.LineNode
	if err := database.GetDB().First(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.AssignNodeGroups(entity.CommercialLineNodeGroupsRequest{NodeIDs: []string{node.ID}, GroupIDs: []string{group.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().First(&node, "id = ?", node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if node.Status != lineStatusChecking {
		t.Fatalf("assigned node status = %s", node.Status)
	}
	managedInbound := model.Inbound{Remark: "managed", Enable: true, Port: 25555, Protocol: model.VLESS, Settings: `{}`, StreamSettings: `{}`, Tag: "managed-line-test"}
	if err := database.GetDB().Create(&managedInbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&node).Updates(map[string]any{"status": lineStatusHealthy, "health_status": lineHealthHealthy, "inbound_id": managedInbound.Id, "public_port": managedInbound.Port}).Error; err != nil {
		t.Fatal(err)
	}
	plan := model.Plan{ID: uuid.NewString(), Slug: "line-union", Name: "Line Union", TrafficBytes: 1, ResetCycle: "monthly", Visibility: "public", ProvisionInboundIDs: `[24443]`}
	if err := database.GetDB().Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.PlanLineGroup{PlanID: plan.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatal(err)
	}
	ids, err := NewWorker().provisionInboundIDs(&plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != managedInbound.Id || ids[1] != 24443 {
		t.Fatalf("inbound union = %v", ids)
	}
}

func TestAssignNodeGroupsReconcilesPlansFromRemovedGroups(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	service := NewLineService()
	oldGroup, err := service.SaveGroup(entity.CommercialLineGroupRequest{Name: "Old", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	newGroup, err := service.SaveGroup(entity.CommercialLineGroupRequest{Name: "New", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	clientEmail := uuid.NewString()
	clientUUID := uuid.NewString()
	settings, err := json.Marshal(map[string][]model.Client{"clients": {{ID: clientUUID, Email: clientEmail, Enable: true}}})
	if err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{Remark: "old group", Enable: true, Port: 25556, Protocol: model.VLESS, Settings: string(settings), StreamSettings: `{}`, Tag: "managed-line-old-group"}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("f", 64), Remark: "move", Protocol: "vless", OutboundTag: lineTagPrefix + "move", OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id, Status: lineStatusHealthy, HealthStatus: lineHealthHealthy}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineGroupNode{GroupID: oldGroup.ID, NodeID: node.ID}).Error; err != nil {
		t.Fatal(err)
	}
	plan := model.Plan{ID: uuid.NewString(), Slug: "old-group-plan", Name: "Old Group Plan", TrafficBytes: 1, ResetCycle: "monthly", Visibility: "public", ProvisionInboundIDs: `[]`}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.PlanLineGroup{PlanID: plan.ID, GroupID: oldGroup.ID}).Error; err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "removed-group@example.com", PasswordHash: "unused", Status: "active", InviteCode: "OLDGROUP1"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	client := model.ClientRecord{Email: clientEmail, UUID: clientUUID, Enable: true, TrafficMultiplierPermille: 1000}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().UTC().Add(time.Hour)
	entitlement := model.SubscriptionEntitlement{ID: uuid.NewString(), CustomerID: customer.ID, PlanID: plan.ID, OrderID: uuid.NewString(), InternalClientID: clientEmail, SubscriptionID: uuid.NewString(), Status: "active", StartsAt: time.Now().UTC(), ExpiresAt: &expiresAt}
	if err := db.Create(&entitlement).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveGroup(entity.CommercialLineGroupRequest{ID: oldGroup.ID, Name: oldGroup.Name, Active: false}); err != nil {
		t.Fatal(err)
	}
	var bindingCount int64
	if err := db.Model(&model.ClientInbound{}).Where("client_id = ?", client.Id).Count(&bindingCount).Error; err != nil {
		t.Fatal(err)
	}
	if bindingCount != 0 {
		t.Fatalf("disabled group left %d stale client bindings", bindingCount)
	}
	if _, err := service.SaveGroup(entity.CommercialLineGroupRequest{ID: oldGroup.ID, Name: oldGroup.Name, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ClientInbound{}).Where("client_id = ?", client.Id).Count(&bindingCount).Error; err != nil {
		t.Fatal(err)
	}
	if bindingCount != 1 {
		t.Fatalf("re-enabled group restored %d client bindings, want 1", bindingCount)
	}
	if err := service.AssignNodeGroups(entity.CommercialLineNodeGroupsRequest{NodeIDs: []string{node.ID}, GroupIDs: []string{newGroup.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ClientInbound{}).Where("client_id = ?", client.Id).Count(&bindingCount).Error; err != nil {
		t.Fatal(err)
	}
	if bindingCount != 0 {
		t.Fatalf("removed group left %d stale client bindings", bindingCount)
	}
}

func TestInactiveLineGroupStaysInactiveOnCreate(t *testing.T) {
	initCommercialTestDB(t)
	group, err := NewLineService().SaveGroup(entity.CommercialLineGroupRequest{Name: "Draft", Active: false})
	if err != nil {
		t.Fatal(err)
	}
	if group.Active {
		t.Fatal("inactive line group was changed to active by database defaults")
	}
}

func TestLineSourceSSRFAndStableFingerprint(t *testing.T) {
	initCommercialTestDB(t)
	service := NewLineService()
	if _, _, err := service.fetchURLSource(context.Background(), "http://127.0.0.1:8080/sub"); err == nil {
		t.Fatal("private subscription URL was accepted")
	}
	links := "vless://same@8.8.8.8:443?type=tcp&security=none#First\n" +
		"vless://same@8.8.8.8:443?type=tcp&security=none#Second"
	preview, err := service.PreviewImport(links)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ValidCount != 1 || preview.DuplicateCount != 1 || preview.Entries[0].Fingerprint != preview.Entries[1].Fingerprint {
		t.Fatalf("remark change altered identity: %+v", preview)
	}
}

func TestLineHealthThresholds(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	inbound := model.Inbound{Remark: "health", Enable: true, Port: 26666, Protocol: model.VLESS, Settings: `{}`, StreamSettings: `{}`, Tag: "health-line-test"}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("a", 64), Remark: "health", Protocol: "vless", OutboundTag: lineTagPrefix + "health", OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id, Status: lineStatusHealthy, HealthStatus: lineHealthHealthy}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	worker := NewWorker()
	base := time.Now().UTC().Unix()
	for index := 1; index <= 3; index++ {
		_ = worker.applyLineHealthEvent(eventbus.Event{Source: node.OutboundTag, Data: &eventbus.OutboundHealthData{Alive: false, Error: "down", LastTryTime: base + int64(index)}})
		if err := db.First(&node, "id = ?", node.ID).Error; err != nil {
			t.Fatal(err)
		}
		if index < 3 && node.Status != lineStatusHealthy {
			t.Fatalf("node went offline after %d failures", index)
		}
	}
	if node.Status != lineStatusOffline {
		t.Fatalf("node status after three failures = %s", node.Status)
	}
	for index := 4; index <= 5; index++ {
		_ = worker.applyLineHealthEvent(eventbus.Event{Source: node.OutboundTag, Data: &eventbus.OutboundHealthData{Alive: true, LastTryTime: base + int64(index)}})
	}
	if err := db.First(&node, "id = ?", node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if node.Status != lineStatusHealthy {
		t.Fatalf("node did not recover after two passes: %s", node.Status)
	}
}

func TestStaleLineRequiresMatureSourceBeforeCleanup(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	old := time.Now().UTC().Add(-8 * 24 * time.Hour)
	source := model.LineSource{ID: uuid.NewString(), Name: "stale", Kind: lineSourceURL, SecretCiphertext: "encrypted", Enabled: true, Status: "ready", RefreshInterval: 1800, ConsecutiveSuccesses: 1}
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("b", 64), Remark: "stale", Protocol: "vless", OutboundTag: lineTagPrefix + "stale", OutboundCiphertext: "encrypted", Status: lineStatusStale, HealthStatus: lineHealthOffline, MissingSince: &old}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineSourceNode{SourceID: source.ID, NodeID: node.ID, LastSeenAt: old, MissingSince: &old}).Error; err != nil {
		t.Fatal(err)
	}
	eligibleMissingSince := old.Add(time.Hour)
	eligible := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("c", 64), Remark: "eligible", Protocol: "vless", OutboundTag: lineTagPrefix + "eligible", OutboundCiphertext: "encrypted", Status: lineStatusStale, HealthStatus: lineHealthOffline, MissingSince: &eligibleMissingSince}
	if err := db.Create(&eligible).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewWorker().cleanupStaleLine(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.LineNode{}).Where("id = ?", node.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("immature stale source removed node: count=%d err=%v", count, err)
	}
	if err := db.Model(&model.LineNode{}).Where("id = ?", eligible.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("immature stale source starved eligible cleanup: count=%d err=%v", count, err)
	}
	if err := db.Model(&source).Update("consecutive_successes", 2).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewWorker().cleanupStaleLine(); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.LineNode{}).Where("id = ?", node.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("mature stale source kept node: count=%d err=%v", count, err)
	}
}
