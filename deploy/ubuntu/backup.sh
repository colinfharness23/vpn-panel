#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "请使用 root 执行。" >&2; exit 1; }
# shellcheck disable=SC1091
source /etc/nova/deploy.env

backup_dir="/var/backups/nova/database"
install -d -m 700 "$backup_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
db_target="$backup_dir/$NOVA_DB_NAME-$stamp.dump"
config_target="$backup_dir/config-$stamp.tar.gz"

PGPASSWORD="$NOVA_DB_PASSWORD" pg_dump \
  --host 127.0.0.1 --username "$NOVA_DB_USER" --dbname "$NOVA_DB_NAME" \
  --format custom --compress 9 --file "$db_target"

config_files=(etc/nova/deploy.env etc/default/x-ui)
[[ -f /etc/nginx/sites-available/nova.conf ]] && config_files+=(etc/nginx/sites-available/nova.conf)
tar -czf "$config_target" -C / "${config_files[@]}"
chmod 600 "$db_target" "$config_target"
find "$backup_dir" -type f \( -name '*.dump' -o -name 'config-*.tar.gz' \) -mtime +14 -delete
echo "备份完成：$db_target"
echo "部署配置备份：$config_target"
