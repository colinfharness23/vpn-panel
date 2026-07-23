package commercial

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	linkutil "github.com/mhsanaei/3x-ui/v3/internal/util/link"
	wgutil "github.com/mhsanaei/3x-ui/v3/internal/util/wireguard"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	outboundservice "github.com/mhsanaei/3x-ui/v3/internal/web/service/outbound"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const lineStatusRetry = "retry"

const (
	managedLineProvisionVersion = 6
	managedLinePublicPort       = 443
	managedLineWSPathPrefix     = "/nova-line/"
)

func (w *Worker) HandleLineHealthEvent(event eventbus.Event) {
	if event.Type != eventbus.EventOutboundSample || !strings.HasPrefix(event.Source, lineTagPrefix) {
		return
	}
	select {
	case w.lineEvents <- event:
	default:
	}
}

func (w *Worker) processNextLineNode(ctx context.Context) error {
	now := time.Now().UTC()
	var node model.LineNode
	err := w.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND missing_since IS NULL AND (next_provision_at IS NULL OR next_provision_at <= ?)", []string{lineStatusChecking, lineStatusRetry}, now).
			Where("EXISTS (SELECT 1 FROM commercial_line_group_nodes memberships WHERE memberships.node_id = commercial_line_nodes.id)").
			Order("updated_at asc")
		if err := query.First(&node).Error; err != nil {
			return err
		}
		return tx.Model(&node).Updates(map[string]any{"status": lineStatusProvisioning, "health_status": lineHealthChecking, "provision_locked_at": now, "provision_attempts": gorm.Expr("provision_attempts + 1"), "last_error": ""}).Error
	})
	if err != nil {
		return err
	}
	if err := w.provisionLineNode(ctx, &node); err != nil {
		attempts := node.ProvisionAttempts + 1
		delay := time.Duration(min(attempts, 10)) * time.Minute
		next := time.Now().UTC().Add(delay)
		return w.db.Model(&model.LineNode{}).Where("id = ? AND status = ?", node.ID, lineStatusProvisioning).Updates(map[string]any{"status": lineStatusRetry, "health_status": lineHealthOffline, "provision_locked_at": nil, "next_provision_at": next, "last_error": err.Error()}).Error
	}
	return nil
}

