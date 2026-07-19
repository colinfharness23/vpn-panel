#!/usr/bin/env bash
set -Eeuo pipefail

readonly INSTALL_DIR="/usr/local/x-ui"
readonly CONFIG_DIR="/etc/nova"
readonly BACKUP_DIR="/var/backups/nova"
readonly DEPLOY_FILE="/etc/nova/deploy.env"
readonly SERVICE_USER="nova"

tmp_dir=""
tty_echo_disabled=0
log() { printf '[NOVA] %s\n' "$*"; }
die() { printf '[NOVA] 安装失败：%s\n' "$*" >&2; exit 1; }
cleanup() {
  if (( tty_echo_disabled == 1 )); then
    stty echo </dev/tty 2>/dev/null || true
  fi
  [[ -z "$tmp_dir" || ! -d "$tmp_dir" ]] || rm -rf -- "$tmp_dir"
}
on_error() {
  printf '[NOVA] 安装在第 %s 行失败。\n' "$1" >&2
  journalctl -u x-ui -n 80 --no-pager >&2 2>/dev/null || true
}
trap 'on_error "$LINENO"' ERR
trap cleanup EXIT

[[ $EUID -eq 0 ]] || die "请使用 root 执行，或在命令前使用 sudo。"
# shellcheck disable=SC1091
source /etc/os-release
[[ $ID == ubuntu ]] || die "仅支持 Ubuntu 22.04/24.04，当前为 $PRETTY_NAME。"
case "$VERSION_ID" in 22.04|24.04) ;; *) die "不支持 Ubuntu $VERSION_ID。" ;; esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "仅支持 amd64 和 arm64。" ;;
esac

saved_value() {
  [[ -f $DEPLOY_FILE ]] || return 0
  sed -n "s/^$1=//p" "$DEPLOY_FILE" | tail -n1
}
read_tty() {
  local value
  printf '%s' "$1" >/dev/tty
  IFS= read -r value </dev/tty
  printf -v "$2" '%s' "$value"
}
read_secret_tty() {
  local value
  printf '%s' "$1" >/dev/tty
  stty -echo </dev/tty
  tty_echo_disabled=1
  if ! IFS= read -r value </dev/tty; then
    stty echo </dev/tty
    tty_echo_disabled=0
    return 1
  fi
  stty echo </dev/tty
  tty_echo_disabled=0
  printf '\n' >/dev/tty
  printf -v "$2" '%s' "$value"
}
generate_admin_path() {
  local result="" byte
  for _ in $(seq 1 18); do
    byte="$(od -An -N1 -tu1 /dev/urandom | tr -d ' ')"
    result="$result$((byte % 10))"
  done
  printf '%s' "$result"
}
assert_port_available() {
  local line
  line="$(ss -ltnp 2>/dev/null | awk -v suffix=":$1" '$4 ~ suffix"$" {print; exit}')"
  [[ -z "$line" || "$line" == *nginx* ]] || die "端口 $1 已被其他服务占用：$line"
}
wait_http() {
  local url="$1" host="$2"
  for _ in $(seq 1 40); do
    curl -fsS --max-time 3 -H "Host: $host" "$url" >/dev/null && return 0
    sleep 1
  done
  return 1
}

first_install=1
[[ -f $DEPLOY_FILE ]] && first_install=0

