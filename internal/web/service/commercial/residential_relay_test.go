package commercial

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestSafeRelayIPRejectsInternalNetworks(t *testing.T) {
	cases := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"100.64.0.1",
		"::1",
		"fe80::1",
	}
	for _, raw := range cases {
		if safeRelayIP(net.ParseIP(raw)) {
			t.Fatalf("expected %s to be rejected", raw)
		}
	}
	if !safeRelayIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("expected public IPv4 address to be accepted")
	}
	if !safeRelayIP(net.ParseIP("2606:4700:4700::1111")) {
		t.Fatal("expected public IPv6 address to be accepted")
	}
}

func TestValidateRelayEndpointChecksResolvedAddresses(t *testing.T) {
	originalLookup := relayLookupIP
	t.Cleanup(func() { relayLookupIP = originalLookup })

	relayLookupIP = func(_ context.Context, host string) ([]net.IPAddr, error) {
		if host == "public.example.com" {
			return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("192.168.1.12")}}, nil
	}

	if err := validateRelayEndpoint(context.Background(), "public.example.com", 1080); err != nil {
		t.Fatalf("expected public endpoint to pass validation: %v", err)
	}
	if err := validateRelayEndpoint(context.Background(), "private.example.com", 1080); err == nil || !strings.Contains(err.Error(), "保留地址") {
		t.Fatalf("expected resolved private endpoint to be rejected, got %v", err)
	}
	if err := validateRelayEndpoint(context.Background(), "localhost", 1080); err == nil {
		t.Fatal("expected unqualified local hostname to be rejected")
	}
	if err := validateRelayEndpoint(context.Background(), "1.1.1.1", 0); err == nil {
		t.Fatal("expected invalid port to be rejected")
	}
}
