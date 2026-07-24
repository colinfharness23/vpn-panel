package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"

	netproxy "golang.org/x/net/proxy"
	"gorm.io/gorm"
)

var managedAnyTLSRuntime = struct {
	sync.Mutex
	cmd        *exec.Cmd
	done       chan struct{}
	cancel     context.CancelFunc
	statsAPI   *xray.XrayAPI
	configHash [sha256.Size]byte
}{}

const managedAnyTLSStatsPort = 62081

const managedAnyTLSStatsMethod = "/v2ray.core.app.stats.command.StatsService/QueryStats"

func managedAnyTLSBinaryPath() string {
	if custom := strings.TrimSpace(os.Getenv("XUI_SINGBOX_BINARY")); custom != "" {
		return custom
	}
	arch := runtime.GOARCH
	if arch == "arm" {
		arch = "arm32"
	}
	return filepath.Join(config.GetBinFolderPath(), fmt.Sprintf("sing-box-%s-%s", runtime.GOOS, arch))
}

func managedAnyTLSConfigPath() string {
	if custom := strings.TrimSpace(os.Getenv("XUI_SINGBOX_CONFIG")); custom != "" {
		return custom
	}
	return filepath.Join(config.GetBinFolderPath(), "sing-box-managed.json")
}

// ReconcileManagedAnyTLS keeps the sing-box sidecar in sync with the managed
// AnyTLS lines. Xray does not implement AnyTLS, so these rows must never be
// inserted into Xray's configuration.
func ReconcileManagedAnyTLS() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	contents, count, err := buildManagedAnyTLSConfig(db)
	if err != nil {
		return err
	}

	managedAnyTLSRuntime.Lock()
	defer managedAnyTLSRuntime.Unlock()
	if count == 0 {
		stopManagedAnyTLSLocked()
		managedAnyTLSRuntime.configHash = [sha256.Size]byte{}
		return nil
	}

	hash := sha256.Sum256(contents)
	if hash == managedAnyTLSRuntime.configHash && managedAnyTLSRunningLocked() {
		return nil
	}
	binaryPath := managedAnyTLSBinaryPath()
	info, err := os.Stat(binaryPath)
	if err != nil || info.IsDir() {
		return fmt.Errorf("AnyTLS runtime is unavailable at %s", binaryPath)
	}
	configPath := managedAnyTLSConfigPath()
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, contents, 0o600); err != nil {
		return err
	}
	defer os.Remove(tempPath)
	checkCtx, cancelCheck := context.WithTimeout(context.Background(), 30*time.Second)
	output, checkErr := exec.CommandContext(checkCtx, binaryPath, "check", "-c", tempPath).CombinedOutput()
	cancelCheck()
	if checkErr != nil {
		return fmt.Errorf("sing-box rejected managed AnyTLS config: %w: %s", checkErr, strings.TrimSpace(string(output)))
	}
	if err := os.Rename(tempPath, configPath); err != nil {
		return err
	}

	stopManagedAnyTLSLocked()
	logPath := filepath.Join(config.GetLogFolder(), "sing-box.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
	cmd := exec.CommandContext(runtimeCtx, binaryPath, "run", "-c", configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		cancelRuntime()
		_ = logFile.Close()
		return err
	}
	done := make(chan struct{})
	managedAnyTLSRuntime.cmd = cmd
	managedAnyTLSRuntime.done = done
	managedAnyTLSRuntime.cancel = cancelRuntime
	managedAnyTLSRuntime.configHash = hash
	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		if err != nil {
			logger.Warning("managed AnyTLS runtime exited:", err)
		}
		close(done)
	}()
	statsAPI := new(xray.XrayAPI)
	if err := statsAPI.Init(managedAnyTLSStatsPort); err != nil {
		cancelRuntime()
		<-done
		managedAnyTLSRuntime.cmd = nil
		managedAnyTLSRuntime.done = nil
		managedAnyTLSRuntime.cancel = nil
		managedAnyTLSRuntime.configHash = [sha256.Size]byte{}
		return fmt.Errorf("initialize managed AnyTLS statistics API: %w", err)
	}
	managedAnyTLSRuntime.statsAPI = statsAPI
	return nil
}

