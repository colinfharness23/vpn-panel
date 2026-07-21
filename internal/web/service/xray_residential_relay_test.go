package service

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

func TestInjectResidentialRelaysScopesRouteToCustomerAndInbound(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	if err := database.InitDB(filepath.Join(dbDir, "x-ui.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	db := database.GetDB()

	inbound := &model.Inbound{
		Tag:      "paid-line-1",
		Remark:   "Paid line 1",
		Enable:   true,
		Port:     18443,
		Protocol: model.VLESS,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	client := &model.ClientRecord{Email: "customer-internal-id", UUID: "dc7e35bb-e77a-474d-ab7b-c626df961957", Enable: true}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("create client: %v", err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatalf("bind client to inbound: %v", err)
	}
	entitlement := &model.SubscriptionEntitlement{
		ID: "entitlement-1", CustomerID: "customer-1", PlanID: "plan-1", OrderID: "order-1",
		InternalClientID: client.Email, SubscriptionID: "subscription-1", Status: "active",
		ResidentialRelayEnabled: true, ResidentialRelayLimit: 1, StartsAt: time.Now().UTC(),
	}
	if err := db.Create(entitlement).Error; err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	username, err := ProtectCredential("proxy-user")
	if err != nil {
		t.Fatalf("encrypt username: %v", err)
	}
	password, err := ProtectCredential("proxy-password")
	if err != nil {
		t.Fatalf("encrypt password: %v", err)
	}
	relay := &model.ResidentialRelay{
		ID: "relay-1", CustomerID: entitlement.CustomerID, EntitlementID: entitlement.ID,
		InboundID: inbound.Id, Name: "Home relay", OutboundTag: "residential-relay-test",
		SOCKSHost: "8.8.8.8", SOCKSPort: 1080, UsernameCiphertext: username,
		PasswordCiphertext: password, Status: "active",
	}
	if err := db.Create(relay).Error; err != nil {
		t.Fatalf("create relay: %v", err)
	}

	cfg := &xray.Config{
		OutboundConfigs: json_util.RawMessage(`[{"tag":"direct","protocol":"freedom"}]`),
		RouterConfig:    json_util.RawMessage(`{"rules":[{"type":"field","network":"tcp","outboundTag":"direct"}]}`),
	}
	injectResidentialRelays(cfg)

	var outbounds []map[string]any
	if err := json.Unmarshal(cfg.OutboundConfigs, &outbounds); err != nil {
		t.Fatalf("decode outbounds: %v", err)
	}
	if len(outbounds) != 2 || outbounds[1]["tag"] != relay.OutboundTag || outbounds[1]["protocol"] != "socks" {
		t.Fatalf("unexpected injected outbound: %#v", outbounds)
	}
	settings := outbounds[1]["settings"].(map[string]any)
	servers := settings["servers"].([]any)
	server := servers[0].(map[string]any)
	users := server["users"].([]any)
	credentials := users[0].(map[string]any)
	if credentials["user"] != "proxy-user" || credentials["pass"] != "proxy-password" {
		t.Fatalf("unexpected decrypted credentials: %#v", credentials)
	}

	var routing map[string]any
	if err := json.Unmarshal(cfg.RouterConfig, &routing); err != nil {
		t.Fatalf("decode routing: %v", err)
	}
	rules := routing["rules"].([]any)
	if len(rules) != 2 {
		t.Fatalf("expected relay rule before existing rule, got %#v", rules)
	}
	rule := rules[0].(map[string]any)
	if rule["outboundTag"] != relay.OutboundTag {
		t.Fatalf("unexpected outbound target: %#v", rule)
	}
	if got := rule["inboundTag"].([]any)[0]; got != inbound.Tag {
		t.Fatalf("route leaked beyond selected inbound: got %v", got)
	}
	if got := rule["user"].([]any)[0]; got != client.Email {
		t.Fatalf("route leaked beyond selected customer: got %v", got)
	}
}