NOVA_GITHUB_REPO="${NOVA_GITHUB_REPO:-$(saved_value NOVA_GITHUB_REPO)}"
NOVA_RELEASE_TAG="${NOVA_RELEASE_TAG:-latest}"
NOVA_DOMAIN="${NOVA_DOMAIN:-$(saved_value NOVA_DOMAIN)}"
NOVA_ACME_EMAIL="${NOVA_ACME_EMAIL:-$(saved_value NOVA_ACME_EMAIL)}"
NOVA_ADMIN_PATH="${NOVA_ADMIN_PATH:-$(saved_value NOVA_ADMIN_PATH)}"
NOVA_PANEL_PORT="${NOVA_PANEL_PORT:-$(saved_value NOVA_PANEL_PORT)}"
NOVA_DB_NAME="${NOVA_DB_NAME:-$(saved_value NOVA_DB_NAME)}"
NOVA_DB_USER="${NOVA_DB_USER:-$(saved_value NOVA_DB_USER)}"
NOVA_DB_PASSWORD="${NOVA_DB_PASSWORD:-$(saved_value NOVA_DB_PASSWORD)}"
NOVA_ADMIN_USERNAME="${NOVA_ADMIN_USERNAME:-$(saved_value NOVA_ADMIN_USERNAME)}"
NOVA_DB_NAME="${NOVA_DB_NAME:-nova}"
NOVA_DB_USER="${NOVA_DB_USER:-nova}"

if (( first_install == 1 )); then
  [[ -n "$NOVA_ADMIN_USERNAME" ]] || read_tty "管理员账号：" NOVA_ADMIN_USERNAME
  if [[ -z ${NOVA_ADMIN_PASSWORD:-} ]]; then
    read_secret_tty "管理员密码（至少 12 位，包含大小写字母和数字）：" NOVA_ADMIN_PASSWORD
    read_secret_tty "再次输入管理员密码：" NOVA_ADMIN_PASSWORD_CONFIRM
  else
    NOVA_ADMIN_PASSWORD_CONFIRM="$NOVA_ADMIN_PASSWORD"
  fi
  [[ -n "$NOVA_DOMAIN" ]] || read_tty "已解析到本机的域名：" NOVA_DOMAIN
  [[ -n "$NOVA_ACME_EMAIL" ]] || read_tty "Let's Encrypt 证书通知邮箱：" NOVA_ACME_EMAIL
else
  log "检测到已有安装，将保留域名、管理员入口、数据库和管理员账号。"
fi

