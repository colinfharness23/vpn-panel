package link

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

const maxStructuredSubscriptionDepth = 4

// parseStructuredSubscription handles provider formats that are not a plain
// newline-delimited share-link list: Clash/Mihomo YAML, Clash JSON, sing-box
// JSON and the small JSON envelopes commonly used by subscription gateways.
func parseStructuredSubscription(text string, depth int) ([]Outbound, []string) {
	if depth > maxStructuredSubscriptionDepth {
		return nil, nil
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if text == "" {
		return nil, nil
	}

	var document any
	if json.Unmarshal([]byte(text), &document) == nil {
		if outbounds, identities := parseStructuredValue(document, depth+1); len(outbounds) > 0 {
			return outbounds, identities
		}
	}

	var yamlDocument map[string]any
	if yaml.Unmarshal([]byte(text), &yamlDocument) == nil {
		if outbounds, identities := parseStructuredValue(yamlDocument, depth+1); len(outbounds) > 0 {
			return outbounds, identities
		}
	}
	return nil, nil
}

func parseStructuredValue(value any, depth int) ([]Outbound, []string) {
	if depth > maxStructuredSubscriptionDepth {
		return nil, nil
	}
	switch typed := value.(type) {
	case string:
		return parseEmbeddedSubscription(typed, depth)
	case []any:
		return parseStructuredArray(typed, depth)
	case map[string]any:
		if proxies, ok := sliceValue(typed["proxies"]); ok {
			if outbounds, identities := parseClashProxies(proxies); len(outbounds) > 0 {
				return outbounds, identities
			}
		}
		if outbounds, ok := sliceValue(typed["outbounds"]); ok {
			if parsed, identities := parseSingBoxOutbounds(outbounds); len(parsed) > 0 {
				return parsed, identities
			}
		}
		// Some JSON APIs return the node objects directly in data/list instead
		// of wrapping them in a Clash `proxies` or sing-box `outbounds` field.
		if mapString(typed, "type") != "" && mapString(typed, "server", "address") != "" {
			if hasMapKey(typed, "server_port") {
				if parsed, identities := parseSingBoxOutbounds([]any{typed}); len(parsed) > 0 {
					return parsed, identities
				}
			}
			if outbound, identity := clashProxyToOutbound(typed); outbound != nil {
				return []Outbound{outbound}, []string{identity}
			}
			if parsed, identities := parseSingBoxOutbounds([]any{typed}); len(parsed) > 0 {
				return parsed, identities
			}
		}
		// Gateways commonly wrap the actual subscription in one of these keys.
		// Traverse only known payload fields so account metadata or arbitrary
		// error objects cannot accidentally be interpreted as credentials.
		for _, key := range []string{"data", "result", "subscription", "content", "payload", "links", "link", "nodes", "items", "list", "servers", "url", "uri"} {
			if nested, exists := typed[key]; exists {
				if outbounds, identities := parseStructuredValue(nested, depth+1); len(outbounds) > 0 {
					return outbounds, identities
				}
			}
		}
	}
	return nil, nil
}

func parseEmbeddedSubscription(text string, depth int) ([]Outbound, []string) {
	text = strings.TrimSpace(strings.TrimPrefix(text, "\ufeff"))
	if text == "" {
		return nil, nil
	}
	if decoded, ok := tryBase64(text); ok {
		decoded = strings.TrimSpace(strings.TrimPrefix(decoded, "\ufeff"))
		if outbounds, identities := parseShareLinkList(decoded); len(outbounds) > 0 {
			return outbounds, identities
		}
		if outbounds, identities := parseStructuredSubscription(decoded, depth+1); len(outbounds) > 0 {
			return outbounds, identities
		}
	}
	if outbounds, identities := parseShareLinkList(text); len(outbounds) > 0 {
		return outbounds, identities
	}
	return parseStructuredSubscription(text, depth+1)
}

func parseStructuredArray(values []any, depth int) ([]Outbound, []string) {
	outbounds := make([]Outbound, 0, len(values))
	identities := make([]string, 0, len(values))
	for _, value := range values {
		parsed, ids := parseStructuredValue(value, depth+1)
		outbounds = append(outbounds, parsed...)
		identities = append(identities, ids...)
	}
	return outbounds, identities
}

func parseClashProxies(values []any) ([]Outbound, []string) {
	outbounds := make([]Outbound, 0, len(values))
	identities := make([]string, 0, len(values))
	for _, raw := range values {
		proxy, ok := stringMap(raw)
		if !ok {
			continue
		}
		outbound, identity := clashProxyToOutbound(proxy)
		if outbound == nil {
			continue
		}
		outbounds = append(outbounds, outbound)
		identities = append(identities, identity)
	}
	return outbounds, identities
}

func clashProxyToOutbound(proxy map[string]any) (Outbound, string) {
	protocol := strings.ToLower(mapString(proxy, "type"))
	if protocol == "hy2" {
		protocol = "hysteria2"
	}
	if protocol == "shadowsocks" {
		protocol = "ss"
	}
	name := mapString(proxy, "name", "tag")
	server := mapString(proxy, "server", "address")
	port := mapInt(proxy, "port", "server_port")
	if server == "" || port <= 0 || port > 65535 {
		return nil, ""
	}

	var outbound Outbound
	switch protocol {
	case "vmess":
		uuid := mapString(proxy, "uuid", "id")
		if uuid == "" {
			return nil, ""
		}
		stream := clashStreamSettings(proxy, "none")
		cipher := firstNonEmpty(mapString(proxy, "cipher", "security"), "auto")
		outbound = Outbound{
			"protocol": "vmess", "tag": name,
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": server, "port": port,
				"users": []any{map[string]any{"id": uuid, "security": cipher}},
			}}},
			"streamSettings": stream,
		}
	case "vless":
		uuid := mapString(proxy, "uuid", "id")
		if uuid == "" {
			return nil, ""
		}
		outbound = Outbound{
			"protocol": "vless", "tag": name,
			"settings": map[string]any{
				"address": server, "port": port, "id": uuid,
				"flow": mapString(proxy, "flow"), "encryption": "none",
			},
			"streamSettings": clashStreamSettings(proxy, "none"),
		}
	case "trojan":
		password := mapString(proxy, "password")
		if password == "" {
			return nil, ""
		}
		outbound = Outbound{
			"protocol": "trojan", "tag": name,
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": server, "port": port, "password": password,
			}}},
			"streamSettings": clashStreamSettings(proxy, "tls"),
		}
	case "ss":
		method := mapString(proxy, "cipher", "method")
		password := mapString(proxy, "password")
		if method == "" || password == "" {
			return nil, ""
		}
		outbound = Outbound{
			"protocol": "shadowsocks", "tag": name,
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": server, "port": port, "method": method, "password": password,
			}}},
		}
	case "hysteria2", "hysteria":
		password := mapString(proxy, "password", "auth", "auth_str")
		if password == "" {
			return nil, ""
		}
		tls := map[string]any{
			"serverName":    mapString(proxy, "sni", "servername", "server_name"),
			"alpn":          stringSlice(proxy["alpn"]),
			"fingerprint":   mapString(proxy, "client-fingerprint", "fingerprint"),
			"allowInsecure": mapBool(proxy, "skip-cert-verify", "insecure"),
			"echConfigList": "", "verifyPeerCertByName": "", "pinnedPeerCertSha256": "",
		}
		outbound = Outbound{
			"protocol": "hysteria", "tag": name,
			"settings": map[string]any{"address": server, "port": port, "version": 2},
			"streamSettings": map[string]any{
				"network": "hysteria", "security": "tls",
				"hysteriaSettings": map[string]any{"version": 2, "auth": password, "udpIdleTimeout": 60},
				"tlsSettings":      tls,
			},
		}
	case "wireguard":
		privateKey := mapString(proxy, "private-key", "private_key", "secretKey")
		publicKey := mapString(proxy, "public-key", "public_key", "peer_public_key")
		if privateKey == "" || publicKey == "" {
			return nil, ""
		}
		peer := map[string]any{
			"publicKey":  publicKey,
			"endpoint":   fmt.Sprintf("%s:%d", server, port),
			"allowedIPs": nonEmptyStringSlice(proxy, []string{"0.0.0.0/0", "::/0"}, "allowed-ips", "allowed_ips"),
		}
		if value := mapString(proxy, "pre-shared-key", "pre_shared_key", "preshared_key"); value != "" {
			peer["preSharedKey"] = value
		}
		settings := map[string]any{
			"secretKey": privateKey,
			"address":   wireGuardAddresses(proxy),
			"peers":     []any{peer},
		}
		if mtu := mapInt(proxy, "mtu"); mtu > 0 {
			settings["mtu"] = mtu
		}
		if reserved := intSlice(proxy["reserved"]); len(reserved) > 0 {
			settings["reserved"] = reserved
		}
		outbound = Outbound{"protocol": "wireguard", "tag": name, "settings": settings}
	default:
		return nil, ""
	}
	return outbound, stableOutboundIdentity(outbound)
}