func (w *Worker) provisionLineNode(ctx context.Context, node *model.LineNode) error {
	if err := w.ensureManagedLineInbound(node); err != nil {
		return err
	}
	if err := w.attachLineSubscribers(node.ID); err != nil {
		return err
	}
	plain, err := service.UnprotectCredential(node.OutboundCiphertext)
	if err != nil {
		return err
	}
	secureOutbound, changed, err := linkutil.SecureTLSOutbound(ctx, plain, node.TLSAutoPinned)
	if err != nil {
		return err
	}
	if changed {
		protected, protectErr := service.ProtectCredential(secureOutbound)
		if protectErr != nil {
			return protectErr
		}
		if updateErr := w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{"outbound_ciphertext": protected, "tls_auto_pinned": true}).Error; updateErr != nil {
			return updateErr
		}
		node.OutboundCiphertext = protected
		node.TLSAutoPinned = true
		plain = secureOutbound
	}
	if node.InboundID == nil {
		return errors.New("托管入站创建后未返回标识")
	}
	now := time.Now().UTC()
	if err := w.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Update("enable", true).Error; err != nil {
			return err
		}
		return tx.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
			"status": lineStatusReady, "health_status": lineHealthChecking, "provision_locked_at": nil,
			"next_provision_at": nil, "last_error": "",
		}).Error
	}); err != nil {
		return err
	}
	managedAnyTLS := strings.EqualFold(strings.TrimSpace(node.Protocol), "anytls")
	if managedAnyTLS {
		// AnyTLS is served by the sing-box sidecar rather than Xray. Reconcile
		// it immediately after enabling the inbound so first-time provisioning
		// does not wait on a listener that cannot start until the next worker
		// tick.
		if err := service.ReconcileManagedAnyTLS(); err != nil {
			return err
		}
	} else {
		if err := (&service.XrayService{}).RestartXray(false); err != nil {
			return err
		}
	}
	if node.PublicPort == nil {
		return errors.New("托管线路没有公网端口")
	}
	waitForListener := true
	if managedAnyTLS {
		var err error
		waitForListener, err = managedAnyTLSNeedsListener(w.db, *node.InboundID)
		if err != nil {
			return err
		}
		// sing-box requires at least one AnyTLS user. A newly imported line
		// commonly has none until the first plan is purchased, so there is no
		// listener to wait for yet. Keep the inbound enabled and complete the
		// upstream probe; ClientService reconciles the sidecar as soon as the
		// first enabled client is attached.
	}
	if waitForListener {
		if err := waitManagedLineListenerForProtocol(ctx, managedLineInboundProtocol(node.Protocol), *node.PublicPort, 8*time.Second); err != nil {
			_ = w.db.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Update("enable", false).Error
			if managedAnyTLS {
				_ = service.ReconcileManagedAnyTLS()
			} else {
				_ = (&service.XrayService{}).RestartXray(true)
			}
			return err
		}
	}

	if managedAnyTLS {
		delay, probeErr := service.ProbeManagedAnyTLSOutbound(ctx, plain)
		if probeErr != nil {
			return w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
				"status": lineStatusReady, "health_status": lineHealthOffline, "consecutive_fails": 1,
				"consecutive_passes": 0, "last_probe_at": now, "last_error": probeErr.Error(),
			}).Error
		}
		return w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
			"status": lineStatusReady, "health_status": lineHealthHealthy, "latency_ms": delay,
			"consecutive_fails": 0, "consecutive_passes": 1, "last_probe_at": now, "last_error": "",
		}).Error
	}

	results, err := (&outboundservice.OutboundService{}).TestOutbounds("["+plain+"]", "", "", "real")
	if err != nil || len(results) != 1 || !results[0].Success {
		message := "线路真实出站探测失败"
		if err != nil {
			message = err.Error()
		} else if len(results) == 1 && strings.TrimSpace(results[0].Error) != "" {
			message = results[0].Error
		}
		return w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
			"status": lineStatusReady, "health_status": lineHealthOffline, "consecutive_fails": 1,
			"consecutive_passes": 0, "last_probe_at": now, "last_error": message,
		}).Error
	}
	return w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
		"status": lineStatusReady, "health_status": lineHealthHealthy, "latency_ms": results[0].Delay,
		"consecutive_fails": 0, "consecutive_passes": 1, "last_probe_at": now, "last_error": "",
	}).Error
}

func managedAnyTLSNeedsListener(db *gorm.DB, inboundID int) (bool, error) {
	var enabledClients int64
	err := db.Table(model.ClientRecord{}.TableName()+" clients").
		Joins("JOIN "+model.ClientInbound{}.TableName()+" bindings ON bindings.client_id = clients.id").
		Where("bindings.inbound_id = ? AND clients.enable = ?", inboundID, true).
		Count(&enabledClients).Error
	return enabledClients > 0, err
}

func waitManagedLineListener(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	consecutive := 0
	for time.Now().Before(deadline) {
		dialCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(port)))
		cancel()
		if err == nil {
			_ = conn.Close()
			consecutive++
			if consecutive >= 2 {
				return nil
			}
		} else {
			consecutive = 0
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("托管线路端口 %d 未成功监听，已停止发布并等待自动重试", port)
}

