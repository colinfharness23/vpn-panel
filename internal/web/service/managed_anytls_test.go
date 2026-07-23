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
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer tlsServer.Close()
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
