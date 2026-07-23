package link

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestParseVmessLink(t *testing.T) {
	// vmess:// + base64 of:
	// {"v":"2","ps":"test","add":"1.2.3.4","port":443,"id":"uuid","aid":"0","net":"ws","type":"","host":"ex.com","path":"/","tls":"tls"}
	link := "vmess://eyJ2IjoiMiIsInBzIjoidGVzdCIsImFkZCI6IjEuMi4zLjQiLCJwb3J0Ijo0NDMsImlkIjoidXVpZCIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiIiLCJob3N0IjoiZXguY29tIiwicGF0aCI6Ii8iLCJ0bHMiOiJ0bHMifQ=="
	res, err := ParseLink(link)
	if err != nil {
		t.Fatalf("parse vmess: %v", err)
	}
	if res.Outbound["protocol"] != "vmess" {
		t.Errorf("expected vmess protocol, got %v", res.Outbound["protocol"])
	}
	if res.Outbound["tag"] != "test" {
		t.Errorf("expected tag 'test', got %v", res.Outbound["tag"])
	}
}

func TestParseVlessLink(t *testing.T) {
	link := "vless://uuid@1.2.3.4:443?type=ws&security=tls&path=/&host=ex.com#node1"
	res, err := ParseLink(link)
	if err != nil {
		t.Fatalf("parse vless: %v", err)
	}
	if res.Outbound["protocol"] != "vless" {
		t.Fatalf("bad protocol")
	}
	if res.Outbound["tag"] != "node1" {
		t.Errorf("tag mismatch: %v", res.Outbound["tag"])
	}
}

func TestParseVlessLink_FinalMaskQuicParamsSanitized(t *testing.T) {
	fm := url.QueryEscape(`{"mask":"dtls","quicParams":{"keepAlivePeriod":"10s","maxIdleTimeout":"30","initStreamReceiveWindow":524288,"maxIncomingStreams":true,"brutalUp":"100 mbps"}}`)
	res, err := ParseLink("vless://uuid@1.2.3.4:443?type=tcp&security=none&fm=" + fm + "#node1")
	if err != nil {
		t.Fatalf("parse vless with fm: %v", err)
	}
	stream, ok := res.Outbound["streamSettings"].(map[string]any)
	if !ok {
		t.Fatalf("missing streamSettings: %v", res.Outbound)
	}
	finalmask, ok := stream["finalmask"].(map[string]any)
	if !ok {
		t.Fatalf("missing finalmask: %v", stream)
	}
	if finalmask["mask"] != "dtls" {
		t.Errorf("mask changed: %v", finalmask["mask"])
	}
	qp, ok := finalmask["quicParams"].(map[string]any)
	if !ok {
		t.Fatalf("missing quicParams: %v", finalmask)
	}
	if got := qp["keepAlivePeriod"]; got != int64(10) {
		t.Errorf("keepAlivePeriod: expected 10, got %v (%T)", got, got)
	}
	if got := qp["maxIdleTimeout"]; got != int64(30) {
		t.Errorf("maxIdleTimeout: expected 30, got %v (%T)", got, got)
	}
	if got := qp["initStreamReceiveWindow"]; got != int64(524288) {
		t.Errorf("initStreamReceiveWindow: expected 524288, got %v (%T)", got, got)
	}
	if _, exists := qp["maxIncomingStreams"]; exists {
		t.Errorf("maxIncomingStreams should be dropped, got %v", qp["maxIncomingStreams"])
	}
	if got := qp["brutalUp"]; got != "100 mbps" {
		t.Errorf("brutalUp should stay a string, got %v (%T)", got, got)
	}
}

func TestSanitizeFinalMaskQuicParams_ClampsAndRejects(t *testing.T) {
	cases := []struct {
		name string
		key  string
		in   any
		want any
	}{
		{"infinite string dropped", "keepAlivePeriod", "inf", nil},
		{"nan string dropped", "keepAlivePeriod", "NaN", nil},
		{"negative dropped", "maxStreamReceiveWindow", float64(-5), nil},
		{"negative duration dropped", "keepAlivePeriod", "-10s", nil},
		{"absurd magnitude dropped", "initConnectionReceiveWindow", float64(1e30), nil},
		{"keepAlive clamped up", "keepAlivePeriod", "1s", int64(2)},
		{"keepAlive clamped down", "keepAlivePeriod", "90s", int64(60)},
		{"idle clamped up", "maxIdleTimeout", float64(1), int64(4)},
		{"idle clamped down", "maxIdleTimeout", "10m", int64(120)},
		{"streams clamped up", "maxIncomingStreams", float64(4), int64(8)},
		{"zero means unset and survives", "maxIdleTimeout", float64(0), int64(0)},
		{"window passes through", "initStreamReceiveWindow", float64(524288), int64(524288)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed := map[string]any{"quicParams": map[string]any{c.key: c.in}}
			sanitizeFinalMaskQuicParams(parsed)
			qp := parsed["quicParams"].(map[string]any)
			got, exists := qp[c.key]
			if c.want == nil {
				if exists {
					t.Fatalf("%s: expected key dropped, got %v (%T)", c.key, got, got)
				}
				return
			}
			if !exists || got != c.want {
				t.Fatalf("%s: expected %v, got %v (%T)", c.key, c.want, got, got)
			}
		})
	}
}