func waitManagedLineListenerForProtocol(ctx context.Context, protocol model.Protocol, port int, timeout time.Duration) error {
	if protocol != model.Hysteria && protocol != model.WireGuard {
		return waitManagedLineListener(ctx, port, timeout)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		listener, err := (&net.ListenConfig{}).ListenPacket(ctx, "udp", net.JoinHostPort("0.0.0.0", fmt.Sprint(port)))
		if err != nil {
			return nil
		}
		_ = listener.Close()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("托管线路 UDP 端口 %d 未成功监听，已停止发布并等待自动重试", port)
}

func (w *Worker) ensureManagedLineInbound(node *model.LineNode) error {
	expectedProtocol := managedLineInboundProtocol(node.Protocol)
	siteURL := NewConfigStore().GetDefault("site.url", "")
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("请先在商业设置中配置站点域名")
	}
	hostname := parsed.Hostname()
	if node.InboundID != nil && node.PublicPort != nil {
		var existing model.Inbound
		if err := w.db.Select("id", "listen", "port", "protocol", "stream_settings").First(&existing, "id = ?", *node.InboundID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing.Id > 0 && existing.Protocol == expectedProtocol && node.ProvisionVersion >= 3 {
			if node.ProvisionVersion < managedLineProvisionVersion {
				return w.upgradeManagedLineIngress(node, &existing, expectedProtocol, hostname)
			}
			return nil
		}
		if existing.Id > 0 {
			oldInboundID := *node.InboundID
			oldPort := *node.PublicPort
			if err := w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{"inbound_id": nil, "public_port": nil}).Error; err != nil {
				return err
			}
			if _, err := w.inbounds.DelInbound(oldInboundID); err != nil {
				_ = w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{"inbound_id": oldInboundID, "public_port": oldPort}).Error
				return fmt.Errorf("migrate managed line protocol: %w", err)
			}
			node.InboundID = nil
			node.PublicPort = nil
		}
	}
	port, err := allocateManagedLinePort(w.db, expectedProtocol, node.ID)
	if err != nil {
		return err
	}
	settings, stream, err := managedLineInboundJSON(node, expectedProtocol, hostname, port)
	if err != nil {
		return err
	}
	sniffing := `{"enabled":true,"destOverride":["http","tls","quic"],"metadataOnly":false,"routeOnly":false}`
	publicName := strings.TrimSpace(node.PublicName)
	if publicName == "" {
		publicName = strings.ToUpper(strings.TrimSpace(node.Protocol)) + " 线路"
	}
	inbound := &model.Inbound{UserId: 1, Remark: publicName, SubSortIndex: 1, Enable: false, Listen: managedLineListenAddress(expectedProtocol), Port: port, Protocol: expectedProtocol, Settings: string(settings), StreamSettings: string(stream), Tag: "commercial-in-" + strings.ReplaceAll(node.ID, "-", "")[:20], Sniffing: sniffing, ShareAddrStrategy: "custom", ShareAddr: hostname}
	created, _, err := w.inbounds.AddInbound(inbound)
	if err != nil {
		return err
	}
	if err := w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{"public_port": port, "inbound_id": created.Id, "provision_version": managedLineProvisionVersion}).Error; err != nil {
		_, _ = w.inbounds.DelInbound(created.Id)
		return err
	}
	node.PublicPort = &port
	node.InboundID = &created.Id
	node.ProvisionVersion = managedLineProvisionVersion
	return nil
}

func (w *Worker) upgradeManagedLineIngress(node *model.LineNode, inbound *model.Inbound, protocol model.Protocol, hostname string) error {
	updates := map[string]any{"listen": managedLineListenAddress(protocol)}
	port := inbound.Port
	if protocol == model.Hysteria && port != managedLinePublicPort && managedLineUDPPortAvailable(w.db, node.ID, managedLinePublicPort) {
		port = managedLinePublicPort
		updates["port"] = port
	}
	switch protocol {
	case model.VMESS, model.VLESS, model.Trojan, model.Hysteria, model.AnyTLS:
		_, stream, err := managedLineInboundJSON(node, protocol, hostname, port)
		if err != nil {
			return err
		}
		updates["stream_settings"] = string(stream)
	}
	if err := w.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Updates(updates).Error; err != nil {
			return err
		}
		if protocol == model.VLESS && managedLineUsesWebSocket443(protocol) {
			if err := tx.Model(&model.ClientInbound{}).Where("inbound_id = ?", inbound.Id).Update("flow_override", "").Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{
			"public_port":       port,
			"provision_version": managedLineProvisionVersion,
		}).Error
	}); err != nil {
		return err
	}
	node.PublicPort = &port
	node.ProvisionVersion = managedLineProvisionVersion
	return nil
}

func managedLineListenAddress(protocol model.Protocol) string {
	if managedLineUsesWebSocket443(protocol) {
		return "127.0.0.1"
	}
	return "0.0.0.0"
}

func managedLineUsesWebSocket443(protocol model.Protocol) bool {
	switch protocol {
	case model.VMESS, model.VLESS, model.Trojan:
		return true
	default:
		return false
	}
}

