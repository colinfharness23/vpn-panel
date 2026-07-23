package commercial

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/link"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	lineSourceURL                = "url"
	lineSourceManual             = "manual"
	lineStatusUnassigned         = "unassigned"
	lineStatusChecking           = "checking"
	lineStatusProvisioning       = "provisioning"
	lineStatusReady              = "ready"
	lineStatusHealthy            = "healthy"
	lineStatusOffline            = "offline"
	lineStatusStale              = "stale"
	lineHealthUnchecked          = "unchecked"
	lineHealthChecking           = "checking"
	lineHealthHealthy            = "healthy"
	lineHealthOffline            = "offline"
	lineTagPrefix                = "commercial-line-"
	lineSourceMaxBytes     int64 = 8 << 20
	lineImportMaxEntries         = 500
	lineSourceUserAgent          = "v2rayN/7.15.0"
)

var supportedLineProtocols = map[string]struct{}{
	"vmess":       {},
	"vless":       {},
	"trojan":      {},
	"shadowsocks": {},
	"hysteria":    {},
	"wireguard":   {},
	"anytls":      {},
}

type LineSourceView struct {
	model.LineSource
	GroupIDs       []string `json:"groupIds"`
	NodeCount      int64    `json:"nodeCount"`
	HealthyCount   int64    `json:"healthyCount"`
	PublishedCount int64    `json:"publishedCount"`
	HasSecret      bool     `json:"hasSecret"`
}

type LineGroupView struct {
	model.LineGroup
	PlanIDs        []string `json:"planIds"`
	NodeCount      int64    `json:"nodeCount"`
	HealthyCount   int64    `json:"healthyCount"`
	PublishedCount int64    `json:"publishedCount"`
}

type LineNodeView struct {
	model.LineNode
	SourceIDs []string `json:"sourceIds"`
	GroupIDs  []string `json:"groupIds"`
	Published bool     `json:"published"`
}

type LineOverview struct {
	Sources []LineSourceView `json:"sources"`
	Groups  []LineGroupView  `json:"groups"`
	Nodes   []LineNodeView   `json:"nodes"`
}

