package commercial

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	webservice "github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func TestWaitManagedLineListenerRequiresRealTCPListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := waitManagedLineListener(context.Background(), port, 2*time.Second); err != nil {
		t.Fatalf("open listener rejected: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := waitManagedLineListener(context.Background(), port, 600*time.Millisecond); err == nil {
		t.Fatal("closed listener was accepted")
	}
}

func TestManagedAnyTLSWithoutSubscribersRemainsEnabled(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	inbound := &model.Inbound{
		UserId: 1, Enable: true, Port: 34567, Protocol: model.AnyTLS,
		Settings: `{"clients":[]}`, StreamSettings: `{"network":"tcp","security":"tls"}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatal(err)
	}

	needsListener, err := managedAnyTLSNeedsListener(db, inbound.Id)
	if err != nil {
		t.Fatal(err)
	}
	if needsListener {
		t.Fatal("subscriber-free managed AnyTLS line must not wait for a listener")
	}
	if !inbound.Enable {
		t.Fatal("subscriber-free managed AnyTLS inbound must remain enabled for the first client")
	}

	client := &model.ClientRecord{Email: "anytls@example.com", Password: "secret", Enable: true}
	if err := db.Create(client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatal(err)
	}
	needsListener, err = managedAnyTLSNeedsListener(db, inbound.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !needsListener {
		t.Fatal("managed AnyTLS line with an enabled subscriber must verify its listener")
	}
}

func TestManagedLineInboundPreservesImportedProtocolAndAliasInputs(t *testing.T) {
	initCommercialTestDB(t)
	if err := NewConfigStore().SetManyProtected(map[string]string{
		"line.reality_private_key": "private-key",
		"line.reality_public_key":  "public-key",
	}, map[string]bool{"line.reality_private_key": true}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile := dir + "/fullchain.pem"
	keyFile := dir + "/privkey.pem"
	if err := os.WriteFile(certFile, []byte("certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("private key"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XUI_LINE_CERT_FILE", certFile)
	t.Setenv("XUI_LINE_KEY_FILE", keyFile)

	tests := []struct {
		imported string
		want     model.Protocol
		network  string
	}{
		{"vmess", model.VMESS, "ws"},
		{"vless", model.VLESS, "ws"},
		{"trojan", model.Trojan, "ws"},
		{"shadowsocks", model.Shadowsocks, "tcp"},
		{"hysteria", model.Hysteria, "hysteria"},
		{"wireguard", model.WireGuard, ""},
		{"anytls", model.AnyTLS, "tcp"},
	}
	for _, tc := range tests {
		t.Run(tc.imported, func(t *testing.T) {
			protocol := managedLineInboundProtocol(tc.imported)
			if protocol != tc.want {
				t.Fatalf("protocol = %q, want %q", protocol, tc.want)
			}
			node := &model.LineNode{Fingerprint: strings.Repeat("a", 64), Protocol: tc.imported, PublicName: "IPLC"}
			settingsJSON, streamJSON, err := managedLineInboundJSON(node, protocol, "vpn.pheero.com", 23456)
			if err != nil {
				t.Fatal(err)
			}
			var settings, stream map[string]any
			if err := json.Unmarshal(settingsJSON, &settings); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(streamJSON, &stream); err != nil {
				t.Fatal(err)
			}
			if tc.want != model.WireGuard {
				if _, ok := settings["clients"]; !ok {
					t.Fatalf("%s settings have no clients array: %s", tc.imported, settingsJSON)
				}
			}
			if got, _ := stream["network"].(string); got != tc.network {
				t.Fatalf("%s network = %q, want %q", tc.imported, got, tc.network)
			}
			if tc.network == "ws" {
				ws, _ := stream["wsSettings"].(map[string]any)
				if got, _ := ws["path"].(string); got != "/nova-line/23456/aaaaaaaaaaaaaaaa" {
					t.Fatalf("%s websocket path = %q", tc.imported, got)
				}
				proxies, _ := stream["externalProxy"].([]any)
				if len(proxies) != 1 {
					t.Fatalf("%s external proxy = %#v", tc.imported, proxies)
				}
				ep, _ := proxies[0].(map[string]any)
				if ep["dest"] != "vpn.pheero.com" || int(ep["port"].(float64)) != 443 || ep["forceTls"] != "tls" {
					t.Fatalf("%s public endpoint = %#v", tc.imported, ep)
				}
			}
		})
	}
}

func TestManagedLineUpgradeMovesVLESSBehindStandard443WebSocket(t *testing.T) {
	initCommercialTestDB(t)
	if err := NewConfigStore().SetManyProtected(map[string]string{
		"site.url":                 "https://vpn.pheero.com",
		"line.reality_private_key": "private-key",
		"line.reality_public_key":  "public-key",
	}, map[string]bool{"line.reality_private_key": true}); err != nil {
		t.Fatal(err)
	}
	db := database.GetDB()
	inbound := model.Inbound{
		UserId: 1, Remark: "IPLC", Enable: true, Listen: "0.0.0.0", Port: 23456,
		Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none","encryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"reality"}`, Tag: "commercial-in-upgrade",
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.ClientRecord{
		Email: "upgrade@example.com", SubID: "upgrade-sub",
		UUID: "11111111-2222-4333-8444-555555555555", Enable: true,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{
		ClientId: client.Id, InboundId: inbound.Id, FlowOverride: "xtls-rprx-vision",
	}).Error; err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{
		ID: uuid.NewString(), Fingerprint: strings.Repeat("a", 64), Remark: "provider",
		PublicName: "IPLC", Protocol: "vless", OutboundTag: lineTagPrefix + "upgrade",
		OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id,
		Status: lineStatusReady, HealthStatus: lineHealthHealthy, ProvisionVersion: 3,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewWorker().ensureManagedLineInbound(&node); err != nil {
		t.Fatal(err)
	}
	var got model.Inbound
	if err := db.First(&got, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Listen != "127.0.0.1" || got.Port != 23456 {
		t.Fatalf("upgraded listener = %s:%d", got.Listen, got.Port)
	}
	var stream map[string]any
	if err := json.Unmarshal([]byte(got.StreamSettings), &stream); err != nil {
		t.Fatal(err)
	}
	if stream["network"] != "ws" || stream["security"] != "none" {
		t.Fatalf("upgraded stream = %#v", stream)
	}
	var binding model.ClientInbound
	if err := db.Where("client_id = ? AND inbound_id = ?", client.Id, inbound.Id).First(&binding).Error; err != nil {
		t.Fatal(err)
	}
	if binding.FlowOverride != "" {
		t.Fatalf("VLESS WebSocket retained incompatible flow %q", binding.FlowOverride)
	}
	if node.ProvisionVersion != managedLineProvisionVersion {
		t.Fatalf("provision version = %d", node.ProvisionVersion)
	}
}

func TestManagedLineUpgradeRepairsEmptyAnyTLSStreamSettings(t *testing.T) {
	initCommercialTestDB(t)
	if err := NewConfigStore().SetManyProtected(map[string]string{
		"site.url": "https://vpn.pheero.com",
	}, nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile := dir + "/fullchain.pem"
	keyFile := dir + "/privkey.pem"
	if err := os.WriteFile(certFile, []byte("certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("private key"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XUI_LINE_CERT_FILE", certFile)
	t.Setenv("XUI_LINE_KEY_FILE", keyFile)

	db := database.GetDB()
	inbound := model.Inbound{
		UserId: 1, Remark: "AnyTLS", Enable: true, Listen: "0.0.0.0", Port: 34567,
		Protocol: model.AnyTLS, Settings: `{"clients":[]}`, StreamSettings: "",
		Tag: "commercial-in-anytls-upgrade",
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{
		ID: uuid.NewString(), Fingerprint: strings.Repeat("b", 64), Remark: "provider",
		PublicName: "AnyTLS", Protocol: "anytls", OutboundTag: lineTagPrefix + "anytls-upgrade",
		OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id,
		Status: lineStatusReady, HealthStatus: lineHealthHealthy, ProvisionVersion: managedLineProvisionVersion - 1,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewWorker().ensureManagedLineInbound(&node); err != nil {
		t.Fatal(err)
	}
	var got model.Inbound
	if err := db.First(&got, inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	var stream map[string]any
	if err := json.Unmarshal([]byte(got.StreamSettings), &stream); err != nil {
		t.Fatalf("repaired AnyTLS stream settings are invalid: %v; raw=%q", err, got.StreamSettings)
	}
	if stream["network"] != "tcp" || stream["security"] != "tls" {
		t.Fatalf("repaired AnyTLS stream = %#v", stream)
	}
	if node.ProvisionVersion != managedLineProvisionVersion {
		t.Fatalf("provision version = %d", node.ProvisionVersion)
	}
}

func TestManagedWebSocketIngressBackendValidatesNodeTokenAndRuntime(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	port := 23456
	node := model.LineNode{
		ID: uuid.NewString(), Fingerprint: strings.Repeat("a", 64), Remark: "provider",
		PublicName: "IPLC", Protocol: "vless", OutboundTag: lineTagPrefix + "ingress-auth",
		OutboundCiphertext: "encrypted", PublicPort: &port, Status: lineStatusReady,
		HealthStatus: lineHealthHealthy, ProvisionVersion: managedLineProvisionVersion,
	}
	path := managedLineWSPathPrefix + "23456/" + managedLineWebSocketPathToken(&node)
	inbound := model.Inbound{
		UserId: 1, Remark: "IPLC", Enable: true, Listen: "127.0.0.1", Port: port,
		Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none"}`,
		StreamSettings: `{"network":"ws","security":"none","wsSettings":{"path":"` + path + `","host":"vpn.pheero.com"}}`,
		Tag:            "commercial-in-ingress-auth",
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	node.InboundID = &inbound.Id
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	lineService := NewLineService()
	if backend, err := lineService.ManagedWebSocketIngressBackend(port, strings.Repeat("a", 16)); err != nil || backend != port {
		t.Fatalf("valid ingress = %d, %v", backend, err)
	}
	if _, err := lineService.ManagedWebSocketIngressBackend(port, strings.Repeat("b", 16)); err == nil {
		t.Fatal("incorrect line token was accepted")
	}
	if err := db.Model(&inbound).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := lineService.ManagedWebSocketIngressBackend(port, strings.Repeat("a", 16)); err == nil {
		t.Fatal("disabled inbound was accepted")
	}
}

const lineTestLinks = `vmess://eyJ2IjoiMiIsInBzIjoidGVzdCIsImFkZCI6IjEuMi4zLjQiLCJwb3J0Ijo0NDMsImlkIjoidXVpZCIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiIiLCJob3N0IjoiZXguY29tIiwicGF0aCI6Ii8iLCJ0bHMiOiJ0bHMifQ==
vless://uuid@1.2.3.5:443?type=ws&security=tls&path=/&host=ex.com#VLESS
trojan://secret@1.2.3.6:443?type=tcp&security=tls&sni=example.com#Trojan
ss://YWVzLTI1Ni1nY206c2VjcmV0cGFzcw==@1.2.3.7:8388#SS
hysteria2://auth-secret@1.2.3.8:443?sni=example.com#HY2
wireguard://private-key@1.2.3.9:51820?publickey=public-key&address=10.0.0.2%2F32#WG
anytls://anytls-secret@1.2.3.10:443?sni=example.com#AnyTLS
vless://uuid@1.2.3.5:443?type=ws&security=tls&path=/&host=ex.com#Renamed
invalid://value`

func TestLineImportSevenProtocolsDeduplicatesAndEncrypts(t *testing.T) {
	initCommercialTestDB(t)
	service := NewLineService()
	preview, err := service.PreviewImport(lineTestLinks)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ValidCount != 7 || preview.DuplicateCount != 1 || preview.InvalidCount != 1 {
		t.Fatalf("preview counts = valid %d duplicate %d invalid %d", preview.ValidCount, preview.DuplicateCount, preview.InvalidCount)
	}
	protocols := map[string]bool{}
	for _, entry := range preview.Entries {
		if entry.Valid && !entry.Duplicate {
			protocols[entry.Protocol] = true
		}
	}
	for _, protocol := range []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria", "wireguard", "anytls"} {
		if !protocols[protocol] {
			t.Fatalf("protocol %s missing from preview: %#v", protocol, protocols)
		}
	}
	source, err := service.CommitImport(entity.CommercialLineImportRequest{Name: "seven protocols", Links: lineTestLinks})
	if err != nil {
		t.Fatal(err)
	}
	if source.NodeCount != 7 {
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
	if len(nodes) != 7 {
		t.Fatalf("stored nodes = %d, want 7", len(nodes))
	}
	for _, node := range nodes {
		if node.OutboundCiphertext == "" || strings.Contains(node.OutboundCiphertext, "secret") {
			t.Fatalf("node %s outbound was not encrypted", node.ID)
		}
		if strings.TrimSpace(node.PublicName) == "" {
			t.Fatalf("node %s has no user-facing alias", node.ID)
		}
	}
	updated, err := service.UpdateNode(nodes[0].ID, entity.CommercialLineNodeUpdateRequest{PublicName: "香港高速线路"})
	if err != nil || updated.PublicName != "香港高速线路" {
		t.Fatalf("update public alias: row=%+v err=%v", updated, err)
	}
}

func TestLineImportAcceptsAnyTLSWithTLSCompatibilityParameters(t *testing.T) {
	initCommercialTestDB(t)
	// Keep the provider credential supplied for live acceptance out of source
	// control. This synthetic link preserves the same scheme, TLS compatibility
	// flags, TCP transport and percent-encoded Unicode fragment.
	const raw = "anytls://11111111-2222-4333-8444-555555555555@anytls.example.com:55551?security=tls&sni=anytls.example.com&insecure=1&allowInsecure=1&type=tcp#%F0%9F%87%B8%F0%9F%87%AC%E6%96%B0%E5%8A%A0%E5%9D%A102-AWS%E7%94%B5%E4%BF%A1%E4%BC%98%E5%8C%96"

	lineService := NewLineService()
	preview, err := lineService.PreviewImport(raw)
	if err != nil {
		t.Fatalf("preview AnyTLS link: %v", err)
	}
	if preview.ValidCount != 1 || preview.InvalidCount != 0 || len(preview.Entries) != 1 {
		t.Fatalf("unexpected AnyTLS preview: %+v", preview)
	}
	entry := preview.Entries[0]
	if !entry.Valid || entry.Protocol != "anytls" || entry.Remark != "🇸🇬新加坡02-AWS电信优化" {
		t.Fatalf("unexpected AnyTLS entry: %+v", entry)
	}

	source, err := lineService.CommitImport(entity.CommercialLineImportRequest{
		Name:  "AnyTLS exact-link regression",
		Links: raw,
	})
	if err != nil {
		t.Fatalf("commit AnyTLS link: %v", err)
	}
	if source.NodeCount != 1 {
		t.Fatalf("AnyTLS source node count = %d, want 1", source.NodeCount)
	}

	var node model.LineNode
	if err := database.GetDB().First(&node, "protocol = ?", "anytls").Error; err != nil {
		t.Fatalf("load imported AnyTLS node: %v", err)
	}
	if node.Protocol != "anytls" || strings.TrimSpace(node.OutboundCiphertext) == "" {
		t.Fatalf("stored AnyTLS node is incomplete: %+v", node)
	}
}

func TestManualLineDefaultAliasDoesNotLeakShareLinkRemark(t *testing.T) {
	source := model.LineSource{Kind: lineSourceManual, Name: "Manual import"}
	entry := preparedLineEntry{Protocol: "trojan", Remark: "A机场 香港 IPLC"}
	if got, want := defaultPublicLineName(source, entry, 2), "TROJAN 线路 #3"; got != want {
		t.Fatalf("manual default alias = %q, want %q", got, want)
	}
}

func TestURLLineDefaultAliasDoesNotLeakSourceOrProviderBrand(t *testing.T) {
	source := model.LineSource{Kind: lineSourceURL, Name: "A机场 Premium"}
	entry := preparedLineEntry{Protocol: "vless", Remark: "A机场 香港 IPLC"}
	got := defaultPublicLineName(source, entry, 0)
	if got != "VLESS 线路 #1" {
		t.Fatalf("URL default alias = %q", got)
	}
	if strings.Contains(got, source.Name) || strings.Contains(got, entry.Remark) {
		t.Fatalf("URL default alias leaked upstream metadata: %q", got)
	}
}

func TestURLRefreshScrubsLegacyGeneratedAliasAndPreservesCustomAlias(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	source := model.LineSource{ID: uuid.NewString(), Kind: lineSourceURL, Name: "A机场", SecretCiphertext: "encrypted", RefreshInterval: 1800, Status: "ready"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	entry := preparedLineEntry{
		Fingerprint: strings.Repeat("9", 64), Remark: "A机场 香港", Protocol: "vless",
		OutboundTag: lineTagPrefix + "legacy", OutboundCiphertext: "encrypted",
	}
	legacy := model.LineNode{
		ID: uuid.NewString(), Fingerprint: entry.Fingerprint, Remark: entry.Remark,
		PublicName: "A机场 · VLESS #1", Protocol: entry.Protocol, OutboundTag: entry.OutboundTag,
		OutboundCiphertext: entry.OutboundCiphertext, Status: lineStatusUnassigned, HealthStatus: lineHealthUnchecked,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	service := NewLineService()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return service.upsertPreparedLines(tx, source.ID, []preparedLineEntry{entry}, nil, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&legacy, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.PublicName != "VLESS 线路 #1" || legacy.PublicNameCustom {
		t.Fatalf("legacy generated alias was not scrubbed: %+v", legacy)
	}
	if _, err := service.UpdateNode(legacy.ID, entity.CommercialLineNodeUpdateRequest{PublicName: "香港高速线路"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return service.upsertPreparedLines(tx, source.ID, []preparedLineEntry{entry}, nil, time.Now().UTC())
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&legacy, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.PublicName != "香港高速线路" || !legacy.PublicNameCustom {
		t.Fatalf("custom alias was overwritten by refresh: %+v", legacy)
	}
}

func TestLineSourceUpstreamErrorsArePreciseAndSafe(t *testing.T) {
	body := []byte(`{"status":"fail","message":"token is error","data":null,"error":null}`)
	if got := lineSourceHTTPError(403, body).Error(); got != "获取订阅失败：订阅令牌无效或已失效（HTTP 403）" {
		t.Fatalf("HTTP error = %q", got)
	}
	if got := lineSourcePayloadError(body); got == nil || got.Error() != "获取订阅失败：订阅令牌无效或已失效" {
		t.Fatalf("200 error envelope = %v", got)
	}
	secretEcho := []byte(`{"status":"fail","message":"denied https://provider.example/s/secret-token"}`)
	if got := lineSourceHTTPError(403, secretEcho).Error(); strings.Contains(got, "secret-token") || got != "获取订阅失败：上游订阅服务拒绝访问（HTTP 403）" {
		t.Fatalf("HTTP error leaked upstream response: %q", got)
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

func TestLiveLineSourceURL(t *testing.T) {
	rawURL := strings.TrimSpace(os.Getenv("NOVA_TEST_LIVE_SUBSCRIPTION_URL"))
	if rawURL == "" {
		t.Skip("set NOVA_TEST_LIVE_SUBSCRIPTION_URL to run the live subscription integration test")
	}
	_, entries, err := NewLineService().fetchURLSource(context.Background(), rawURL)
	if err != nil {
		t.Fatalf("live subscription fetch failed: %v", err)
	}
	valid := 0
	for _, entry := range entries {
		if entry.Valid {
			valid++
		}
	}
	if valid == 0 {
		t.Fatal("live subscription returned no valid supported nodes")
	}
	t.Logf("live subscription parsed %d supported nodes", valid)
}

func TestLineHealthIsAdvisoryAndDoesNotUnpublish(t *testing.T) {
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
	}
	if node.Status != lineStatusHealthy || node.HealthStatus != lineHealthOffline {
		t.Fatalf("probe result changed publication status: status=%s health=%s", node.Status, node.HealthStatus)
	}
	if err := db.First(&inbound, "id = ?", inbound.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !inbound.Enable {
		t.Fatal("three failed probes disabled a published inbound")
	}
	views, err := NewLineService().Nodes("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || !views[0].Published {
		t.Fatalf("published API state no longer reflects the enabled inbound: %+v", views)
	}
	for index := 4; index <= 5; index++ {
		_ = worker.applyLineHealthEvent(eventbus.Event{Source: node.OutboundTag, Data: &eventbus.OutboundHealthData{Alive: true, LastTryTime: base + int64(index)}})
	}
	if err := db.First(&node, "id = ?", node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if node.Status != lineStatusHealthy || node.HealthStatus != lineHealthHealthy {
		t.Fatalf("probe health did not recover without changing publication: status=%s health=%s", node.Status, node.HealthStatus)
	}
}

func TestRestoreManagedLinePublicationQueuesLegacyVLESSWrapperForProtocolMigration(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	group := model.LineGroup{ID: uuid.NewString(), Name: "Published", Active: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{Remark: "legacy", Enable: true, Port: 26667, Protocol: model.VLESS, Settings: `{}`, StreamSettings: `{}`, Tag: "legacy-line-test"}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("d", 64), Remark: "legacy", Protocol: "trojan", OutboundTag: lineTagPrefix + "legacy", OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id, Status: lineStatusOffline, HealthStatus: lineHealthOffline}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineGroupNode{GroupID: group.ID, NodeID: node.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewWorker().restoreManagedLinePublication(); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&node, "id = ?", node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if node.Status != lineStatusChecking || node.HealthStatus != lineHealthChecking {
		t.Fatalf("legacy wrapper was not queued for protocol migration: status=%s health=%s", node.Status, node.HealthStatus)
	}
}

func TestRestoreManagedLinePublicationQueuesLegacyInsecureTLSNode(t *testing.T) {
	initCommercialTestDB(t)
	db := database.GetDB()
	group := model.LineGroup{ID: uuid.NewString(), Name: "Legacy TLS", Active: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{Remark: "legacy-tls", Enable: true, Port: 26668, Protocol: model.VLESS, Settings: `{}`, StreamSettings: `{}`, Tag: "legacy-tls-line-test"}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	outbound := `{"protocol":"trojan","settings":{"servers":[{"address":"example.com","port":443}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com","allowInsecure":true}}}`
	protected, err := webservice.ProtectCredential(outbound)
	if err != nil {
		t.Fatal(err)
	}
	port := inbound.Port
	node := model.LineNode{ID: uuid.NewString(), Fingerprint: strings.Repeat("e", 64), Remark: "legacy-tls", Protocol: "trojan", OutboundTag: lineTagPrefix + "legacy-tls", OutboundCiphertext: protected, PublicPort: &port, InboundID: &inbound.Id, Status: lineStatusOffline, HealthStatus: lineHealthOffline}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineGroupNode{GroupID: group.ID, NodeID: node.ID}).Error; err != nil {
		t.Fatal(err)
	}

	if err := NewWorker().restoreManagedLinePublication(); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&node, "id = ?", node.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !node.TLSAutoPinned || node.Status != lineStatusChecking || node.HealthStatus != lineHealthChecking {
		t.Fatalf("legacy insecure TLS node was not queued for certificate pinning: %+v", node)
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