func TestParseShadowsocks(t *testing.T) {
	modernUser := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secretpass"))
	legacyBody := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secretpass@1.2.3.4:8388"))
	cases := []struct {
		name   string
		link   string
		host   string
		port   int
		method string
		pass   string
	}{
		{
			name:   "modern",
			link:   "ss://" + modernUser + "@1.2.3.4:8388#node",
			host:   "1.2.3.4",
			port:   8388,
			method: "aes-256-gcm",
			pass:   "secretpass",
		},
		{
			name:   "modern with plugin query",
			link:   "ss://" + modernUser + "@1.2.3.4:8388?plugin=v2ray-plugin#node",
			host:   "1.2.3.4",
			port:   8388,
			method: "aes-256-gcm",
			pass:   "secretpass",
		},
		{
			name:   "modern sip002 slash query",
			link:   "ss://" + modernUser + "@1.2.3.4:8388/?plugin=obfs-local%3Bobfs%3Dhttp#node",
			host:   "1.2.3.4",
			port:   8388,
			method: "aes-256-gcm",
			pass:   "secretpass",
		},
		{
			name:   "legacy",
			link:   "ss://" + legacyBody + "#node",
			host:   "1.2.3.4",
			port:   8388,
			method: "aes-256-gcm",
			pass:   "secretpass",
		},
		{
			name:   "base64url userinfo with plugin and trailing slash",
			link:   "ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:pa+ss/word")) + "@1.2.3.4:8388/?plugin=obfs-local%3Bobfs%3Dhttp#node",
			host:   "1.2.3.4",
			port:   8388,
			method: "aes-128-gcm",
			pass:   "pa+ss/word",
		},
		{
			name:   "sip022 percent-encoded userinfo",
			link:   "ss://2022-blake3-aes-256-gcm:YctPZ6U7xPPcU%2Bgp3u%2B0tx%2FtRizJN9K8y%2BuKlW2qjlI%3D@example.com:8888#Example3",
			host:   "example.com",
			port:   8888,
			method: "2022-blake3-aes-256-gcm",
			pass:   "YctPZ6U7xPPcU+gp3u+0tx/tRizJN9K8y+uKlW2qjlI=",
		},
		{
			name:   "sip022 dual-key password with type query preserves inner colon",
			link:   "ss://2022-blake3-aes-256-gcm:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA%3D:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB%3D@1.2.3.4:9999?type=tcp#node",
			host:   "1.2.3.4",
			port:   9999,
			method: "2022-blake3-aes-256-gcm",
			pass:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := ParseLink(c.link)
			if err != nil {
				t.Fatalf("parse ss: %v", err)
			}
			if res.Outbound["protocol"] != "shadowsocks" {
				t.Fatalf("protocol = %v, want shadowsocks", res.Outbound["protocol"])
			}
			srv := res.Outbound["settings"].(map[string]any)["servers"].([]any)[0].(map[string]any)
			if srv["address"] != c.host {
				t.Errorf("address = %v, want %v", srv["address"], c.host)
			}
			if srv["port"] != c.port {
				t.Errorf("port = %v, want %v", srv["port"], c.port)
			}
			if srv["method"] != c.method {
				t.Errorf("method = %v, want %v", srv["method"], c.method)
			}
			if srv["password"] != c.pass {
				t.Errorf("password = %v, want %v", srv["password"], c.pass)
			}
		})
	}
}

func TestParseShadowsocksBadPort(t *testing.T) {
	user := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secretpass"))
	cases := map[string]string{
		"modern": "ss://" + user + "@1.2.3.4:notaport#node",
		"legacy": "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secretpass@1.2.3.4:notaport")) + "#node",
	}
	for name, link := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseLink(link); err == nil {
				t.Errorf("expected parse error for non-numeric port, got nil")
			}
		})
	}
}

