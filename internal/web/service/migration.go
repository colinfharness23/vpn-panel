package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"golang.org/x/crypto/ssh"
)

const (
	migrationMinDiskKB      = 2 * 1024 * 1024
	migrationDialTimeout    = 12 * time.Second
	migrationCommandTimeout = 20 * time.Minute
)

var migrationUsernamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,63}$`)

type ServerMigrationRequest struct {
	Host        string `json:"host" form:"host"`
	Port        int    `json:"port" form:"port"`
	Username    string `json:"username" form:"username"`
	Password    string `json:"password" form:"password"`
	Fingerprint string `json:"fingerprint" form:"fingerprint"`
	Confirmed   bool   `json:"confirmed" form:"confirmed"`
}

type MigrationCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type MigrationPreflight struct {
	Ready           bool             `json:"ready"`
	TargetIP        string           `json:"targetIp"`
	TargetOS        string           `json:"targetOs"`
	TargetArch      string           `json:"targetArch"`
	TargetDiskFree  uint64           `json:"targetDiskFree"`
	Fingerprint     string           `json:"fingerprint"`
	Domain          string           `json:"domain"`
	DNSAddresses    []string         `json:"dnsAddresses"`
	DNSReady        bool             `json:"dnsReady"`
	ExistingInstall bool             `json:"existingInstall"`
	Checks          []MigrationCheck `json:"checks"`
}

type MigrationJobSnapshot struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Progress  int       `json:"progress"`
	Step      string    `json:"step"`
	Logs      []string  `json:"logs"`
	PortalURL string    `json:"portalUrl,omitempty"`
	AdminURL  string    `json:"adminUrl,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt,omitempty"`
}

type migrationJob struct {
	MigrationJobSnapshot
}

type MigrationManager struct {
	mu       sync.RWMutex
	jobs     map[string]*migrationJob
	activeID string
	server   *ServerService
}

type migrationDeployConfig struct {
	Repository             string
	ReleaseTag             string
	Domain                 string
	ACMEEmail              string
	AdminPath              string
	AdminUsername          string
	BootstrapAdminPassword string
	PanelPort              string
	WebBasePath            string
	DBName                 string
	DBUser                 string
}

type migrationRemoteInfo struct {
	OSID            string
	OSVersion       string
	Arch            string
	MachineID       string
	DiskFreeKB      uint64
	ExistingInstall bool
}

func NewMigrationManager(server *ServerService) *MigrationManager {
	return &MigrationManager{jobs: make(map[string]*migrationJob), server: server}
}

func (m *MigrationManager) SourceInfo() map[string]any {
	deploy, err := loadMigrationDeployConfig()
	return map[string]any{
		"supported":  runtime.GOOS == "linux" && err == nil,
		"platform":   runtime.GOOS,
		"database":   config.GetDBKind(),
		"domain":     deploy.Domain,
		"configured": err == nil,
	}
}

func (m *MigrationManager) Preflight(ctx context.Context, req ServerMigrationRequest) (*MigrationPreflight, error) {
	return m.preflight(ctx, req, "")
}

func (m *MigrationManager) preflight(ctx context.Context, req ServerMigrationRequest, expectedFingerprint string) (*MigrationPreflight, error) {
	if runtime.GOOS != "linux" {
		return nil, errors.New("一键迁移只能在已部署的 Ubuntu 生产服务器上执行")
	}
	targetIP, err := validateMigrationRequest(req)
	if err != nil {
		return nil, err
	}
	deploy, err := loadMigrationDeployConfig()
	if err != nil {
		return nil, err
	}
	if _, err := migrationInstallScriptPath(); err != nil {
		return nil, err
	}

	client, fingerprint, err := dialMigrationSSH(ctx, req, expectedFingerprint)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败：%w", err)
	}
	defer client.Close()

	probe := `set -eu
