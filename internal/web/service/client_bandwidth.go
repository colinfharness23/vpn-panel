package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
)

const bandwidthTCChain = "31802"

type ClientBandwidthService struct{}

type clientBandwidthPolicy struct {
	Email             string
	Port              int
	UploadLimitMbps   int
	DownloadLimitMbps int
}

type bandwidthRule struct {
	Direction string
	Family    string
	IPField   string
	IP        string
	PortField string
	Port      int
	Protocol  string
	RateMbps  int
}

var bandwidthCommandContext = exec.CommandContext

func (s *ClientBandwidthService) Reconcile(ctx context.Context, observed map[string]map[string]int64) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("tc"); err != nil {
		return errors.New("tc is unavailable; install iproute2 to enforce client speed limits")
	}
	if _, err := exec.LookPath("ip"); err != nil {
		return errors.New("ip is unavailable; install iproute2 to enforce client speed limits")
	}
	iface, err := bandwidthDefaultInterface(ctx)
	if err != nil {
		return err
	}

	policies, err := loadClientBandwidthPolicies(observed)
	if err != nil {
		return err
	}
	rules := buildBandwidthRules(observed, policies)
	return applyBandwidthRules(ctx, iface, rules)
}

func loadClientBandwidthPolicies(observed map[string]map[string]int64) ([]clientBandwidthPolicy, error) {
	emails := make([]string, 0, len(observed))
	for email := range observed {
		emails = append(emails, email)
	}
	if len(emails) == 0 {
		return nil, nil
	}
	var rows []clientBandwidthPolicy
	err := database.GetDB().Table("clients").
		Select("clients.email, inbounds.port, clients.upload_limit_mbps, clients.download_limit_mbps").
		Joins("JOIN client_inbounds ON client_inbounds.client_id = clients.id").
		Joins("JOIN inbounds ON inbounds.id = client_inbounds.inbound_id").
		Where("clients.email IN ?", emails).
		Where("clients.enable = ? AND inbounds.enable = ?", true, true).
		Where("inbounds.node_id IS NULL").
		Where("clients.upload_limit_mbps > 0 OR clients.download_limit_mbps > 0").
		Scan(&rows).Error
	return rows, err
}

func bandwidthDefaultInterface(ctx context.Context) (string, error) {
	out, err := bandwidthCommandContext(ctx, "ip", "-o", "route", "show", "default").Output()
	if err != nil {
		return "", fmt.Errorf("detect default network interface: %w", err)
	}
	fields := strings.Fields(string(out))
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "dev" && fields[i+1] != "" {
			return fields[i+1], nil
		}
	}
	return "", errors.New("default network interface was not found")
}

func buildBandwidthRules(observed map[string]map[string]int64, policies []clientBandwidthPolicy) []bandwidthRule {
	type key struct {
		direction string
		family    string
		ip        string
		port      int
		protocol  string
	}
	rates := make(map[key]int)
	setRate := func(k key, rate int) {
		if rate <= 0 {
			return
		}
		if current, ok := rates[k]; !ok || rate < current {
			rates[k] = rate
		}
	}
	for _, policy := range policies {
		for rawIP := range observed[policy.Email] {
			parsed := net.ParseIP(strings.TrimSpace(rawIP))
			if parsed == nil || policy.Port <= 0 || policy.Port > 65535 {
				continue
			}
			family := "ip"
			ip := parsed.String()
			if parsed.To4() == nil {
				family = "ipv6"
			}
			for _, protocol := range []string{"tcp", "udp"} {
				setRate(key{"ingress", family, ip, policy.Port, protocol}, policy.UploadLimitMbps)
				setRate(key{"egress", family, ip, policy.Port, protocol}, policy.DownloadLimitMbps)
			}
		}
	}

	keys := make([]key, 0, len(rates))
	for k := range rates {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%s|%s|%s|%05d|%s", keys[i].direction, keys[i].family, keys[i].ip, keys[i].port, keys[i].protocol) <
			fmt.Sprintf("%s|%s|%s|%05d|%s", keys[j].direction, keys[j].family, keys[j].ip, keys[j].port, keys[j].protocol)
	})
	rules := make([]bandwidthRule, 0, len(keys))
	for _, k := range keys {
		rule := bandwidthRule{Direction: k.direction, Family: k.family, IP: k.ip, Port: k.port, Protocol: k.protocol, RateMbps: rates[k]}
		if k.direction == "ingress" {
			rule.IPField, rule.PortField = "src_ip", "dst_port"
		} else {
			rule.IPField, rule.PortField = "dst_ip", "src_port"
		}
		rules = append(rules, rule)
	}
	return rules
}

func applyBandwidthRules(ctx context.Context, iface string, rules []bandwidthRule) error {
	out, err := bandwidthCommandContext(ctx, "tc", "qdisc", "add", "dev", iface, "clsact").CombinedOutput()
	if err != nil && !strings.Contains(strings.ToLower(string(out)), "file exists") {
		return fmt.Errorf("attach clsact to %s: %w: %s", iface, err, strings.TrimSpace(string(out)))
	}
	for _, direction := range []string{"ingress", "egress"} {
		_, _ = bandwidthCommandContext(ctx, "tc", "filter", "del", "dev", iface, direction, "protocol", "all", "pref", bandwidthTCChain).CombinedOutput()
		_, _ = bandwidthCommandContext(ctx, "tc", "filter", "del", "dev", iface, direction, "chain", bandwidthTCChain).CombinedOutput()
	}
	if len(rules) == 0 {
		return nil
	}
	for index, rule := range rules {
		pref := strconv.Itoa(100 + index)
		args := []string{"filter", "add", "dev", iface, rule.Direction, "chain", bandwidthTCChain, "protocol", rule.Family, "pref", pref, "flower", rule.IPField, rule.IP, "ip_proto", rule.Protocol, rule.PortField, strconv.Itoa(rule.Port), "action", "police", "rate", strconv.Itoa(rule.RateMbps) + "mbit", "burst", "256k", "conform-exceed", "drop"}
		out, err := bandwidthCommandContext(ctx, "tc", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("install bandwidth rule for %s:%d: %w: %s", rule.IP, rule.Port, err, strings.TrimSpace(string(out)))
		}
	}
	for _, direction := range []string{"ingress", "egress"} {
		out, err := bandwidthCommandContext(ctx, "tc", "filter", "add", "dev", iface, direction, "protocol", "all", "pref", bandwidthTCChain, "matchall", "action", "goto", "chain", bandwidthTCChain).CombinedOutput()
		if err != nil {
			return fmt.Errorf("activate bandwidth chain on %s %s: %w: %s", iface, direction, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}