type LineImportEntry struct {
	Index       int    `json:"index"`
	Remark      string `json:"remark,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Valid       bool   `json:"valid"`
	Duplicate   bool   `json:"duplicate"`
	Error       string `json:"error,omitempty"`
	outbound    link.Outbound
	identity    string
}

type LineImportPreview struct {
	Entries        []LineImportEntry `json:"entries"`
	ValidCount     int               `json:"validCount"`
	InvalidCount   int               `json:"invalidCount"`
	DuplicateCount int               `json:"duplicateCount"`
}

type LineService struct {
	db   *gorm.DB
	xray service.XrayService
}

func NewLineService() *LineService {
	return &LineService{db: database.GetDB()}
}

func (s *LineService) Overview() (*LineOverview, error) {
	sources, err := s.Sources()
	if err != nil {
		return nil, err
	}
	groups, err := s.Groups()
	if err != nil {
		return nil, err
	}
	nodes, err := s.Nodes("", "", "")
	if err != nil {
		return nil, err
	}
	return &LineOverview{Sources: sources, Groups: groups, Nodes: nodes}, nil
}

func (s *LineService) Sources() ([]LineSourceView, error) {
	var rows []model.LineSource
	if err := s.db.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]LineSourceView, 0, len(rows))
	for i := range rows {
		view := LineSourceView{LineSource: rows[i], GroupIDs: []string{}, HasSecret: rows[i].SecretCiphertext != ""}
		if err := s.db.Model(&model.LineSourceGroup{}).Where("source_id = ?", rows[i].ID).Order("group_id asc").Pluck("group_id", &view.GroupIDs).Error; err != nil {
			return nil, err
		}
		if err := s.db.Model(&model.LineSourceNode{}).Where("source_id = ? AND missing_since IS NULL", rows[i].ID).Count(&view.NodeCount).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(model.LineSourceNode{}.TableName()+" AS links").Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = links.node_id").Where("links.source_id = ? AND links.missing_since IS NULL AND nodes.health_status = ?", rows[i].ID, lineHealthHealthy).Count(&view.HealthyCount).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(model.LineSourceNode{}.TableName()+" AS links").Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = links.node_id").Joins("JOIN inbounds ON inbounds.id = nodes.inbound_id").Where("links.source_id = ? AND links.missing_since IS NULL AND nodes.inbound_id IS NOT NULL AND inbounds.enable = ?", rows[i].ID, true).Count(&view.PublishedCount).Error; err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *LineService) Groups() ([]LineGroupView, error) {
	var rows []model.LineGroup
	if err := s.db.Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]LineGroupView, 0, len(rows))
	for i := range rows {
		view := LineGroupView{LineGroup: rows[i], PlanIDs: []string{}}
		if err := s.db.Model(&model.PlanLineGroup{}).Where("group_id = ?", rows[i].ID).Order("plan_id asc").Pluck("plan_id", &view.PlanIDs).Error; err != nil {
			return nil, err
		}
		if err := s.db.Model(&model.LineGroupNode{}).Where("group_id = ?", rows[i].ID).Count(&view.NodeCount).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(model.LineGroupNode{}.TableName()+" AS memberships").Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = memberships.node_id").Where("memberships.group_id = ? AND nodes.health_status = ?", rows[i].ID, lineHealthHealthy).Count(&view.HealthyCount).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(model.LineGroupNode{}.TableName()+" AS memberships").Joins("JOIN "+model.LineNode{}.TableName()+" AS nodes ON nodes.id = memberships.node_id").Joins("JOIN inbounds ON inbounds.id = nodes.inbound_id").Where("memberships.group_id = ? AND nodes.missing_since IS NULL AND nodes.inbound_id IS NOT NULL AND inbounds.enable = ?", rows[i].ID, true).Count(&view.PublishedCount).Error; err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *LineService) Nodes(sourceID, groupID, status string) ([]LineNodeView, error) {
	query := s.db.Model(&model.LineNode{}).Distinct("commercial_line_nodes.*")
	if sourceID != "" {
		query = query.Joins("JOIN "+model.LineSourceNode{}.TableName()+" ON commercial_line_source_nodes.node_id = commercial_line_nodes.id").Where("commercial_line_source_nodes.source_id = ?", sourceID)
	}
	if groupID != "" {
		query = query.Joins("JOIN "+model.LineGroupNode{}.TableName()+" ON commercial_line_group_nodes.node_id = commercial_line_nodes.id").Where("commercial_line_group_nodes.group_id = ?", groupID)
	}
	if status != "" {
		query = query.Where("commercial_line_nodes.status = ?", status)
	}
	var rows []model.LineNode
	if err := query.Order("commercial_line_nodes.created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]LineNodeView, 0, len(rows))
	for i := range rows {
		view := LineNodeView{LineNode: rows[i], SourceIDs: []string{}, GroupIDs: []string{}}
		if err := s.db.Model(&model.LineSourceNode{}).Where("node_id = ? AND missing_since IS NULL", rows[i].ID).Pluck("source_id", &view.SourceIDs).Error; err != nil {
			return nil, err
		}
		if err := s.db.Model(&model.LineGroupNode{}).Where("node_id = ?", rows[i].ID).Pluck("group_id", &view.GroupIDs).Error; err != nil {
			return nil, err
		}
		if rows[i].InboundID != nil && rows[i].MissingSince == nil {
			var inbound model.Inbound
			err := s.db.Select("id", "enable").First(&inbound, "id = ?", *rows[i].InboundID).Error
			switch {
			case err == nil:
				view.Published = inbound.Enable
			case !errors.Is(err, gorm.ErrRecordNotFound):
				return nil, err
			}
		}
		sort.Strings(view.SourceIDs)
		sort.Strings(view.GroupIDs)
		views = append(views, view)
	}
	return views, nil
}

func (s *LineService) UpdateNode(id string, request entity.CommercialLineNodeUpdateRequest) (*model.LineNode, error) {
	id = strings.TrimSpace(id)
	publicName := strings.TrimSpace(request.PublicName)
	if id == "" || publicName == "" || len([]rune(publicName)) > 160 || strings.ContainsAny(publicName, "\r\n\x00") {
		return nil, errors.New("线路别名无效")
	}
	result := s.db.Model(&model.LineNode{}).Where("id = ?", id).Updates(map[string]any{"public_name": publicName, "public_name_custom": true})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("线路节点不存在")
	}
	var row model.LineNode
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *LineService) SaveGroup(request entity.CommercialLineGroupRequest) (*model.LineGroup, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	if request.Name == "" || len([]rune(request.Name)) > 120 || len([]rune(request.Description)) > 500 {
		return nil, errors.New("线路组名称或说明无效")
	}
	row := &model.LineGroup{ID: request.ID, Name: request.Name, Description: request.Description, Active: request.Active}
	if row.ID == "" {
		row.ID = uuid.NewString()
		if err := s.db.Create(row).Error; err != nil {
			return nil, lineDuplicateError(err, "线路组名称已存在")
		}
		if err := s.db.Model(row).Update("active", request.Active).Error; err != nil {
			return nil, err
		}
		row.Active = request.Active
		return row, nil
	}
	result := s.db.Model(row).Where("id = ?", row.ID).Updates(map[string]any{"name": row.Name, "description": row.Description, "active": row.Active})
	if result.Error != nil {
		return nil, lineDuplicateError(result.Error, "线路组名称已存在")
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("线路组不存在")
	}
	if err := s.db.First(row, "id = ?", row.ID).Error; err != nil {
		return nil, err
	}
	if err := s.reconcilePlansForGroups([]string{row.ID}); err != nil {
		return row, err
	}
	return row, nil
}

func (s *LineService) DeleteGroup(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("线路组不存在")
	}
	var nodeIDs []string
	if err := s.db.Model(&model.LineGroupNode{}).Where("group_id = ?", id).Pluck("node_id", &nodeIDs).Error; err != nil {
		return err
	}
	var planIDs []string
	if err := s.db.Model(&model.PlanLineGroup{}).Where("group_id = ?", id).Pluck("plan_id", &planIDs).Error; err != nil {
		return err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", id).Delete(&model.PlanLineGroup{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", id).Delete(&model.LineGroupNode{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.LineGroup{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("线路组不存在")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.normalizeAssignmentStates(nodeIDs); err != nil {
		return err
	}
	return s.reconcilePlans(planIDs)
}

func (s *LineService) PreviewImport(raw string) (*LineImportPreview, error) {
	entries, err := parseLineEntries(raw)
	if err != nil {
		return nil, err
	}
	fingerprints := make([]string, 0, len(entries))
	for i := range entries {
		if entries[i].Valid {
			fingerprints = append(fingerprints, entries[i].Fingerprint)
		}
	}
	existing := map[string]bool{}
	if len(fingerprints) > 0 {
		var values []string
		if err := s.db.Model(&model.LineNode{}).Where("fingerprint IN ?", fingerprints).Pluck("fingerprint", &values).Error; err != nil {
			return nil, err
		}
		for _, value := range values {
			existing[value] = true
		}
	}
	preview := &LineImportPreview{Entries: entries}
	seen := map[string]bool{}
	for i := range preview.Entries {
		entry := &preview.Entries[i]
		if !entry.Valid {
			preview.InvalidCount++
			continue
		}
		entry.Duplicate = existing[entry.Fingerprint] || seen[entry.Fingerprint]
		seen[entry.Fingerprint] = true
		if entry.Duplicate {
			preview.DuplicateCount++
		} else {
			preview.ValidCount++
		}
	}
	return preview, nil
}

func (s *LineService) CommitImport(request entity.CommercialLineImportRequest) (*LineSourceView, error) {
	preview, err := s.PreviewImport(request.Links)
	if err != nil {
		return nil, err
	}
	prepared, err := prepareLineEntries(preview.Entries)
	if err != nil {
		return nil, err
	}
	if len(prepared) == 0 {
		return nil, errors.New("没有可导入的新协议链接")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "手动导入 " + time.Now().Format("2006-01-02 15:04")
	}
	secret, err := service.ProtectCredential(request.Links)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	source := model.LineSource{ID: uuid.NewString(), Name: name, Kind: lineSourceManual, SecretCiphertext: secret, RefreshInterval: 0, Enabled: false, Status: "ready", LastSuccessAt: &now, ConsecutiveSuccesses: 1}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&source).Error; err != nil {
			return err
		}
		if err := tx.Model(&source).Update("enabled", false).Error; err != nil {
			return err
		}
		source.Enabled = false
		return s.upsertPreparedLines(tx, source.ID, prepared, nil, now)
	}); err != nil {
		return nil, err
	}
	views, err := s.Sources()
	if err != nil {
		return nil, err
	}
	for i := range views {
		if views[i].ID == source.ID {
			return &views[i], nil
		}
	}
	return nil, errors.New("导入来源保存失败")
}

func (s *LineService) SaveURLSource(ctx context.Context, request entity.CommercialLineSourceRequest) (*LineSourceView, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.URL = strings.TrimSpace(request.URL)
	if request.RefreshInterval == 0 {
		request.RefreshInterval = 1800
	}
	if request.Name == "" || len([]rune(request.Name)) > 120 {
		return nil, errors.New("订阅来源名称无效")
	}
	if request.RefreshInterval < 300 || request.RefreshInterval > 86400 {
		return nil, errors.New("刷新周期必须为 5 分钟到 24 小时")
	}
	if len(request.GroupIDs) == 0 {
		return nil, errors.New("订阅 URL 必须选择至少一个线路组")
	}
	if err := s.validateGroupsAndPlans(request.GroupIDs, request.PlanIDs); err != nil {
		return nil, err
	}
	var existing model.LineSource
	if request.ID != "" {
		if err := s.db.First(&existing, "id = ? AND kind = ?", request.ID, lineSourceURL).Error; err != nil {
			return nil, errors.New("订阅来源不存在")
		}
		if request.URL == "" {
			plain, err := service.UnprotectCredential(existing.SecretCiphertext)
			if err != nil {
				return nil, err
			}
			request.URL = plain
		}
	}
	cleanURL, entries, err := s.fetchURLSource(ctx, request.URL)
	if err != nil {
		return nil, err
	}
	prepared, err := prepareLineEntries(entries)
	if err != nil {
		return nil, err
	}
	if len(prepared) == 0 {
		return nil, errors.New("订阅中没有可用的受支持节点")
	}
	secret, err := service.ProtectCredential(cleanURL)
	if err != nil {
		return nil, err
	}
	parsedURL, _ := url.Parse(cleanURL)
	now := time.Now().UTC()
	next := now.Add(time.Duration(request.RefreshInterval) * time.Second)
	source := model.LineSource{ID: request.ID, Name: request.Name, Kind: lineSourceURL, SecretCiphertext: secret, URLHost: parsedURL.Hostname(), RefreshInterval: request.RefreshInterval, Enabled: request.Enabled, Status: "ready", LastSuccessAt: &now, NextRefreshAt: &next, ConsecutiveSuccesses: existing.ConsecutiveSuccesses + 1}
	if source.ID == "" {
		source.ID = uuid.NewString()
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if request.ID == "" {
			if err := tx.Create(&source).Error; err != nil {
				return err
			}
			if err := tx.Model(&source).Update("enabled", request.Enabled).Error; err != nil {
				return err
			}
			source.Enabled = request.Enabled
		} else {
			if err := tx.Model(&model.LineSource{}).Where("id = ?", source.ID).Updates(map[string]any{"name": source.Name, "secret_ciphertext": source.SecretCiphertext, "url_host": source.URLHost, "refresh_interval": source.RefreshInterval, "enabled": source.Enabled, "status": source.Status, "last_error": "", "last_success_at": now, "next_refresh_at": next, "consecutive_successes": source.ConsecutiveSuccesses, "locked_at": nil}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("source_id = ?", source.ID).Delete(&model.LineSourceGroup{}).Error; err != nil {
			return err
		}
		for _, groupID := range uniqueStrings(request.GroupIDs) {
			if err := tx.Create(&model.LineSourceGroup{SourceID: source.ID, GroupID: groupID}).Error; err != nil {
				return err
			}
		}
		for _, planID := range uniqueStrings(request.PlanIDs) {
			for _, groupID := range uniqueStrings(request.GroupIDs) {
				row := model.PlanLineGroup{PlanID: planID, GroupID: groupID}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
					return err
				}
			}
		}
		return s.upsertPreparedLines(tx, source.ID, prepared, request.GroupIDs, now)
	}); err != nil {
		return nil, err
	}
	var missingNodeIDs []string
	if err := s.db.Model(&model.LineSourceNode{}).Where("source_id = ? AND missing_since IS NOT NULL", source.ID).Pluck("node_id", &missingNodeIDs).Error; err != nil {
		return nil, err
	}
	if err := s.markOrphanedNodesStale(missingNodeIDs); err != nil {
		return nil, err
	}
	if err := s.reconcilePlansForGroups(request.GroupIDs); err != nil {
		return nil, err
	}
	views, err := s.Sources()
	if err != nil {
		return nil, err
	}
	for i := range views {
		if views[i].ID == source.ID {
			return &views[i], nil
		}
	}
	return nil, errors.New("订阅来源保存失败")
}

func (s *LineService) QueueRefresh(sourceID string) error {
	now := time.Now().UTC()
	result := s.db.Model(&model.LineSource{}).Where("id = ? AND kind = ? AND enabled = ?", sourceID, lineSourceURL, true).Updates(map[string]any{"status": "pending", "next_refresh_at": now, "last_error": "", "locked_at": nil})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("启用的订阅来源不存在")
	}
	return nil
}

func (s *LineService) DeleteSource(sourceID string) error {
	var nodeIDs []string
	if err := s.db.Model(&model.LineSourceNode{}).Where("source_id = ?", sourceID).Pluck("node_id", &nodeIDs).Error; err != nil {
		return err
	}
	result := s.db.Delete(&model.LineSource{}, "id = ?", sourceID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("线路来源不存在")
	}
	return s.markOrphanedNodesStale(nodeIDs)
}

func (s *LineService) AssignNodeGroups(request entity.CommercialLineNodeGroupsRequest) error {
	nodeIDs := uniqueStrings(request.NodeIDs)
	groupIDs := uniqueStrings(request.GroupIDs)
	if len(nodeIDs) == 0 {
		return errors.New("请选择线路节点")
	}
	if len(groupIDs) > 0 {
		if err := s.validateGroupsAndPlans(groupIDs, nil); err != nil {
			return err
		}
	}
	var previousGroupIDs []string
	if err := s.db.Model(&model.LineGroupNode{}).Where("node_id IN ?", nodeIDs).Distinct("group_id").Pluck("group_id", &previousGroupIDs).Error; err != nil {
		return err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_id IN ?", nodeIDs).Delete(&model.LineGroupNode{}).Error; err != nil {
			return err
		}
		for _, nodeID := range nodeIDs {
			for _, groupID := range groupIDs {
				if err := tx.Create(&model.LineGroupNode{GroupID: groupID, NodeID: nodeID}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := s.normalizeAssignmentStates(nodeIDs); err != nil {
		return err
	}
	return s.reconcilePlansForGroups(append(previousGroupIDs, groupIDs...))
}

func (s *LineService) QueueProbe(nodeID string) error {
	result := s.db.Model(&model.LineNode{}).Where("id = ? AND missing_since IS NULL", nodeID).Updates(map[string]any{"status": lineStatusChecking, "health_status": lineHealthChecking, "provision_locked_at": nil, "last_error": ""})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("线路节点不存在或已失效")
	}
	return nil
}

func (s *LineService) validateGroupsAndPlans(groupIDs, planIDs []string) error {
	groupIDs = uniqueStrings(groupIDs)
	planIDs = uniqueStrings(planIDs)
	var groupCount int64
	if err := s.db.Model(&model.LineGroup{}).Where("id IN ? AND active = ?", groupIDs, true).Count(&groupCount).Error; err != nil {
		return err
	}
	if groupCount != int64(len(groupIDs)) {
		return errors.New("包含不存在或已停用的线路组")
	}
	if len(planIDs) == 0 {
		return nil
	}
	var planCount int64
	if err := s.db.Model(&model.Plan{}).Where("id IN ?", planIDs).Count(&planCount).Error; err != nil {
		return err
	}
	if planCount != int64(len(planIDs)) {
		return errors.New("包含不存在的套餐")
	}
	return nil
}

func (s *LineService) fetchURLSource(ctx context.Context, rawURL string) (string, []LineImportEntry, error) {
	cleanURL, err := service.SanitizePublicHTTPURL(rawURL, false)
	if err != nil {
		return "", nil, fmt.Errorf("订阅 URL 无效: %w", err)
	}
	if cleanURL == "" {
		return "", nil, errors.New("订阅 URL 无效")
	}
	client := service.NewPublicHTTPClient(30 * time.Second)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("订阅重定向次数过多")
		}
		_, err := service.SanitizePublicHTTPURL(req.URL.String(), false)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cleanURL, nil)
	if err != nil {
		return "", nil, err
	}
	// A number of subscription gateways choose their output format (or reject
	// the request entirely) based on a known client User-Agent. Request the
	// broadly supported URI-list representation; the parser also accepts Clash
	// YAML and JSON responses when a provider ignores this preference.
	req.Header.Set("User-Agent", lineSourceUserAgent)
	req.Header.Set("Accept", "text/plain, application/yaml, application/json, application/octet-stream, */*")
	// cleanURL is restricted to public HTTP(S) targets above. NewPublicHTTPClient
	// resolves and rejects private/link-local addresses again at connect time,
	// and CheckRedirect repeats URL validation for every redirect hop. This
	// second DNS check is what blocks rebinding between validation and dialing.
	resp, err := client.Do(req) // lgtm[go/request-forgery]
	if err != nil {
		return "", nil, errors.New("获取订阅失败，请检查地址、证书和网络")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, lineSourceMaxBytes+1))
	if err != nil {
		return "", nil, err
	}
	if int64(len(body)) > lineSourceMaxBytes {
		return "", nil, errors.New("订阅内容超过 8 MiB")
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, lineSourceHTTPError(resp.StatusCode, body)
	}
	if err := lineSourcePayloadError(body); err != nil {
		return "", nil, err
	}
	outbounds, identities, err := link.ParseSubscriptionBody(body)
	if err != nil {
		return "", nil, fmt.Errorf("无法解析订阅内容: %w", err)
	}
	if len(outbounds) > lineImportMaxEntries {
		return "", nil, fmt.Errorf("订阅节点超过 %d 条上限", lineImportMaxEntries)
	}
	entries := make([]LineImportEntry, 0, len(outbounds))
	for i := range outbounds {
		identity := ""
		if i < len(identities) {
			identity = identities[i]
		}
		entries = append(entries, buildLineImportEntry(i+1, outbounds[i], identity))
	}
	return cleanURL, entries, nil
}

func lineSourceHTTPError(status int, body []byte) error {
	message := lineSourceJSONMessage(body)
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "token") && (strings.Contains(lower, "error") || strings.Contains(lower, "invalid") || strings.Contains(lower, "expired")):
		return fmt.Errorf("获取订阅失败：订阅令牌无效或已失效（HTTP %d）", status)
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Errorf("获取订阅失败：上游订阅服务拒绝访问（HTTP %d）", status)
	case status == http.StatusTooManyRequests:
		return errors.New("获取订阅失败：上游请求过于频繁（HTTP 429），请稍后重试")
	default:
		return fmt.Errorf("获取订阅失败：上游返回 HTTP %d", status)
	}
}

func lineSourcePayloadError(body []byte) error {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	status := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["status"])))
	success, hasSuccess := payload["success"].(bool)
	if status != "fail" && status != "failed" && status != "error" && (!hasSuccess || success) {
		return nil
	}
	message := strings.ToLower(lineSourceJSONMessage(body))
	if strings.Contains(message, "token") && (strings.Contains(message, "error") || strings.Contains(message, "invalid") || strings.Contains(message, "expired")) {
		return errors.New("获取订阅失败：订阅令牌无效或已失效")
	}
	return errors.New("获取订阅失败：上游订阅服务返回错误")
}

func lineSourceJSONMessage(body []byte) string {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	for _, key := range []string{"message", "msg", "error"} {
		if value, ok := payload[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseLineEntries(raw string) ([]LineImportEntry, error) {
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	lines := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	if len(lines) > lineImportMaxEntries {
		return nil, fmt.Errorf("单次最多导入 %d 条协议链接", lineImportMaxEntries)
	}
	entries := make([]LineImportEntry, 0, len(lines))
	index := 0
	for _, rawLine := range lines {
		value := strings.TrimSpace(rawLine)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		index++
		parsed, err := link.ParseLink(value)
		if err != nil || parsed == nil {
			message := "无法解析协议链接"
			if err != nil {
				message = err.Error()
			}
			entries = append(entries, LineImportEntry{Index: index, Valid: false, Error: message})
			continue
		}
		entries = append(entries, buildLineImportEntry(index, parsed.Outbound, parsed.Identity))
	}
	if len(entries) == 0 {
		return nil, errors.New("没有找到协议链接")
	}
	return entries, nil
}

func buildLineImportEntry(index int, outbound link.Outbound, identity string) LineImportEntry {
	protocol, _ := outbound["protocol"].(string)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if _, supported := supportedLineProtocols[protocol]; !supported {
		return LineImportEntry{Index: index, Protocol: protocol, Valid: false, Error: "仅支持 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 和 AnyTLS"}
	}
	remark, _ := outbound["tag"].(string)
	identity = strings.TrimSpace(identity)
	if identity == "" {
		serialized, _ := json.Marshal(outbound)
		identity = string(serialized)
	}
	sum := sha256.Sum256([]byte(identity))
	fingerprint := hex.EncodeToString(sum[:])
	if strings.TrimSpace(remark) == "" {
		remark = strings.ToUpper(protocol) + "-" + fingerprint[:8]
	}
	outbound["tag"] = lineTagPrefix + fingerprint[:24]
	return LineImportEntry{Index: index, Remark: remark, Protocol: protocol, Fingerprint: fingerprint, Valid: true, outbound: outbound, identity: identity}
}

type preparedLineEntry struct {
	Fingerprint        string
	Remark             string
	Protocol           string
	OutboundTag        string
	OutboundCiphertext string
	TLSAutoPinned      bool
}

func prepareLineEntries(entries []LineImportEntry) ([]preparedLineEntry, error) {
	prepared := make([]preparedLineEntry, 0, len(entries))
	seen := map[string]bool{}
	for i := range entries {
		entry := entries[i]
		if !entry.Valid || seen[entry.Fingerprint] {
			continue
		}
		seen[entry.Fingerprint] = true
		serialized, err := json.Marshal(entry.outbound)
		if err != nil {
			return nil, err
		}
		protected, err := service.ProtectCredential(string(serialized))
		if err != nil {
			return nil, err
		}
		outboundTag, _ := entry.outbound["tag"].(string)
		prepared = append(prepared, preparedLineEntry{Fingerprint: entry.Fingerprint, Remark: entry.Remark, Protocol: entry.Protocol, OutboundTag: outboundTag, OutboundCiphertext: protected, TLSAutoPinned: outboundNeedsTLSAutoPin(entry.outbound)})
	}
	return prepared, nil
}

func (s *LineService) upsertPreparedLines(tx *gorm.DB, sourceID string, entries []preparedLineEntry, groupIDs []string, now time.Time) error {
	groupIDs = uniqueStrings(groupIDs)
	var source model.LineSource
	if err := tx.Select("id", "name", "kind").First(&source, "id = ?", sourceID).Error; err != nil {
		return err
	}
	seenNodeIDs := make([]string, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		publicName := defaultPublicLineName(source, entry, i)
		candidate := model.LineNode{ID: uuid.NewString(), Fingerprint: entry.Fingerprint, Remark: entry.Remark, PublicName: publicName, Protocol: entry.Protocol, OutboundTag: entry.OutboundTag, OutboundCiphertext: entry.OutboundCiphertext, TLSAutoPinned: entry.TLSAutoPinned, Status: lineStatusUnassigned, HealthStatus: lineHealthUnchecked, LastSeenAt: &now}
		if len(groupIDs) > 0 {
			candidate.Status = lineStatusChecking
			candidate.HealthStatus = lineHealthChecking
		}
		// The fingerprint already covers every connection parameter. Preserve an
		// existing encrypted outbound so a refresh cannot overwrite the exact
		// certificate pin learned during first provisioning with insecure=1.
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "fingerprint"}}, DoUpdates: clause.Assignments(map[string]any{"remark": entry.Remark, "protocol": entry.Protocol, "last_seen_at": now, "missing_since": nil})}).Create(&candidate).Error; err != nil {
			return err
		}
		var node model.LineNode
		if err := tx.Where("fingerprint = ?", entry.Fingerprint).First(&node).Error; err != nil {
			return err
		}
		if !node.PublicNameCustom {
			currentName := strings.TrimSpace(node.PublicName)
			if currentName != "" && currentName != publicName && !isLegacyGeneratedPublicName(source, entry, i, currentName) {
				// Rows created before PublicNameCustom existed have false here even
				// when an administrator edited the alias. Preserve those edits and
				// mark them explicitly on the first refresh after upgrading.
				if err := tx.Model(&node).Update("public_name_custom", true).Error; err != nil {
					return err
				}
				node.PublicNameCustom = true
			} else {
				if err := tx.Model(&node).Update("public_name", publicName).Error; err != nil {
					return err
				}
				node.PublicName = publicName
			}
		}
		if len(groupIDs) > 0 && (node.Status == lineStatusUnassigned || node.Status == lineStatusStale || node.InboundID == nil) {
			if err := tx.Model(&node).Updates(map[string]any{"status": lineStatusChecking, "health_status": lineHealthChecking, "provision_locked_at": nil, "next_provision_at": nil}).Error; err != nil {
				return err
			}
		}
		seenNodeIDs = append(seenNodeIDs, node.ID)
		membership := model.LineSourceNode{SourceID: sourceID, NodeID: node.ID, LastSeenAt: now, MissingSince: nil}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_id"}, {Name: "node_id"}}, DoUpdates: clause.Assignments(map[string]any{"last_seen_at": now, "missing_since": nil})}).Create(&membership).Error; err != nil {
			return err
		}
		for _, groupID := range groupIDs {
			row := model.LineGroupNode{GroupID: groupID, NodeID: node.ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	missingQuery := tx.Model(&model.LineSourceNode{}).Where("source_id = ? AND missing_since IS NULL", sourceID)
	if len(seenNodeIDs) > 0 {
		missingQuery = missingQuery.Where("node_id NOT IN ?", seenNodeIDs)
	}
	return missingQuery.Update("missing_since", now).Error
}

func outboundNeedsTLSAutoPin(outbound map[string]any) bool {
	stream, _ := outbound["streamSettings"].(map[string]any)
	tlsSettings, _ := stream["tlsSettings"].(map[string]any)
	value, _ := tlsSettings["allowInsecure"].(bool)
	return value
}

func defaultPublicLineName(_ model.LineSource, entry preparedLineEntry, index int) string {
	protocol := strings.ToUpper(strings.TrimSpace(entry.Protocol))
	if protocol == "HYSTERIA" {
		protocol = "Hysteria2"
	}
	// Both the imported fragment and the source name may contain an upstream
	// provider's brand. Keep them private to the administrator and publish a
	// neutral alias by default. The node-pool alias editor is the only place
	// that controls the customer-visible name.
	return truncateRunes(fmt.Sprintf("%s 线路 #%d", protocol, index+1), 160)
}

func isLegacyGeneratedPublicName(source model.LineSource, entry preparedLineEntry, index int, current string) bool {
	protocol := strings.ToUpper(strings.TrimSpace(entry.Protocol))
	if protocol == "HYSTERIA" {
		protocol = "Hysteria2"
	}
	name := strings.TrimSpace(source.Name)
	if name == "" {
		name = "订阅线路"
	}
	legacy := []string{
		fmt.Sprintf("%s · %s #%d", name, protocol, index+1),
		fmt.Sprintf("%s 路 %s #%d", name, protocol, index+1),
	}
	for _, candidate := range legacy {
		if current == candidate {
			return true
		}
	}
	return false
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func (s *LineService) normalizeAssignmentStates(nodeIDs []string) error {
	nodeIDs = uniqueStrings(nodeIDs)
	if len(nodeIDs) == 0 {
		return nil
	}
	restart := false
	for _, nodeID := range nodeIDs {
		var count int64
		if err := s.db.Model(&model.LineGroupNode{}).Where("node_id = ?", nodeID).Count(&count).Error; err != nil {
			return err
		}
		var node model.LineNode
		if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
			return err
		}
		if count == 0 {
			if node.InboundID != nil {
				if err := s.db.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Update("enable", false).Error; err != nil {
					return err
				}
				restart = true
			}
			if err := s.db.Model(&node).Updates(map[string]any{"status": lineStatusUnassigned, "health_status": lineHealthUnchecked, "provision_locked_at": nil}).Error; err != nil {
				return err
			}
			continue
		}
		if node.Status == lineStatusUnassigned || node.Status == lineStatusStale || node.InboundID == nil {
			if err := s.db.Model(&node).Updates(map[string]any{"status": lineStatusChecking, "health_status": lineHealthChecking, "missing_since": nil, "provision_locked_at": nil}).Error; err != nil {
				return err
			}
		}
	}
	if restart {
		return s.xray.RestartXray(false)
	}
	return nil
}

func (s *LineService) markOrphanedNodesStale(nodeIDs []string) error {
	now := time.Now().UTC()
	restart := false
	for _, nodeID := range uniqueStrings(nodeIDs) {
		var sources int64
		if err := s.db.Model(&model.LineSourceNode{}).Where("node_id = ? AND missing_since IS NULL", nodeID).Count(&sources).Error; err != nil {
			return err
		}
		if sources > 0 {
			continue
		}
		var node model.LineNode
		if err := s.db.First(&node, "id = ?", nodeID).Error; err != nil {
			continue
		}
		if node.InboundID != nil {
			if err := s.db.Model(&model.Inbound{}).Where("id = ?", *node.InboundID).Update("enable", false).Error; err != nil {
				return err
			}
			restart = true
		}
		if err := s.db.Model(&node).Updates(map[string]any{"status": lineStatusStale, "health_status": lineHealthOffline, "missing_since": now, "provision_locked_at": nil}).Error; err != nil {
			return err
		}
	}
	if restart {
		return s.xray.RestartXray(false)
	}
	return nil
}

func (s *LineService) reconcilePlansForGroups(groupIDs []string) error {
	if len(groupIDs) == 0 {
		return nil
	}
	var planIDs []string
	if err := s.db.Model(&model.PlanLineGroup{}).Where("group_id IN ?", uniqueStrings(groupIDs)).Distinct("plan_id").Pluck("plan_id", &planIDs).Error; err != nil {
		return err
	}
	return s.reconcilePlans(planIDs)
}

func (s *LineService) reconcilePlans(planIDs []string) error {
	worker := NewWorker()
	for _, planID := range uniqueStrings(planIDs) {
		if err := worker.reconcileActivePlanClients(planID); err != nil {
			return err
		}
	}
	return nil
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func lineDuplicateError(err error, message string) error {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return errors.New(message)
	}
	return err
}
