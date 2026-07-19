#!/usr/bin/env bash
set -Eeuo pipefail

[[ $EUID -eq 0 ]] || { echo "请使用 root 执行。" >&2; exit 1; }
# shellcheck disable=SC1091
source /etc/nova/deploy.env
[[ $NOVA_ADMIN_PATH =~ ^[0-9]{18}$ ]] || { echo "部署配置中的管理员入口无效。" >&2; exit 1; }

tmp_dir=""
backup=""
rollback_armed=0
cleanup() { [[ -z $tmp_dir || ! -d $tmp_dir ]] || rm -rf -- "$tmp_dir"; }
restore_previous() {
  local failed_status="$1"
  trap - ERR
  set +e
  if (( rollback_armed == 1 )) && [[ -n $backup && -f $backup ]]; then
    echo "更新失败，正在自动恢复 $NOVA_RELEASE_TAG。" >&2
    systemctl stop x-ui
    rm -rf -- /usr/local/x-ui
    tar -xzf "$backup" -C /usr/local
    chown -R nova:nova /usr/local/x-ui
    sed -i "s|^NOVA_RELEASE_TAG=.*$|NOVA_RELEASE_TAG=$NOVA_RELEASE_TAG|" /etc/nova/deploy.env
    systemctl restart x-ui
    curl -fsS --max-time 10 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" >/dev/null ||
      journalctl -u x-ui -n 80 --no-pager >&2
  fi
  exit "$failed_status"
}
trap 'restore_previous "$?"' ERR
trap cleanup EXIT

requested_tag="${1:-latest}"
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "不支持当前 CPU 架构。" >&2; exit 1 ;;
esac
if [[ $requested_tag == latest ]]; then
  requested_tag="$(curl -fsSL --retry 4 "https://api.github.com/repos/$NOVA_GITHUB_REPO/releases/latest" | jq -r '.tag_name // empty')"
fi
[[ $requested_tag =~ ^[A-Za-z0-9._-]+$ ]] || { echo "Release 标签无效。" >&2; exit 1; }
[[ $requested_tag != "$NOVA_RELEASE_TAG" ]] || { echo "当前已经是 $requested_tag。"; exit 0; }

/usr/local/sbin/nova-backup
tmp_dir="$(mktemp -d)"
asset="x-ui-linux-$arch.tar.gz"
asset_url="https://github.com/$NOVA_GITHUB_REPO/releases/download/$requested_tag/$asset"
curl -fL --retry 5 --retry-delay 3 --max-time 600 -o "$tmp_dir/$asset" "$asset_url"
curl -fL --retry 5 --retry-delay 3 --max-time 60 -o "$tmp_dir/$asset.sha256" "$asset_url.sha256"
(cd "$tmp_dir" && sha256sum --check "$asset.sha256")
tar -tzf "$tmp_dir/$asset" | grep -Eq '(^/|(^|/)\.\.(/|$))' && { echo "安装包包含不安全路径。" >&2; exit 1; }
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
[[ -x $tmp_dir/x-ui/x-ui && -f $tmp_dir/x-ui/bin/config.json && -x $tmp_dir/x-ui/bin/xray-linux-$arch ]] ||
  { echo "Release 缺少面板、Xray 或初始配置。" >&2; exit 1; }

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="/var/backups/nova/releases/x-ui-$NOVA_RELEASE_TAG-$stamp.tar.gz"
tar -czf "$backup" -C /usr/local x-ui
printf '%s\n' "$NOVA_RELEASE_TAG" >"$backup.tag"
chmod 600 "$backup" "$backup.tag"

rollback_armed=1
systemctl stop x-ui
rm -rf -- /usr/local/x-ui.new
install -d -m 755 /usr/local/x-ui.new
cp -a "$tmp_dir/x-ui/." /usr/local/x-ui.new/
chmod 755 /usr/local/x-ui.new/x-ui /usr/local/x-ui.new/bin/* 2>/dev/null || true
rm -rf -- /usr/local/x-ui
mv /usr/local/x-ui.new /usr/local/x-ui
chown -R nova:nova /usr/local/x-ui
systemctl restart x-ui

healthy=0
for _ in $(seq 1 40); do
  if curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/$NOVA_ADMIN_PATH/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/api/v1/guest/bootstrap" | jq -e '.success == true' >/dev/null &&
     pgrep -u nova -f "/usr/local/x-ui/bin/xray-linux-$arch" >/dev/null; then
    healthy=1
    break
  fi
  sleep 1
done
if (( healthy == 0 )); then
  echo "新版本健康检查失败。" >&2
  false
fi

status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/panel/")"
[[ $status == 404 ]]

sed -i "s|^NOVA_RELEASE_TAG=.*$|NOVA_RELEASE_TAG=$requested_tag|" /etc/nova/deploy.env
for script in update rollback backup rotate-admin-path; do
  [[ -f /usr/local/x-ui/deploy/ubuntu/$script.sh ]] &&
    install -m 755 "/usr/local/x-ui/deploy/ubuntu/$script.sh" "/usr/local/sbin/nova-$script"
done
rollback_armed=0
echo "更新完成：$NOVA_RELEASE_TAG -> $requested_tag"