. /etc/os-release
printf 'OS_ID=%s\n' "${ID:-}"
printf 'OS_VERSION=%s\n' "${VERSION_ID:-}"
printf 'ARCH=%s\n' "$(uname -m)"
printf 'MACHINE_ID=%s\n' "$(cat /etc/machine-id 2>/dev/null || true)"
printf 'DISK_FREE_KB=%s\n' "$(df -Pk / | awk 'NR==2 {print $4}')"
if [ -e /usr/local/x-ui ] || [ -e /etc/nova/deploy.env ]; then printf 'EXISTING_INSTALL=1\n'; else printf 'EXISTING_INSTALL=0\n'; fi`
	out, err := runMigrationRootCommand(ctx, client, req, probe, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("目标服务器权限或环境检测失败：%w", err)
	}
	remote, err := parseMigrationRemoteInfo(out)
	if err != nil {
		return nil, err
	}

	localMachineID := strings.TrimSpace(readOptionalFile("/etc/machine-id"))
	dnsReady, dnsAddresses := migrationDNSReady(ctx, deploy.Domain, targetIP)
	checks := []MigrationCheck{
		{Key: "ssh", Label: "SSH 连接", Status: "success", Detail: "账号密码验证成功，主机指纹已锁定"},
		migrationBoolCheck("ubuntu", "Ubuntu 系统", remote.OSID == "ubuntu" && (strings.HasPrefix(remote.OSVersion, "22.04") || strings.HasPrefix(remote.OSVersion, "24.04")), "支持 Ubuntu 22.04/24.04", fmt.Sprintf("检测到 %s %s", remote.OSID, remote.OSVersion)),
		migrationBoolCheck("arch", "服务器架构", remote.Arch == "x86_64" || remote.Arch == "amd64" || remote.Arch == "aarch64" || remote.Arch == "arm64", "支持 amd64/arm64", "不支持的架构："+remote.Arch),
		migrationBoolCheck("disk", "可用磁盘空间", remote.DiskFreeKB >= migrationMinDiskKB, formatMigrationBytes(remote.DiskFreeKB*1024)+" 可用", "至少需要 2 GB 可用空间"),
		migrationBoolCheck("machine", "目标服务器身份", localMachineID == "" || remote.MachineID == "" || localMachineID != remote.MachineID, "已确认不是当前服务器", "目标地址指向当前服务器，已阻止迁移"),
	}
	if deploy.Domain == "" {
		checks = append(checks, MigrationCheck{Key: "dns", Label: "域名解析", Status: "warning", Detail: "当前未配置域名，迁移后将使用目标 IP 访问"})
		dnsReady = true
	} else {
		checks = append(checks, migrationBoolCheck("dns", "域名解析", dnsReady, "域名已解析到目标服务器", "请先把域名解析到目标 IP，然后重新检测"))
	}
	if remote.ExistingInstall {
		checks = append(checks, MigrationCheck{Key: "existing", Label: "目标端现有数据", Status: "warning", Detail: "检测到现有安装，执行时会先备份并由本机数据完整覆盖"})
	} else {
		checks = append(checks, MigrationCheck{Key: "existing", Label: "目标端现有数据", Status: "success", Detail: "目标服务器为干净环境"})
	}

	ready := true
	for _, check := range checks {
		if check.Status == "error" {
			ready = false
			break
		}
	}
	return &MigrationPreflight{
		Ready: ready, TargetIP: targetIP.String(), TargetOS: strings.TrimSpace(remote.OSID + " " + remote.OSVersion),
		TargetArch: remote.Arch, TargetDiskFree: remote.DiskFreeKB * 1024, Fingerprint: fingerprint,
		Domain: deploy.Domain, DNSAddresses: dnsAddresses, DNSReady: dnsReady,
		ExistingInstall: remote.ExistingInstall, Checks: checks,
	}, nil
}

func (m *MigrationManager) Start(ctx context.Context, req ServerMigrationRequest) (*MigrationJobSnapshot, error) {
	if !req.Confirmed {
		return nil, errors.New("请先确认已了解迁移会覆盖目标服务器上的同名服务和数据库")
	}
	if strings.TrimSpace(req.Fingerprint) == "" {
		return nil, errors.New("请先完成连接检测并锁定目标服务器指纹")
	}
	preflight, err := m.preflight(ctx, req, req.Fingerprint)
	if err != nil {
		return nil, err
	}
	if !preflight.Ready {
		return nil, errors.New("迁移前检查未通过")
	}

	m.mu.Lock()
	if m.activeID != "" {
		if active := m.jobs[m.activeID]; active != nil && active.Status == "running" {
			m.mu.Unlock()
			return nil, errors.New("已有迁移任务正在执行，请勿重复提交")
		}
	}
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	job := &migrationJob{MigrationJobSnapshot: MigrationJobSnapshot{
		ID: id, Status: "running", Progress: 2, Step: "准备迁移", StartedAt: time.Now().UTC(),
		Logs: []string{"迁移任务已创建；SSH 密码仅保存在本次任务内存中，不会写入数据库或日志。"},
	}}
	m.jobs[id] = job
	m.activeID = id
	snapshot := cloneMigrationSnapshot(job)
	m.mu.Unlock()

	go m.run(id, req)
	return snapshot, nil
}

func (m *MigrationManager) Status(id string) (*MigrationJobSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job := m.jobs[id]
	if job == nil {
		return nil, errors.New("迁移任务不存在或服务已重启")
	}
	return cloneMigrationSnapshot(job), nil
}

func (m *MigrationManager) run(id string, req ServerMigrationRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()
	defer func() { req.Password = "" }()
	fail := func(err error) {
		logger.Warningf("server migration %s failed: %v", id, err)
		m.update(id, -1, "迁移失败", "", func(job *migrationJob) {
			job.Status = "failed"
			job.Error = err.Error()
			job.EndedAt = time.Now().UTC()
		})
	}

	deploy, err := loadMigrationDeployConfig()
	if err != nil {
		fail(err)
		return
	}
	deploy.BootstrapAdminPassword, err = generateMigrationBootstrapPassword()
	if err != nil {
		fail(fmt.Errorf("生成目标端临时管理员凭据失败：%w", err))
		return
	}
	installPath, err := migrationInstallScriptPath()
	if err != nil {
		fail(err)
		return
	}
	installScript, err := os.ReadFile(installPath)
	if err != nil {
		fail(fmt.Errorf("读取迁移安装程序失败：%w", err))
		return
	}

	m.update(id, 8, "生成数据库快照", "正在为迁移生成一致性数据库快照。", nil)
	backup, err := m.server.GetDb()
	if err != nil {
		fail(fmt.Errorf("生成数据库快照失败：%w", err))
		return
	}
	backupName := "source.db"
	if database.IsPostgres() {
		backupName = "source.dump"
	}

	m.update(id, 16, "连接目标服务器", "正在使用已锁定的主机指纹重新连接。", nil)
	client, _, err := dialMigrationSSH(ctx, req, req.Fingerprint)
	if err != nil {
		fail(fmt.Errorf("重新连接目标服务器失败：%w", err))
		return
	}
	defer client.Close()
	m.update(id, 20, "备份目标服务器", "如目标端已存在本项目，会先备份原有 PostgreSQL 数据库。", nil)
	targetBackupCommand := "if [ -f /etc/nova/deploy.env ]; then\n" +
		"  if [ -x /usr/local/sbin/nova-backup ]; then\n" +
		"    /usr/local/sbin/nova-backup\n" +
		"  else\n" +
		"    . /etc/nova/deploy.env\n" +
		"    install -d -m 700 /var/backups/nova/database\n" +
		"    target=\"/var/backups/nova/database/$NOVA_DB_NAME-before-migration-$(date -u +%Y%m%dT%H%M%SZ).dump\"\n" +
		"    PGPASSWORD=\"$NOVA_DB_PASSWORD\" pg_dump --host 127.0.0.1 --username \"$NOVA_DB_USER\" --dbname \"$NOVA_DB_NAME\" --format custom --compress 9 --file \"$target\"\n" +
		"    chmod 600 \"$target\"\n" +
		"  fi\n" +
		"fi"
	if _, err := runMigrationRootCommand(ctx, client, req, targetBackupCommand, migrationCommandTimeout); err != nil {
		fail(fmt.Errorf("目标服务器原有数据库备份失败，已停止迁移：%w", err))
		return
	}
	tmpBase := "/tmp/nova-migration-" + id
	installRemote := tmpBase + "-install.sh"
	backupRemote := tmpBase + "-" + backupName
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		_, _ = runMigrationRootCommand(cleanupCtx, client, req, "rm -f "+shellQuote(installRemote)+" "+shellQuote(backupRemote), 15*time.Second)
	}()

	m.update(id, 28, "上传安装程序", "正在通过加密 SSH 通道传输目标端安装程序。", nil)
	if err := uploadMigrationFile(ctx, client, installRemote, installScript); err != nil {
		fail(err)
		return
	}
	m.update(id, 38, "上传数据库快照", fmt.Sprintf("正在传输 %s 数据库快照。", formatMigrationBytes(uint64(len(backup)))), nil)
	if err := uploadMigrationFile(ctx, client, backupRemote, backup); err != nil {
		fail(err)
		return
	}
	backup = nil

	m.update(id, 48, "安装目标服务", "正在安装运行依赖、应用程序、PostgreSQL、Nginx 与 TLS。", nil)
	installCommand := migrationInstallCommand(deploy, installRemote)
	if _, err := runMigrationRootCommand(ctx, client, req, installCommand, migrationCommandTimeout); err != nil {
		fail(fmt.Errorf("目标服务安装失败：%w", err))
		return
	}

	m.update(id, 72, "恢复业务数据", "正在停止目标端服务并以事务方式恢复账户、套餐、订单、订阅和节点数据。", nil)
	restoreCommand := migrationRestoreCommand(backupRemote, database.IsPostgres())
	if _, err := runMigrationRootCommand(ctx, client, req, restoreCommand, migrationCommandTimeout); err != nil {
		fail(fmt.Errorf("目标端数据库恢复失败：%w", err))
		return
	}

	m.update(id, 90, "验证目标服务", "正在检查目标端前台、后台和服务进程。", nil)
	healthCommand := `. /etc/nova/deploy.env
