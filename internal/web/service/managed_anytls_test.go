package service

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestManagedAnyTLSOutboundConfig(t *testing.T) {
	raw := `{
	  "protocol":"anytls",
	  "settings":{"server":"203.0.113.45","serverPort":443,"password":"upstream-secret"},
	  "streamSettings":{"security":"tls","tlsSettings":{
	    "serverName":"edge.example.com",
	    "fingerprint":"chrome",
	    "alpn":["h2"],
	    "pinnedPeerPublicKeySha256":"KioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKio="
	  }}
	}`
	outbound, err := managedAnyTLSOutboundConfig(raw, "commercial-line-anytls")
	if err != nil {
		t.Fatal(err)
	}
	if outbound["type"] != "anytls" || outbound["tag"] != "commercial-line-anytls" ||
		outbound["server"] != "203.0.113.45" || outbound["server_port"] != 443 ||
		outbound["password"] != "upstream-secret" {
		t.Fatalf("unexpected outbound: %#v", outbound)
	}
	tlsConfig := outbound["tls"].(map[string]any)
	if tlsConfig["server_name"] != "edge.example.com" {
		t.Fatalf("server_name = %v", tlsConfig["server_name"])
	}
	pins, _ := tlsConfig["certificate_public_key_sha256"].([]string)
	if len(pins) != 1 || pins[0] != "KioqKioqKioqKioqKioqKioqKioqKioqKioqKioqKio=" {
		t.Fatalf("public-key pins = %#v", pins)
	}
	utls := tlsConfig["utls"].(map[string]any)
	if utls["enabled"] != true || utls["fingerprint"] != "chrome" {
		t.Fatalf("uTLS = %#v", utls)
	}
	if _, err := json.Marshal(outbound); err != nil {
		t.Fatal(err)
	}
}

