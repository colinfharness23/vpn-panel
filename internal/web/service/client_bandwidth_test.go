package service

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBuildBandwidthRules(t *testing.T) {
	observed := map[string]map[string]int64{
		"slow@example.com": {"203.0.113.10": 1},
		"fast@example.com": {"203.0.113.10": 1},
		"v6@example.com":   {"2001:db8::10": 1},
	}
	policies := []clientBandwidthPolicy{
		{Email: "slow@example.com", Port: 443, UploadLimitMbps: 10, DownloadLimitMbps: 20},
		{Email: "fast@example.com", Port: 443, UploadLimitMbps: 50, DownloadLimitMbps: 80},
		{Email: "v6@example.com", Port: 8443, DownloadLimitMbps: 30},
	}
	rules := buildBandwidthRules(observed, policies)
	if len(rules) != 6 {
		t.Fatalf("got %d rules, want 6: %#v", len(rules), rules)
	}
	for _, rule := range rules {
		if rule.IP == "203.0.113.10" {
			want := 20
			if rule.Direction == "ingress" {
				want = 10
			}
			if rule.RateMbps != want {
				t.Errorf("shared IPv4 %s rate = %d, want %d", rule.Direction, rule.RateMbps, want)
			}
		}
		if rule.IP == "2001:db8::10" && (rule.Family != "ipv6" || rule.Direction != "egress" || rule.RateMbps != 30) {
			t.Errorf("unexpected IPv6 rule: %#v", rule)
		}
	}
}

func TestBuildBandwidthRulesSkipsInvalidAndUnlimited(t *testing.T) {
	rules := buildBandwidthRules(
		map[string]map[string]int64{"u": {"not-an-ip": 1, "198.51.100.1": 1}},
		[]clientBandwidthPolicy{{Email: "u", Port: 443}},
	)
	if len(rules) != 0 {
		t.Fatalf("got %d rules for unlimited policy, want 0", len(rules))
	}
}

func TestApplyBandwidthRulesActivatesPrivateChain(t *testing.T) {
	original := bandwidthCommandContext
	t.Cleanup(func() { bandwidthCommandContext = original })

	var calls []string
	bandwidthCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestBandwidthCommandHelper")
		cmd.Env = append(os.Environ(), "GO_WANT_BANDWIDTH_HELPER=1")
		return cmd
	}

	rules := []bandwidthRule{{Direction: "ingress", Family: "ip", IPField: "src_ip", IP: "203.0.113.10", PortField: "dst_port", Port: 443, Protocol: "tcp", RateMbps: 10}}
	if err := applyBandwidthRules(context.Background(), "eth0", rules); err != nil {
		t.Fatalf("applyBandwidthRules: %v", err)
	}

	for _, direction := range []string{"ingress", "egress"} {
		want := "tc filter add dev eth0 " + direction + " protocol all pref " + bandwidthTCChain + " matchall action goto chain " + bandwidthTCChain
		found := false
		for _, call := range calls {
			if call == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s chain activation; calls: %#v", direction, calls)
		}
	}
}

func TestBandwidthCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_BANDWIDTH_HELPER") != "1" {
		return
	}
	os.Exit(0)
}