systemctl is-active --quiet x-ui
curl -fsS --max-time 8 "http://127.0.0.1:${NOVA_PANEL_PORT}/portal/" >/dev/null
curl -fsS --max-time 8 "http://127.0.0.1:${NOVA_PANEL_PORT}/${NOVA_ADMIN_PATH}/" >/dev/null
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 8 "http://127.0.0.1:${NOVA_PANEL_PORT}/panel/")"
[ "$status" = 404 ]
pgrep -u nova -f '/usr/local/x-ui/bin/xray-linux-' >/dev/null`
	if _, err := runMigrationRootCommand(ctx, client, req, healthCommand, 60*time.Second); err != nil {
		fail(fmt.Errorf("目标服务健康检查失败：%w", err))
		return
	}

	portalURL, adminURL := migrationResultURLs(deploy, req.Host)
	m.update(id, 100, "迁移完成", "目标服务器健康检查已通过；旧服务器仍保持运行，确认新站正常后再手动停用旧服务器。", func(job *migrationJob) {
		job.Status = "completed"
		job.PortalURL = portalURL
		job.AdminURL = adminURL
		job.EndedAt = time.Now().UTC()
	})
}

func (m *MigrationManager) update(id string, progress int, step, logLine string, mutate func(*migrationJob)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[id]
	if job == nil {
		return
	}
	if progress >= 0 {
		job.Progress = progress
	}
	job.Step = step
	if logLine != "" {
		job.Logs = append(job.Logs, logLine)
		if len(job.Logs) > 40 {
			job.Logs = append([]string(nil), job.Logs[len(job.Logs)-40:]...)
		}
	}
	if mutate != nil {
		mutate(job)
	}
	if job.Status != "running" && m.activeID == id {
		m.activeID = ""
	}
}

func cloneMigrationSnapshot(job *migrationJob) *MigrationJobSnapshot {
	if job == nil {
		return nil
	}
	copy := job.MigrationJobSnapshot
	copy.Logs = append([]string(nil), job.Logs...)
	return &copy
}

func validateMigrationRequest(req ServerMigrationRequest) (net.IP, error) {
	ip := net.ParseIP(strings.TrimSpace(req.Host))
	if ip == nil {
		return nil, errors.New("目标服务器必须填写有效的 IPv4 或 IPv6 地址")
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
		return nil, errors.New("目标服务器 IP 不能是本机、未指定或组播地址")
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Port < 1 || req.Port > 65535 {
		return nil, errors.New("SSH 端口必须在 1-65535 之间")
	}
	if !migrationUsernamePattern.MatchString(strings.TrimSpace(req.Username)) {
		return nil, errors.New("SSH 用户名格式无效")
	}
	if len(req.Password) < 1 || len(req.Password) > 512 {
		return nil, errors.New("SSH 密码不能为空且不能超过 512 个字符")
	}
	return ip, nil
}

func dialMigrationSSH(ctx context.Context, req ServerMigrationRequest, expectedFingerprint string) (*ssh.Client, string, error) {
	port := req.Port
	if port == 0 {
		port = 22
	}
	address := net.JoinHostPort(strings.TrimSpace(req.Host), strconv.Itoa(port))
	var fingerprint string
	hostKeyCallback := func(_ string, _ net.Addr, key ssh.PublicKey) error {
		fingerprint = ssh.FingerprintSHA256(key)
		if expectedFingerprint != "" && subtle.ConstantTimeCompare([]byte(fingerprint), []byte(expectedFingerprint)) != 1 {
			return fmt.Errorf("目标服务器主机指纹发生变化（期望 %s，实际 %s）", expectedFingerprint, fingerprint)
		}
		return nil
	}
	sshConfig := &ssh.ClientConfig{
		User: strings.TrimSpace(req.Username), Auth: []ssh.AuthMethod{ssh.Password(req.Password)},
		HostKeyCallback: hostKeyCallback, Timeout: migrationDialTimeout,
	}
	dialer := net.Dialer{Timeout: migrationDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, "", err
	}
	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		conn.Close()
		return nil, "", err
	}
	return ssh.NewClient(sshConn, channels, requests), fingerprint, nil
}

func runMigrationRootCommand(ctx context.Context, client *ssh.Client, req ServerMigrationRequest, command string, timeout time.Duration) (string, error) {
	rootCommand := "sh -c " + shellQuote(command)
	stdin := ""
	if strings.TrimSpace(req.Username) != "root" {
		rootCommand = "sudo -S -p '' -- sh -c " + shellQuote(command)
		stdin = req.Password + "\n"
	}
	return runMigrationCommand(ctx, client, rootCommand, stdin, timeout)
}

func runMigrationCommand(ctx context.Context, client *ssh.Client, command, stdin string, timeout time.Duration) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	if stdin != "" {
		session.Stdin = strings.NewReader(stdin)
	}
	var output bytes.Buffer
	session.Stdout = &output
	session.Stderr = &output
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case err := <-done:
		if err != nil {
			return output.String(), fmt.Errorf("%w: %s", err, migrationErrorTail(output.String()))
		}
		return output.String(), nil
	case <-commandCtx.Done():
		_ = session.Close()
		return output.String(), fmt.Errorf("命令执行超时：%w", commandCtx.Err())
	}
}

func uploadMigrationFile(ctx context.Context, client *ssh.Client, remotePath string, data []byte) error {
	commandCtx, cancel := context.WithTimeout(ctx, migrationCommandTimeout)
	defer cancel()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = bytes.NewReader(data)
	var output bytes.Buffer
	session.Stderr = &output
	done := make(chan error, 1)
	go func() { done <- session.Run("umask 077; cat > " + shellQuote(remotePath)) }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("上传迁移文件失败：%w: %s", err, migrationErrorTail(output.String()))
		}
		return nil
	case <-commandCtx.Done():
		_ = session.Close()
		return fmt.Errorf("上传迁移文件超时：%w", commandCtx.Err())
	}
}

func parseMigrationRemoteInfo(output string) (migrationRemoteInfo, error) {
	values := parseMigrationEnv(output)
	disk, err := strconv.ParseUint(values["DISK_FREE_KB"], 10, 64)
	if err != nil {
		return migrationRemoteInfo{}, errors.New("无法读取目标服务器磁盘空间")
	}
	return migrationRemoteInfo{
		OSID: values["OS_ID"], OSVersion: values["OS_VERSION"], Arch: values["ARCH"],
		MachineID: values["MACHINE_ID"], DiskFreeKB: disk, ExistingInstall: values["EXISTING_INSTALL"] == "1",
	}, nil
}

func parseMigrationEnv(contents string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func loadMigrationDeployConfig() (migrationDeployConfig, error) {
	path := strings.TrimSpace(os.Getenv("NOVA_DEPLOY_ENV"))
	if path == "" {
		path = "/etc/nova/deploy.env"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return migrationDeployConfig{}, errors.New("未找到 /etc/nova/deploy.env；请先使用本项目的 Ubuntu 一键部署脚本安装当前服务器")
	}
	v := parseMigrationEnv(string(data))
	cfg := migrationDeployConfig{
		Repository: v["NOVA_GITHUB_REPO"], ReleaseTag: v["NOVA_RELEASE_TAG"], Domain: v["NOVA_DOMAIN"],
		ACMEEmail: v["NOVA_ACME_EMAIL"], AdminPath: v["NOVA_ADMIN_PATH"], AdminUsername: v["NOVA_ADMIN_USERNAME"],
		PanelPort: v["NOVA_PANEL_PORT"], WebBasePath: v["NOVA_WEB_BASE_PATH"],
		DBName: v["NOVA_DB_NAME"], DBUser: v["NOVA_DB_USER"],
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(cfg.Repository) {
		return migrationDeployConfig{}, errors.New("当前服务器部署配置缺少有效的 GitHub 仓库")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._-]+$`).MatchString(cfg.ReleaseTag) {
		return migrationDeployConfig{}, errors.New("当前服务器部署配置缺少有效的 Release 版本")
	}
	if !regexp.MustCompile(`^[0-9]{18}$`).MatchString(cfg.AdminPath) {
		return migrationDeployConfig{}, errors.New("当前服务器部署配置缺少有效的 18 位管理员入口")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]{4,64}$`).MatchString(cfg.AdminUsername) {
		return migrationDeployConfig{}, errors.New("当前服务器部署配置缺少有效的管理员账号")
	}
	if cfg.PanelPort == "" {
		cfg.PanelPort = "2053"
	}
	if _, err := strconv.Atoi(cfg.PanelPort); err != nil {
		return migrationDeployConfig{}, errors.New("当前服务器面板端口配置无效")
	}
	if cfg.WebBasePath == "" {
		cfg.WebBasePath = "/"
	}
	if cfg.DBName == "" {
		cfg.DBName = "nova"
	}
	if cfg.DBUser == "" {
		cfg.DBUser = "nova"
	}
	return cfg, nil
}

func migrationInstallScriptPath() (string, error) {
	candidates := []string{}
	if explicit := strings.TrimSpace(os.Getenv("NOVA_MIGRATION_INSTALL_SCRIPT")); explicit != "" {
		candidates = append(candidates, explicit)
	}
	if mainFolder := strings.TrimSpace(os.Getenv("XUI_MAIN_FOLDER")); mainFolder != "" {
		candidates = append(candidates, filepath.Join(mainFolder, "deploy", "ubuntu", "install.sh"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "deploy", "ubuntu", "install.sh"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "deploy", "ubuntu", "install.sh"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("当前安装包缺少 deploy/ubuntu/install.sh，无法启动迁移")
}

func migrationInstallCommand(cfg migrationDeployConfig, remoteScript string) string {
	values := map[string]string{
		"NOVA_GITHUB_REPO": cfg.Repository, "NOVA_RELEASE_TAG": cfg.ReleaseTag, "NOVA_DOMAIN": cfg.Domain,
		"NOVA_ACME_EMAIL": cfg.ACMEEmail, "NOVA_ADMIN_PATH": cfg.AdminPath, "NOVA_ADMIN_USERNAME": cfg.AdminUsername,
		"NOVA_ADMIN_PASSWORD": cfg.BootstrapAdminPassword, "NOVA_PANEL_PORT": cfg.PanelPort, "NOVA_WEB_BASE_PATH": cfg.WebBasePath,
		"NOVA_DB_NAME": cfg.DBName, "NOVA_DB_USER": cfg.DBUser,
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	envParts := []string{"env"}
	for _, key := range keys {
		envParts = append(envParts, key+"="+shellQuote(values[key]))
	}
	envParts = append(envParts, "bash", shellQuote(remoteScript))
	return "chmod 700 " + shellQuote(remoteScript) + "\n" + strings.Join(envParts, " ")
}

func migrationRestoreCommand(backupPath string, postgresSource bool) string {
	if postgresSource {
		return `. /etc/nova/deploy.env