func managedLineInboundProtocol(protocol string) model.Protocol {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "vmess":
		return model.VMESS
	case "trojan":
		return model.Trojan
	case "shadowsocks", "ss":
		return model.Shadowsocks
	case "hysteria", "hysteria2", "hy2":
		return model.Hysteria
	case "wireguard", "wg":
		return model.WireGuard
	case "anytls":
		return model.AnyTLS
	default:
		return model.VLESS
	}
}

func managedLineWebSocketStream(node *model.LineNode, hostname string, port int) map[string]any {
	token := managedLineWebSocketPathToken(node)
	path := managedLineWSPathPrefix + strconv.Itoa(port) + "/" + token
	return map[string]any{
		"network":  "ws",
		"security": "none",
		"wsSettings": map[string]any{
			"path": path,
			"host": hostname,
		},
		"externalProxy": []any{map[string]any{
			"dest":                 hostname,
			"port":                 managedLinePublicPort,
			"forceTls":             "tls",
			"sni":                  hostname,
			"fingerprint":          "chrome",
			"verifyPeerCertByName": hostname,
		}},
	}
}

func managedLineWebSocketPathToken(node *model.LineNode) string {
	candidate := strings.ToLower(strings.TrimSpace(node.Fingerprint))
	if len(candidate) >= 16 {
		candidate = candidate[:16]
		valid := true
		for _, char := range candidate {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
				valid = false
				break
			}
		}
		if valid {
			return candidate
		}
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(node.ID)))
	return hex.EncodeToString(sum[:8])
}

func managedLineInboundJSON(node *model.LineNode, protocol model.Protocol, hostname string, port int) ([]byte, []byte, error) {
	var settings map[string]any
	stream := map[string]any{}
	switch protocol {
	case model.VLESS:
		if len(node.Fingerprint) < 16 {
			return nil, nil, errors.New("线路节点指纹无效")
		}
		settings = map[string]any{"clients": []any{}, "decryption": "none", "encryption": "none", "fallbacks": []any{}}
		stream = managedLineWebSocketStream(node, hostname, port)
	case model.VMESS:
		settings = map[string]any{"clients": []any{}}
		stream = managedLineWebSocketStream(node, hostname, port)
	case model.Trojan:
		settings = map[string]any{"clients": []any{}, "fallbacks": []any{}}
		stream = managedLineWebSocketStream(node, hostname, port)
	case model.Hysteria:
		certFile, keyFile, err := managedLineTLSFiles()
		if err != nil {
			return nil, nil, err
		}
		tlsSettings := map[string]any{
			"serverName": hostname,
			"alpn":       []string{"h3", "h2", "http/1.1"},
			"certificates": []any{map[string]any{
				"certificateFile": certFile,
				"keyFile":         keyFile,
			}},
			"settings": map[string]any{"verifyPeerCertByName": hostname},
		}
		settings = map[string]any{"version": 2, "clients": []any{}}
		stream = map[string]any{"network": "hysteria", "security": "tls", "tlsSettings": tlsSettings, "hysteriaSettings": map[string]any{"version": 2, "udpIdleTimeout": 60}}
	case model.Shadowsocks:
		masterKey, err := randomManagedLineKey(32)
		if err != nil {
			return nil, nil, err
		}
		settings = map[string]any{"method": "2022-blake3-aes-256-gcm", "password": masterKey, "network": "tcp,udp", "clients": []any{}, "ivCheck": false}
		stream = map[string]any{"network": "tcp", "security": "none"}
	case model.WireGuard:
		privateKey, _, err := wgutil.GenerateWireguardKeypair()
		if err != nil {
			return nil, nil, err
		}
		settings = map[string]any{"secretKey": privateKey, "peers": []any{}, "mtu": 1420}
	case model.AnyTLS:
		certFile, keyFile, err := managedLineTLSFiles()
		if err != nil {
			return nil, nil, err
		}
		settings = map[string]any{"clients": []any{}}
		stream = map[string]any{
			"network": "tcp", "security": "tls",
			"tlsSettings": map[string]any{
				"serverName": hostname,
				"certificates": []any{map[string]any{
					"certificateFile": certFile,
					"keyFile":         keyFile,
				}},
			},
		}
	default:
		return nil, nil, fmt.Errorf("unsupported managed line protocol %q", protocol)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, nil, err
	}
	streamJSON, err := json.Marshal(stream)
	if err != nil {
		return nil, nil, err
	}
	return settingsJSON, streamJSON, nil
}

