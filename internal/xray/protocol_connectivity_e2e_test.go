package xray

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestXrayProtocolConnectivity_E2E proves that the Xray binary shipped in the
// Ubuntu archive can carry a real HTTP request through the non-VLESS protocols
// exposed by this panel. It deliberately tests traffic, rather than only
// asking Xray to parse a configuration file.
//
// The test is skipped during ordinary unit-test runs. Release/candidate builds
// set XRAY_E2E_BINARY after downloading the exact Xray runtime that will be
// packaged.
func TestXrayProtocolConnectivity_E2E(t *testing.T) {
	bin := os.Getenv("XRAY_E2E_BINARY")
	if bin == "" {
		t.Skip("set XRAY_E2E_BINARY to the packaged xray binary to run protocol connectivity tests")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("XRAY_E2E_BINARY: %v", err)
	}

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("nova-protocol-e2e-ok"))
	}))
	defer origin.Close()

	certFile, keyFile, certPin := writeProtocolTestCertificate(t)

	t.Run("trojan_tls", func(t *testing.T) {
		password := "trojan-e2e-password"
		serverPort := freePort(t)
		proxyPort := freePort(t)

		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "trojan", "tag": "trojan-e2e-in",
				"settings": map[string]any{"clients": []any{map[string]any{"password": password, "email": "trojan-e2e"}}},
				"streamSettings": map[string]any{
					"network": "tcp", "security": "tls",
					"tlsSettings": protocolTestTLSServerSettings(certFile, keyFile),
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "trojan", "tag": "trojan-e2e-out",
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": "127.0.0.1", "port": serverPort, "password": password,
			}}},
			"streamSettings": map[string]any{
				"network": "tcp", "security": "tls",
				"tlsSettings": protocolTestTLSClientSettings(certPin),
			},
		})

		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, origin.URL, false)
	})

	t.Run("hysteria2_tls", func(t *testing.T) {
		auth := "hysteria2-e2e-auth"
		serverPort := freeUDPPort(t)
		proxyPort := freePort(t)

		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "hysteria", "tag": "hysteria2-e2e-in",
				"settings": map[string]any{
					"version": 2,
					"clients": []any{map[string]any{"auth": auth, "email": "hysteria2-e2e"}},
				},
				"streamSettings": map[string]any{
					"network": "hysteria", "security": "tls",
					"tlsSettings":      protocolTestTLSServerSettings(certFile, keyFile),
					"hysteriaSettings": map[string]any{"version": 2, "udpIdleTimeout": 60},
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "hysteria", "tag": "hysteria2-e2e-out",
			"settings": map[string]any{"address": "127.0.0.1", "port": serverPort, "version": 2},
			"streamSettings": map[string]any{
				"network": "hysteria", "security": "tls",
				"tlsSettings": protocolTestTLSClientSettings(certPin),
				"hysteriaSettings": map[string]any{
					"version": 2, "auth": auth, "udpIdleTimeout": 60,
				},
			},
		})

		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, origin.URL, true)
	})
}

func protocolTestDirectOutbounds() []any {
	return []any{map[string]any{"protocol": "freedom", "settings": map[string]any{}, "tag": "direct"}}
}

func protocolTestHTTPClientConfig(proxyPort int, outbound map[string]any) map[string]any {
	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"listen": "127.0.0.1", "port": proxyPort, "protocol": "http", "tag": "local-http-proxy",
			"settings": map[string]any{},
		}},
		"outbounds": []any{outbound},
	}
}

func protocolTestTLSServerSettings(certFile, keyFile string) map[string]any {
	return map[string]any{
		"serverName": "e2e.local",
		"alpn":       []string{"h3", "http/1.1"},
		"certificates": []any{map[string]any{
			"certificateFile": certFile,
			"keyFile":         keyFile,
		}},
	}
}

func protocolTestTLSClientSettings(certPin string) map[string]any {
	return map[string]any{
		"serverName":           "e2e.local",
		"verifyPeerCertByName": "e2e.local",
		"pinnedPeerCertSha256": certPin,
		"alpn":                 []string{"h3", "http/1.1"},
	}
}

func runProtocolProxyCheck(
	t *testing.T,
	bin string,
	serverConfig, clientConfig map[string]any,
	serverPort, proxyPort int,
	originURL string,
	udpServer bool,
) {
	t.Helper()
	server := startProtocolXray(t, bin, serverConfig, "server")
	if udpServer {
		// A Hysteria2 listener is UDP-only, so there is no TCP readiness socket.
		// Keep the delay short; the actual proxied request below is the readiness
		// and functionality assertion.
		time.Sleep(500 * time.Millisecond)
	} else {
		waitForProtocolPort(t, serverPort, server)
	}
	client := startProtocolXray(t, bin, clientConfig, "client")
	waitForProtocolPort(t, proxyPort, client)

	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatal(err)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   12 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, originURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request through protocol tunnel failed: %v\nserver log:\n%s\nclient log:\n%s",
			err, server.log.String(), client.log.String())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read tunneled response: %v", err)
	}
	if string(body) != "nova-protocol-e2e-ok" {
		t.Fatalf("unexpected tunneled response %q", body)
	}
}

type protocolXrayProcess struct {
	cmd *exec.Cmd
	log *lockedBuffer
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func waitForProtocolPort(t *testing.T, port int, process *protocolXrayProcess) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("xray port %d did not open in time\nprocess log:\n%s", port, process.log.String())
}

func startProtocolXray(t *testing.T, bin string, config map[string]any, role string) *protocolXrayProcess {
	t.Helper()
	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), role+".json")
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	logs := &lockedBuffer{}
	cmd := exec.Command(bin, "-c", configPath)
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start xray %s: %v", role, err)
	}
	p := &protocolXrayProcess{cmd: cmd, log: logs}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	})
	return p
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

func writeProtocolTestCertificate(t *testing.T) (string, string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "e2e.local"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"e2e.local"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile := filepath.Join(dir, "e2e.crt")
	keyFile := filepath.Join(dir, "e2e.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		t.Fatal(err)
	}
	pin := sha256.Sum256(certDER)
	return certFile, keyFile, hex.EncodeToString(pin[:])
}
