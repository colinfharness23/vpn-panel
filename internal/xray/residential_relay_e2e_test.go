package xray

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestResidentialRelayChainsHysteria2CarrierToSOCKS5_E2E(t *testing.T) {
	bin := os.Getenv("XRAY_E2E_BINARY")
	if bin == "" {
		t.Skip("set XRAY_E2E_BINARY to the packaged xray binary")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatal(err)
	}

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("residential-exit-ok"))
	}))
	defer origin.Close()
	originURL, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatal(err)
	}

	socksPort, socksHits := startAuthenticatedRelaySOCKS5(t, "residential-user", "residential-password")
	certFile, keyFile, certPin := writeProtocolTestCertificate(t)
	carrierPort := freeUDPPort(t)
	panelPort := freeUDPPort(t)
	localProxyPort := freePort(t)
	carrierAuth := "carrier-hysteria-auth"
	customerAuth := "customer-hysteria-auth"
	customerEmail := "relay-customer"

	carrierConfig := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"listen": "127.0.0.1", "port": carrierPort, "protocol": "hysteria", "tag": "carrier-in",
			"settings": map[string]any{
				"version": 2,
				"clients": []any{map[string]any{"auth": carrierAuth, "email": "carrier-user"}},
			},
			"streamSettings": map[string]any{
				"network": "hysteria", "security": "tls",
				"tlsSettings":      protocolTestTLSServerSettings(certFile, keyFile),
				"hysteriaSettings": map[string]any{"version": 2, "udpIdleTimeout": 60},
			},
		}},
		"outbounds": protocolTestDirectOutbounds(),
	}
	carrierOutbound := map[string]any{
		"protocol": "hysteria", "tag": "commercial-line-hysteria",
		"settings": map[string]any{"address": "127.0.0.1", "port": carrierPort, "version": 2},
		"streamSettings": map[string]any{
			"network": "hysteria", "security": "tls",
			"tlsSettings": protocolTestTLSClientSettings(certPin),
			"hysteriaSettings": map[string]any{
				"version": 2, "auth": carrierAuth, "udpIdleTimeout": 60,
			},
		},
	}
	panelConfig := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"listen": "127.0.0.1", "port": panelPort, "protocol": "hysteria", "tag": "commercial-in-hysteria",
			"settings": map[string]any{
				"version": 2,
				"clients": []any{map[string]any{"auth": customerAuth, "email": customerEmail}},
			},
			"streamSettings": map[string]any{
				"network": "hysteria", "security": "tls",
				"tlsSettings":      protocolTestTLSServerSettings(certFile, keyFile),
				"hysteriaSettings": map[string]any{"version": 2, "udpIdleTimeout": 60},
			},
		}},
		"outbounds": []any{
			carrierOutbound,
			map[string]any{
				"protocol": "socks", "tag": "residential-relay-london",
				"settings": map[string]any{"servers": []any{map[string]any{
					"address": "127.0.0.1", "port": socksPort,
					"users": []any{map[string]any{"user": "residential-user", "pass": "residential-password"}},
				}}},
				"proxySettings": map[string]any{"tag": "commercial-line-hysteria"},
			},
			map[string]any{"protocol": "freedom", "tag": "direct"},
		},
		"routing": map[string]any{"rules": []any{
			map[string]any{
				"type": "field", "inboundTag": []string{"commercial-in-hysteria"},
				"user": []string{customerEmail}, "outboundTag": "residential-relay-london",
			},
			map[string]any{
				"type": "field", "inboundTag": []string{"commercial-in-hysteria"},
				"outboundTag": "commercial-line-hysteria",
			},
		}},
	}
	clientConfig := protocolTestHTTPClientConfig(localProxyPort, map[string]any{
		"protocol": "hysteria", "tag": "customer-out",
		"settings": map[string]any{"address": "127.0.0.1", "port": panelPort, "version": 2},
		"streamSettings": map[string]any{
			"network": "hysteria", "security": "tls",
			"tlsSettings": protocolTestTLSClientSettings(certPin),
			"hysteriaSettings": map[string]any{
				"version": 2, "auth": customerAuth, "udpIdleTimeout": 60,
			},
		},
	})

	carrier := startProtocolXray(t, bin, carrierConfig, "residential-carrier")
	time.Sleep(400 * time.Millisecond)
	panel := startProtocolXray(t, bin, panelConfig, "residential-panel")
	time.Sleep(400 * time.Millisecond)
	client := startProtocolXray(t, bin, clientConfig, "residential-client")
	waitForProtocolPort(t, localProxyPort, client)

	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", localProxyPort))
	if err != nil {
		t.Fatal(err)
	}
	httpClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 15 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("residential chain request failed: %v\ncarrier:\n%s\npanel:\n%s\nclient:\n%s",
			err, carrier.log.String(), panel.log.String(), client.log.String())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "residential-exit-ok" {
		t.Fatalf("unexpected exit response %q", body)
	}
	select {
	case target := <-socksHits:
		if target != originURL.Host {
			t.Fatalf("SOCKS5 exit dialed %q, want %q", target, originURL.Host)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request bypassed the residential SOCKS5 exit")
	}
}

func startAuthenticatedRelaySOCKS5(t *testing.T, username, password string) (int, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	hits := make(chan string, 8)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleAuthenticatedRelaySOCKS5(conn, username, password, hits)
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, hits
}

func handleAuthenticatedRelaySOCKS5(conn net.Conn, username, password string, hits chan<- string) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil || header[0] != 5 {
		return
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	_, _ = conn.Write([]byte{5, 2})

	if _, err := io.ReadFull(conn, header); err != nil || header[0] != 1 {
		return
	}
	usernameBytes := make([]byte, int(header[1]))
	if _, err := io.ReadFull(conn, usernameBytes); err != nil {
		return
	}
	length := []byte{0}
	if _, err := io.ReadFull(conn, length); err != nil {
		return
	}
	passwordBytes := make([]byte, int(length[0]))
	if _, err := io.ReadFull(conn, passwordBytes); err != nil {
		return
	}
	if string(usernameBytes) != username || string(passwordBytes) != password {
		_, _ = conn.Write([]byte{1, 1})
		return
	}
	_, _ = conn.Write([]byte{1, 0})

	request := make([]byte, 4)
	if _, err := io.ReadFull(conn, request); err != nil || request[0] != 5 || request[1] != 1 {
		return
	}
	var host string
	switch request[3] {
	case 1:
		raw := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return
		}
		host = net.IP(raw).String()
	case 3:
		if _, err := io.ReadFull(conn, length); err != nil {
			return
		}
		raw := make([]byte, int(length[0]))
		if _, err := io.ReadFull(conn, raw); err != nil {
			return
		}
		host = string(raw)
	case 4:
		raw := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return
		}
		host = net.IP(raw).String()
	default:
		return
	}
	rawPort := make([]byte, 2)
	if _, err := io.ReadFull(conn, rawPort); err != nil {
		return
	}
	target := net.JoinHostPort(host, fmt.Sprintf("%d", binary.BigEndian.Uint16(rawPort)))
	upstream, err := net.DialTimeout("tcp", target, 8*time.Second)
	if err != nil {
		_, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer upstream.Close()
	select {
	case hits <- target:
	default:
	}
	_, _ = conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	_ = conn.SetDeadline(time.Time{})
	_ = upstream.SetDeadline(time.Time{})
	go func() {
		_, _ = io.Copy(upstream, conn)
		_ = upstream.Close()
	}()
	_, _ = io.Copy(conn, upstream)
}
