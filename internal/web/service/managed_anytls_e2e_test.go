package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/xray"

	netproxy "golang.org/x/net/proxy"
)

func TestManagedAnyTLSChainE2E(t *testing.T) {
	binary := os.Getenv("SINGBOX_E2E_BINARY")
	if binary == "" {
		t.Skip("set SINGBOX_E2E_BINARY to run the AnyTLS chain test")
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
	if err := os.WriteFile(certificatePath, certificatePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}), 0o600); err != nil {
		t.Fatal(err)
	}

	upstreamPort := reserveAnyTLSTestPort(t)
	managedPort := reserveAnyTLSTestPort(t)
	statsPort := reserveAnyTLSTestPort(t)
	clientPort := reserveAnyTLSTestPort(t)
	upstream := map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type": "anytls", "tag": "upstream-in", "listen": "127.0.0.1", "listen_port": upstreamPort,
			"users": []any{map[string]any{"name": "nova", "password": "upstream-secret"}},
			"tls":   map[string]any{"enabled": true, "certificate_path": certificatePath, "key_path": keyPath},
		}},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		"route":     map[string]any{"rules": []any{}, "final": "direct", "auto_detect_interface": true},
	}
	managed := map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type": "anytls", "tag": "managed-in", "listen": "127.0.0.1", "listen_port": managedPort,
			"users": []any{map[string]any{"name": "customer", "password": "customer-secret"}},
			"tls":   map[string]any{"enabled": true, "certificate_path": certificatePath, "key_path": keyPath},
		}},
		"outbounds": []any{map[string]any{
			"type": "anytls", "tag": "upstream-out", "server": "127.0.0.1",
			"server_port": upstreamPort, "password": "upstream-secret",
			"tls": map[string]any{"enabled": true, "insecure": true},
		}},
		"route": map[string]any{
			"rules": []any{map[string]any{
				"inbound": []string{"managed-in"}, "action": "route", "outbound": "upstream-out",
			}},
			"final": "upstream-out", "auto_detect_interface": true,
		},
		"experimental": map[string]any{"v2ray_api": map[string]any{
			"listen": net.JoinHostPort("127.0.0.1", strconv.Itoa(statsPort)),
			"stats": map[string]any{
				"enabled": true, "inbounds": []string{"managed-in"}, "users": []string{"customer"},
			},
		}},
	}
	clientConfig := map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type": "mixed", "tag": "client-in", "listen": "127.0.0.1", "listen_port": clientPort,
		}},
		"outbounds": []any{map[string]any{
			"type": "anytls", "tag": "managed-out", "server": "127.0.0.1",
			"server_port": managedPort, "password": "customer-secret",
			"tls": map[string]any{"enabled": true, "insecure": true},
		}},
		"route": map[string]any{"rules": []any{}, "final": "managed-out", "auto_detect_interface": true},
	}

	stopUpstream := startAnyTLSTestRuntime(t, binary, upstream, upstreamPort)
	defer stopUpstream()
	stopManaged := startAnyTLSTestRuntime(t, binary, managed, managedPort)
	defer stopManaged()
	stopClient := startAnyTLSTestRuntime(t, binary, clientConfig, clientPort)
	defer stopClient()

	statsAPI := new(xray.XrayAPI)
	if err := statsAPI.Init(statsPort); err != nil {
		t.Fatal(err)
	}
	defer statsAPI.Close()
	if _, _, err := statsAPI.GetTrafficWithMethod(managedAnyTLSStatsMethod); err != nil {
		t.Fatalf("initialize managed AnyTLS traffic baseline: %v", err)
	}

	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(response, "managed-anytls-ok")
	}))
	defer target.Close()
	dialer, err := netproxy.SOCKS5("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(clientPort)), nil, netproxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{DialContext: func(_ context.Context, network, address string) (net.Conn, error) {
		return dialer.Dial(network, address)
	}}
	defer transport.CloseIdleConnections()
	response, err := (&http.Client{Transport: transport, Timeout: 10 * time.Second}).Get(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "managed-anytls-ok" {
		t.Fatalf("response body = %q", body)
	}
	var inboundTraffic []*xray.Traffic
	var clientTraffic []*xray.ClientTraffic
	var inboundBytes int64
	var clientBytes int64
	deadline := time.Now().Add(3 * time.Second)
	for {
		inboundTraffic, clientTraffic, err = statsAPI.GetTrafficWithMethod(managedAnyTLSStatsMethod)
		inboundBytes += anyTLSTestInboundBytes(inboundTraffic)
		clientBytes += anyTLSTestClientBytes(clientTraffic)
		if err == nil && inboundBytes > 0 && clientBytes > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("managed AnyTLS traffic was not reported: inbound=%s clients=%s totals=%d/%d err=%v",
				mustAnyTLSTestJSON(inboundTraffic), mustAnyTLSTestJSON(clientTraffic), inboundBytes, clientBytes, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func mustAnyTLSTestJSON(value any) string {
	contents, _ := json.Marshal(value)
	return string(contents)
}

func anyTLSTestInboundBytes(inbound []*xray.Traffic) int64 {
	var inboundBytes int64
	for _, traffic := range inbound {
		if traffic != nil && traffic.IsInbound && traffic.Tag == "managed-in" {
			inboundBytes += traffic.Up + traffic.Down
		}
	}
	return inboundBytes
}

func anyTLSTestClientBytes(clients []*xray.ClientTraffic) int64 {
	var clientBytes int64
	for _, traffic := range clients {
		if traffic != nil && traffic.Email == "customer" {
			clientBytes += traffic.Up + traffic.Down
		}
	}
	return clientBytes
}

func reserveAnyTLSTestPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}

func startAnyTLSTestRuntime(t *testing.T, binary string, config map[string]any, port int) func() {
	t.Helper()
	contents, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(binary, "check", "-c", configPath).CombinedOutput(); err != nil {
		t.Fatalf("sing-box check failed: %v\n%s", err, output)
	}
	cmd := exec.Command(binary, "run", "-c", configPath)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, output)
		_ = cmd.Wait()
		close(done)
	}()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(5 * time.Second)
	for {
		connection, dialErr := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if dialErr == nil {
			_ = connection.Close()
			break
		}
		select {
		case <-done:
			t.Fatal("sing-box exited before its listener became ready")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("sing-box listener did not become ready")
		}
		time.Sleep(50 * time.Millisecond)
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("sing-box did not stop")
		}
	}
}
