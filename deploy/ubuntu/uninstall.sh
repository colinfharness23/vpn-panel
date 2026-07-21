#!/usr/bin/env bash
set -Eeuo pipefail

readonly INSTALL_DIR="/usr/local/x-ui"
readonly CONFIG_DIR="/etc/nova"
readonly DEPLOY_FILE="/etc/nova/deploy.env"
readonly DATA_DIR="/var/lib/x-ui"
readonly LOG_DIR="/var/log/x-ui"
readonly BACKUP_ROOT="/var/backups/nova"
readonly SERVICE_USER="nova"

log() { printf '[NOVA] %s\n' "$*"; }
die() { printf '[NOVA] 卸载失败：%s\n' "$*" >&2; exit 1; }

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

[[ $EUID -eq 0 ]] || die "请使用 root 执行，或在命令前使用 sudo。"

installed=0
for path in "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR" /etc/default/x-ui /etc/systemd/system/x-ui.service /etc/nginx/sites-available/nova.conf; do
  if [[ -e $path || -L $path ]]; then
    installed=1
    break
  fi
done
if ((installed == 0)); then
  log "没有检测到 NOVA 面板安装，无需卸载。"
  exit 0
fi

domain="$(existing_env_value "$DEPLOY_FILE" NOVA_DOMAIN)"
db_name="$(existing_env_value "$DEPLOY_FILE" NOVA_DB_NAME)"
db_user="$(existing_env_value "$DEPLOY_FILE" NOVA_DB_USER)"
db_name="${db_name:-nova}"
db_user="${db_user:-nova}"
[[ $db_name =~ ^[A-Za-z][A-Za-z0-9_]*$ ]] || die "部署配置中的数据库名称无效，已拒绝卸载。"
[[ $db_user =~ ^[A-Za-z][A-Za-z0-9_]*$ && $db_user != postgres ]] || die "部署配置中的数据库用户无效，已拒绝卸载。"
case "$db_name" in
  postgres|template0|template1) die "部署配置指向 PostgreSQL 系统数据库，已拒绝卸载。" ;;
esac

printf '\n即将卸载 NOVA 面板并删除当前 PostgreSQL 数据库“%s”。\n' "$db_name" >/dev/tty
printf '卸载前会自动备份到 %s；证书、Nginx/PostgreSQL 软件包和历史备份不会删除。\n' "$BACKUP_ROOT" >/dev/tty
if [[ ${NOVA_UNINSTALL_CONFIRM:-} == UNINSTALL ]]; then
  confirmation=UNINSTALL
else
  printf '请输入 UNINSTALL 确认：' >/dev/tty
  IFS= read -r confirmation </dev/tty
fi
[[ $confirmation == UNINSTALL ]] || die "确认内容不正确，已取消。"

backup_dir="$BACKUP_ROOT/uninstall-$(date -u +%Y%m%dT%H%M%SZ)"
install -d -m 700 "$backup_dir"
log "正在创建卸载前备份：$backup_dir"

backup_paths=()
for path in "$DEPLOY_FILE" /etc/default/x-ui /etc/systemd/system/x-ui.service /etc/systemd/system/nova-backup.service /etc/systemd/system/nova-backup.timer /etc/nginx/sites-available/nova.conf /root/nova-install-result.txt /etc/x-ui; do
  [[ -e $path || -L $path ]] && backup_paths+=("$path")
done
if ((${#backup_paths[@]} > 0)); then
  tar -czf "$backup_dir/configuration.tar.gz" "${backup_paths[@]}"
  tar -tzf "$backup_dir/configuration.tar.gz" >/dev/null
fi
if [[ -d $INSTALL_DIR ]]; then
  tar -czf "$backup_dir/program.tar.gz" -C "$(dirname "$INSTALL_DIR")" "$(basename "$INSTALL_DIR")"
  tar -tzf "$backup_dir/program.tar.gz" >/dev/null
fi
if [[ -d $DATA_DIR ]]; then
  tar -czf "$backup_dir/data.tar.gz" -C "$(dirname "$DATA_DIR")" "$(basename "$DATA_DIR")"
  tar -tzf "$backup_dir/data.tar.gz" >/dev/null
fi
if command -v runuser >/dev/null 2>&1 && id -u postgres >/dev/null 2>&1; then
  if runuser -u postgres -- psql -tAc "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -qx 1; then
    runuser -u postgres -- pg_dump --format=custom "$db_name" >"$backup_dir/postgresql.dump"
    pg_restore --list "$backup_dir/postgresql.dump" >/dev/null
  fi
fi
find "$backup_dir" -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum >"$backup_dir/SHA256SUMS"
[[ -s $backup_dir/SHA256SUMS ]] || die "卸载前备份为空，已拒绝继续。"
(cd "$backup_dir" && sha256sum --check SHA256SUMS >/dev/null)
log "备份已校验。"

systemctl disable --now nova-backup.timer 2>/dev/null || true
systemctl stop x-ui 2>/dev/null || true
systemctl disable x-ui 2>/dev/null || true

if command -v runuser >/dev/null 2>&1 && id -u postgres >/dev/null 2>&1; then
  runuser -u postgres -- dropdb --if-exists --force "$db_name"
  runuser -u postgres -- dropuser --if-exists "$db_user"
fi

rm -f -- /etc/systemd/system/x-ui.service /etc/systemd/system/nova-backup.service /etc/systemd/system/nova-backup.timer
rm -f -- /etc/nginx/sites-enabled/nova.conf /etc/nginx/sites-available/nova.conf
rm -f -- /etc/default/x-ui /root/nova-install-result.txt
rm -f -- /usr/local/sbin/nova-update /usr/local/sbin/nova-rollback /usr/local/sbin/nova-backup /usr/local/sbin/nova-rotate-admin-path /usr/local/sbin/nova-uninstall

[[ $INSTALL_DIR == /usr/local/x-ui ]] || die "程序目录安全检查失败。"
[[ $CONFIG_DIR == /etc/nova ]] || die "配置目录安全检查失败。"
[[ $DATA_DIR == /var/lib/x-ui ]] || die "数据目录安全检查失败。"
[[ $LOG_DIR == /var/log/x-ui ]] || die "日志目录安全检查失败。"
rm -rf -- "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR" /etc/x-ui

if id -u "$SERVICE_USER" >/dev/null 2>&1; then
  userdel "$SERVICE_USER" 2>/dev/null || true
fi
if [[ -f /etc/nginx/sites-available/default && ! -e /etc/nginx/sites-enabled/default ]]; then
  ln -s /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default
fi
systemctl daemon-reload
systemctl reset-failed x-ui 2>/dev/null || true
if command -v nginx >/dev/null 2>&1 && nginx -t; then
  systemctl reload nginx 2>/dev/null || true
fi

log "卸载完成。可恢复备份：$backup_dir"
[[ -z $domain ]] || log "域名 $domain 的证书仍保留在 /etc/letsencrypt。"
log "现在可以执行最新版一键安装命令。"
