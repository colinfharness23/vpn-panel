package commercial

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/eventbus"
	linkutil "github.com/mhsanaei/3x-ui/v3/internal/util/link"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	outboundservice "github.com/mhsanaei/3x-ui/v3/internal/web/service/outbound"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const lineStatusRetry = "retry"

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
	if err := (&service.XrayService{}).RestartXray(false); err != nil {
		return err
	}
	if node.PublicPort == nil {
		return errors.New("托管线路没有公网端口")
	}
	if err := waitManagedLineListener(ctx, *node.PublicPort, 8*time.Second); err != nil {
		_ = w.db.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Update("enable", false).Error
		_ = (&service.XrayService{}).RestartXray(true)
		return err
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

func (w *Worker) ensureManagedLineInbound(node *model.LineNode) error {
	if node.InboundID != nil && node.PublicPort != nil {
		var count int64
		if err := w.db.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Count(&count).Error; err != nil {
			return err
		}
		if count == 1 {
			return nil
		}
	}
	port, err := allocateManagedLinePort(w.db)
	if err != nil {
		return err
	}
	privateKey, publicKey, err := lineRealityKeys()
	if err != nil {
		return err
	}
	siteURL := NewConfigStore().GetDefault("site.url", "")
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("请先在商业设置中配置站点域名")
	}
	settings, _ := json.Marshal(map[string]any{"clients": []any{}, "decryption": "none", "encryption": "none", "fallbacks": []any{}})
	stream, _ := json.Marshal(map[string]any{
		"network":     "tcp",
		"security":    "reality",
		"tcpSettings": map[string]any{"header": map[string]any{"type": "none"}},
		"realitySettings": map[string]any{
			"show": false, "xver": 0, "target": "127.0.0.1:443", "serverNames": []string{parsed.Hostname()},
			"privateKey": privateKey, "minClientVer": "", "maxClientVer": "", "maxTimediff": 0,
			"shortIds": []string{node.Fingerprint[:16]}, "mldsa65Seed": "",
			"settings": map[string]any{"publicKey": publicKey, "fingerprint": "chrome", "serverName": parsed.Hostname(), "spiderX": "/", "mldsa65Verify": ""},
		},
	})
	sniffing := `{"enabled":true,"destOverride":["http","tls","quic"],"metadataOnly":false,"routeOnly":false}`
	siteName := NewConfigStore().GetDefault("site.name", "NOVA")
	publicName := strings.TrimSpace(node.PublicName)
	if publicName == "" {
		publicName = strings.ToUpper(strings.TrimSpace(node.Protocol)) + " 线路"
	}
	inbound := &model.Inbound{UserId: 1, Remark: fmt.Sprintf("%s · %s", siteName, publicName), SubSortIndex: 1, Enable: false, Listen: "0.0.0.0", Port: port, Protocol: model.VLESS, Settings: string(settings), StreamSettings: string(stream), Tag: "commercial-in-" + strings.ReplaceAll(node.ID, "-", "")[:20], Sniffing: sniffing, ShareAddrStrategy: "custom", ShareAddr: parsed.Hostname()}
	created, _, err := w.inbounds.AddInbound(inbound)
	if err != nil {
		return err
	}
	if err := w.db.Model(&model.LineNode{}).Where("id = ?", node.ID).Updates(map[string]any{"public_port": port, "inbound_id": created.Id}).Error; err != nil {
		_, _ = w.inbounds.DelInbound(created.Id)
		return err
	}
	node.PublicPort = &port
	node.InboundID = &created.Id
	return nil
}

func lineRealityKeys() (string, string, error) {
	config := NewConfigStore()
	privateKey, privateErr := config.Get("line.reality_private_key")
	publicKey, publicErr := config.Get("line.reality_public_key")
	if privateErr == nil && publicErr == nil && privateKey != "" && publicKey != "" {
		return privateKey, publicKey, nil
	}
	raw, err := (&service.ServerService{}).GetNewX25519Cert()
	if err != nil {
		return "", "", err
	}
	pair, ok := raw.(map[string]any)
	if !ok {
		return "", "", errors.New("Xray 未返回有效 Reality 密钥")
	}
	privateKey, _ = pair["privateKey"].(string)
	publicKey, _ = pair["publicKey"].(string)
	if privateKey == "" || publicKey == "" {
		return "", "", errors.New("Xray 返回的 Reality 密钥为空")
	}
	if err := config.SetManyProtected(map[string]string{"line.reality_private_key": privateKey, "line.reality_public_key": publicKey}, map[string]bool{"line.reality_private_key": true}); err != nil {
		return "", "", err
	}
	return privateKey, publicKey, nil
}

func allocateManagedLinePort(db *gorm.DB) (int, error) {
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
		_ = listener.Close()
		return port, nil
	}
	return 0, errors.New("无法分配可用的随机线路端口")
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
		if err := w.db.Select("id", "enable").First(&inbound, "id = ?", *nodes[i].InboundID).Error; err != nil {
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
		if _, err := w.clients.AttachByEmail(&w.inbounds, email, desired); err != nil {
			return err
		}
		obsolete := make([]int, 0)
		for _, inboundID := range current {
			if _, keep := desiredSet[inboundID]; !keep {
				obsolete = append(obsolete, inboundID)
			}
		}
		if len(obsolete) > 0 {
			if _, err := w.clients.DetachByEmailMany(&w.inbounds, email, obsolete); err != nil {
				return err
			}
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
