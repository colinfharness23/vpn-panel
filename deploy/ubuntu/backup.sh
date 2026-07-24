#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "请使用 root 执行。" >&2; exit 1; }
# shellcheck disable=SC1091
source /etc/nova/deploy.env
valid_admin_path() {
  local value="$1"
  [[ $value =~ ^[0-9]{18}$ ]] ||
    [[ $value =~ ^[A-Za-z0-9_-]{40}$ &&
       $value =~ [A-Z] && $value =~ [a-z] && $value =~ [0-9] &&
       $value =~ [_-] ]]
}
valid_admin_path "$NOVA_ADMIN_PATH" || { echo "部署配置中的管理员入口无效。" >&2; exit 1; }

backup_dir="/var/backups/nova/database"
install -d -m 700 "$backup_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
db_target="$backup_dir/$NOVA_DB_NAME-$stamp.dump"
config_target="$backup_dir/config-$stamp.tar.gz"
uploads_target="$backup_dir/uploads-$stamp.tar.gz"
manifest_target="$backup_dir/SHA256SUMS-$stamp"

PGPASSWORD="$NOVA_DB_PASSWORD" pg_dump \
  --host 127.0.0.1 --username "$NOVA_DB_USER" --dbname "$NOVA_DB_NAME" \
  --format custom --compress 9 --file "$db_target"
pg_restore --list "$db_target" >/dev/null

config_files=(etc/nova/deploy.env etc/default/x-ui)
[[ -f /etc/nginx/sites-available/nova.conf ]] && config_files+=(etc/nginx/sites-available/nova.conf)
tar -czf "$config_target" -C / "${config_files[@]}"
if [[ -d /var/lib/x-ui/uploads ]]; then
  tar -czf "$uploads_target" -C /var/lib/x-ui uploads
else
  tar -czf "$uploads_target" --files-from /dev/null
fi
tar -tzf "$config_target" >/dev/null
tar -tzf "$uploads_target" >/dev/null
sha256sum "$db_target" "$config_target" "$uploads_target" >"$manifest_target"
sha256sum --check "$manifest_target" >/dev/null
chmod 600 "$db_target" "$config_target" "$uploads_target" "$manifest_target"
find "$backup_dir" -type f \( -name '*.dump' -o -name 'config-*.tar.gz' -o -name 'uploads-*.tar.gz' -o -name 'SHA256SUMS-*' \) -mtime +14 -delete
echo "数据库备份完成：$db_target"
echo "部署配置备份完成：$config_target"
echo "安装包备份完成：$uploads_target"