[[ $NOVA_GITHUB_REPO =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  die "请在安装命令中设置 NOVA_GITHUB_REPO=用户名/仓库名。"
[[ $NOVA_ADMIN_USERNAME =~ ^[A-Za-z0-9_.-]{4,64}$ ]] || die "管理员账号格式无效。"
[[ $NOVA_DOMAIN =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$ ]] || die "域名格式无效。"
[[ $NOVA_ACME_EMAIL =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]] || die "证书邮箱格式无效。"
[[ $NOVA_DB_NAME =~ ^[A-Za-z][A-Za-z0-9_]*$ ]] || die "数据库名称格式无效。"
[[ $NOVA_DB_USER =~ ^[A-Za-z][A-Za-z0-9_]*$ ]] || die "数据库用户名格式无效。"
if (( first_install == 1 )); then
  [[ $NOVA_ADMIN_PASSWORD == "$NOVA_ADMIN_PASSWORD_CONFIRM" ]] || die "两次输入的密码不一致。"
  password_length="$(printf '%s' "$NOVA_ADMIN_PASSWORD" | wc -c)"
  ((password_length >= 12 && password_length <= 128)) || die "管理员密码长度必须为 12-128 位。"
  [[ $NOVA_ADMIN_PASSWORD =~ [A-Z] && $NOVA_ADMIN_PASSWORD =~ [a-z] && $NOVA_ADMIN_PASSWORD =~ [0-9] ]] ||
    die "管理员密码必须包含大写字母、小写字母和数字。"
fi

[[ -n "$NOVA_ADMIN_PATH" ]] || NOVA_ADMIN_PATH="$(generate_admin_path)"
[[ $NOVA_ADMIN_PATH =~ ^[0-9]{18}$ ]] || die "管理员入口必须为 18 位数字。"
[[ -n "$NOVA_PANEL_PORT" ]] || NOVA_PANEL_PORT="$(shuf -i 20000-59999 -n1)"
[[ $NOVA_PANEL_PORT =~ ^[0-9]+$ ]] && ((NOVA_PANEL_PORT >= 1024 && NOVA_PANEL_PORT <= 65535)) ||
  die "内部面板端口必须位于 1024-65535。"

export DEBIAN_FRONTEND=noninteractive
log "安装 Ubuntu 运行依赖……"
apt-get update -y
apt-get install -y --no-install-recommends ca-certificates curl jq tar openssl nginx \
  postgresql postgresql-contrib certbot python3-certbot-nginx iproute2
systemctl enable --now postgresql
assert_port_available 80
assert_port_available 443
systemctl enable --now nginx
getent ahosts "$NOVA_DOMAIN" >/dev/null || die "域名 $NOVA_DOMAIN 尚未完成 DNS 解析。"

id -u "$SERVICE_USER" >/dev/null 2>&1 ||
  useradd --system --home-dir /var/lib/x-ui --create-home --shell /usr/sbin/nologin "$SERVICE_USER"
install -d -m 700 "$CONFIG_DIR" "$BACKUP_DIR/database" "$BACKUP_DIR/releases"
install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 750 /var/lib/x-ui /var/log/x-ui

[[ -n "$NOVA_DB_PASSWORD" ]] || NOVA_DB_PASSWORD="$(openssl rand -hex 24)"
[[ $NOVA_DB_PASSWORD =~ ^[A-Za-z0-9]{32,128}$ ]] || die "数据库密码格式无效。"

log "配置本机 PostgreSQL……"
if ! runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$NOVA_DB_USER'" | grep -q 1; then
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "CREATE ROLE \"$NOVA_DB_USER\" LOGIN PASSWORD '$NOVA_DB_PASSWORD'"
else
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "ALTER ROLE \"$NOVA_DB_USER\" WITH LOGIN PASSWORD '$NOVA_DB_PASSWORD'"
fi
runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_database WHERE datname='$NOVA_DB_NAME'" | grep -q 1 ||
  runuser -u postgres -- createdb -O "$NOVA_DB_USER" "$NOVA_DB_NAME"

if [[ $NOVA_RELEASE_TAG == latest ]]; then
  NOVA_RELEASE_TAG="$(curl -fsSL --retry 4 "https://api.github.com/repos/$NOVA_GITHUB_REPO/releases/latest" | jq -r '.tag_name // empty')"
fi
[[ $NOVA_RELEASE_TAG =~ ^[A-Za-z0-9._-]+$ ]] || die "没有找到可安装的 GitHub Release。"

tmp_dir="$(mktemp -d)"
asset="x-ui-linux-$ARCH.tar.gz"
asset_url="https://github.com/$NOVA_GITHUB_REPO/releases/download/$NOVA_RELEASE_TAG/$asset"
log "下载并校验 $NOVA_RELEASE_TAG ($ARCH)……"
curl -fL --retry 5 --retry-delay 3 --max-time 600 -o "$tmp_dir/$asset" "$asset_url"
curl -fL --retry 5 --retry-delay 3 --max-time 60 -o "$tmp_dir/$asset.sha256" "$asset_url.sha256"
(cd "$tmp_dir" && sha256sum --check "$asset.sha256")
tar -tzf "$tmp_dir/$asset" | grep -Eq '(^/|(^|/)\.\.(/|$))' && die "安装包包含不安全路径。"
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
[[ -x $tmp_dir/x-ui/x-ui ]] || die "安装包缺少 x-ui。"
[[ -f $tmp_dir/x-ui/bin/config.json ]] || die "安装包缺少 bin/config.json。"
[[ -x $tmp_dir/x-ui/bin/xray-linux-$ARCH ]] || die "安装包缺少对应架构的 Xray。"

if [[ -d $INSTALL_DIR ]]; then
  systemctl stop x-ui 2>/dev/null || true
  release_backup="$BACKUP_DIR/releases/x-ui-before-$NOVA_RELEASE_TAG-$(date -u +%Y%m%dT%H%M%SZ).tar.gz"
  tar -czf "$release_backup" -C /usr/local x-ui
  printf '%s\n' "$(saved_value NOVA_RELEASE_TAG)" >"$release_backup.tag"
  chmod 600 "$release_backup" "$release_backup.tag"
fi
rm -rf -- "$INSTALL_DIR.new"
install -d -m 755 "$INSTALL_DIR.new"
cp -a "$tmp_dir/x-ui/." "$INSTALL_DIR.new/"
chmod 755 "$INSTALL_DIR.new/x-ui" "$INSTALL_DIR.new/bin/"* 2>/dev/null || true
rm -rf -- "$INSTALL_DIR"
mv "$INSTALL_DIR.new" "$INSTALL_DIR"
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

cat >/etc/default/x-ui <<EOF
XUI_DB_TYPE=postgres
XUI_DB_DSN=postgres://$NOVA_DB_USER:$NOVA_DB_PASSWORD@127.0.0.1:5432/$NOVA_DB_NAME?sslmode=disable
XUI_DB_MAX_OPEN_CONNS=40
XUI_DB_MAX_IDLE_CONNS=10
XUI_MAIN_FOLDER=$INSTALL_DIR
XUI_DB_FOLDER=/var/lib/x-ui
XUI_LOG_FOLDER=/var/log/x-ui
XUI_BIN_FOLDER=$INSTALL_DIR/bin
XUI_PORT=$NOVA_PANEL_PORT
XUI_INIT_WEB_BASE_PATH=/
XUI_ADMIN_BASE_PATH=/$NOVA_ADMIN_PATH/
XUI_BEHIND_HTTPS_PROXY=true
XUI_SKIP_HSTS=false
XUI_DEBUG=false
XUI_LOG_LEVEL=info
XUI_COMMERCIAL_DEMO=false
XUI_COMMERCIAL_ENV=production
GIN_MODE=release
EOF
chmod 600 /etc/default/x-ui

cat >/etc/systemd/system/x-ui.service <<'EOF'
[Unit]
Description=NOVA VPN Commercial Panel
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

[Service]
Type=simple
User=nova
Group=nova
Environment=HOME=/var/lib/x-ui
EnvironmentFile=/etc/default/x-ui
WorkingDirectory=/usr/local/x-ui
ExecStart=/usr/local/x-ui/x-ui
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576
PrivateTmp=true
NoNewPrivileges=true
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
UMask=0027

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable x-ui

set -a
# shellcheck disable=SC1091
source /etc/default/x-ui
set +a
if (( first_install == 1 )); then
  runuser -u "$SERVICE_USER" --preserve-environment -- "$INSTALL_DIR/x-ui" setting \
    -username "$NOVA_ADMIN_USERNAME" -password "$NOVA_ADMIN_PASSWORD" \
    -port "$NOVA_PANEL_PORT" -webBasePath "/" -listenIP "127.0.0.1"
else
  runuser -u "$SERVICE_USER" --preserve-environment -- "$INSTALL_DIR/x-ui" setting \
    -port "$NOVA_PANEL_PORT" -webBasePath "/" -listenIP "127.0.0.1"
fi
systemctl restart x-ui

cat >/etc/nginx/sites-available/nova.conf <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name $NOVA_DOMAIN;
    client_max_body_size 16m;
    location / {
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
EOF
ln -sfn /etc/nginx/sites-available/nova.conf /etc/nginx/sites-enabled/nova.conf
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx

wait_http "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" "$NOVA_DOMAIN" || die "用户门户启动检查失败。"
wait_http "http://127.0.0.1:$NOVA_PANEL_PORT/$NOVA_ADMIN_PATH/" "$NOVA_DOMAIN" || die "管理员入口启动检查失败。"

log "申请并验证 Let's Encrypt 证书……"
certbot --nginx --non-interactive --agree-tos --redirect --email "$NOVA_ACME_EMAIL" -d "$NOVA_DOMAIN"
systemctl enable --now certbot.timer
certbot renew --cert-name "$NOVA_DOMAIN" --dry-run --no-random-sleep-on-renew

for script in update rollback backup rotate-admin-path; do
  [[ -f $INSTALL_DIR/deploy/ubuntu/$script.sh ]] &&
    install -m 755 "$INSTALL_DIR/deploy/ubuntu/$script.sh" "/usr/local/sbin/nova-$script"
done

cat >"$DEPLOY_FILE" <<EOF
NOVA_GITHUB_REPO=$NOVA_GITHUB_REPO
NOVA_RELEASE_TAG=$NOVA_RELEASE_TAG
NOVA_DOMAIN=$NOVA_DOMAIN
NOVA_ACME_EMAIL=$NOVA_ACME_EMAIL
NOVA_ADMIN_PATH=$NOVA_ADMIN_PATH
NOVA_PANEL_PORT=$NOVA_PANEL_PORT
NOVA_DB_NAME=$NOVA_DB_NAME
NOVA_DB_USER=$NOVA_DB_USER
NOVA_DB_PASSWORD=$NOVA_DB_PASSWORD
NOVA_ADMIN_USERNAME=$NOVA_ADMIN_USERNAME
NOVA_WEB_BASE_PATH=/
EOF
chmod 600 "$DEPLOY_FILE"

cat >/etc/systemd/system/nova-backup.service <<'EOF'
[Unit]
Description=Backup NOVA database and deployment configuration
After=postgresql.service
[Service]
Type=oneshot
ExecStart=/usr/local/sbin/nova-backup
EOF
cat >/etc/systemd/system/nova-backup.timer <<'EOF'
[Unit]
Description=Daily NOVA backup
[Timer]
OnCalendar=*-*-* 03:30:00
Persistent=true
RandomizedDelaySec=20m
[Install]
WantedBy=timers.target
EOF
systemctl daemon-reload
systemctl enable --now nova-backup.timer

log "执行数据库、门户、后台、Xray、订阅和证书健康检查……"
systemctl is-active --quiet postgresql
systemctl is-active --quiet nginx
systemctl is-active --quiet x-ui
PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" -tAc 'SELECT 1' | grep -qx 1
pgrep -u "$SERVICE_USER" -f "$INSTALL_DIR/bin/xray-linux-$ARCH" >/dev/null || die "Xray 进程未运行。"
curl -fsS --max-time 10 "https://$NOVA_DOMAIN/portal/" >/dev/null
curl -fsS --max-time 10 "https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/" >/dev/null
curl -fsS --max-time 10 "https://$NOVA_DOMAIN/api/v1/guest/bootstrap" | jq -e '.success == true' >/dev/null
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "https://$NOVA_DOMAIN/panel/")"
[[ $status == 404 ]] || die "公开 /panel/ 未返回 404（HTTP $status）。"
if [[ $NOVA_ADMIN_PATH != 123456789012345678 ]]; then
  status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "https://$NOVA_DOMAIN/123456789012345678/")"
  [[ $status == 404 ]] || die "错误管理员路径未返回 404（HTTP $status）。"
fi
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "https://$NOVA_DOMAIN/sub/health-probe")"
[[ $status != 000 && $status -lt 500 ]] || die "订阅服务健康检查失败（HTTP $status）。"
systemctl is-enabled --quiet certbot.timer

cat >/root/nova-install-result.txt <<EOF
网站地址: https://$NOVA_DOMAIN/portal/
管理员后台: https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/
管理员账号: $NOVA_ADMIN_USERNAME
已安装版本: $NOVA_RELEASE_TAG
更新命令: sudo nova-update
回滚命令: sudo nova-rollback
备份命令: sudo nova-backup
轮换后台入口: sudo nova-rotate-admin-path
服务状态: sudo systemctl status x-ui
服务日志: sudo journalctl -u x-ui -f
EOF
chmod 600 /root/nova-install-result.txt
unset NOVA_ADMIN_PASSWORD NOVA_ADMIN_PASSWORD_CONFIRM
log "安装和全部健康检查完成。结果仅保存在 /root/nova-install-result.txt（权限 600）。"
cat /root/nova-install-result.txt