func TestManagedAnyTLSConfigRoutesCustomerThroughResidentialSOCKS(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	if err := database.InitDB(filepath.Join(dbDir, "x-ui.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	db := database.GetDB()

	certificatePath, keyPath := writeManagedAnyTLSTestCertificate(t)
	inbound := &model.Inbound{
		Tag: "managed-anytls-in", Remark: "AnyTLS", Enable: true, Listen: "127.0.0.1", Port: 34443,
		Protocol: model.AnyTLS, Settings: `{}`,
		StreamSettings: `{"tlsSettings":{"certificates":[{"certificateFile":"` + filepath.ToSlash(certificatePath) + `","keyFile":"` + filepath.ToSlash(keyPath) + `"}]}}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := &model.ClientRecord{Email: "relay-customer", Password: "customer-secret", Enable: true}
	if err := db.Create(client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatal(err)
	}
	upstream, err := ProtectCredential(`{
	  "protocol":"anytls",
	  "settings":{"server":"203.0.113.45","serverPort":443,"password":"upstream-secret"},
	  "streamSettings":{"security":"tls","tlsSettings":{"serverName":"edge.example.com","allowInsecure":true}}
	}`)
	if err != nil {
		t.Fatal(err)
	}
	carrier := &model.LineNode{
		ID: "anytls-node", Fingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Remark: "AnyTLS upstream", PublicName: "AnyTLS upstream", Protocol: "anytls",
		OutboundTag: "commercial-line-anytls", OutboundCiphertext: upstream,
		InboundID: &inbound.Id, Status: "healthy", HealthStatus: "healthy",
	}
	if err := db.Create(carrier).Error; err != nil {
		t.Fatal(err)
	}
	entitlement := &model.SubscriptionEntitlement{
		ID: "entitlement-anytls", CustomerID: "customer-anytls", PlanID: "plan-anytls", OrderID: "order-anytls",
		InternalClientID: client.Email, SubscriptionID: "sub-anytls", Status: "active",
		ResidentialRelayEnabled: true, ResidentialRelayLimit: 1, StartsAt: time.Now().UTC(),
	}
	if err := db.Create(entitlement).Error; err != nil {
		t.Fatal(err)
	}
	username, err := ProtectCredential("socks-user")
	if err != nil {
		t.Fatal(err)
	}
	password, err := ProtectCredential("socks-password")
	if err != nil {
		t.Fatal(err)
	}
	relay := &model.ResidentialRelay{
		ID: "relay-anytls", CustomerID: entitlement.CustomerID, EntitlementID: entitlement.ID,
		InboundID: inbound.Id, Name: "London", OutboundTag: "residential-relay-anytls",
		SOCKSHost: "8.8.8.8", SOCKSPort: 1080, UsernameCiphertext: username,
		PasswordCiphertext: password, Status: "active",
	}
	if err := db.Create(relay).Error; err != nil {
		t.Fatal(err)
	}

	contents, count, err := buildManagedAnyTLSConfig(db)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("managed AnyTLS inbound count = %d", count)
	}
	var cfg managedAnyTLSConfig
	if err := json.Unmarshal(contents, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Outbounds) != 3 {
		t.Fatalf("outbounds = %#v", cfg.Outbounds)
	}
	relayOutbound := cfg.Outbounds[1].(map[string]any)
	if relayOutbound["tag"] != relay.OutboundTag || relayOutbound["detour"] != carrier.OutboundTag {
		t.Fatalf("residential AnyTLS outbound is not chained: %#v", relayOutbound)
	}
	rules := cfg.Route["rules"].([]any)
	relayRule := rules[0].(map[string]any)
	authUsers := relayRule["auth_user"].([]any)
	if relayRule["outbound"] != relay.OutboundTag || len(authUsers) != 1 || authUsers[0] != client.Email {
		t.Fatalf("residential AnyTLS rule is not customer-scoped: %#v", relayRule)
	}
	if binary := os.Getenv("SINGBOX_E2E_BINARY"); binary != "" {
		path := filepath.Join(t.TempDir(), "managed-anytls-residential.json")
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(binary, "check", "-c", path).CombinedOutput(); err != nil {
			t.Fatalf("sing-box rejected residential AnyTLS config: %v\n%s", err, output)
		}
	}
}

func TestAnyTLSCertificatePaths(t *testing.T) {
	certificate, key, err := anyTLSCertificatePaths(`{
	  "tlsSettings":{"certificates":[{"certificateFile":"/cert/fullchain.pem","keyFile":"/cert/privkey.pem"}]}
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if certificate != "/cert/fullchain.pem" || key != "/cert/privkey.pem" {
		t.Fatalf("certificate=%q key=%q", certificate, key)
	}
}

func TestManagedAnyTLSConfigAcceptedBySingBox(t *testing.T) {
	binary := os.Getenv("SINGBOX_E2E_BINARY")
	if binary == "" {
		t.Skip("set SINGBOX_E2E_BINARY to run the packaged-runtime contract")
	}
	outbound, err := managedAnyTLSOutboundConfig(`{
	  "protocol":"anytls",
	  "settings":{"server":"203.0.113.45","serverPort":443,"password":"upstream-secret"},
	  "streamSettings":{"security":"tls","tlsSettings":{"serverName":"edge.example.com","fingerprint":"chrome"}}
	}`, "probe-out")
	if err != nil {
		t.Fatal(err)
	}
	cfg := managedAnyTLSConfig{
		Log:       map[string]any{"level": "warn", "timestamp": true},
		Outbounds: []any{outbound},
	}
	certificatePath, keyPath := writeManagedAnyTLSTestCertificate(t)
	cfg.Inbounds = []any{map[string]any{
		"type": "anytls", "tag": "managed-anytls", "listen": "127.0.0.1", "listen_port": 32080,
		"users": []any{map[string]any{"name": "customer", "password": "customer-secret"}},
		"tls": map[string]any{
			"enabled": true, "certificate_path": certificatePath, "key_path": keyPath,
		},
	}}
	cfg.Route = map[string]any{
		"rules": []any{map[string]any{
			"inbound": []string{"managed-anytls"}, "action": "route", "outbound": "probe-out",
		}},
		"final": "probe-out", "auto_detect_interface": true,
	}
	contents, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sing-box.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(binary, "check", "-c", path).CombinedOutput(); err != nil {
		t.Fatalf("sing-box rejected generated AnyTLS config: %v\n%s", err, output)
	}
}

func writeManagedAnyTLSTestCertificate(t *testing.T) (string, string) {
	t.Helper()
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(tlsServer.Close)
	certificate := tlsServer.TLS.Certificates[0]
	certificatePath := filepath.Join(t.TempDir(), "fullchain.pem")
	keyPath := filepath.Join(t.TempDir(), "privkey.pem")
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]})
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})
	if err := os.WriteFile(certificatePath, certificatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, privateKeyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certificatePath, keyPath
}
