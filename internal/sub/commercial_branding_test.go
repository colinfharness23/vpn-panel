package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestManagedLineSubscriptionHidesImportedProviderRemark(t *testing.T) {
	initSubDB(t)
	db := database.GetDB()
	if err := db.Create(&model.CommercialSetting{Key: "site.name", Value: "Pheero VPN"}).Error; err != nil {
		t.Fatalf("seed site name: %v", err)
	}
	inbound := &model.Inbound{
		UserId: 1, Remark: "A机场 香港 IPLC", Enable: true, Port: 24443,
		Protocol: model.VLESS, Settings: `{"clients":[]}`, StreamSettings: `{"network":"tcp","security":"none"}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("seed inbound: %v", err)
	}
	client := &model.ClientRecord{Email: "customer@example.com", SubID: "brand-sub", UUID: "11111111-2222-4333-8444-555555555555", Enable: true}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatalf("seed client inbound: %v", err)
	}
	lineID := "abcd1234-1111-4222-8333-abcdefabcdef"
	if err := db.Create(&model.LineNode{
		ID: lineID, Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Remark: "A机场 香港 IPLC", PublicName: "香港 IPLC", Protocol: "vless", OutboundTag: "commercial-line-test",
		OutboundCiphertext: "encrypted", InboundID: &inbound.Id, Status: "healthy", HealthStatus: "healthy",
	}).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}

	rows, err := (&SubService{}).getInboundsBySubId("brand-sub")
	if err != nil {
		t.Fatalf("get inbounds: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d inbounds, want 1", len(rows))
	}
	if got, want := rows[0].Remark, "香港 IPLC"; got != want {
		t.Fatalf("managed line remark = %q, want %q", got, want)
	}

	links, _, _, _, err := NewSubService("").GetSubs("brand-sub", "vpn.pheero.com")
	if err != nil || len(links) != 1 {
		t.Fatalf("render managed subscription: links=%v err=%v", links, err)
	}
	if strings.Contains(links[0], "A机场") {
		t.Fatalf("rendered subscription leaked upstream branding: %q", links[0])
	}
	parsed, err := url.Parse(links[0])
	if err != nil {
		t.Fatalf("parse managed subscription link: %v", err)
	}
	if got, want := parsed.Fragment, "香港 IPLC"; got != want {
		t.Fatalf("rendered node name = %q, want %q", got, want)
	}
}

func TestManagedWebSocketIngressPublishesStandardTLS443(t *testing.T) {
	initSubDB(t)
	db := database.GetDB()
	inbound := &model.Inbound{
		UserId: 1, Remark: "IPLC", Enable: true, Listen: "127.0.0.1", Port: 23456,
		Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none","encryption":"none"}`,
		StreamSettings: `{
			"network":"ws",
			"security":"none",
			"wsSettings":{"path":"/nova-line/23456/aaaaaaaaaaaaaaaa","host":"vpn.pheero.com"},
			"externalProxy":[{
				"dest":"vpn.pheero.com",
				"port":443,
				"forceTls":"tls",
				"sni":"vpn.pheero.com",
				"fingerprint":"chrome",
				"verifyPeerCertByName":"vpn.pheero.com"
			}]
		}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := &model.ClientRecord{
		Email: "standard-port@example.com", SubID: "standard-port-sub",
		UUID: "11111111-2222-4333-8444-555555555555", Enable: true,
	}
	if err := db.Create(client).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LineNode{
		ID: "11111111-1111-4111-8111-111111111111", Fingerprint: strings.Repeat("a", 64),
		Remark: "provider", PublicName: "IPLC", Protocol: "vless", OutboundTag: "commercial-line-standard-port",
		OutboundCiphertext: "encrypted", InboundID: &inbound.Id, Status: "ready", HealthStatus: "healthy",
	}).Error; err != nil {
		t.Fatal(err)
	}

	links, _, _, _, err := NewSubService("").GetSubs("standard-port-sub", "vpn.pheero.com")
	if err != nil || len(links) != 1 {
		t.Fatalf("links=%v err=%v", links, err)
	}
	parsed, err := url.Parse(links[0])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "vless" || parsed.Hostname() != "vpn.pheero.com" || parsed.Port() != "443" {
		t.Fatalf("public endpoint = %s://%s", parsed.Scheme, parsed.Host)
	}
	for key, want := range map[string]string{
		"type": "ws", "security": "tls", "sni": "vpn.pheero.com",
		"host": "vpn.pheero.com", "path": "/nova-line/23456/aaaaaaaaaaaaaaaa",
		"fp": "chrome", "vcn": "vpn.pheero.com",
	} {
		if got := parsed.Query().Get(key); got != want {
			t.Fatalf("%s = %q, want %q in %q", key, got, want, links[0])
		}
	}
	if parsed.Fragment != "IPLC" {
		t.Fatalf("alias = %q, want IPLC", parsed.Fragment)
	}
}

func TestManagedSubscriptionPreservesSevenProtocolTypesAndExactAliases(t *testing.T) {
	initSubDB(t)
	db := database.GetDB()
	if err := db.Create(&model.CommercialSetting{Key: "site.name", Value: "PHEERO"}).Error; err != nil {
		t.Fatal(err)
	}
	privateKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("p", 32)))
	client := &model.ClientRecord{
		Email: "six@example.com", SubID: "six-protocols", UUID: "11111111-2222-4333-8444-555555555555",
		Password: "trojan-and-ss-password", Auth: "hysteria-auth", Security: "auto",
		PrivateKey: privateKey, AllowedIPs: "10.0.0.2/32", Enable: true,
	}
	if err := db.Create(client).Error; err != nil {
		t.Fatal(err)
	}
	streams := map[model.Protocol]string{
		model.VMESS:       `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"vpn.pheero.com","settings":{"fingerprint":"chrome"}}}`,
		model.VLESS:       `{"network":"tcp","security":"reality","realitySettings":{"serverNames":["vpn.pheero.com"],"shortIds":["abcd1234"],"settings":{"publicKey":"PBK","fingerprint":"chrome","spiderX":"/"}}}`,
		model.Trojan:      `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"vpn.pheero.com","settings":{"fingerprint":"chrome"}}}`,
		model.Shadowsocks: `{}`,
		model.Hysteria:    `{"network":"hysteria","security":"tls","tlsSettings":{"serverName":"vpn.pheero.com","settings":{"fingerprint":"chrome"}},"hysteriaSettings":{"version":2}}`,
		model.WireGuard:   `{}`,
		model.AnyTLS:      `{"network":"tcp","security":"tls","tlsSettings":{"serverName":"vpn.pheero.com"}}`,
	}
	settings := map[model.Protocol]string{
		model.VMESS:       `{"clients":[]}`,
		model.VLESS:       `{"clients":[],"decryption":"none"}`,
		model.Trojan:      `{"clients":[]}`,
		model.Shadowsocks: `{"method":"2022-blake3-aes-256-gcm","password":"cHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHA=","network":"tcp,udp","clients":[]}`,
		model.Hysteria:    `{"version":2,"clients":[]}`,
		model.WireGuard:   `{"secretKey":"` + privateKey + `","mtu":1420,"peers":[]}`,
		model.AnyTLS:      `{"clients":[]}`,
	}
	protocols := []model.Protocol{model.VMESS, model.VLESS, model.Trojan, model.Shadowsocks, model.Hysteria, model.WireGuard, model.AnyTLS}
	for index, protocol := range protocols {
		alias := "Alias-" + strings.ToUpper(string(protocol))
		inbound := &model.Inbound{UserId: 1, Remark: "private provider", Enable: true, Port: 31000 + index, Protocol: protocol, Settings: settings[protocol], StreamSettings: streams[protocol], Tag: fmt.Sprintf("managed-%s-%d", protocol, index), ShareAddrStrategy: "custom", ShareAddr: "vpn.pheero.com"}
		if err := db.Create(inbound).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
			t.Fatal(err)
		}
		node := &model.LineNode{
			ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", index+1), Fingerprint: fmt.Sprintf("%064d", index+1),
			Remark: "private provider", PublicName: alias, Protocol: string(protocol), OutboundTag: fmt.Sprintf("commercial-line-%d", index+1),
			OutboundCiphertext: "encrypted", InboundID: &inbound.Id, Status: "healthy", HealthStatus: "healthy",
		}
		if err := db.Create(node).Error; err != nil {
			t.Fatal(err)
		}
	}

	links, _, _, _, err := NewSubService("").GetSubs("six-protocols", "vpn.pheero.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != len(protocols) {
		t.Fatalf("got %d links, want %d: %v", len(links), len(protocols), links)
	}
	wantSchemes := map[string]bool{"vmess": false, "vless": false, "trojan": false, "ss": false, "hysteria2": false, "wireguard": false, "anytls": false}
	wantAliases := map[string]string{"vless": "Alias-VLESS", "trojan": "Alias-TROJAN", "ss": "Alias-SHADOWSOCKS", "hysteria2": "Alias-HYSTERIA", "wireguard": "Alias-WIREGUARD", "anytls": "Alias-ANYTLS"}
	for _, rawLink := range links {
		if strings.HasPrefix(rawLink, "vmess://") {
			decoded, decodeErr := base64.StdEncoding.DecodeString(strings.TrimPrefix(rawLink, "vmess://"))
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			var payload map[string]any
			if err := json.Unmarshal(decoded, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["ps"] != "Alias-VMESS" {
				t.Fatalf("VMess alias = %v", payload["ps"])
			}
			wantSchemes["vmess"] = true
			continue
		}
		parsed, parseErr := url.Parse(rawLink)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if _, ok := wantSchemes[parsed.Scheme]; !ok {
			t.Fatalf("unexpected subscription scheme %q in %q", parsed.Scheme, rawLink)
		}
		wantSchemes[parsed.Scheme] = true
		if got, want := parsed.Fragment, wantAliases[parsed.Scheme]; got != want {
			t.Fatalf("%s alias = %q, want %q", parsed.Scheme, got, want)
		}
		if strings.Contains(parsed.Fragment, "PHEERO") || strings.Contains(rawLink, "private provider") {
			t.Fatalf("managed link leaked or prefixed branding: %q", rawLink)
		}
		if parsed.Hostname() != "vpn.pheero.com" {
			t.Fatalf("%s address = %q, want vpn.pheero.com", parsed.Scheme, parsed.Hostname())
		}
		query := parsed.Query()
		switch parsed.Scheme {
		case "vless":
			for key, want := range map[string]string{
				"type": "tcp", "security": "reality", "sni": "vpn.pheero.com",
				"fp": "chrome", "pbk": "PBK", "sid": "abcd1234",
			} {
				if got := query.Get(key); got != want {
					t.Fatalf("VLESS %s = %q, want %q in %q", key, got, want, rawLink)
				}
			}
			if parsed.User.Username() != client.UUID {
				t.Fatalf("VLESS UUID = %q, want %q", parsed.User.Username(), client.UUID)
			}
		case "trojan":
			for key, want := range map[string]string{
				"type": "tcp", "security": "tls", "sni": "vpn.pheero.com",
			} {
				if got := query.Get(key); got != want {
					t.Fatalf("Trojan %s = %q, want %q in %q", key, got, want, rawLink)
				}
			}
			if parsed.User.Username() != client.Password {
				t.Fatalf("Trojan password did not round-trip")
			}
		case "hysteria2":
			for key, want := range map[string]string{
				"security": "tls", "sni": "vpn.pheero.com", "fp": "chrome",
			} {
				if got := query.Get(key); got != want {
					t.Fatalf("Hysteria2 %s = %q, want %q in %q", key, got, want, rawLink)
				}
			}
			if parsed.User.Username() != client.Auth {
				t.Fatalf("Hysteria2 auth did not round-trip")
			}
		case "anytls":
			if query.Get("security") != "tls" || query.Get("sni") != "vpn.pheero.com" {
				t.Fatalf("AnyTLS TLS parameters are incomplete: %q", rawLink)
			}
			if parsed.User.Username() != client.Password {
				t.Fatalf("AnyTLS password did not round-trip")
			}
		}
	}
	for scheme, present := range wantSchemes {
		if !present {
			t.Fatalf("managed subscription omitted %s: %v", scheme, links)
		}
	}
}

func TestCommercialSubscriptionHeadersUseSiteBrandOnly(t *testing.T) {
	initSubDB(t)
	t.Setenv("XUI_COMMERCIAL_ENV", "production")
	db := database.GetDB()
	for key, value := range map[string]string{
		"site.name":        "Pheero VPN",
		"site.url":         "https://vpn.pheero.com",
		"site.support_url": "https://vpn.pheero.com/tickets",
	} {
		if err := db.Create(&model.CommercialSetting{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	controller := &SUBController{subTitle: "A机场", subSupportUrl: "https://a.example/support", subProfileUrl: "https://a.example"}
	controller.ApplyCommonHeaders(context, "", "30", "A机场", "https://a.example/support", "https://a.example", "", false, "", false)

	wantTitle := "base64:" + base64.StdEncoding.EncodeToString([]byte("Pheero VPN"))
	if got := recorder.Header().Get("Profile-Title"); got != wantTitle {
		t.Fatalf("Profile-Title = %q, want %q", got, wantTitle)
	}
	if got := recorder.Header().Get("Profile-Web-Page-Url"); got != "https://vpn.pheero.com" {
		t.Fatalf("Profile-Web-Page-Url = %q", got)
	}
	if got := recorder.Header().Get("Support-Url"); got != "https://vpn.pheero.com/tickets" {
		t.Fatalf("Support-Url = %q", got)
	}
}
