#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "请使用 root 执行。" >&2; exit 1; }
# shellcheck disable=SC1091
source /etc/nova/deploy.env
[[ $NOVA_ADMIN_PATH =~ ^[0-9]{18}$ ]] || { echo "部署配置中的管理员入口无效。" >&2; exit 1; }

current=""
recovery_armed=0
recover_current() {
  local failed_status="$1"
  trap - ERR
  set +e
  if (( recovery_armed == 1 )) && [[ -n $current && -f $current ]]; then
    echo "回滚失败，正在恢复回滚前运行版本。" >&2
    systemctl stop x-ui
    rm -rf -- /usr/local/x-ui
    tar -xzf "$current" -C /usr/local
    chown -R nova:nova /usr/local/x-ui
    sed -i "s|^NOVA_RELEASE_TAG=.*$|NOVA_RELEASE_TAG=$NOVA_RELEASE_TAG|" /etc/nova/deploy.env
    systemctl restart x-ui
    curl -fsS --max-time 10 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" >/dev/null ||
      journalctl -u x-ui -n 80 --no-pager >&2
  fi
  exit "$failed_status"
}
trap 'recover_current "$?"' ERR

backup="${1:-}"
if [[ -z $backup ]]; then
  backup="$(find /var/backups/nova/releases -maxdepth 1 -type f -name 'x-ui-*.tar.gz' -printf '%T@ %p\n' | sort -nr | head -n1 | cut -d' ' -f2-)"
fi
[[ -n $backup && -f $backup ]] || { echo "没有找到可回滚版本。" >&2; exit 1; }
tar -tzf "$backup" | grep -Eq '(^/|(^|/)\.\.(/|$))' && { echo "备份包包含不安全路径。" >&2; exit 1; }
tar -tzf "$backup" | grep -Ev '^x-ui(/|$)' | grep -q . && { echo "备份包含非 x-ui 路径。" >&2; exit 1; }

/usr/local/sbin/nova-backup
systemctl stop x-ui
current="/var/backups/nova/releases/x-ui-before-rollback-$(date -u +%Y%m%dT%H%M%SZ).tar.gz"
tar -czf "$current" -C /usr/local x-ui
printf '%s\n' "$NOVA_RELEASE_TAG" >"$current.tag"
chmod 600 "$current" "$current.tag"
recovery_armed=1
rm -rf -- /usr/local/x-ui
tar -xzf "$backup" -C /usr/local
chown -R nova:nova /usr/local/x-ui
systemctl restart x-ui

for _ in $(seq 1 40); do
  if curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/$NOVA_ADMIN_PATH/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/api/v1/guest/bootstrap" | jq -e '.success == true' >/dev/null &&
     pgrep -u nova -f '/usr/local/x-ui/bin/xray-linux-' >/dev/null; then
    if [[ -f $backup.tag ]]; then
      rollback_tag="$(tr -d '\r\n' <"$backup.tag")"
      [[ $rollback_tag =~ ^[A-Za-z0-9._-]+$ ]] &&
        sed -i "s|^NOVA_RELEASE_TAG=.*$|NOVA_RELEASE_TAG=$rollback_tag|" /etc/nova/deploy.env
    fi
    status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/panel/")"
    [[ $status == 404 ]]
    recovery_armed=0
    echo "回滚完成：$backup"
    exit 0
  fi
  sleep 1
done

echo "回滚后健康检查失败，将恢复回滚前版本。" >&2
false