systemctl stop x-ui
restore_ok=0
trap 'if [ "${restore_ok}" -eq 0 ]; then systemctl start x-ui || true; fi' EXIT
PGPASSWORD="${NOVA_DB_PASSWORD}" pg_restore --host 127.0.0.1 --username "${NOVA_DB_USER}" --dbname "${NOVA_DB_NAME}" --clean --if-exists --no-owner --no-privileges --single-transaction ` + shellQuote(backupPath) + `
restore_ok=1
systemctl start x-ui
trap - EXIT`
	}
	return `. /etc/nova/deploy.env
systemctl stop x-ui
restore_ok=0
trap 'if [ "${restore_ok}" -eq 0 ]; then systemctl start x-ui || true; fi' EXIT
dsn="postgres://${NOVA_DB_USER}:${NOVA_DB_PASSWORD}@127.0.0.1:5432/${NOVA_DB_NAME}?sslmode=disable"
/usr/local/x-ui/x-ui migrate-db --src ` + shellQuote(backupPath) + ` --dsn "${dsn}"
restore_ok=1
systemctl start x-ui
trap - EXIT`
}

func migrationResultURLs(cfg migrationDeployConfig, targetHost string) (string, string) {
	scheme := "http"
	host := targetHost
	if cfg.Domain != "" {
		scheme, host = "https", cfg.Domain
	}
	base := cfg.WebBasePath
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = "/" + strings.Trim(base, "/")
	if base == "/" {
		base = ""
	}
	return scheme + "://" + host + base + "/portal/", scheme + "://" + host + base + "/" + cfg.AdminPath + "/"
}

func generateMigrationBootstrapPassword() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "Aa1-" + hex.EncodeToString(buffer), nil
}

func migrationDNSReady(ctx context.Context, domain string, target net.IP) (bool, []string) {
	if strings.TrimSpace(domain) == "" {
		return true, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupIPAddr(lookupCtx, domain)
	if err != nil {
		return false, nil
	}
	result := make([]string, 0, len(addresses))
	ready := false
	for _, address := range addresses {
		result = append(result, address.IP.String())
		if address.IP.Equal(target) {
			ready = true
		}
	}
	sort.Strings(result)
	return ready, result
}

func migrationBoolCheck(key, label string, ok bool, success, failure string) MigrationCheck {
	if ok {
		return MigrationCheck{Key: key, Label: label, Status: "success", Detail: success}
	}
	return MigrationCheck{Key: key, Label: label, Status: "error", Detail: failure}
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func migrationErrorTail(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	sensitiveMarkers := []string{
		"NOVA_ADMIN_PASSWORD", "NOVA_DB_PASSWORD", "PGPASSWORD", "管理员密码", "数据库密码",
	}
	for index, line := range lines {
		for _, marker := range sensitiveMarkers {
			if strings.Contains(line, marker) {
				lines[index] = "[敏感凭据已隐藏]"
				break
			}
		}
	}
	out := strings.Join(lines, " | ")
	if len(out) > 1200 {
		out = out[len(out)-1200:]
	}
	return out
}

func formatMigrationBytes(value uint64) string {
	const mb = 1024 * 1024
	const gb = 1024 * mb
	if value >= gb {
		return fmt.Sprintf("%.1f GB", float64(value)/float64(gb))
	}
	return fmt.Sprintf("%.1f MB", float64(value)/float64(mb))
}

func readOptionalFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
