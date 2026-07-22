package link

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	quic "github.com/apernet/quic-go"
)

func TestPinInsecureLineOutboundReplacesAllowInsecure(t *testing.T) {
	raw := `{"protocol":"trojan","settings":{"servers":[{"address":"203.0.113.8","port":443}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.test","allowInsecure":true}}}`
	const pin = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	result, changed, err := secureTLSOutbound(context.Background(), raw, false, func(_ context.Context, request lineCertPinRequest) (string, error) {
		if request.Address != "203.0.113.8:443" || request.ServerName != "example.test" || request.QUIC {
			t.Fatalf("unexpected pin request: %+v", request)
		}
		return pin, nil
	})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Fatal(err)
	}
	tlsSettings := decoded["streamSettings"].(map[string]any)["tlsSettings"].(map[string]any)
	if _, exists := tlsSettings["allowInsecure"]; exists {
		t.Fatal("allowInsecure must be removed after pinning")
	}
	if tlsSettings["pinnedPeerCertSha256"] != pin {
		t.Fatalf("pin = %v", tlsSettings["pinnedPeerCertSha256"])
	}
}

func TestPinInsecureLineOutboundUsesHysteriaQUIC(t *testing.T) {
	raw := `{"protocol":"hysteria","settings":{"address":"198.51.100.9","port":8443},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"hy.test","alpn":["h3"],"allowInsecure":true}}}`
	const pin = "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	_, changed, err := secureTLSOutbound(context.Background(), raw, false, func(_ context.Context, request lineCertPinRequest) (string, error) {
		if !request.QUIC || request.Address != "198.51.100.9:8443" || len(request.ALPN) != 1 || request.ALPN[0] != "h3" {
			t.Fatalf("unexpected Hysteria pin request: %+v", request)
		}
		return pin, nil
	})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}

func TestSecureTLSOutboundRefreshesAnExistingAutomaticPin(t *testing.T) {
	raw := `{"protocol":"trojan","settings":{"servers":[{"address":"203.0.113.8","port":443}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.test","pinnedPeerCertSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}`
	const replacement = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	result, changed, err := secureTLSOutbound(context.Background(), raw, true, func(context.Context, lineCertPinRequest) (string, error) {
		return replacement, nil
	})
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if !strings.Contains(result, replacement) || strings.Contains(result, strings.Repeat("a", 64)) {
		t.Fatalf("automatic pin was not refreshed: %s", result)
	}
}

func TestSecureTLSOutboundRejectsNonHexCertificatePin(t *testing.T) {
	raw := `{"protocol":"trojan","settings":{"servers":[{"address":"203.0.113.8","port":443}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.test","allowInsecure":true}}}`
	invalid := strings.Repeat("z", sha256.Size*2)
	if _, _, err := secureTLSOutbound(context.Background(), raw, false, func(context.Context, lineCertPinRequest) (string, error) {
		return invalid, nil
	}); err == nil {
		t.Fatal("non-hex certificate pin was accepted")
	}
}

func TestFetchLineCertificatePinTCPAndQUIC(t *testing.T) {
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer tlsServer.Close()
	expectedSum := sha256.Sum256(tlsServer.Certificate().Raw)
	expected := hex.EncodeToString(expectedSum[:])

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tcpPin, err := fetchLineCertificatePin(ctx, lineCertPinRequest{Address: tlsServer.Listener.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	if tcpPin != expected {
		t.Fatalf("TCP pin = %s, want %s", tcpPin, expected)
	}

	quicListener, err := quic.ListenAddr("127.0.0.1:0", &tls.Config{
		Certificates: tlsServer.TLS.Certificates,
		NextProtos:   []string{"h3"},
		MinVersion:   tls.VersionTLS13,
	}, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer quicListener.Close()
	go func() {
		connection, acceptErr := quicListener.Accept(ctx)
		if acceptErr == nil {
			<-connection.Context().Done()
		}
	}()
	quicPin, err := fetchLineCertificatePin(ctx, lineCertPinRequest{
		Address: quicListener.Addr().String(), ALPN: []string{"h3"}, QUIC: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if quicPin != expected {
		t.Fatalf("QUIC pin = %s, want %s", quicPin, expected)
	}
}