func StopManagedAnyTLS() {
	managedAnyTLSRuntime.Lock()
	defer managedAnyTLSRuntime.Unlock()
	stopManagedAnyTLSLocked()
	managedAnyTLSRuntime.configHash = [sha256.Size]byte{}
}

func managedAnyTLSRunningLocked() bool {
	if managedAnyTLSRuntime.cmd == nil || managedAnyTLSRuntime.done == nil {
		return false
	}
	select {
	case <-managedAnyTLSRuntime.done:
		return false
	default:
		return true
	}
}

func stopManagedAnyTLSLocked() {
	if managedAnyTLSRuntime.statsAPI != nil {
		managedAnyTLSRuntime.statsAPI.Close()
		managedAnyTLSRuntime.statsAPI = nil
	}
	if !managedAnyTLSRunningLocked() {
		if managedAnyTLSRuntime.cancel != nil {
			managedAnyTLSRuntime.cancel()
		}
		managedAnyTLSRuntime.cmd = nil
		managedAnyTLSRuntime.done = nil
		managedAnyTLSRuntime.cancel = nil
		return
	}
	cmd := managedAnyTLSRuntime.cmd
	done := managedAnyTLSRuntime.done
	cancelRuntime := managedAnyTLSRuntime.cancel
	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		if cancelRuntime != nil {
			cancelRuntime()
		}
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
	}
	if cancelRuntime != nil {
		cancelRuntime()
	}
	managedAnyTLSRuntime.cmd = nil
	managedAnyTLSRuntime.done = nil
	managedAnyTLSRuntime.cancel = nil
}

// GetManagedAnyTLSTraffic returns per-line and per-customer traffic deltas from
// sing-box's loopback-only V2Ray-compatible statistics service.
func GetManagedAnyTLSTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error) {
	managedAnyTLSRuntime.Lock()
	defer managedAnyTLSRuntime.Unlock()
	if !managedAnyTLSRunningLocked() || managedAnyTLSRuntime.statsAPI == nil {
		return nil, nil, nil
	}
	return managedAnyTLSRuntime.statsAPI.GetTrafficWithMethod(managedAnyTLSStatsMethod)
}

type managedAnyTLSConfig struct {
	Log          map[string]any `json:"log"`
	Inbounds     []any          `json:"inbounds"`
	Outbounds    []any          `json:"outbounds"`
	Route        map[string]any `json:"route"`
	Experimental map[string]any `json:"experimental,omitempty"`
}

