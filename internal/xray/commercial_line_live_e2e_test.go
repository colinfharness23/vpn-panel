package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	linkutil "github.com/mhsanaei/3x-ui/v3/internal/util/link"
)

// TestCommercialLineLiveLinks_E2E is intentionally opt-in because it uses
// administrator-supplied private upstream credentials. Links are read only
// from the process environment and are never printed or stored in fixtures.
// Each entry is tested twice: directly, then through the same local VLESS
// Reality wrapper and inbound-tag routing used by the commercial line worker.
func TestCommercialLineLiveLinks_E2E(t *testing.T) {
	bin := strings.TrimSpace(os.Getenv("XRAY_E2E_BINARY"))
	raw := strings.TrimSpace(os.Getenv("XRAY_LIVE_LINE_LINKS"))
	if bin == "" || raw == "" {
		t.Skip("set XRAY_E2E_BINARY and XRAY_LIVE_LINE_LINKS to run live commercial-line checks")
	}
	links := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	if len(links) == 0 {
		t.Fatal("XRAY_LIVE_LINE_LINKS did not contain any links")
	}

	decoy := newRealityDecoy(t)
	for index, rawLink := range links {
		parsed, err := linkutil.ParseLink(strings.TrimSpace(rawLink))
		if err != nil {
			t.Fatalf("parse live link %d: %v", index+1, err)
		}
		serialized, err := json.Marshal(parsed.Outbound)
		if err != nil {
			t.Fatalf("serialize live link %d: %v", index+1, err)
		}
		secured, _, err := linkutil.SecureTLSOutbound(context.Background(), string(serialized))
		if err != nil {
			t.Fatalf("secure live link %d: %v", index+1, err)
		}
		if err := json.Unmarshal([]byte(secured), &parsed.Outbound); err != nil {
			t.Fatalf("decode secured live link %d: %v", index+1, err)
		}
		protocol, _ := parsed.Outbound["protocol"].(string)
		name := fmt.Sprintf("%02d_%s", index+1, protocol)
		t.Run(name+"_direct", func(t *testing.T) {
			proxyPort := freePort(t)
			parsed.Outbound["tag"] = "live-upstream"
			client := protocolTestHTTPClientConfig(proxyPort, map[string]any(parsed.Outbound))
			process := startProtocolXray(t, bin, client, "live-direct")
			waitForProtocolPort(t, proxyPort, process)
			requestGenerate204(t, proxyPort, process)
		})
		t.Run(name+"_managed_reality", func(t *testing.T) {
			proxyPort := freePort(t)
			serverPort := freePort(t)
			privateKey, publicKey := realityTestKeyPair(t)
			shortID := fmt.Sprintf("%016x", index+1)
			clientID := fmt.Sprintf("33333333-4444-4555-8666-%012d", index+1)
			parsed.Outbound["tag"] = "live-upstream"
			server := map[string]any{
				"log": map[string]any{"loglevel": "warning"},
				"inbounds": []any{map[string]any{
					"listen": "127.0.0.1", "port": serverPort, "protocol": "vless", "tag": "commercial-managed-in",
					"settings": map[string]any{"clients": []any{map[string]any{"id": clientID, "email": "e2e"}}, "decryption": "none", "encryption": "none", "fallbacks": []any{}},
					"streamSettings": map[string]any{
						"network": "tcp", "security": "reality",
						"tcpSettings": map[string]any{"header": map[string]any{"type": "none"}},
						"realitySettings": map[string]any{
							"show": false, "xver": 0, "target": decoy.host, "serverNames": []string{"e2e.local"},
							"privateKey": privateKey, "shortIds": []string{shortID},
						},
					},
				}},
				"outbounds": []any{parsed.Outbound},
				"routing": map[string]any{"domainStrategy": "AsIs", "rules": []any{map[string]any{
					"type": "field", "inboundTag": []string{"commercial-managed-in"}, "outboundTag": "live-upstream",
				}}},
			}
			client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
				"protocol": "vless", "tag": "managed-reality-out",
				"settings": map[string]any{"vnext": []any{map[string]any{
					"address": "127.0.0.1", "port": serverPort,
					"users": []any{map[string]any{"id": clientID, "encryption": "none"}},
				}}},
				"streamSettings": map[string]any{
					"network": "tcp", "security": "reality",
					"tcpSettings": map[string]any{"header": map[string]any{"type": "none"}},
					"realitySettings": map[string]any{
						"show": false, "fingerprint": "chrome", "serverName": "e2e.local",
						"publicKey": publicKey, "shortId": shortID, "spiderX": "/",
					},
				},
			})
			serverProcess := startProtocolXray(t, bin, server, "live-managed-server")
			waitForProtocolPort(t, serverPort, serverProcess)
			clientProcess := startProtocolXray(t, bin, client, "live-managed-client")
			waitForProtocolPort(t, proxyPort, clientProcess)
			requestGenerate204(t, proxyPort, clientProcess)
		})
	}
}

type realityDecoy struct {
	host string
}

func newRealityDecoy(t *testing.T) realityDecoy {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	return realityDecoy{host: strings.TrimPrefix(server.URL, "https://")}
}

func requestGenerate204(t *testing.T, proxyPort int, process *protocolXrayProcess) {
	t.Helper()
	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 20 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.google.com/generate_204", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("generate_204 through line failed: %v\nxray log:\n%s", err, process.log.String())
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		t.Fatalf("generate_204 returned %d\nxray log:\n%s", response.StatusCode, process.log.String())
	}
}