func managedLineTLSFiles() (string, string, error) {
	certFile := strings.TrimSpace(os.Getenv("XUI_LINE_CERT_FILE"))
	keyFile := strings.TrimSpace(os.Getenv("XUI_LINE_KEY_FILE"))
	if certFile == "" {
		certFile = "/var/lib/x-ui/certs/fullchain.pem"
	}
	if keyFile == "" {
		keyFile = "/var/lib/x-ui/certs/privkey.pem"
	}
	if _, err := os.Stat(certFile); err != nil {
		return "", "", fmt.Errorf("本站线路 TLS 证书不可用: %w", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return "", "", fmt.Errorf("本站线路 TLS 私钥不可用: %w", err)
	}
	return certFile, keyFile, nil
}

func randomManagedLineKey(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(value), nil
}

func allocateManagedLinePort(db *gorm.DB, protocol model.Protocol, nodeID string) (int, error) {
	if protocol == model.Hysteria && managedLineUDPPortAvailable(db, nodeID, managedLinePublicPort) {
		return managedLinePublicPort, nil
	}
	span := big.NewInt(40000)
	for range 512 {
		value, err := rand.Int(rand.Reader, span)
		if err != nil {
			return 0, err
		}
		port := 20000 + int(value.Int64())
		var inboundCount int64
		if err := db.Model(&model.Inbound{}).Where("node_id IS NULL AND port = ?", port).Count(&inboundCount).Error; err != nil {
			return 0, err
		}
		if inboundCount > 0 {
			continue
		}
		var lineCount int64
		if err := db.Model(&model.LineNode{}).Where("public_port = ?", port).Count(&lineCount).Error; err != nil {
			return 0, err
		}
		if lineCount > 0 {
			continue
		}
		listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		packet, packetErr := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", fmt.Sprintf(":%d", port))
		if packetErr != nil {
			_ = listener.Close()
			continue
		}
		_ = packet.Close()
		_ = listener.Close()
		return port, nil
	}
	return 0, errors.New("无法分配可用的随机线路端口")
}

func managedLineUDPPortAvailable(db *gorm.DB, nodeID string, port int) bool {
	var lineCount int64
	query := db.Model(&model.LineNode{}).Where("public_port = ?", port)
	if strings.TrimSpace(nodeID) != "" {
		query = query.Where("id <> ?", nodeID)
	}
	if err := query.Count(&lineCount).Error; err != nil || lineCount > 0 {
		return false
	}
	packet, err := (&net.ListenConfig{}).ListenPacket(context.Background(), "udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = packet.Close()
	return true
}

func (w *Worker) attachLineSubscribers(nodeID string) error {
	var emails []string
	err := w.db.Table(model.SubscriptionEntitlement{}.TableName()+" AS entitlements").
		Distinct("entitlements.internal_client_id").
		Joins("JOIN "+model.PlanLineGroup{}.TableName()+" AS plan_groups ON plan_groups.plan_id = entitlements.plan_id").
		Joins("JOIN "+model.LineGroupNode{}.TableName()+" AS group_nodes ON group_nodes.group_id = plan_groups.group_id").
		Where("group_nodes.node_id = ? AND entitlements.status = ? AND (entitlements.expires_at IS NULL OR entitlements.expires_at > ?)", nodeID, "active", time.Now().UTC()).
		Pluck("entitlements.internal_client_id", &emails).Error
	if err != nil {
		return err
	}
	var node model.LineNode
	if err := w.db.First(&node, "id = ?", nodeID).Error; err != nil {
		return err
	}
	if node.InboundID == nil {
		return errors.New("线路节点没有托管入站")
	}
	for _, email := range emails {
		if _, err := w.clients.AttachByEmail(&w.inbounds, email, []int{*node.InboundID}); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processNextLineSource(ctx context.Context) error {
	now := time.Now().UTC()
	var source model.LineSource
	err := w.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("kind = ? AND enabled = ? AND status <> ? AND next_refresh_at <= ?", lineSourceURL, true, "refreshing", now).Order("next_refresh_at asc")
		if err := query.First(&source).Error; err != nil {
			return err
		}
		return tx.Model(&source).Updates(map[string]any{"status": "refreshing", "locked_at": now, "last_error": ""}).Error
	})
	if err != nil {
		return err
	}
	plain, err := service.UnprotectCredential(source.SecretCiphertext)
	if err != nil {
		return w.failLineSource(&source, err)
	}
	_, entries, err := NewLineService().fetchURLSource(ctx, plain)
	if err != nil {
		return w.failLineSource(&source, err)
	}
	prepared, err := prepareLineEntries(entries)
	if err != nil || len(prepared) == 0 {
		if err == nil {
			err = errors.New("订阅刷新没有返回受支持节点")
		}
		return w.failLineSource(&source, err)
	}
	var groupIDs []string
	if err := w.db.Model(&model.LineSourceGroup{}).Where("source_id = ?", source.ID).Pluck("group_id", &groupIDs).Error; err != nil {
		return w.failLineSource(&source, err)
	}
	var previousNodeIDs []string
	if err := w.db.Model(&model.LineSourceNode{}).Where("source_id = ?", source.ID).Pluck("node_id", &previousNodeIDs).Error; err != nil {
		return w.failLineSource(&source, err)
	}
	next := now.Add(time.Duration(source.RefreshInterval) * time.Second)
	err = w.db.Transaction(func(tx *gorm.DB) error {
		if err := NewLineService().upsertPreparedLines(tx, source.ID, prepared, groupIDs, now); err != nil {
			return err
		}
		return tx.Model(&model.LineSource{}).Where("id = ? AND status = ?", source.ID, "refreshing").Updates(map[string]any{"status": "ready", "last_error": "", "last_success_at": now, "next_refresh_at": next, "locked_at": nil, "consecutive_successes": gorm.Expr("consecutive_successes + 1")}).Error
	})
	if err != nil {
		return w.failLineSource(&source, err)
	}
	return NewLineService().markOrphanedNodesStale(previousNodeIDs)
}

func (w *Worker) failLineSource(source *model.LineSource, cause error) error {
	next := time.Now().UTC().Add(time.Duration(source.RefreshInterval) * time.Second)
	_ = w.db.Model(&model.LineSource{}).Where("id = ?", source.ID).Updates(map[string]any{"status": "error", "last_error": cause.Error(), "next_refresh_at": next, "locked_at": nil, "consecutive_successes": 0}).Error
	return cause
}

func (w *Worker) recoverLineLocks() {
	now := time.Now().UTC()
	cutoff := now.Add(-15 * time.Minute)
	_ = w.db.Model(&model.LineNode{}).Where("status = ? AND provision_locked_at < ?", lineStatusProvisioning, cutoff).Updates(map[string]any{"status": lineStatusRetry, "provision_locked_at": nil, "next_provision_at": now, "last_error": "recovered stale line provisioning lock"}).Error
	_ = w.db.Model(&model.LineSource{}).Where("status = ? AND locked_at < ?", "refreshing", cutoff).Updates(map[string]any{"status": "error", "locked_at": nil, "next_refresh_at": now, "last_error": "recovered stale line refresh lock"}).Error
}

// restoreManagedLinePublication upgrades the previous health-gated behavior.
// A failed external probe is monitoring information only: syntactically valid,
// provisioned lines remain published until an administrator unassigns them or
// their source enters the stale retention flow.
func (w *Worker) restoreManagedLinePublication() error {
	var nodes []model.LineNode
	err := w.db.Model(&model.LineNode{}).
		Where("missing_since IS NULL AND inbound_id IS NOT NULL").
		Where("EXISTS (SELECT 1 FROM "+model.LineGroupNode{}.TableName()+" memberships JOIN "+model.LineGroup{}.TableName()+" groups ON groups.id = memberships.group_id WHERE memberships.node_id = "+model.LineNode{}.TableName()+".id AND groups.active = ?)", true).
		Find(&nodes).Error
	if err != nil {
		return err
	}
	restart := false
	for i := range nodes {
		// Nodes imported before certificate pinning was introduced may still
		// contain allowInsecure=true. Queue every such node once at startup so
		// existing installations repair themselves without a delete/re-import.
		if !nodes[i].TLSAutoPinned {
			plain, decryptErr := service.UnprotectCredential(nodes[i].OutboundCiphertext)
			if decryptErr == nil {
				var outbound map[string]any
				if json.Unmarshal([]byte(plain), &outbound) == nil && outboundNeedsTLSAutoPin(outbound) {
					if err := w.db.Model(&model.LineNode{}).Where("id = ?", nodes[i].ID).Updates(map[string]any{
						"tls_auto_pinned": true, "status": lineStatusChecking, "health_status": lineHealthChecking,
						"provision_locked_at": nil, "next_provision_at": nil, "last_error": "",
					}).Error; err != nil {
						return err
					}
					nodes[i].TLSAutoPinned = true
					nodes[i].Status = lineStatusChecking
					nodes[i].HealthStatus = lineHealthChecking
				}
			}
		}
		var inbound model.Inbound
		if err := w.db.Select("id", "enable", "protocol").First(&inbound, "id = ?", *nodes[i].InboundID).Error; err != nil {
			continue
		}
		if inbound.Protocol != managedLineInboundProtocol(nodes[i].Protocol) || nodes[i].ProvisionVersion < managedLineProvisionVersion {
			if err := w.db.Model(&model.LineNode{}).Where("id = ?", nodes[i].ID).Updates(map[string]any{
				"status": lineStatusChecking, "health_status": lineHealthChecking, "provision_locked_at": nil, "next_provision_at": nil, "last_error": "",
			}).Error; err != nil {
				return err
			}
			continue
		}
		if !inbound.Enable {
			if err := w.db.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("enable", true).Error; err != nil {
				return err
			}
			restart = true
		}
		if nodes[i].Status == lineStatusHealthy || nodes[i].Status == lineStatusOffline {
			if err := w.db.Model(&model.LineNode{}).Where("id = ?", nodes[i].ID).Update("status", lineStatusReady).Error; err != nil {
				return err
			}
		}
	}
	if restart {
		return (&service.XrayService{}).RestartXray(false)
	}
	return nil
}

func (w *Worker) cleanupStaleLine() error {
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour)
	var node model.LineNode
	err := w.db.Where("status = ? AND missing_since < ?", lineStatusStale, cutoff).
		Where("NOT EXISTS (SELECT 1 FROM "+model.LineSourceNode{}.TableName()+" memberships JOIN "+model.LineSource{}.TableName()+" sources ON sources.id = memberships.source_id WHERE memberships.node_id = "+model.LineNode{}.TableName()+".id AND sources.consecutive_successes < ?)", 2).
		Order("missing_since asc").First(&node).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if node.InboundID != nil {
		if _, err := w.inbounds.DelInbound(*node.InboundID); err != nil {
			return err
		}
	}
	return w.db.Delete(&model.LineNode{}, "id = ?", node.ID).Error
}

func (w *Worker) applyLineHealthEvent(event eventbus.Event) error {
	data, ok := event.Data.(*eventbus.OutboundHealthData)
	if !ok || data == nil || data.LastTryTime <= 0 {
		return nil
	}
	var node model.LineNode
	if err := w.db.Where("outbound_tag = ? AND missing_since IS NULL", event.Source).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	probeAt := normalizeProbeTime(data.LastTryTime)
	if node.LastProbeAt != nil && !probeAt.After(*node.LastProbeAt) {
		return nil
	}
	updates := map[string]any{"last_probe_at": probeAt, "latency_ms": data.Delay}
	if data.Alive {
		passes := node.ConsecutivePasses + 1
		updates["consecutive_passes"] = passes
		updates["consecutive_fails"] = 0
		updates["last_error"] = ""
		if node.HealthStatus != lineHealthHealthy && (node.HealthStatus != lineHealthOffline || passes >= 2) {
			updates["health_status"] = lineHealthHealthy
		}
	} else {
		fails := node.ConsecutiveFails + 1
		updates["consecutive_fails"] = fails
		updates["consecutive_passes"] = 0
		updates["last_error"] = data.Error
		if node.HealthStatus != lineHealthOffline && fails >= 3 {
			updates["health_status"] = lineHealthOffline
		}
	}
	if err := w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

func normalizeProbeTime(value int64) time.Time {
	if value > 1_000_000_000_000_000 {
		return time.Unix(0, value).UTC()
	}
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC()
	}
	return time.Unix(value, 0).UTC()
}

func (w *Worker) reconcileActivePlanClients(planID string) error {
	var plan model.Plan
	if err := w.db.First(&plan, "id = ?", planID).Error; err != nil {
		return err
	}
	desired, err := w.provisionInboundIDs(&plan)
	if err != nil {
		return err
	}
	var entitlements []model.SubscriptionEntitlement
	if err := w.db.Where("plan_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", planID, "active", time.Now().UTC()).Find(&entitlements).Error; err != nil {
		return err
	}
	desiredSet := make(map[int]struct{}, len(desired))
	for _, inboundID := range desired {
		desiredSet[inboundID] = struct{}{}
	}
	for i := range entitlements {
		email := entitlements[i].InternalClientID
		current, err := w.clients.GetInboundIdsForEmail(w.db, email)
		if err != nil {
			return err
		}
		needRuntimeRestart, err := w.clients.AttachByEmail(&w.inbounds, email, desired)
		if err != nil {
			return err
		}
		obsolete := make([]int, 0)
		for _, inboundID := range current {
			if _, keep := desiredSet[inboundID]; !keep {
				obsolete = append(obsolete, inboundID)
			}
		}
		if len(obsolete) > 0 {
			needRestart, detachErr := w.clients.DetachByEmailMany(&w.inbounds, email, obsolete)
			if detachErr != nil {
				return detachErr
			}
			needRuntimeRestart = needRuntimeRestart || needRestart
		}
		if err := w.applyClientRuntimeMutation(needRuntimeRestart); err != nil {
			return fmt.Errorf("apply reconciled plan client to proxy runtime: %w", err)
		}
	}
	return nil
}

func planHasPublishedLineDB(db *gorm.DB, planID string, manualInboundIDs []int) (bool, error) {
	if len(manualInboundIDs) > 0 {
		var count int64
		if err := db.Model(&model.Inbound{}).Where("id IN ? AND enable = ?", manualInboundIDs, true).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	var count int64
	err := db.Table(model.PlanLineGroup{}.TableName()+" AS plan_groups").
		Joins("JOIN "+model.LineGroup{}.TableName()+" AS groups ON groups.id = plan_groups.group_id AND groups.active = true").
		Joins("JOIN "+model.LineGroupNode{}.TableName()+" AS group_nodes ON group_nodes.group_id = plan_groups.group_id").
		Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = group_nodes.node_id").
		Joins("JOIN inbounds ON inbounds.id = nodes.inbound_id").
		Where("plan_groups.plan_id = ? AND nodes.missing_since IS NULL AND nodes.inbound_id IS NOT NULL AND inbounds.enable = ?", planID, true).
		Count(&count).Error
	return count > 0, err
}

func hasPublishedLineForGroupsDB(db *gorm.DB, groupIDs []string, manualInboundIDs []int) (bool, error) {
	if len(manualInboundIDs) > 0 {
		var count int64
		if err := db.Model(&model.Inbound{}).Where("id IN ? AND enable = ?", manualInboundIDs, true).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	if len(groupIDs) == 0 {
		return false, nil
	}
	var count int64
	err := db.Table(model.LineGroupNode{}.TableName()+" AS group_nodes").
		Joins("JOIN "+model.LineGroup{}.TableName()+" AS groups ON groups.id = group_nodes.group_id AND groups.active = true").
		Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = group_nodes.node_id").
		Joins("JOIN inbounds ON inbounds.id = nodes.inbound_id").
		Where("group_nodes.group_id IN ? AND nodes.missing_since IS NULL AND nodes.inbound_id IS NOT NULL AND inbounds.enable = ?", groupIDs, true).
		Count(&count).Error
	return count > 0, err
}

func sortedUniqueInts(values []int) []int {
	seen := map[int]bool{}
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}