func buildManagedAnyTLSConfig(db *gorm.DB) ([]byte, int, error) {
	var nodes []model.LineNode
	if err := db.Where(
		"protocol = ? AND missing_since IS NULL AND inbound_id IS NOT NULL AND status IN ?",
		"anytls", []string{"checking", "provisioning", "retry", "ready", "healthy", "offline"},
	).Order("created_at asc").Find(&nodes).Error; err != nil {
		return nil, 0, err
	}
	cfg := managedAnyTLSConfig{
		Log:       map[string]any{"level": "warn", "timestamp": true},
		Inbounds:  []any{},
		Outbounds: []any{},
		Route:     map[string]any{"rules": []any{}, "final": "direct", "auto_detect_interface": true},
	}
	rules := []any{}
	inboundTags := []string{}
	statUsers := map[string]struct{}{}
	count := 0
	clientService := ClientService{}
	for i := range nodes {
		var inbound model.Inbound
		if err := db.First(&inbound, "id = ? AND enable = ? AND protocol = ?", *nodes[i].InboundID, true, model.AnyTLS).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, 0, err
		}
		clients, err := clientService.ListForInbound(db, inbound.Id)
		if err != nil {
			return nil, 0, err
		}
		users := make([]any, 0, len(clients))
		activeUsers := make(map[string]struct{}, len(clients))
		for j := range clients {
			if clients[j].Enable && clients[j].Email != "" && clients[j].Password != "" {
				users = append(users, map[string]any{"name": clients[j].Email, "password": clients[j].Password})
				statUsers[clients[j].Email] = struct{}{}
				activeUsers[clients[j].Email] = struct{}{}
			}
		}
		if len(users) == 0 {
			continue
		}
		plain, err := UnprotectCredential(nodes[i].OutboundCiphertext)
		if err != nil {
			return nil, 0, err
		}
		outbound, err := managedAnyTLSOutboundConfig(plain, nodes[i].OutboundTag)
		if err != nil {
			return nil, 0, fmt.Errorf("AnyTLS line %s: %w", nodes[i].ID, err)
		}
		certificatePath, keyPath, err := anyTLSCertificatePaths(inbound.StreamSettings)
		if err != nil {
			return nil, 0, err
		}
		cfg.Inbounds = append(cfg.Inbounds, map[string]any{
			"type": "anytls", "tag": inbound.Tag, "listen": inbound.Listen,
			"listen_port": inbound.Port, "users": users,
			"tls": map[string]any{
				"enabled": true, "certificate_path": certificatePath, "key_path": keyPath,
			},
		})
		cfg.Outbounds = append(cfg.Outbounds, outbound)
		relayRules, relayOutbounds, err := managedAnyTLSResidentialRelays(db, inbound.Id, inbound.Tag, nodes[i].OutboundTag, activeUsers)
		if err != nil {
			return nil, 0, err
		}
		cfg.Outbounds = append(cfg.Outbounds, relayOutbounds...)
		rules = append(rules, relayRules...)
		rules = append(rules, map[string]any{
			"inbound": []string{inbound.Tag}, "action": "route", "outbound": nodes[i].OutboundTag,
		})
		inboundTags = append(inboundTags, inbound.Tag)
		count++
	}
	if count == 0 {
		return nil, 0, nil
	}
	cfg.Outbounds = append(cfg.Outbounds, map[string]any{"type": "direct", "tag": "direct"})
	cfg.Route["rules"] = rules
	users := make([]string, 0, len(statUsers))
	for email := range statUsers {
		users = append(users, email)
	}
	sort.Strings(users)
	cfg.Experimental = map[string]any{
		"v2ray_api": map[string]any{
			"listen": fmt.Sprintf("127.0.0.1:%d", managedAnyTLSStatsPort),
			"stats": map[string]any{
				"enabled": true, "inbounds": inboundTags, "users": users,
			},
		},
	}
	contents, err := json.MarshalIndent(cfg, "", "  ")
	return contents, count, err
}

// managedAnyTLSResidentialRelays adds per-customer SOCKS5 exits to the
// sing-box sidecar. The SOCKS outbound itself detours through the selected
// AnyTLS airport line; auth_user keeps the override scoped to the customer who
// configured it.
func managedAnyTLSResidentialRelays(db *gorm.DB, inboundID int, inboundTag, carrierTag string, activeUsers map[string]struct{}) ([]any, []any, error) {
	type relayRow struct {
		model.ResidentialRelay
		AuthUser string `gorm:"column:auth_user"`
	}
	var rows []relayRow
	now := time.Now().UTC()
	err := db.Table(model.ResidentialRelay{}.TableName()+" AS relays").
		Select("relays.*, entitlements.internal_client_id AS auth_user").
		Joins("JOIN "+model.SubscriptionEntitlement{}.TableName()+" AS entitlements ON entitlements.id = relays.entitlement_id AND entitlements.customer_id = relays.customer_id").
		Where("relays.inbound_id = ? AND relays.status IN ?", inboundID, []string{"active", "pending"}).
		Where("entitlements.status = ? AND entitlements.residential_relay_enabled = ? AND entitlements.residential_relay_limit > 0", "active", true).
		Where("entitlements.expires_at IS NULL OR entitlements.expires_at > ?", now).
		Order("relays.created_at asc").
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	rules := make([]any, 0, len(rows))
	outbounds := make([]any, 0, len(rows))
	for i := range rows {
		if rows[i].AuthUser == "" {
			continue
		}
		if _, ok := activeUsers[rows[i].AuthUser]; !ok {
			continue
		}
		username, err := UnprotectCredential(rows[i].UsernameCiphertext)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt AnyTLS residential relay username %s: %w", rows[i].ID, err)
		}
		password, err := UnprotectCredential(rows[i].PasswordCiphertext)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt AnyTLS residential relay password %s: %w", rows[i].ID, err)
		}
		relayOutbound := map[string]any{
			"type":        "socks",
			"tag":         rows[i].OutboundTag,
			"server":      rows[i].SOCKSHost,
			"server_port": rows[i].SOCKSPort,
			"version":     "5",
			"detour":      carrierTag,
		}
		if username != "" && password != "" {
			relayOutbound["username"] = username
			relayOutbound["password"] = password
		}
		outbounds = append(outbounds, relayOutbound)
		rules = append(rules, map[string]any{
			"inbound":   []string{inboundTag},
			"auth_user": []string{rows[i].AuthUser},
			"action":    "route",
			"outbound":  rows[i].OutboundTag,
		})
	}
	return rules, outbounds, nil
}