func clashStreamSettings(proxy map[string]any, defaultSecurity string) map[string]any {
	network := strings.ToLower(mapString(proxy, "network", "transport"))
	if network == "" {
		network = "tcp"
	}
	if network == "http" || network == "h2" {
		network = "xhttp"
	}

	reality, _ := nestedStringMap(proxy, "reality-opts", "reality_opts")
	security := defaultSecurity
	if reality != nil {
		security = "reality"
	} else if mapBool(proxy, "tls") {
		security = "tls"
	} else if defaultSecurity == "tls" && hasMapKey(proxy, "tls") && !mapBool(proxy, "tls") {
		security = "none"
	}

	params := url.Values{}
	params.Set("type", network)
	params.Set("security", security)
	if value := mapString(proxy, "servername", "server-name", "sni", "server_name"); value != "" {
		params.Set("sni", value)
	}
	if value := mapString(proxy, "client-fingerprint", "fingerprint"); value != "" {
		params.Set("fp", value)
	}
	if mapBool(proxy, "skip-cert-verify", "insecure") {
		params.Set("allowInsecure", "1")
	}
	if values := stringSlice(proxy["alpn"]); len(values) > 0 {
		params.Set("alpn", strings.Join(values, ","))
	}
	if reality != nil {
		params.Set("pbk", mapString(reality, "public-key", "public_key"))
		params.Set("sid", mapString(reality, "short-id", "short_id"))
	}

	switch network {
	case "ws":
		if opts, ok := nestedStringMap(proxy, "ws-opts", "ws_opts"); ok {
			params.Set("path", firstNonEmpty(mapString(opts, "path"), "/"))
			if headers, ok := nestedStringMap(opts, "headers"); ok {
				params.Set("host", mapString(headers, "Host", "host"))
			}
		}
	case "grpc":
		if opts, ok := nestedStringMap(proxy, "grpc-opts", "grpc_opts"); ok {
			params.Set("serviceName", mapString(opts, "grpc-service-name", "service-name", "service_name"))
			params.Set("authority", mapString(opts, "authority"))
		}
	case "xhttp", "httpupgrade":
		for _, key := range []string{"xhttp-opts", "xhttp_opts", "http-opts", "http_opts", "http-upgrade-opts", "http_upgrade_opts"} {
			if opts, ok := nestedStringMap(proxy, key); ok {
				params.Set("path", firstNonEmpty(firstString(opts["path"]), "/"))
				if headers, ok := nestedStringMap(opts, "headers"); ok {
					params.Set("host", firstNonEmpty(mapString(headers, "Host", "host"), firstString(headers["host"])))
				}
				break
			}
		}
	}
	stream := buildStream(network, security)
	applyTransport(stream, params)
	applySecurity(stream, params)
	return stream
}

