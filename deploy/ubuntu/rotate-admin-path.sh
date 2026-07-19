#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "请使用 root 执行。" >&2; exit 1; }
# shellcheck disable=SC1091
source /etc/nova/deploy.env

old_path=""
revert_armed=0
restore_old_path() {
  local failed_status="$1"
  trap - ERR
  set +e
  if (( revert_armed == 1 )) && [[ $old_path =~ ^[0-9]{18}$ ]]; then
    sed -i "s|^NOVA_ADMIN_PATH=.*$|NOVA_ADMIN_PATH=$old_path|" /etc/nova/deploy.env
    sed -i "s|^XUI_ADMIN_BASE_PATH=.*$|XUI_ADMIN_BASE_PATH=/$old_path/|" /etc/default/x-ui
    systemctl restart x-ui
  fi
  echo "新入口健康检查失败，已恢复原入口。" >&2
  exit "$failed_status"
}
trap 'restore_old_path "$?"' ERR

generate_path() {
  local result="" byte
  for _ in $(seq 1 18); do
    byte="$(od -An -N1 -tu1 /dev/urandom | tr -d ' ')"
    result="$result$((byte % 10))"
  done
  printf '%s' "$result"
}

old_path="$NOVA_ADMIN_PATH"
new_path="$(generate_path)"
while [[ $new_path == "$old_path" ]]; do new_path="$(generate_path)"; done

revert_armed=1
sed -i "s|^NOVA_ADMIN_PATH=.*$|NOVA_ADMIN_PATH=$new_path|" /etc/nova/deploy.env
sed -i "s|^XUI_ADMIN_BASE_PATH=.*$|XUI_ADMIN_BASE_PATH=/$new_path/|" /etc/default/x-ui
systemctl restart x-ui

healthy=0
for _ in $(seq 1 30); do
  if curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/$new_path/" >/dev/null; then
    healthy=1
    break
  fi
  sleep 1
done

if (( healthy == 0 )); then
  false
fi

status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/$old_path/")"
[[ $status == 404 ]]

if [[ -f /root/nova-install-result.txt ]]; then
  sed -i "s|^管理员后台:.*$|管理员后台: https://$NOVA_DOMAIN/$new_path/|" /root/nova-install-result.txt
fi
revert_armed=0
echo "管理员入口已轮换：https://$NOVA_DOMAIN/$new_path/"