func managedAnyTLSOutboundConfig(raw, tag string) (map[string]any, error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	if strings.ToLower(anyTLSString(parsed["protocol"])) != "anytls" {
		return nil, errors.New("upstream protocol is not AnyTLS")
	}
	settings, _ := parsed["settings"].(map[string]any)
	server := anyTLSString(settings["server"])
	port := anyTLSInt(settings["serverPort"])
	password := anyTLSString(settings["password"])
	if server == "" || port < 1 || port > 65535 || password == "" {
		return nil, errors.New("upstream server, port or password is invalid")
	}
	tlsConfig := map[string]any{"enabled": true}
	if stream, ok := parsed["streamSettings"].(map[string]any); ok {
		if tlsSettings, ok := stream["tlsSettings"].(map[string]any); ok {
			if value := anyTLSString(tlsSettings["serverName"]); value != "" {
				tlsConfig["server_name"] = value
			}
			if value := anyTLSString(tlsSettings["pinnedPeerPublicKeySha256"]); value != "" {
				tlsConfig["certificate_public_key_sha256"] = []string{value}
			}
			if anyTLSBool(tlsSettings["allowInsecure"]) {
				tlsConfig["insecure"] = true
			}
			if values := anyTLSStrings(tlsSettings["alpn"]); len(values) > 0 {
				tlsConfig["alpn"] = values
			}
			if fingerprint := anyTLSString(tlsSettings["fingerprint"]); fingerprint != "" {
				tlsConfig["utls"] = map[string]any{"enabled": true, "fingerprint": fingerprint}
			}
		}
	}
	return map[string]any{
		"type": "anytls", "tag": tag, "server": server, "server_port": port,
		"password": password, "tls": tlsConfig,
	}, nil
}

