#!/usr/bin/env bash
set -Eeuo pipefail

readonly INSTALL_DIR="/usr/local/x-ui"
readonly CONFIG_DIR="/etc/nova"
readonly BACKUP_ROOT="/var/backups/nova"
readonly DEPLOY_FILE="/etc/nova/deploy.env"
readonly DATA_DIR="/var/lib/x-ui"
readonly UPLOAD_DIR="/var/lib/x-ui/uploads/client-applications"
readonly LOG_DIR="/var/log/x-ui"
readonly SERVICE_USER="nova"

tmp_dir=""
tty_echo_disabled=0
log() { printf '[NOVA] %s\n' "$*"; }
die() { printf '[NOVA] 安装失败：%s\n' "$*" >&2; exit 1; }
cleanup() {
  if ((tty_echo_disabled == 1)); then
    stty echo </dev/tty 2>/dev/null || true
  fi
  if [[ -n $tmp_dir && -d $tmp_dir ]]; then
    case "$tmp_dir" in
      /tmp/nova-install.*) rm -rf -- "$tmp_dir" ;;
    esac
  fi
}
on_error() {
  printf '[NOVA] 安装在第 %s 行失败。\n' "$1" >&2
  journalctl -u x-ui -n 100 --no-pager >&2 2>/dev/null || true
}
trap 'on_error "$LINENO"' ERR
trap cleanup EXIT

[[ $EUID -eq 0 ]] || die "请使用 root 执行，或在命令前使用 sudo。"
source /etc/os-release
[[ $ID == ubuntu ]] || die "仅支持 Ubuntu 22.04/24.04，当前为 $PRETTY_NAME。"
case "$VERSION_ID" in
  22.04|24.04) ;;
  *) die "不支持 Ubuntu $VERSION_ID。" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "仅支持 amd64 和 arm64。" ;;
esac

