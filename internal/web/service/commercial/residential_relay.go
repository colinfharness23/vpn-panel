package commercial

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

const (
	residentialRelayStatusActive  = "active"
	residentialRelayStatusPending = "pending"
)

var (
	relayHostnamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
	relayLookupIP        = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return net.DefaultResolver.LookupIPAddr(ctx, host)
	}
)

type ResidentialRelayLine struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Protocol string         `json:"protocol"`
	Inbound  *model.Inbound `json:"-"`
}

type ResidentialRelayView struct {
	ID          string         `json:"id"`
	InboundID   int            `json:"inboundId"`
	LineName    string         `json:"lineName"`
	Protocol    string         `json:"protocol"`
	Name        string         `json:"name"`
	Host        string         `json:"host"`
	Port        int            `json:"port"`
	Username    string         `json:"username,omitempty"`
	HasPassword bool           `json:"hasPassword"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	Links       []string       `json:"links,omitempty"`
	Inbound     *model.Inbound `json:"-"`
	ClientID    string         `json:"-"`
}

type ResidentialRelayOverview struct {
	Enabled bool                   `json:"enabled"`
	Limit   int                    `json:"limit"`
	Lines   []ResidentialRelayLine `json:"lines"`
	Relays  []ResidentialRelayView `json:"relays"`
}

type ResidentialRelayService struct {
	db   *gorm.DB
	xray service.XrayService
}

func NewResidentialRelayService() *ResidentialRelayService {
	return &ResidentialRelayService{db: database.GetDB()}
}

func (s *ResidentialRelayService) Overview(customerID string) (*ResidentialRelayOverview, error) {
	entitlement, err := s.activeEntitlement(customerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ResidentialRelayOverview{Lines: []ResidentialRelayLine{}, Relays: []ResidentialRelayView{}}, nil
		}
		return nil, err
	}
	overview := &ResidentialRelayOverview{
		Enabled: entitlement.ResidentialRelayEnabled && entitlement.ResidentialRelayLimit > 0,
		Limit:   entitlement.ResidentialRelayLimit,
		Lines:   []ResidentialRelayLine{},
		Relays:  []ResidentialRelayView{},
	}
	if !overview.Enabled {
		return overview, nil
	}

	lines, err := s.availableLines(entitlement)
	if err != nil {
		return nil, err
	}
	overview.Lines = lines

	var rows []model.ResidentialRelay
	if err := s.db.Where("customer_id = ? AND entitlement_id = ?", customerID, entitlement.ID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	lineByID := make(map[int]ResidentialRelayLine, len(lines))
	for _, line := range lines {
		lineByID[line.ID] = line
	}
	for i := range rows {
		line, ok := lineByID[rows[i].InboundID]
		if !ok {
			continue
		}
		view, err := s.toView(&rows[i], entitlement, line)
		if err != nil {
			return nil, err
		}
		overview.Relays = append(overview.Relays, view)
	}
	return overview, nil
}

func (s *ResidentialRelayService) Create(ctx context.Context, customerID string, request entity.ResidentialRelayRequest) (*ResidentialRelayOverview, error) {
	entitlement, line, err := s.validateRequest(ctx, customerID, false, request)
	if err != nil {
		return nil, err
	}
	// Treat a repeated POST for the same entitlement and line as the same
	// operation. A runtime reconciliation can take several seconds; without
	// this check a second click observes the first committed row and reports a
	// misleading plan-limit error even though the relay was created.
	var existing model.ResidentialRelay
	if err := s.db.Where("entitlement_id = ? AND inbound_id = ?", entitlement.ID, line.ID).First(&existing).Error; err == nil {
		return s.Overview(customerID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&model.ResidentialRelay{}).Where("customer_id = ? AND entitlement_id = ?", customerID, entitlement.ID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count >= int64(entitlement.ResidentialRelayLimit) {
		return nil, fmt.Errorf("当前套餐最多可创建 %d 条住宅中转", entitlement.ResidentialRelayLimit)
	}
	username, password, err := protectRelayCredentials(request.Username, request.Password)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = line.Name + "住宅中转"
	}
	id := uuid.NewString()
	relay := &model.ResidentialRelay{
		ID:                 id,
		CustomerID:         customerID,
		EntitlementID:      entitlement.ID,
		InboundID:          line.ID,
		Name:               name,
		OutboundTag:        "residential-relay-" + strings.ReplaceAll(id, "-", "")[:20],
		SOCKSHost:          strings.TrimSpace(request.Host),
		SOCKSPort:          request.Port,
		UsernameCiphertext: username,
		PasswordCiphertext: password,
		Status:             residentialRelayStatusPending,
	}
	if err := s.db.Create(relay).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return s.Overview(customerID)
		}
		return nil, err
	}
	if err := s.applyRuntime(relay.ID); err != nil {
		return nil, err
	}
	return s.Overview(customerID)
}

func (s *ResidentialRelayService) Update(ctx context.Context, customerID, relayID string, request entity.ResidentialRelayRequest) (*ResidentialRelayOverview, error) {
	entitlement, line, err := s.validateRequest(ctx, customerID, true, request)
	if err != nil {
		return nil, err
	}
	var relay model.ResidentialRelay
	if err := s.db.Where("id = ? AND customer_id = ? AND entitlement_id = ?", relayID, customerID, entitlement.ID).First(&relay).Error; err != nil {
		return nil, errors.New("住宅中转配置不存在")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = line.Name + "住宅中转"
	}
	updates := map[string]any{
		"inbound_id": request.InboundID,
		"name":       name,
		"socks_host": strings.TrimSpace(request.Host),
		"socks_port": request.Port,
		"status":     residentialRelayStatusPending,
		"last_error": "",
	}

	existingUsername, err := service.UnprotectCredential(relay.UsernameCiphertext)
	if err != nil {
		return nil, errors.New("现有 SOCKS5 凭据无法解密，请删除后重新创建")
	}
	requestedUsername := strings.TrimSpace(request.Username)
	if request.Password != "" {
		if requestedUsername == "" {
			return nil, errors.New("SOCKS5 用户名和密码需要同时填写")
		}
		username, password, err := protectRelayCredentials(request.Username, request.Password)
		if err != nil {
			return nil, err
		}
		updates["username_ciphertext"] = username
		updates["password_ciphertext"] = password
	} else if requestedUsername != "" && requestedUsername != existingUsername {
		return nil, errors.New("修改 SOCKS5 用户名时请同时填写密码")
	}

	if err := s.db.Model(&relay).Updates(updates).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("这条线路已经配置了住宅中转，请直接编辑现有配置")
		}
		return nil, err
	}
	if err := s.applyRuntime(relay.ID); err != nil {
		return nil, err
	}
	return s.Overview(customerID)
}

func (s *ResidentialRelayService) Delete(customerID, relayID string) (*ResidentialRelayOverview, error) {
	var relay model.ResidentialRelay
	if err := s.db.Where("id = ? AND customer_id = ?", relayID, customerID).First(&relay).Error; err != nil {
		return nil, errors.New("住宅中转配置不存在")
	}
	if err := s.db.Delete(&relay).Error; err != nil {
		return nil, err
	}
	if s.xray.IsXrayRunning() {
		_ = s.xray.RestartXray(false)
	} else {
		s.xray.SetToNeedRestart()
	}
	return s.Overview(customerID)
}

func (s *ResidentialRelayService) Relay(customerID, relayID string) (*ResidentialRelayView, error) {
	overview, err := s.Overview(customerID)
	if err != nil {
		return nil, err
	}
	for i := range overview.Relays {
		if overview.Relays[i].ID == relayID {
			return &overview.Relays[i], nil
		}
	}
	return nil, errors.New("住宅中转配置不存在")
}

func (s *ResidentialRelayService) activeEntitlement(customerID string) (*model.SubscriptionEntitlement, error) {
	var entitlement model.SubscriptionEntitlement
	query := s.db.Where("customer_id = ? AND status = ?", customerID, "active")
	query = query.Where("expires_at IS NULL OR expires_at > ?", time.Now().UTC())
	err := query.Order("created_at desc").First(&entitlement).Error
	return &entitlement, err
}

func (s *ResidentialRelayService) availableLines(entitlement *model.SubscriptionEntitlement) ([]ResidentialRelayLine, error) {
	var client model.ClientRecord
	if err := s.db.Where("email = ?", entitlement.InternalClientID).First(&client).Error; err != nil {
		return nil, err
	}
	var inboundIDs []int
	if err := s.db.Model(&model.ClientInbound{}).Where("client_id = ?", client.Id).Pluck("inbound_id", &inboundIDs).Error; err != nil {
		return nil, err
	}
	if len(inboundIDs) == 0 {
		return []ResidentialRelayLine{}, nil
	}
	var inbounds []*model.Inbound
	if err := s.db.Where("id IN ? AND enable = ? AND node_id IS NULL", inboundIDs, true).Order("sub_sort_index asc, id asc").Find(&inbounds).Error; err != nil {
		return nil, err
	}
	lines := make([]ResidentialRelayLine, 0, len(inbounds))
	for _, inbound := range inbounds {
		if inbound.Tag == "" || !relaySupportedProtocol(inbound.Protocol) {
			continue
		}
		name := strings.TrimSpace(inbound.Remark)
		if name == "" {
			name = fmt.Sprintf("线路 %d", inbound.Id)
		}
		lines = append(lines, ResidentialRelayLine{ID: inbound.Id, Name: name, Protocol: string(inbound.Protocol), Inbound: inbound})
	}
	return lines, nil
}

func (s *ResidentialRelayService) validateRequest(ctx context.Context, customerID string, updating bool, request entity.ResidentialRelayRequest) (*model.SubscriptionEntitlement, ResidentialRelayLine, error) {
	entitlement, err := s.activeEntitlement(customerID)
	if err != nil {
		return nil, ResidentialRelayLine{}, errors.New("当前没有可用订阅")
	}
	if !entitlement.ResidentialRelayEnabled || entitlement.ResidentialRelayLimit <= 0 {
		return nil, ResidentialRelayLine{}, errors.New("当前套餐未包含住宅中转功能")
	}
	if err := validateRelayEndpoint(ctx, request.Host, request.Port); err != nil {
		return nil, ResidentialRelayLine{}, err
	}
	username := strings.TrimSpace(request.Username)
	if !updating && (username == "") != (request.Password == "") {
		return nil, ResidentialRelayLine{}, errors.New("SOCKS5 用户名和密码需要同时填写")
	}
	lines, err := s.availableLines(entitlement)
	if err != nil {
		return nil, ResidentialRelayLine{}, err
	}
	for _, line := range lines {
		if line.ID == request.InboundID {
			return entitlement, line, nil
		}
	}
	return nil, ResidentialRelayLine{}, errors.New("所选线路不属于当前套餐，或暂不支持住宅中转")
}

func (s *ResidentialRelayService) toView(relay *model.ResidentialRelay, entitlement *model.SubscriptionEntitlement, line ResidentialRelayLine) (ResidentialRelayView, error) {
	username, err := service.UnprotectCredential(relay.UsernameCiphertext)
	if err != nil {
		return ResidentialRelayView{}, errors.New("住宅中转凭据无法解密，请删除后重新创建")
	}
	return ResidentialRelayView{
		ID: relay.ID, InboundID: relay.InboundID, LineName: line.Name, Protocol: line.Protocol,
		Name: relay.Name, Host: relay.SOCKSHost, Port: relay.SOCKSPort, Username: username,
		HasPassword: relay.PasswordCiphertext != "", Status: relay.Status, CreatedAt: relay.CreatedAt,
		Inbound: line.Inbound, ClientID: entitlement.InternalClientID,
	}, nil
}

func (s *ResidentialRelayService) applyRuntime(relayID string) error {
	if !s.xray.IsXrayRunning() {
		s.xray.SetToNeedRestart()
		return errors.New("代理运行时尚未启动，住宅中转已保存但暂未生效")
	}
	status := residentialRelayStatusActive
	lastError := ""
	runtimeErr := s.xray.RestartXray(false)
	if runtimeErr != nil {
		status = residentialRelayStatusPending
		lastError = runtimeErr.Error()
		if len(lastError) > 500 {
			lastError = lastError[:500]
		}
	}
	if err := s.db.Model(&model.ResidentialRelay{}).Where("id = ?", relayID).Updates(map[string]any{"status": status, "last_error": lastError}).Error; err != nil {
		return err
	}
	if runtimeErr != nil {
		return fmt.Errorf("住宅中转已保存，但应用到代理运行时失败: %w", runtimeErr)
	}
	return nil
}

func protectRelayCredentials(username, password string) (string, string, error) {
	username = strings.TrimSpace(username)
	if (username == "") != (password == "") {
		return "", "", errors.New("SOCKS5 用户名和密码需要同时填写")
	}
	protectedUsername, err := service.ProtectCredential(username)
	if err != nil {
		return "", "", err
	}
	protectedPassword, err := service.ProtectCredential(password)
	if err != nil {
		return "", "", err
	}
	return protectedUsername, protectedPassword, nil
}

func validateRelayEndpoint(parent context.Context, rawHost string, port int) error {
	host := strings.Trim(strings.TrimSpace(rawHost), "[]")
	if host == "" || len(host) > 253 || port < 1 || port > 65535 {
		return errors.New("请填写有效的 SOCKS5 地址和端口")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !safeRelayIP(ip) {
			return errors.New("住宅代理不能指向本机、局域网或保留地址")
		}
		return nil
	}
	lower := strings.ToLower(host)
	if !relayHostnamePattern.MatchString(host) || strings.Contains(host, "..") || strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".localhost") || !strings.Contains(host, ".") {
		return errors.New("请填写可公开解析的 SOCKS5 主机名")
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	addresses, err := relayLookupIP(ctx, host)
	if err != nil || len(addresses) == 0 {
		return errors.New("SOCKS5 主机名暂时无法解析")
	}
	for _, address := range addresses {
		if !safeRelayIP(address.IP) {
			return errors.New("住宅代理不能解析到本机、局域网或保留地址")
		}
	}
	return nil
}

func safeRelayIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64 {
		return false
	}
	return true
}

func relaySupportedProtocol(protocol model.Protocol) bool {
	switch protocol {
	case model.VLESS, model.VMESS, model.Trojan, model.Shadowsocks, model.Hysteria, model.AnyTLS:
		return true
	default:
		return false
	}
}