func TestParseSubscriptionBody_Base64(t *testing.T) {
	// base64 of the two joined links:
	// vless://u@h:443?type=tcp#A\nvless://u2@h2:443?type=tcp#B
	b64 := "dmxlc3M6Ly91QGg6NDQzP3R5cGU9dGNwI0EKdmxlc3M6Ly91MkBoMjo0NDM/dHlwZT10Y3AjQg=="
	obs, ids, err := ParseSubscriptionBody([]byte(b64))
	if err != nil {
		t.Fatalf("parse sub body: %v", err)
	}
	if len(obs) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(obs))
	}
	if !strings.HasPrefix(ids[0], "vless:") || !strings.HasPrefix(ids[1], "vless:") {
		t.Errorf("bad identities: %v", ids)
	}
}

func TestParseSubscriptionBody_JSONEnvelope(t *testing.T) {
	links := "\ufeffvless://11111111-2222-4333-8444-555555555555@203.0.113.10:443?type=tcp&security=none#Provider"
	body := `{"status":"success","data":"` + base64.StdEncoding.EncodeToString([]byte(links)) + `"}`
	outbounds, identities, err := ParseSubscriptionBody([]byte(body))
	if err != nil {
		t.Fatalf("parse JSON envelope: %v", err)
	}
	if len(outbounds) != 1 || len(identities) != 1 || outbounds[0]["protocol"] != "vless" {
		t.Fatalf("unexpected JSON envelope result: outbounds=%v identities=%v", outbounds, identities)
	}
}

func TestParseSubscriptionBody_JSONNodeArray(t *testing.T) {
	body := `{"status":"success","data":[{"name":"Provider","type":"trojan","server":"203.0.113.40","port":443,"password":"secret","sni":"www.example.com"},{"link":"vless://11111111-2222-4333-8444-555555555555@203.0.113.41:443?type=tcp&security=none#Provider"}]}`
	outbounds, identities, err := ParseSubscriptionBody([]byte(body))
	if err != nil {
		t.Fatalf("parse JSON node array: %v", err)
	}
	if len(outbounds) != 2 || len(identities) != 2 || outbounds[0]["protocol"] != "trojan" || outbounds[1]["protocol"] != "vless" {
		t.Fatalf("unexpected JSON node array result: outbounds=%v identities=%v", outbounds, identities)
	}
}

func TestParseSubscriptionBody_ClashYAMLSixProtocols(t *testing.T) {
	body := `proxies:
  - name: Provider VMess
    type: vmess
    server: 203.0.113.11
    port: 443
    uuid: 11111111-2222-4333-8444-555555555551
    cipher: auto
    tls: true
    servername: edge.example.com
    network: ws
    ws-opts:
      path: /ws
      headers:
        Host: edge.example.com
  - name: Provider VLESS
    type: vless
    server: 203.0.113.12
    port: 443
    uuid: 11111111-2222-4333-8444-555555555552
    flow: xtls-rprx-vision
    tls: true
    servername: www.example.com
    reality-opts:
      public-key: test-public-key
      short-id: abcd
  - name: Provider Trojan
    type: trojan
    server: 203.0.113.13
    port: 443
    password: trojan-secret
    sni: www.example.com
    skip-cert-verify: true
  - name: Provider SS
    type: ss
    server: 203.0.113.14
    port: 8388
    cipher: aes-256-gcm
    password: ss-secret
  - name: Provider HY2
    type: hysteria2
    server: 203.0.113.15
    port: 443
    password: hy2-secret
    sni: www.example.com
    skip-cert-verify: true
  - name: Provider WG
    type: wireguard
    server: 203.0.113.16
    port: 51820
    private-key: private-key
    public-key: public-key
    ip: 10.0.0.2/32
`
	outbounds, identities, err := ParseSubscriptionBody([]byte(body))
	if err != nil {
		t.Fatalf("parse Clash YAML: %v", err)
	}
	if len(outbounds) != 6 || len(identities) != 6 {
		t.Fatalf("Clash YAML parsed %d outbounds and %d identities, want 6", len(outbounds), len(identities))
	}
	want := []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria", "wireguard"}
	for index, protocol := range want {
		if got := outbounds[index]["protocol"]; got != protocol {
			t.Fatalf("outbound %d protocol = %v, want %s", index, got, protocol)
		}
		if identities[index] == "" {
			t.Fatalf("outbound %d has no stable identity", index)
		}
	}
	stream := outbounds[1]["streamSettings"].(map[string]any)
	if stream["security"] != "reality" {
		t.Fatalf("VLESS security = %v, want reality", stream["security"])
	}
	reality := stream["realitySettings"].(map[string]any)
	if reality["publicKey"] != "test-public-key" || reality["shortId"] != "abcd" {
		t.Fatalf("VLESS reality options were lost: %#v", reality)
	}
}