saved_value() {
  existing_env_value "$DEPLOY_FILE" "$1"
}
existing_env_value() {
  local file="$1" key="$2" value first last
  [[ -f $file ]] || return 0
  value="$(sed -n "s/^$key=//p" "$file" | tail -n1)"
  value="${value%$'\r'}"
  if ((${#value} >= 2)); then
    first="${value:0:1}"
    last="${value: -1}"
    if [[ $first == '"' && $last == '"' ]] || [[ $first == "'" && $last == "'" ]]; then
      value="${value:1:${#value}-2}"
    fi
  fi
  printf '%s' "$value"
}
read_tty_default() {
  local prompt="$1" variable="$2" default_value="${3:-}" value
  if [[ -n $default_value ]]; then
    printf '%s [%s]：' "$prompt" "$default_value" >/dev/tty
  else
    printf '%s：' "$prompt" >/dev/tty
  fi
  IFS= read -r value </dev/tty
  [[ -n $value ]] || value="$default_value"
  printf -v "$variable" '%s' "$value"
}
read_secret_tty() {
  local value
  printf '%s' "$1" >/dev/tty
  stty -echo </dev/tty
  tty_echo_disabled=1
  IFS= read -r value </dev/tty
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
generate_port() {
  local minimum="$1" maximum="$2" excluded="${3:-}" candidate
  for _ in $(seq 1 300); do
    candidate="$(shuf -i "$minimum-$maximum" -n1)"
    [[ $candidate != "$excluded" ]] || continue
    if ! ss -H -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "(^|:)$candidate$"; then
      printf '%s' "$candidate"
      return 0
    fi
  done
  return 1
}
valid_internal_port() {
  [[ ${1:-} =~ ^1[0-9]{4}$ ]]
}
reset_saved_internal_port() {
  local explicit="$1" variable="$2" label="$3" excluded="${4:-}" value
  value="${!variable:-}"
  if ((explicit == 0)) && [[ -n $value ]] && { ! valid_internal_port "$value" || [[ -n $excluded && $value == "$excluded" ]]; }; then
    log "检测到无效或冲突的旧版${label}端口 $value，正在重新随机生成。"
    printf -v "$variable" '%s' ''
  fi
}
assert_port_available() {
  local line
  line="$(ss -H -ltnp 2>/dev/null | awk -v suffix=":$1" '$4 ~ suffix"$" {print; exit}')"
  [[ -z $line || $line == *x-ui* ]] || die "端口 $1 已被其他服务占用：$line"
}
wait_http() {
  local url="$1" host="$2" expected="${3:-200}"
  local status
  for _ in $(seq 1 60); do
    status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 -H "Host: $host" "$url" || true)"
    [[ $status == "$expected" ]] && return 0
    sleep 1
  done
  return 1
}
wait_xray_process() {
  local timeout_seconds="${1:-60}"
  for _ in $(seq 1 "$timeout_seconds"); do
    if pgrep -u "$SERVICE_USER" -f "$INSTALL_DIR/bin/xray-linux-$ARCH" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}
print_xray_diagnostics() {
  printf '[NOVA] Xray 未在等待时间内进入运行状态，以下是诊断信息：\n' >&2
  systemctl status x-ui --no-pager -l >&2 2>/dev/null || true
  journalctl -u x-ui -n 160 --no-pager >&2 2>/dev/null || true
  if [[ -x $INSTALL_DIR/bin/xray-linux-$ARCH && -f $INSTALL_DIR/bin/config.json ]]; then
    runuser -u "$SERVICE_USER" -- "$INSTALL_DIR/bin/xray-linux-$ARCH" run -test \
      -c "$INSTALL_DIR/bin/config.json" >&2 2>/dev/null || true
  fi
}

if [[ ${NOVA_INSTALL_PORT_SELF_TEST:-0} == 1 ]]; then
  NOVA_PANEL_PORT=54321
  reset_saved_internal_port 0 NOVA_PANEL_PORT "内部面板"
  [[ -z $NOVA_PANEL_PORT ]] || die "旧版面板端口自检失败。"

  NOVA_PANEL_PORT=15432
  reset_saved_internal_port 0 NOVA_PANEL_PORT "内部面板"
  [[ $NOVA_PANEL_PORT == 15432 ]] || die "有效面板端口自检失败。"

  NOVA_SUB_PORT=15432
  reset_saved_internal_port 0 NOVA_SUB_PORT "订阅服务" "$NOVA_PANEL_PORT"
  [[ -z $NOVA_SUB_PORT ]] || die "冲突订阅端口自检失败。"

  NOVA_PANEL_PORT=54321
  reset_saved_internal_port 1 NOVA_PANEL_PORT "内部面板"
  [[ $NOVA_PANEL_PORT == 54321 ]] || die "显式端口自检失败。"

  log "内部端口兼容性自检通过。"
  exit 0
fi

export DEBIAN_FRONTEND=noninteractive
log "安装 Ubuntu 运行依赖…"
# Fresh Ubuntu hosts commonly start apt-daily/unattended-upgrades during the
# first login. Let apt wait for dpkg's frontend lock instead of failing the
# entire installation while that legitimate system update is still running.
readonly APT_LOCK_TIMEOUT_SECONDS="${NOVA_APT_LOCK_TIMEOUT_SECONDS:-1800}"
[[ $APT_LOCK_TIMEOUT_SECONDS =~ ^[0-9]+$ ]] || die "NOVA_APT_LOCK_TIMEOUT_SECONDS 必须是非负整数。"
apt-get -o "DPkg::Lock::Timeout=$APT_LOCK_TIMEOUT_SECONDS" -o Acquire::Retries=5 update -y
apt-get -o "DPkg::Lock::Timeout=$APT_LOCK_TIMEOUT_SECONDS" -o Acquire::Retries=5 install -y --no-install-recommends ca-certificates curl jq tar openssl nginx postgresql postgresql-contrib postgresql-client certbot python3-certbot-nginx iproute2 sqlite3
systemctl enable --now postgresql
systemctl enable --now nginx

panel_port_explicit=0
sub_port_explicit=0
domain_explicit=0
admin_username_explicit=0
[[ -n ${NOVA_PANEL_PORT:-} ]] && panel_port_explicit=1
[[ -n ${NOVA_SUB_PORT:-} ]] && sub_port_explicit=1
[[ -n ${NOVA_DOMAIN:-} ]] && domain_explicit=1
[[ -n ${NOVA_ADMIN_USERNAME:-} ]] && admin_username_explicit=1

NOVA_GITHUB_REPO="${NOVA_GITHUB_REPO:-$(saved_value NOVA_GITHUB_REPO)}"
NOVA_RELEASE_TAG="${NOVA_RELEASE_TAG:-latest}"
NOVA_RELEASE_ASSET_DIR="${NOVA_RELEASE_ASSET_DIR:-}"
NOVA_DOMAIN="${NOVA_DOMAIN:-$(saved_value NOVA_DOMAIN)}"
NOVA_ADMIN_PATH="${NOVA_ADMIN_PATH:-$(saved_value NOVA_ADMIN_PATH)}"
NOVA_PANEL_PORT="${NOVA_PANEL_PORT:-$(saved_value NOVA_PANEL_PORT)}"
NOVA_SUB_PORT="${NOVA_SUB_PORT:-$(saved_value NOVA_SUB_PORT)}"
NOVA_SUB_PATH="${NOVA_SUB_PATH:-$(saved_value NOVA_SUB_PATH)}"
NOVA_SUB_JSON_PATH="${NOVA_SUB_JSON_PATH:-$(saved_value NOVA_SUB_JSON_PATH)}"
NOVA_SUB_CLASH_PATH="${NOVA_SUB_CLASH_PATH:-$(saved_value NOVA_SUB_CLASH_PATH)}"
NOVA_DB_NAME="${NOVA_DB_NAME:-$(saved_value NOVA_DB_NAME)}"
NOVA_DB_USER="${NOVA_DB_USER:-$(saved_value NOVA_DB_USER)}"
NOVA_DB_PASSWORD="${NOVA_DB_PASSWORD:-$(saved_value NOVA_DB_PASSWORD)}"
NOVA_ADMIN_USERNAME="${NOVA_ADMIN_USERNAME:-$(saved_value NOVA_ADMIN_USERNAME)}"
NOVA_DEFER_TLS="${NOVA_DEFER_TLS:-false}"
NOVA_EXPECTED_PUBLIC_IP="${NOVA_EXPECTED_PUBLIC_IP:-$(saved_value NOVA_EXPECTED_PUBLIC_IP)}"
NOVA_PREVIOUS_DNS_IPS="${NOVA_PREVIOUS_DNS_IPS:-$(saved_value NOVA_PREVIOUS_DNS_IPS)}"
NOVA_DB_NAME="${NOVA_DB_NAME:-nova}"
NOVA_DB_USER="${NOVA_DB_USER:-nova}"

if ((domain_explicit == 0)); then
  read_tty_default "需要绑定的域名（例如 vpn.example.com）" NOVA_DOMAIN "$NOVA_DOMAIN"
fi
if ((admin_username_explicit == 0)); then
  read_tty_default "管理员登录账号" NOVA_ADMIN_USERNAME "$NOVA_ADMIN_USERNAME"
fi
if [[ -z ${NOVA_ADMIN_PASSWORD:-} ]]; then
  read_secret_tty "管理员密码（至少 12 位，包含大小写字母和数字）：" NOVA_ADMIN_PASSWORD
  read_secret_tty "再次输入管理员密码：" NOVA_ADMIN_PASSWORD_CONFIRM
else
  NOVA_ADMIN_PASSWORD_CONFIRM="$NOVA_ADMIN_PASSWORD"
fi
[[ $NOVA_GITHUB_REPO =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "请设置 NOVA_GITHUB_REPO=用户名/仓库名。"
[[ $NOVA_ADMIN_USERNAME =~ ^[A-Za-z0-9_.-]{4,64}$ ]] || die "管理员账号格式无效。"
NOVA_DOMAIN="${NOVA_DOMAIN,,}"
[[ $NOVA_DOMAIN =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$ ]] || die "域名格式无效。"
[[ $NOVA_DB_NAME =~ ^[A-Za-z][A-Za-z0-9_]*$ ]] || die "数据库名称格式无效。"
[[ $NOVA_DB_USER =~ ^[A-Za-z][A-Za-z0-9_]*$ ]] || die "数据库用户名格式无效。"
[[ $NOVA_DB_USER != postgres ]] || die "数据库用户不能使用 PostgreSQL 超级用户 postgres。"
case "$NOVA_DB_NAME" in
  postgres|template0|template1) die "数据库名称不能使用 PostgreSQL 系统数据库。" ;;
esac
[[ $NOVA_ADMIN_PASSWORD == "$NOVA_ADMIN_PASSWORD_CONFIRM" ]] || die "两次输入的密码不一致。"
[[ $NOVA_DEFER_TLS == true || $NOVA_DEFER_TLS == false ]] || die "NOVA_DEFER_TLS 只能是 true 或 false。"
if [[ $NOVA_DEFER_TLS == true ]]; then
  [[ -n $NOVA_EXPECTED_PUBLIC_IP && $NOVA_EXPECTED_PUBLIC_IP != *[[:space:]]* ]] || die "延迟 HTTPS 模式缺少有效的迁移目标公网 IP。"
  [[ $NOVA_PREVIOUS_DNS_IPS != *[[:space:]]* ]] || die "迁移前 DNS 地址列表无效。"
fi
password_length="$(printf '%s' "$NOVA_ADMIN_PASSWORD" | wc -c)"
((password_length >= 12 && password_length <= 128)) || die "管理员密码长度必须为 12-128 位。"
[[ $NOVA_ADMIN_PASSWORD =~ [A-Z] && $NOVA_ADMIN_PASSWORD =~ [a-z] && $NOVA_ADMIN_PASSWORD =~ [0-9] ]] || die "管理员密码必须包含大写字母、小写字母和数字。"

[[ -n $NOVA_ADMIN_PATH ]] || NOVA_ADMIN_PATH="$(generate_admin_path)"
[[ $NOVA_ADMIN_PATH =~ ^[0-9]{18}$ ]] || die "管理员入口必须为 18 位数字。"
reset_saved_internal_port "$panel_port_explicit" NOVA_PANEL_PORT "内部面板"
[[ -n $NOVA_PANEL_PORT ]] || NOVA_PANEL_PORT="$(generate_port 10000 19999)"
reset_saved_internal_port "$sub_port_explicit" NOVA_SUB_PORT "订阅服务" "$NOVA_PANEL_PORT"
[[ -n $NOVA_SUB_PORT ]] || NOVA_SUB_PORT="$(generate_port 10000 19999 "$NOVA_PANEL_PORT")"
valid_internal_port "$NOVA_PANEL_PORT" || die "内部面板端口必须位于 10000-19999。"
valid_internal_port "$NOVA_SUB_PORT" || die "订阅服务端口必须位于 10000-19999。"
[[ $NOVA_SUB_PORT != "$NOVA_PANEL_PORT" ]] || die "订阅端口不能与面板端口相同。"
[[ -n $NOVA_SUB_PATH ]] || NOVA_SUB_PATH="/s-$(openssl rand -hex 10)/"
[[ -n $NOVA_SUB_JSON_PATH ]] || NOVA_SUB_JSON_PATH="/j-$(openssl rand -hex 10)/"
[[ -n $NOVA_SUB_CLASH_PATH ]] || NOVA_SUB_CLASH_PATH="/c-$(openssl rand -hex 10)/"
for path_value in "$NOVA_SUB_PATH" "$NOVA_SUB_JSON_PATH" "$NOVA_SUB_CLASH_PATH"; do
  [[ $path_value =~ ^/[a-z]-[a-f0-9]{20}/$ ]] || die "订阅路径格式无效。"
done
assert_port_available "$NOVA_PANEL_PORT"
assert_port_available "$NOVA_SUB_PORT"

getent ahosts "$NOVA_DOMAIN" >/dev/null || die "域名 $NOVA_DOMAIN 尚未完成 DNS 解析。"

id -u "$SERVICE_USER" >/dev/null 2>&1 || useradd --system --home-dir "$DATA_DIR" --create-home --shell /usr/sbin/nologin "$SERVICE_USER"
install -d -m 700 "$CONFIG_DIR" "$BACKUP_ROOT"
install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 750 "$DATA_DIR" "$UPLOAD_DIR" "$LOG_DIR"

legacy_pg_dsn="$(existing_env_value /etc/default/x-ui XUI_DB_DSN)"
old_install=0
[[ -f $DEPLOY_FILE || -d $INSTALL_DIR || -f /etc/default/x-ui || -f /etc/systemd/system/x-ui.service ]] && old_install=1
if runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_database WHERE datname='$NOVA_DB_NAME'" | grep -qx 1; then
  old_install=1
fi
if find /etc/x-ui "$DATA_DIR" -maxdepth 1 -type f -name '*.db' -print -quit 2>/dev/null | grep -q .; then
  old_install=1
fi

if ((old_install == 1)); then
  preinstall_dir="$BACKUP_ROOT/preinstall-$(date -u +%Y%m%dT%H%M%SZ)"
  install -d -m 700 "$preinstall_dir" "$preinstall_dir/sqlite"
  log "检测到旧安装，先创建可恢复备份：$preinstall_dir"
  config_paths=()
  for existing_path in "$DEPLOY_FILE" /etc/default/x-ui /etc/systemd/system/x-ui.service /etc/nginx/sites-available/nova.conf /etc/x-ui; do
    [[ -e $existing_path ]] && config_paths+=("$existing_path")
  done
  if ((${#config_paths[@]} > 0)); then
    tar -czf "$preinstall_dir/configuration.tar.gz" "${config_paths[@]}"
    tar -tzf "$preinstall_dir/configuration.tar.gz" >/dev/null
  fi
  if [[ -d $INSTALL_DIR ]]; then
    tar -czf "$preinstall_dir/program.tar.gz" -C "$(dirname "$INSTALL_DIR")" "$(basename "$INSTALL_DIR")"
    tar -tzf "$preinstall_dir/program.tar.gz" >/dev/null
  fi
  if [[ -d $DATA_DIR/uploads ]]; then
    tar -czf "$preinstall_dir/uploads.tar.gz" -C "$DATA_DIR" uploads
    tar -tzf "$preinstall_dir/uploads.tar.gz" >/dev/null
  fi
  if [[ -n $legacy_pg_dsn ]]; then
    PGCONNECT_TIMEOUT=10 pg_dump --format=custom --dbname="$legacy_pg_dsn" >"$preinstall_dir/legacy-postgresql.dump"
    pg_restore --list "$preinstall_dir/legacy-postgresql.dump" >/dev/null
  fi
  if runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_database WHERE datname='$NOVA_DB_NAME'" | grep -qx 1; then
    runuser -u postgres -- pg_dump --format=custom "$NOVA_DB_NAME" >"$preinstall_dir/postgresql.dump"
    pg_restore --list "$preinstall_dir/postgresql.dump" >/dev/null
  fi
  while IFS= read -r -d '' sqlite_file; do
    sqlite3 "$sqlite_file" 'PRAGMA integrity_check;' | grep -qx ok || die "SQLite 备份前完整性检查失败：$sqlite_file"
    sqlite_name="$(printf '%s' "$sqlite_file" | sed 's#/#_#g; s/^_//')"
    cp -a "$sqlite_file" "$preinstall_dir/sqlite/$sqlite_name"
    sqlite3 "$preinstall_dir/sqlite/$sqlite_name" 'PRAGMA integrity_check;' | grep -qx ok || die "SQLite 备份校验失败：$sqlite_file"
  done < <(find /etc/x-ui "$DATA_DIR" -maxdepth 1 -type f -name '*.db' -print0 2>/dev/null)
  find "$preinstall_dir" -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum >"$preinstall_dir/SHA256SUMS"
  [[ -s $preinstall_dir/SHA256SUMS ]] || die "旧安装备份为空，已拒绝继续。"
  sha256sum --check "$preinstall_dir/SHA256SUMS" >/dev/null
  log "旧安装备份已验证。"
fi

if [[ -z $NOVA_RELEASE_ASSET_DIR && $NOVA_RELEASE_TAG == latest ]]; then
  NOVA_RELEASE_TAG="$(curl -fsSL --retry 4 "https://api.github.com/repos/$NOVA_GITHUB_REPO/releases/latest" | jq -r '.tag_name // empty')"
fi
if [[ -n $NOVA_RELEASE_ASSET_DIR && $NOVA_RELEASE_TAG == latest ]]; then
  NOVA_RELEASE_TAG="candidate-local"
fi
[[ $NOVA_RELEASE_TAG =~ ^[A-Za-z0-9._-]+$ ]] || die "没有找到可安装的 GitHub Release。"
tmp_dir="$(mktemp -d /tmp/nova-install.XXXXXX)"
asset="x-ui-linux-$ARCH.tar.gz"
if [[ -n $NOVA_RELEASE_ASSET_DIR ]]; then
  [[ $NOVA_RELEASE_ASSET_DIR == /* && -d $NOVA_RELEASE_ASSET_DIR ]] || die "NOVA_RELEASE_ASSET_DIR 必须是存在的绝对目录。"
  cp "$NOVA_RELEASE_ASSET_DIR/$asset" "$tmp_dir/$asset"
  cp "$NOVA_RELEASE_ASSET_DIR/$asset.sha256" "$tmp_dir/$asset.sha256"
else
  release_url="https://github.com/$NOVA_GITHUB_REPO/releases/download/$NOVA_RELEASE_TAG/$asset"
  log "下载候选版本 $NOVA_RELEASE_TAG ($ARCH)…"
  curl -fL --retry 5 --retry-delay 3 --max-time 900 -o "$tmp_dir/$asset" "$release_url"
  curl -fL --retry 5 --retry-delay 3 --max-time 60 -o "$tmp_dir/$asset.sha256" "$release_url.sha256"
fi
expected_sha="$(awk -v name="$asset" '$2 == name || $2 == "*"name {print $1; exit}' "$tmp_dir/$asset.sha256")"
[[ $expected_sha =~ ^[a-fA-F0-9]{64}$ ]] || die "Release SHA-256 文件格式无效。"
actual_sha="$(sha256sum "$tmp_dir/$asset" | awk '{print $1}')"
[[ ${actual_sha,,} == "${expected_sha,,}" ]] || die "Release SHA-256 校验失败。"
if tar -tzf "$tmp_dir/$asset" | awk '$0 ~ /(^\/|(^|\/)\.\.(\/|$))/ { found=1 } END { exit !found }'; then
  die "Release 包含不安全路径。"
fi
if tar -tzf "$tmp_dir/$asset" | awk '$0 !~ /^x-ui(\/|$)/ { found=1 } END { exit !found }'; then
  die "Release 包含 x-ui 目录之外的文件。"
fi
if tar -tvzf "$tmp_dir/$asset" | awk 'substr($1,1,1) !~ /^[-d]$/ { found=1 } END { exit !found }'; then
  die "Release 包含符号链接、硬链接或设备文件。"
fi
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
[[ -x $tmp_dir/x-ui/x-ui ]] || die "Release 缺少 x-ui。"
[[ -f $tmp_dir/x-ui/bin/config.json ]] || die "Release 缺少 Xray 配置。"
[[ -x $tmp_dir/x-ui/bin/xray-linux-$ARCH ]] || die "Release 缺少 $ARCH Xray。"
[[ -x $tmp_dir/x-ui/bin/sing-box-linux-$ARCH ]] || die "Release 缺少 $ARCH AnyTLS 运行时。"
for required_script in install update rollback backup rotate-admin-path uninstall finalize-domain sync-line-cert; do
  [[ -f $tmp_dir/x-ui/deploy/ubuntu/$required_script.sh ]] || die "Release 缺少运维脚本 $required_script.sh。"
done
for required_diagnostic in diagnose-active-subscription diagnose-managed-ingress; do
  [[ -f $tmp_dir/x-ui/deploy/ubuntu/$required_diagnostic.py ]] || die "Release 缺少诊断脚本 $required_diagnostic.py。"
done

systemctl stop x-ui 2>/dev/null || true
if ((old_install == 1)); then
  find /etc/x-ui "$DATA_DIR" -maxdepth 1 -type f -name '*.db' -delete 2>/dev/null || true
  [[ $DATA_DIR == /var/lib/x-ui ]] || die "数据目录安全检查失败。"
  rm -rf -- /etc/x-ui/uploads
  rm -rf -- "$DATA_DIR/uploads"
  install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 750 "$UPLOAD_DIR"
fi
[[ -n $NOVA_DB_PASSWORD ]] || NOVA_DB_PASSWORD="$(openssl rand -hex 24)"
[[ $NOVA_DB_PASSWORD =~ ^[A-Za-z0-9]{32,128}$ ]] || die "数据库密码格式无效。"
runuser -u postgres -- dropdb --if-exists --force "$NOVA_DB_NAME"
if runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$NOVA_DB_USER'" | grep -qx 1; then
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "ALTER ROLE \"$NOVA_DB_USER\" WITH LOGIN PASSWORD '$NOVA_DB_PASSWORD'"
else
  runuser -u postgres -- psql -v ON_ERROR_STOP=1 -c "CREATE ROLE \"$NOVA_DB_USER\" LOGIN PASSWORD '$NOVA_DB_PASSWORD'"
fi
runuser -u postgres -- createdb -O "$NOVA_DB_USER" "$NOVA_DB_NAME"

[[ $INSTALL_DIR == /usr/local/x-ui ]] || die "安装目录安全检查失败。"
rm -rf -- "$INSTALL_DIR.new"
install -d -m 755 "$INSTALL_DIR.new"
cp -a "$tmp_dir/x-ui/." "$INSTALL_DIR.new/"
chmod 755 "$INSTALL_DIR.new/x-ui" "$INSTALL_DIR.new/bin/"* 2>/dev/null || true
rm -rf -- "$INSTALL_DIR"
mv "$INSTALL_DIR.new" "$INSTALL_DIR"
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR" "$DATA_DIR" "$LOG_DIR"

cat >/etc/default/x-ui <<EOF
XUI_DB_TYPE=postgres
XUI_DB_DSN=postgres://$NOVA_DB_USER:$NOVA_DB_PASSWORD@127.0.0.1:5432/$NOVA_DB_NAME?sslmode=disable
XUI_DB_MAX_OPEN_CONNS=40
XUI_DB_MAX_IDLE_CONNS=10
XUI_MAIN_FOLDER=$INSTALL_DIR
XUI_DB_FOLDER=$DATA_DIR
XUI_LOG_FOLDER=$LOG_DIR
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
XUI_LINE_CERT_FILE=/var/lib/x-ui/certs/fullchain.pem
XUI_LINE_KEY_FILE=/var/lib/x-ui/certs/privkey.pem
GIN_MODE=release
EOF
chown root:"$SERVICE_USER" /etc/default/x-ui
chmod 640 /etc/default/x-ui

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
AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_NET_ADMIN CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_NET_ADMIN CAP_NET_RAW
ProtectSystem=full
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ProtectHostname=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
SystemCallArchitectures=native
ReadWritePaths=/usr/local/x-ui/bin /var/lib/x-ui /var/log/x-ui
UMask=0027

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable x-ui
set -a
source /etc/default/x-ui
set +a
runuser -u "$SERVICE_USER" --preserve-environment -- "$INSTALL_DIR/x-ui" setting -username "$NOVA_ADMIN_USERNAME" -password "$NOVA_ADMIN_PASSWORD" -port "$NOVA_PANEL_PORT" -webBasePath "/" -listenIP "127.0.0.1"

PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" -v ON_ERROR_STOP=1 <<SQL
BEGIN;
DELETE FROM settings WHERE key IN ('webDomain','subEnable','subJsonEnable','subClashEnable','subListen','subPort','subPath','subJsonPath','subClashPath','subDomain','subURI','subJsonURI','subClashURI','subTitle','subProfileUrl','subSupportUrl');
INSERT INTO settings (key,value) VALUES
('webDomain','$NOVA_DOMAIN'),
('subEnable','true'),('subJsonEnable','true'),('subClashEnable','true'),
('subListen','127.0.0.1'),('subPort','$NOVA_SUB_PORT'),
('subPath','$NOVA_SUB_PATH'),('subJsonPath','$NOVA_SUB_JSON_PATH'),('subClashPath','$NOVA_SUB_CLASH_PATH'),
('subDomain','$NOVA_DOMAIN'),('subURI','https://$NOVA_DOMAIN'),('subJsonURI','https://$NOVA_DOMAIN'),('subClashURI','https://$NOVA_DOMAIN'),
('subTitle','NOVA'),('subProfileUrl','https://$NOVA_DOMAIN'),('subSupportUrl','https://$NOVA_DOMAIN/tickets');
INSERT INTO commercial_settings (key,value,encrypted,updated_at) VALUES
('site.url','https://$NOVA_DOMAIN',false,NOW()),
('site.force_https','true',false,NOW()),
('security.safe_mode','true',false,NOW()),
('security.password_attempt_limit','true',false,NOW()),
('security.max_password_attempts','5',false,NOW()),
('security.password_lock_minutes','60',false,NOW()),
('security.ip_registration_limit','true',false,NOW())
ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, encrypted=false, updated_at=NOW();
COMMIT;
SQL

cat >/etc/nginx/sites-available/nova.conf <<EOF
server_tokens off;
limit_req_zone \$binary_remote_addr zone=nova_login:10m rate=5r/m;
limit_req_zone \$binary_remote_addr zone=nova_auth:10m rate=20r/m;
limit_conn_zone \$binary_remote_addr zone=nova_conn:10m;

server {
    listen 80;
    listen [::]:80;
    server_name $NOVA_DOMAIN;
    client_max_body_size 10m;
    client_body_timeout 30s;
    client_header_timeout 15s;
    keepalive_timeout 30s;
    send_timeout 30s;
    limit_conn nova_conn 40;
    limit_req_status 429;
    location = /api/v1/passport/login {
        limit_req zone=nova_login burst=5 nodelay;
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
    }
    location = /api/v1/passport/send-code {
        limit_req zone=nova_login burst=3 nodelay;
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
    }
    location ~ ^/$NOVA_ADMIN_PATH/panel/api/commercial/applications/[^/]+/package\$ {
        client_max_body_size 1025m;
        client_body_timeout 3600s;
        proxy_request_buffering off;
        proxy_buffering off;
        proxy_next_upstream off;
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
    location ~ ^/(api/v1/passport/(register|reset-password)|$NOVA_ADMIN_PATH/login)\$ {
        limit_req zone=nova_auth burst=10 nodelay;
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
    }
    location ^~ $NOVA_SUB_PATH {
        proxy_pass http://127.0.0.1:$NOVA_SUB_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header X-Nova-Service subscription always;
        proxy_read_timeout 300s;
    }
    location ^~ $NOVA_SUB_JSON_PATH {
        proxy_pass http://127.0.0.1:$NOVA_SUB_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header X-Nova-Service subscription always;
        proxy_read_timeout 300s;
    }
    location ^~ $NOVA_SUB_CLASH_PATH {
        proxy_pass http://127.0.0.1:$NOVA_SUB_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        add_header X-Nova-Service subscription always;
        proxy_read_timeout 300s;
    }
    location ~ ^/nova-line/[2-5][0-9]{4}/[0-9a-f]{16}\$ {
        proxy_pass http://127.0.0.1:$NOVA_PANEL_PORT;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_buffering off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
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
systemctl restart x-ui
wait_http "http://127.0.0.1:$NOVA_PANEL_PORT/" "$NOVA_DOMAIN" 200 || die "根门户启动失败。"
wait_http "http://127.0.0.1:$NOVA_PANEL_PORT/$NOVA_ADMIN_PATH/" "$NOVA_DOMAIN" 200 || die "隐藏管理员入口启动失败。"

if [[ $NOVA_DEFER_TLS == false ]]; then
  log "申请并验证 Let's Encrypt 证书…"
  certbot --nginx --non-interactive --agree-tos --redirect --register-unsafely-without-email -d "$NOVA_DOMAIN"
  systemctl enable --now certbot.timer
  certbot renew --cert-name "$NOVA_DOMAIN" --dry-run --no-random-sleep-on-renew
else
  log "同域名迁移预部署模式：暂不申请证书，待数据迁移完成后再切换 DNS。"
fi

for script in update rollback backup rotate-admin-path uninstall finalize-domain sync-line-cert; do
  [[ -f $INSTALL_DIR/deploy/ubuntu/$script.sh ]] && install -m 755 "$INSTALL_DIR/deploy/ubuntu/$script.sh" "/usr/local/sbin/nova-$script"
done
for diagnostic in diagnose-active-subscription diagnose-managed-ingress; do
  [[ -f $INSTALL_DIR/deploy/ubuntu/$diagnostic.py ]] &&
    install -m 755 "$INSTALL_DIR/deploy/ubuntu/$diagnostic.py" "/usr/local/sbin/nova-$diagnostic"
done

cat >"$DEPLOY_FILE" <<EOF
NOVA_GITHUB_REPO=$NOVA_GITHUB_REPO
NOVA_RELEASE_TAG=$NOVA_RELEASE_TAG
NOVA_DOMAIN=$NOVA_DOMAIN
NOVA_ADMIN_PATH=$NOVA_ADMIN_PATH
NOVA_PANEL_PORT=$NOVA_PANEL_PORT
NOVA_SUB_PORT=$NOVA_SUB_PORT
NOVA_SUB_PATH=$NOVA_SUB_PATH
NOVA_SUB_JSON_PATH=$NOVA_SUB_JSON_PATH
NOVA_SUB_CLASH_PATH=$NOVA_SUB_CLASH_PATH
NOVA_DB_NAME=$NOVA_DB_NAME
NOVA_DB_USER=$NOVA_DB_USER
NOVA_DB_PASSWORD=$NOVA_DB_PASSWORD
NOVA_ADMIN_USERNAME=$NOVA_ADMIN_USERNAME
NOVA_WEB_BASE_PATH=/
NOVA_EXPECTED_PUBLIC_IP=$NOVA_EXPECTED_PUBLIC_IP
NOVA_PREVIOUS_DNS_IPS=$NOVA_PREVIOUS_DNS_IPS
NOVA_TLS_READY=$([[ $NOVA_DEFER_TLS == false ]] && printf true || printf false)
EOF
chmod 600 "$DEPLOY_FILE"

install -d -m 755 /etc/letsencrypt/renewal-hooks/deploy
ln -sfn /usr/local/sbin/nova-sync-line-cert /etc/letsencrypt/renewal-hooks/deploy/nova-sync-line-cert
if [[ $NOVA_DEFER_TLS == false ]]; then
  NOVA_SYNC_NO_RESTART=true /usr/local/sbin/nova-sync-line-cert
  systemctl restart x-ui
  wait_http "http://127.0.0.1:$NOVA_PANEL_PORT/" "$NOVA_DOMAIN" 200 || {
    print_xray_diagnostics
    die "证书同步后面板未在 60 秒内恢复。"
  }
fi

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

log "执行最终放行检查…"
systemctl is-active --quiet postgresql
systemctl is-active --quiet nginx
systemctl is-active --quiet x-ui
PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" -tAc 'SELECT 1' | grep -qx 1
if ! wait_xray_process 60; then
  enabled_inbounds="$(
    PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" \
      -tAc 'SELECT COUNT(*) FROM inbounds WHERE enable = true' | tr -d '[:space:]'
  )"
  if [[ $enabled_inbounds == 0 ]]; then
    # A fresh commercial installation intentionally starts without public
    # lines. The panel launches Xray as soon as the first managed line is
    # provisioned, so an idle core process is not required yet. Still validate
    # the bundled baseline config before accepting the installation.
    runuser -u "$SERVICE_USER" -- "$INSTALL_DIR/bin/xray-linux-$ARCH" run -test \
      -c "$INSTALL_DIR/bin/config.json" >/dev/null ||
      {
        print_xray_diagnostics
        die "Xray 初始配置校验失败。"
      }
    log "当前没有已启用线路；Xray 配置有效，将在线路创建后自动启动。"
  else
    print_xray_diagnostics
    die "数据库已有 $enabled_inbounds 条启用线路，但 Xray 未在 60 秒内启动。"
  fi
fi
if [[ $NOVA_DEFER_TLS == false ]]; then
  curl -fsS --max-time 15 "https://$NOVA_DOMAIN/" | grep -qi '<div id="portal"></div>' || die "HTTPS 根页面不是用户门户。"
  curl -fsS --max-time 15 "https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/" >/dev/null || die "HTTPS 管理后台不可用。"
  curl -fsS --max-time 15 "https://$NOVA_DOMAIN/api/v1/guest/auth-config" | jq -e '.success == true and (.obj.site.siteName | length > 0)' >/dev/null
  status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "https://$NOVA_DOMAIN/api/v1/user/bootstrap")"
  [[ $status == 401 ]] || die "未登录用户 bootstrap 未返回 401（HTTP $status）。"
  status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "https://$NOVA_DOMAIN/portal/")"
  [[ $status == 308 ]] || die "/portal/ 未永久重定向到根页面（HTTP $status）。"
  status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "https://$NOVA_DOMAIN/panel/")"
  [[ $status == 404 ]] || die "公开 /panel/ 未返回 404（HTTP $status）。"
  if [[ $NOVA_ADMIN_PATH != 123456789012345678 ]]; then
    status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "https://$NOVA_DOMAIN/123456789012345678/")"
    [[ $status == 404 ]] || die "错误管理员路径未返回 404（HTTP $status）。"
  fi
  status="$(curl -sS -D "$tmp_dir/sub-health.headers" -o /dev/null -w '%{http_code}' --max-time 15 "https://$NOVA_DOMAIN${NOVA_SUB_PATH}health-probe")"
  [[ $status != 000 && $status -lt 500 ]] || die "订阅服务健康检查失败（HTTP $status）。"
  tr -d '\r' <"$tmp_dir/sub-health.headers" | grep -qi '^X-Nova-Service: subscription$' || die "订阅路径未进入独立订阅服务。"
fi
ss -H -ltn | awk '{print $4}' | grep -Eq "127\.0\.0\.1:$NOVA_PANEL_PORT$" || die "面板未仅监听本机回环地址。"
ss -H -ltn | awk '{print $4}' | grep -Eq "127\.0\.0\.1:$NOVA_SUB_PORT$" || die "订阅服务未仅监听本机回环地址。"
if find /etc/x-ui "$DATA_DIR" -maxdepth 1 -type f -name '*.db' -print -quit 2>/dev/null | grep -q .; then
  die "生产安装后检测到 SQLite 文件。"
fi
if [[ $NOVA_DEFER_TLS == false ]]; then
  systemctl is-enabled --quiet certbot.timer
fi

cat >/root/nova-install-result.txt <<EOF
网站地址: https://$NOVA_DOMAIN/
管理员后台: https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/
管理员账号: $NOVA_ADMIN_USERNAME
已安装版本: $NOVA_RELEASE_TAG
更新命令: sudo nova-update
回滚命令: sudo nova-rollback
备份命令: sudo nova-backup
轮换后台入口: sudo nova-rotate-admin-path
服务日志: sudo journalctl -u x-ui -f
手工放行线路端口: TCP 与 UDP 20000-59999（安装器未修改 UFW 或云安全组）
EOF
if [[ $NOVA_DEFER_TLS == true ]]; then
  cat >>/root/nova-install-result.txt <<EOF
迁移目标 IP: $NOVA_EXPECTED_PUBLIC_IP
迁移最后一步: 在新服务器运行 sudo nova-finalize-domain，随后按提示切换 DNS
EOF
fi
chmod 600 /root/nova-install-result.txt
unset NOVA_ADMIN_PASSWORD NOVA_ADMIN_PASSWORD_CONFIRM
log "安装和全部健康检查完成。结果已保存到 /root/nova-install-result.txt。"
cat /root/nova-install-result.txt