func parseSingBoxOutbounds(values []any) ([]Outbound, []string) {
	proxies := make([]any, 0, len(values))
	for _, value := range values {
		outbound, ok := stringMap(value)
		if !ok {
			continue
		}
		protocol := strings.ToLower(mapString(outbound, "type"))
		switch protocol {
		case "vmess", "vless", "trojan", "hysteria2", "wireguard":
		case "shadowsocks":
			protocol = "ss"
		default:
			continue
		}
		proxy := map[string]any{
			"type": protocol, "name": mapString(outbound, "tag", "name"),
			"server":         mapString(outbound, "server", "address"),
			"port":           mapInt(outbound, "server_port", "port"),
			"uuid":           mapString(outbound, "uuid", "id"),
			"password":       mapString(outbound, "password"),
			"cipher":         mapString(outbound, "method", "security"),
			"flow":           mapString(outbound, "flow"),
			"private-key":    mapString(outbound, "private_key", "private-key"),
			"public-key":     mapString(outbound, "peer_public_key", "public_key", "public-key"),
			"pre-shared-key": mapString(outbound, "pre_shared_key", "pre-shared-key"),
			"allowed-ips":    outbound["allowed_ips"], "local_address": outbound["local_address"],
			"mtu": outbound["mtu"], "reserved": outbound["reserved"],
		}
		if tls, ok := nestedStringMap(outbound, "tls"); ok && mapBool(tls, "enabled") {
			proxy["tls"] = true
			proxy["servername"] = mapString(tls, "server_name")
			proxy["skip-cert-verify"] = mapBool(tls, "insecure")
			proxy["alpn"] = tls["alpn"]
			if utls, ok := nestedStringMap(tls, "utls"); ok {
				proxy["client-fingerprint"] = mapString(utls, "fingerprint")
			}
			if reality, ok := nestedStringMap(tls, "reality"); ok && mapBool(reality, "enabled") {
				proxy["reality-opts"] = map[string]any{
					"public-key": mapString(reality, "public_key"),
					"short-id":   mapString(reality, "short_id"),
				}
			}
		}
		if transport, ok := nestedStringMap(outbound, "transport"); ok {
			network := strings.ToLower(mapString(transport, "type"))
			proxy["network"] = network
			switch network {
			case "ws":
				proxy["ws-opts"] = map[string]any{"path": mapString(transport, "path"), "headers": transport["headers"]}
			case "grpc":
				proxy["grpc-opts"] = map[string]any{"grpc-service-name": mapString(transport, "service_name")}
			case "http", "httpupgrade":
				proxy["http-opts"] = map[string]any{"path": transport["path"], "headers": transport["headers"]}
			}
		}
		proxies = append(proxies, proxy)
	}
	return parseClashProxies(proxies)
}