func TestParseSubscriptionBody_ClashIdentityIgnoresProviderName(t *testing.T) {
	first := "proxies:\n  - {name: Provider A, type: vless, server: 203.0.113.20, port: 443, uuid: 11111111-2222-4333-8444-555555555555}"
	second := strings.Replace(first, "Provider A", "Provider B", 1)
	_, firstIDs, firstErr := ParseSubscriptionBody([]byte(first))
	_, secondIDs, secondErr := ParseSubscriptionBody([]byte(second))
	if firstErr != nil || secondErr != nil || len(firstIDs) != 1 || len(secondIDs) != 1 {
		t.Fatalf("parse renamed Clash proxy: first=%v second=%v ids=%v/%v", firstErr, secondErr, firstIDs, secondIDs)
	}
	if firstIDs[0] != secondIDs[0] {
		t.Fatalf("provider rename changed identity:\n%s\n%s", firstIDs[0], secondIDs[0])
	}
}

func TestParseSubscriptionBody_SingBoxJSON(t *testing.T) {
	body := `{"outbounds":[{"type":"vless","tag":"Provider","server":"203.0.113.30","server_port":443,"uuid":"11111111-2222-4333-8444-555555555555","flow":"xtls-rprx-vision","tls":{"enabled":true,"server_name":"www.example.com","reality":{"enabled":true,"public_key":"public","short_id":"1234"},"utls":{"enabled":true,"fingerprint":"chrome"}}}]}`
	outbounds, _, err := ParseSubscriptionBody([]byte(body))
	if err != nil || len(outbounds) != 1 {
		t.Fatalf("parse sing-box JSON: outbounds=%v err=%v", outbounds, err)
	}
	stream := outbounds[0]["streamSettings"].(map[string]any)
	if stream["security"] != "reality" {
		t.Fatalf("sing-box security = %v", stream["security"])
	}
}

func TestParseSubscriptionBody_UnsupportedIsAnError(t *testing.T) {
	if _, _, err := ParseSubscriptionBody([]byte(`{"status":"success","data":"not a subscription"}`)); err == nil {
		t.Fatal("unsupported non-empty subscription returned success")
	}
	if _, _, err := ParseSubscriptionBody(nil); err == nil {
		t.Fatal("empty subscription returned success")
	}
}

func TestParseTLSLinksPreserveAllowInsecure(t *testing.T) {
	cases := []struct {
		name string
		link string
	}{
		{name: "trojan insecure", link: "trojan://password@example.com:443?security=tls&type=ws&allowInsecure=1#node"},
		{name: "hysteria2 insecure", link: "hysteria2://password@example.com:443?sni=example.com&insecure=true#node"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseLink(tc.link)
			if err != nil {
				t.Fatal(err)
			}
			stream := result.Outbound["streamSettings"].(map[string]any)
			tlsSettings := stream["tlsSettings"].(map[string]any)
			if tlsSettings["allowInsecure"] != true {
				t.Fatalf("allowInsecure = %#v, want true", tlsSettings["allowInsecure"])
			}
		})
	}
}

func TestParseAnyTLSLink(t *testing.T) {
	result, err := ParseLink("anytls://p%40ss%2Bkey@203.0.113.41:8443?security=tls&sni=edge.example.com&fp=chrome&insecure=1#Hong%20Kong")
	if err != nil {
		t.Fatal(err)
	}
	if result.Identity == "" || result.Outbound["protocol"] != "anytls" || result.Outbound["tag"] != "Hong Kong" {
		t.Fatalf("unexpected AnyTLS parse result: %#v", result)
	}
	settings := result.Outbound["settings"].(map[string]any)
	if settings["server"] != "203.0.113.41" || settings["serverPort"] != 8443 || settings["password"] != "p@ss+key" {
		t.Fatalf("AnyTLS endpoint was not preserved: %#v", settings)
	}
	tlsSettings := result.Outbound["streamSettings"].(map[string]any)["tlsSettings"].(map[string]any)
	if tlsSettings["serverName"] != "edge.example.com" || tlsSettings["fingerprint"] != "chrome" || tlsSettings["allowInsecure"] != true {
		t.Fatalf("AnyTLS TLS settings were not preserved: %#v", tlsSettings)
	}
}

func TestSlugAndSuggest(t *testing.T) {
	if SlugRemark("Hello World!") != "hello-world" {
		t.Errorf("slug failed")
	}
	tag := SuggestTag("hk-", "  SG 01 !! ", 0)
	if tag != "hk-sg-01" {
		t.Errorf("suggest tag got %q", tag)
	}
	// Non-ASCII letters/digits are preserved rather than stripped.
	if got := SlugRemark("Москва 🇷🇺 01"); got != "москва-01" {
		t.Errorf("unicode slug got %q", got)
	}
	if got := SuggestTag("ru-", "Сервер 2", 0); got != "ru-сервер-2" {
		t.Errorf("unicode suggest tag got %q", got)
	}
}