// ProbeManagedAnyTLSOutbound performs a real HTTP request through a temporary
// local SOCKS listener and the imported AnyTLS upstream. This verifies the
// protocol handshake and usable egress rather than treating a parsed URI or an
// open TCP port as a healthy line.
func ProbeManagedAnyTLSOutbound(ctx context.Context, raw string) (int, error) {
	binaryPath := managedAnyTLSBinaryPath()
	if info, err := os.Stat(binaryPath); err != nil || info.IsDir() {
		return 0, fmt.Errorf("AnyTLS runtime is unavailable at %s", binaryPath)
	}
	outbound, err := managedAnyTLSOutboundConfig(raw, "probe-out")
	if err != nil {
		return 0, err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	probeConfig := managedAnyTLSConfig{
		Log: map[string]any{"level": "warn", "timestamp": true},
		Inbounds: []any{map[string]any{
			"type": "mixed", "tag": "probe-in", "listen": "127.0.0.1", "listen_port": port,
		}},
		Outbounds: []any{outbound},
		Route:     map[string]any{"rules": []any{}, "final": "probe-out", "auto_detect_interface": true},
	}
	contents, err := json.Marshal(probeConfig)
	if err != nil {
		return 0, err
	}
	tempDir, err := os.MkdirTemp("", "nova-anytls-probe.")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tempDir)
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		return 0, err
	}
	if output, checkErr := exec.CommandContext(ctx, binaryPath, "check", "-c", configPath).CombinedOutput(); checkErr != nil {
		return 0, fmt.Errorf("sing-box rejected AnyTLS probe config: %w: %s", checkErr, strings.TrimSpace(string(output)))
	}

	var processOutput bytes.Buffer
	cmd := exec.CommandContext(ctx, binaryPath, "run", "-c", configPath)
	cmd.Stdout = &processOutput
	cmd.Stderr = &processOutput
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
		}
	}()

	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(5 * time.Second)
	tcpDialer := &net.Dialer{Timeout: 250 * time.Millisecond}
	for {
		connection, dialErr := tcpDialer.DialContext(ctx, "tcp", address)
		if dialErr == nil {
			_ = connection.Close()
			break
		}
		select {
		case <-done:
			return 0, fmt.Errorf("AnyTLS probe runtime exited: %s", strings.TrimSpace(processOutput.String()))
		default:
		}
		if time.Now().After(deadline) {
			return 0, errors.New("AnyTLS probe listener did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}

	dialer, err := netproxy.SOCKS5("tcp", address, nil, netproxy.Direct)
	if err != nil {
		return 0, err
	}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(_ context.Context, network, destination string) (net.Conn, error) {
			return dialer.Dial(network, destination)
		},
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	started := time.Now()
	targets := []string{
		"https://cp.cloudflare.com/generate_204",
		"https://www.gstatic.com/generate_204",
	}
	var failures []error
	for attempt := 0; attempt < 2; attempt++ {
		failures = failures[:0]
		for _, target := range targets {
			request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
			if requestErr != nil {
				failures = append(failures, requestErr)
				continue
			}
			response, requestErr := client.Do(request)
			if requestErr == nil {
				_ = response.Body.Close()
				if response.StatusCode >= 200 && response.StatusCode < 500 {
					return max(1, int(time.Since(started).Milliseconds())), nil
				}
				requestErr = fmt.Errorf("%s returned HTTP %d", target, response.StatusCode)
			}
			failures = append(failures, requestErr)
		}
		if attempt == 0 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
	}
	runtimeLog := strings.TrimSpace(processOutput.String())
	if runtimeLog != "" {
		return 0, fmt.Errorf("AnyTLS egress probe failed: %w; sing-box: %s", errors.Join(failures...), runtimeLog)
	}
	return 0, fmt.Errorf("AnyTLS egress probe failed: %w", errors.Join(failures...))
}

func anyTLSCertificatePaths(streamSettings string) (string, string, error) {
	var stream map[string]any
	if err := json.Unmarshal([]byte(streamSettings), &stream); err != nil {
		return "", "", err
	}
	tlsSettings, _ := stream["tlsSettings"].(map[string]any)
	certificates, _ := tlsSettings["certificates"].([]any)
	if len(certificates) == 0 {
		return "", "", errors.New("managed AnyTLS inbound has no TLS certificate")
	}
	certificate, _ := certificates[0].(map[string]any)
	certificatePath := anyTLSString(certificate["certificateFile"])
	keyPath := anyTLSString(certificate["keyFile"])
	if certificatePath == "" || keyPath == "" {
		return "", "", errors.New("managed AnyTLS certificate paths are empty")
	}
	return certificatePath, keyPath, nil
}

func anyTLSString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func anyTLSInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
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

func anyTLSBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case string:
		result, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return result
	default:
		return false
	}
}

func anyTLSStrings(value any) []string {
	result := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := anyTLSString(item); text != "" {
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
