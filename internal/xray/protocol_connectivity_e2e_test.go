package xray

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestXrayProtocolConnectivity_E2E(t *testing.T) {
	bin := os.Getenv("XRAY_E2E_BINARY")
	if bin == "" {
		t.Skip("set XRAY_E2E_BINARY to the packaged xray binary to run protocol connectivity tests")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("XRAY_E2E_BINARY: %v", err)
	}

	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("nova-protocol-e2e-ok"))
	}))
	origin.Listener = protocolTestListener(t)
	origin.Start()
	defer origin.Close()
	originPort := origin.Listener.Addr().(*net.TCPAddr).Port
	loopbackOriginURL := fmt.Sprintf("http://127.0.0.1:%d", originPort)

	certFile, keyFile, certPin := writeProtocolTestCertificate(t)
	decoy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer decoy.Close()

	t.Run("vmess", func(t *testing.T) {
		id := "11111111-2222-4333-8444-555555555555"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "vmess", "tag": "vmess-e2e-in",
				"settings": map[string]any{"clients": []any{map[string]any{"id": id, "security": "auto"}}},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "vmess", "tag": "vmess-e2e-out",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": "127.0.0.1", "port": serverPort,
				"users": []any{map[string]any{"id": id, "security": "auto"}},
			}}},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

	t.Run("vless", func(t *testing.T) {
		id := "22222222-3333-4444-8555-666666666666"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "vless", "tag": "vless-e2e-in",
				"settings": map[string]any{"decryption": "none", "clients": []any{map[string]any{"id": id}}},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "vless", "tag": "vless-e2e-out",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": "127.0.0.1", "port": serverPort,
				"users": []any{map[string]any{"id": id, "encryption": "none"}},
			}}},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

	t.Run("managed_vless_reality", func(t *testing.T) {
		id := "33333333-4444-4555-8666-777777777777"
		privateKey, publicKey := realityTestKeyPair(t)
		shortID := "a1b2c3d4e5f60708"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		decoyURL, err := url.Parse(decoy.URL)
		if err != nil {
			t.Fatal(err)
		}
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "vless", "tag": "managed-reality-e2e-in",
				"settings": map[string]any{"decryption": "none", "clients": []any{map[string]any{"id": id}}},
				"streamSettings": map[string]any{
					"network": "tcp", "security": "reality",
					"realitySettings": map[string]any{
						"show": false, "target": decoyURL.Host, "serverNames": []string{"e2e.local"},
						"privateKey": privateKey, "shortIds": []string{shortID},
					},
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "vless", "tag": "managed-reality-e2e-out",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": "127.0.0.1", "port": serverPort,
				"users": []any{map[string]any{"id": id, "encryption": "none"}},
			}}},
			"streamSettings": map[string]any{
				"network": "tcp", "security": "reality",
				"realitySettings": map[string]any{
					"show": false, "fingerprint": "chrome", "serverName": "e2e.local",
					"publicKey": publicKey, "shortId": shortID, "spiderX": "/",
				},
			},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

	t.Run("managed_vless_tls_websocket", func(t *testing.T) {
		id := "44444444-5555-4666-8777-888888888888"
		path := "/nova-line/23456/aaaaaaaaaaaaaaaa"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		publicPort := startProtocolTLSWebSocketProxy(t, certFile, keyFile, serverPort, path, "e2e.local")
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "vless", "tag": "managed-vless-ws-e2e-in",
				"settings": map[string]any{"decryption": "none", "clients": []any{map[string]any{"id": id}}},
				"streamSettings": map[string]any{
					"network": "ws", "security": "none",
					"wsSettings": map[string]any{"path": path, "host": "e2e.local"},
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "vless", "tag": "managed-vless-ws-e2e-out",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": "127.0.0.1", "port": publicPort,
				"users": []any{map[string]any{"id": id, "encryption": "none"}},
			}}},
			"streamSettings": map[string]any{
				"network": "ws", "security": "tls",
				"wsSettings":  map[string]any{"path": path, "host": "e2e.local"},
				"tlsSettings": protocolTestTLSClientSettings(certPin),
			},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

	t.Run("shadowsocks", func(t *testing.T) {
		password := "shadowsocks-e2e-password"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "shadowsocks", "tag": "shadowsocks-e2e-in",
				"settings": map[string]any{"method": "aes-256-gcm", "password": password, "network": "tcp,udp"},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "shadowsocks", "tag": "shadowsocks-e2e-out",
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": "127.0.0.1", "port": serverPort, "method": "aes-256-gcm", "password": password,
			}}},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

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

		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
	})

	t.Run("managed_trojan_tls_websocket", func(t *testing.T) {
		password := "managed-trojan-ws-e2e-password"
		path := "/nova-line/23457/bbbbbbbbbbbbbbbb"
		serverPort := freePort(t)
		proxyPort := freePort(t)
		publicPort := startProtocolTLSWebSocketProxy(t, certFile, keyFile, serverPort, path, "e2e.local")
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "trojan", "tag": "managed-trojan-ws-e2e-in",
				"settings": map[string]any{"clients": []any{map[string]any{"password": password, "email": "managed-trojan-ws-e2e"}}},
				"streamSettings": map[string]any{
					"network": "ws", "security": "none",
					"wsSettings": map[string]any{"path": path, "host": "e2e.local"},
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "trojan", "tag": "managed-trojan-ws-e2e-out",
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": "127.0.0.1", "port": publicPort, "password": password,
			}}},
			"streamSettings": map[string]any{
				"network": "ws", "security": "tls",
				"wsSettings":  map[string]any{"path": path, "host": "e2e.local"},
				"tlsSettings": protocolTestTLSClientSettings(certPin),
			},
		})
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, false)
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

		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, loopbackOriginURL, true)
	})

	t.Run("wireguard", func(t *testing.T) {
		serverPrivate, serverPublic := wireguardTestKeyPair(t)
		clientPrivate, clientPublic := wireguardTestKeyPair(t)
		serverPort := freeUDPPort(t)
		proxyPort := freePort(t)
		server := map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{
				"listen": "127.0.0.1", "port": serverPort, "protocol": "wireguard", "tag": "wireguard-e2e-in",
				"settings": map[string]any{
					"secretKey": serverPrivate,
					"peers":     []any{map[string]any{"publicKey": clientPublic, "allowedIPs": []string{"10.77.0.2/32"}}},
				},
			}},
			"outbounds": protocolTestDirectOutbounds(),
		}
		client := protocolTestHTTPClientConfig(proxyPort, map[string]any{
			"protocol": "wireguard", "tag": "wireguard-e2e-out",
			"settings": map[string]any{
				"secretKey": clientPrivate, "address": []string{"10.77.0.2/32"}, "noKernelTun": true,
				"peers": []any{map[string]any{"endpoint": fmt.Sprintf("127.0.0.1:%d", serverPort), "publicKey": serverPublic, "allowedIPs": []string{"0.0.0.0/0"}}},
			},
		})
		wireguardOriginURL := fmt.Sprintf("http://%s:%d", protocolTestPrimaryIPv4(t), originPort)
		runProtocolProxyCheck(t, bin, server, client, serverPort, proxyPort, wireguardOriginURL, true)
	})
}

func protocolTestListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener
}

func protocolTestPrimaryIPv4(t *testing.T) string {
	t.Helper()
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ip := conn.LocalAddr().(*net.UDPAddr).IP
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
		t.Fatalf("could not determine a non-loopback test address: %v", ip)
	}
	return ip.String()
}

func wireguardTestKeyPair(t *testing.T) (string, string) {
	t.Helper()
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(privateKey.Bytes()), base64.StdEncoding.EncodeToString(privateKey.PublicKey().Bytes())
}

func realityTestKeyPair(t *testing.T) (string, string) {
	t.Helper()
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(privateKey.Bytes()), base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes())
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

func startProtocolTLSWebSocketProxy(t *testing.T, certFile, keyFile string, backendPort int, path, host string) int {
	t.Helper()
	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", backendPort))
	if err != nil {
		t.Fatal(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path || r.Host != host {
			http.NotFound(w, r)
			return
		}
		reverseProxy.ServeHTTP(w, r)
	}))
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	proxy.TLS = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
	}
	proxy.StartTLS()
	t.Cleanup(proxy.Close)
	return proxy.Listener.Addr().(*net.TCPAddr).Port
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
