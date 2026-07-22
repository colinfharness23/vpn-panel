package link

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	quic "github.com/apernet/quic-go"
)

type lineCertPinRequest struct {
	Address    string
	ServerName string
	ALPN       []string
	QUIC       bool
}

type lineCertPinFetcher func(context.Context, lineCertPinRequest) (string, error)

// secureTLSOutbound converts legacy insecure=1 links to certificate
// pinning before their outbound is ever inserted into the running Xray config.
// This preserves compatibility with self-signed upstreams without leaving the
// long-lived connection open to arbitrary certificates.
func secureTLSOutbound(ctx context.Context, raw string, refresh bool, fetch lineCertPinFetcher) (string, bool, error) {
	var outbound map[string]any
	if err := json.Unmarshal([]byte(raw), &outbound); err != nil {
		return "", false, fmt.Errorf("decode line outbound for certificate pinning: %w", err)
	}
	stream, _ := outbound["streamSettings"].(map[string]any)
	tlsSettings, _ := stream["tlsSettings"].(map[string]any)
	if tlsSettings == nil || (!lineBool(tlsSettings["allowInsecure"]) && !refresh) {
		return raw, false, nil
	}
	if strings.TrimSpace(lineString(tlsSettings["pinnedPeerCertSha256"])) != "" && !refresh {
		delete(tlsSettings, "allowInsecure")
		encoded, err := json.Marshal(outbound)
		return string(encoded), true, err
	}

	request, err := lineOutboundPinRequest(outbound, tlsSettings)
	if err != nil {
		return "", false, err
	}
	pin, err := fetch(ctx, request)
	if err != nil {
		return "", false, fmt.Errorf("securely pin upstream certificate: %w", err)
	}
	normalizedPin := strings.ReplaceAll(strings.TrimSpace(pin), ":", "")
	decodedPin, decodeErr := hex.DecodeString(normalizedPin)
	if decodeErr != nil || len(decodedPin) != sha256.Size {
		return "", false, errors.New("upstream certificate pin is not a valid SHA-256 value")
	}
	tlsSettings["pinnedPeerCertSha256"] = hex.EncodeToString(decodedPin)
	delete(tlsSettings, "allowInsecure")
	encoded, err := json.Marshal(outbound)
	if err != nil {
		return "", false, err
	}
	return string(encoded), true, nil
}

// SecureTLSOutbound upgrades a legacy insecure=1 TLS outbound to an exact
// trust-on-first-use certificate pin. Outbounds that do not request insecure
// verification are returned byte-for-byte unchanged.
func SecureTLSOutbound(ctx context.Context, raw string, refresh ...bool) (string, bool, error) {
	forceRefresh := len(refresh) > 0 && refresh[0]
	return secureTLSOutbound(ctx, raw, forceRefresh, fetchLineCertificatePin)
}

func lineOutboundPinRequest(outbound, tlsSettings map[string]any) (lineCertPinRequest, error) {
	protocol := strings.ToLower(strings.TrimSpace(lineString(outbound["protocol"])))
	settings, _ := outbound["settings"].(map[string]any)
	if settings == nil {
		return lineCertPinRequest{}, errors.New("TLS outbound has no settings")
	}
	host := ""
	port := 0
	switch protocol {
	case "vless", "hysteria":
		host, port = lineString(settings["address"]), lineInt(settings["port"])
	case "vmess":
		host, port = firstLineEndpoint(settings["vnext"])
	case "trojan":
		host, port = firstLineEndpoint(settings["servers"])
	default:
		return lineCertPinRequest{}, fmt.Errorf("automatic certificate pinning is unsupported for %s", protocol)
	}
	if strings.TrimSpace(host) == "" || port < 1 || port > 65535 {
		return lineCertPinRequest{}, errors.New("TLS outbound has no valid upstream endpoint")
	}
	serverName := strings.TrimSpace(lineString(tlsSettings["serverName"]))
	if serverName == "" && net.ParseIP(host) == nil {
		serverName = host
	}
	alpn := lineStringSlice(tlsSettings["alpn"])
	if protocol == "hysteria" && len(alpn) == 0 {
		alpn = []string{"h3"}
	}
	return lineCertPinRequest{
		Address: net.JoinHostPort(host, strconv.Itoa(port)), ServerName: serverName,
		ALPN: alpn, QUIC: protocol == "hysteria",
	}, nil
}

func firstLineEndpoint(value any) (string, int) {
	items, _ := value.([]any)
	if len(items) == 0 {
		return "", 0
	}
	endpoint, _ := items[0].(map[string]any)
	return lineString(endpoint["address"]), lineInt(endpoint["port"])
}

func fetchLineCertificatePin(ctx context.Context, request lineCertPinRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	tlsConfig := &tls.Config{
		ServerName: request.ServerName,
		NextProtos: append([]string(nil), request.ALPN...),
		MinVersion: tls.VersionTLS12,
		// This is a single trust-on-first-use handshake for a link whose owner
		// explicitly requested insecure=1. The resulting leaf hash is persisted,
		// and every Xray data connection verifies that exact certificate.
		InsecureSkipVerify: true, // #nosec G402 -- replaced immediately by an exact SHA-256 pin
	}
	if request.QUIC {
		if len(tlsConfig.NextProtos) == 0 {
			tlsConfig.NextProtos = []string{"h3"}
		}
		connection, err := quic.DialAddr(ctx, request.Address, tlsConfig, &quic.Config{
			HandshakeIdleTimeout: 10 * time.Second,
			MaxIdleTimeout:       10 * time.Second,
		})
		if err != nil {
			return "", err
		}
		defer connection.CloseWithError(0, "certificate pin captured")
		return leafCertificatePin(connection.ConnectionState().TLS.PeerCertificates)
	}

	rawConnection, err := (&net.Dialer{}).DialContext(ctx, "tcp", request.Address)
	if err != nil {
		return "", err
	}
	defer rawConnection.Close()
	tlsConnection := tls.Client(rawConnection, tlsConfig)
	if err := tlsConnection.HandshakeContext(ctx); err != nil {
		return "", err
	}
	defer tlsConnection.Close()
	return leafCertificatePin(tlsConnection.ConnectionState().PeerCertificates)
}

func leafCertificatePin(certificates []*x509.Certificate) (string, error) {
	if len(certificates) == 0 || certificates[0] == nil || len(certificates[0].Raw) == 0 {
		return "", errors.New("upstream did not present a certificate")
	}
	sum := sha256.Sum256(certificates[0].Raw)
	return hex.EncodeToString(sum[:]), nil
}

func lineString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func lineInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		result, _ := strconv.Atoi(string(typed))
		return result
	case string:
		result, _ := strconv.Atoi(strings.TrimSpace(typed))
		return result
	default:
		return 0
	}
}

func lineBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

func lineStringSlice(value any) []string {
	var result []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := lineString(item); text != "" {
				result = append(result, text)
			}
		}
	case []string:
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				result = append(result, text)
			}
		}
	case string:
		for _, item := range strings.Split(typed, ",") {
			if text := strings.TrimSpace(item); text != "" {
				result = append(result, text)
			}
		}
	}
	return result
}