func stableOutboundIdentity(outbound Outbound) string {
	clean := make(map[string]any, len(outbound))
	for key, value := range outbound {
		if key != "tag" {
			clean[key] = value
		}
	}
	serialized, _ := json.Marshal(clean)
	return strings.ToLower(mapString(outbound, "protocol")) + ":" + string(serialized)
}

func sliceValue(value any) ([]any, bool) {
	values, ok := value.([]any)
	return values, ok
}

func stringMap(value any) (map[string]any, bool) {
	valueMap, ok := value.(map[string]any)
	return valueMap, ok
}

func nestedStringMap(values map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if value, ok := stringMap(values[key]); ok {
			return value, true
		}
	}
	return nil, false
}

func hasMapKey(values map[string]any, key string) bool {
	_, ok := values[key]
	return ok
}

func mapString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != nil {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func mapInt(values map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch typed := value.(type) {
			case int:
				return typed
			case int64:
				return int(typed)
			case uint64:
				return int(typed)
			case float64:
				return int(typed)
			case json.Number:
				parsed, _ := strconv.Atoi(typed.String())
				return parsed
			case string:
				parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
				return parsed
			}
		}
	}
	return 0
}

func mapBool(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch typed := value.(type) {
			case bool:
				return typed
			case string:
				parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
				if parsed || typed == "1" {
					return true
				}
			case int:
				return typed != 0
			case float64:
				return typed != 0
			}
		}
	}
	return false
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
		return result
	case []string:
		return typed
	case string:
		if strings.TrimSpace(typed) != "" {
			return splitComma(typed)
		}
	}
	return nil
}

func intSlice(value any) []int {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]int, 0, len(values))
	for _, value := range values {
		parsed := mapInt(map[string]any{"value": value}, "value")
		if parsed >= 0 && parsed <= 255 {
			result = append(result, parsed)
		}
	}
	return result
}

func firstString(value any) string {
	values := stringSlice(value)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func nonEmptyStringSlice(values map[string]any, fallback []string, keys ...string) []string {
	for _, key := range keys {
		if parsed := stringSlice(values[key]); len(parsed) > 0 {
			return parsed
		}
	}
	return fallback
}

func wireGuardAddresses(proxy map[string]any) []string {
	if values := stringSlice(proxy["local_address"]); len(values) > 0 {
		return values
	}
	addresses := make([]string, 0, 2)
	if value := mapString(proxy, "ip"); value != "" {
		addresses = append(addresses, value)
	}
	if value := mapString(proxy, "ipv6"); value != "" {
		addresses = append(addresses, value)
	}
	if len(addresses) == 0 {
		return []string{"0.0.0.0/0", "::/0"}
	}
	return addresses
}
