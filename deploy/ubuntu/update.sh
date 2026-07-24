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
[[ $NOVA_GITHUB_REPO =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || { echo "部署配置中的 GitHub 仓库无效。" >&2; exit 1; }

tmp_dir=""
backup=""
nginx_backup=""
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
    if [[ -n $nginx_backup && -f $nginx_backup ]]; then
      cp -a "$nginx_backup" /etc/nginx/sites-available/nova.conf
      nginx -t && systemctl reload nginx
    fi
    curl -fsS --max-time 10 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/" >/dev/null ||
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
expected_sha="$(awk -v name="$asset" '$2 == name || $2 == "*"name {print $1; exit}' "$tmp_dir/$asset.sha256")"
[[ $expected_sha =~ ^[a-fA-F0-9]{64}$ ]] || { echo "Release SHA-256 文件格式无效。" >&2; exit 1; }
actual_sha="$(sha256sum "$tmp_dir/$asset" | awk '{print $1}')"
[[ ${actual_sha,,} == "${expected_sha,,}" ]] || { echo "Release SHA-256 校验失败。" >&2; exit 1; }
tar -tzf "$tmp_dir/$asset" | awk '$0 ~ /(^\/|(^|\/)\.\.(\/|$))/ { found=1 } END { exit !found }' && { echo "安装包包含不安全路径。" >&2; exit 1; }
tar -tzf "$tmp_dir/$asset" | awk '$0 !~ /^x-ui(\/|$)/ { found=1 } END { exit !found }' && { echo "安装包包含 x-ui 目录之外的文件。" >&2; exit 1; }
tar -tvzf "$tmp_dir/$asset" | awk 'substr($1,1,1) !~ /^[-d]$/ { found=1 } END { exit !found }' && { echo "安装包包含符号链接、硬链接或设备文件。" >&2; exit 1; }
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"
[[ -x $tmp_dir/x-ui/x-ui && -f $tmp_dir/x-ui/bin/config.json && -x $tmp_dir/x-ui/bin/xray-linux-$arch && -x $tmp_dir/x-ui/bin/sing-box-linux-$arch ]] ||
  { echo "Release 缺少面板、Xray、AnyTLS 运行时或初始配置。" >&2; exit 1; }
for required_script in install update rollback backup rotate-admin-path uninstall finalize-domain sync-line-cert; do
  [[ -f $tmp_dir/x-ui/deploy/ubuntu/$required_script.sh ]] || { echo "Release 缺少运维脚本 $required_script.sh。" >&2; exit 1; }
done
for required_diagnostic in diagnose-active-subscription diagnose-managed-ingress; do
  [[ -f $tmp_dir/x-ui/deploy/ubuntu/$required_diagnostic.py ]] || { echo "Release 缺少诊断脚本 $required_diagnostic.py。" >&2; exit 1; }
done

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
install -d -m 700 /var/backups/nova/releases
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
install -m 755 "/usr/local/x-ui/deploy/ubuntu/sync-line-cert.sh" /usr/local/sbin/nova-sync-line-cert
install -d -m 755 /etc/letsencrypt/renewal-hooks/deploy
ln -sfn /usr/local/sbin/nova-sync-line-cert /etc/letsencrypt/renewal-hooks/deploy/nova-sync-line-cert
if [[ -r /etc/letsencrypt/live/$NOVA_DOMAIN/fullchain.pem && -r /etc/letsencrypt/live/$NOVA_DOMAIN/privkey.pem ]]; then
  NOVA_SYNC_NO_RESTART=true /usr/local/sbin/nova-sync-line-cert
fi

if ! grep -q '/nova-line/' /etc/nginx/sites-available/nova.conf; then
  nginx_backup="$tmp_dir/nova.conf.before-line-ingress"
  cp -a /etc/nginx/sites-available/nova.conf "$nginx_backup"
  line_location="$tmp_dir/nova-line-location.conf"
  cat >"$line_location" <<EOF
    location ~ "^/nova-line/[2-5][0-9]{4}/[0-9a-f]{16}\$" {
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
EOF
  updated_nginx="$tmp_dir/nova.conf.with-line-ingress"
  awk -v snippet="$line_location" '
    !inserted && /^[[:space:]]*location \/[[:space:]]*\{/ {
      while ((getline line < snippet) > 0) print line
      close(snippet)
      inserted=1
    }
    { print }
    END { if (!inserted) exit 42 }
  ' /etc/nginx/sites-available/nova.conf >"$updated_nginx"
  install -m 644 "$updated_nginx" /etc/nginx/sites-available/nova.conf
  nginx -t
  systemctl reload nginx
fi
if ! grep -q '/_nova_internal/client-applications/' /etc/nginx/sites-available/nova.conf; then
  if [[ -z $nginx_backup ]]; then
    nginx_backup="$tmp_dir/nova.conf.before-download-acceleration"
    cp -a /etc/nginx/sites-available/nova.conf "$nginx_backup"
  fi
  download_location="$tmp_dir/nova-download-location.conf"
  cat >"$download_location" <<'EOF'
    location ^~ /_nova_internal/client-applications/ {
        internal;
        alias /var/lib/x-ui/uploads/client-applications/;
        sendfile on;
        tcp_nopush on;
        limit_rate 0;
        open_file_cache max=100 inactive=60s;
        open_file_cache_valid 60s;
        open_file_cache_min_uses 1;
        open_file_cache_errors off;
    }
EOF
  updated_nginx="$tmp_dir/nova.conf.with-download-acceleration"
  awk -v snippet="$download_location" '
    !inserted && /^[[:space:]]*location \/[[:space:]]*\{/ {
      while ((getline line < snippet) > 0) print line
      close(snippet)
      inserted=1
    }
    { print }
    END { if (!inserted) exit 42 }
  ' /etc/nginx/sites-available/nova.conf >"$updated_nginx"
  install -m 644 "$updated_nginx" /etc/nginx/sites-available/nova.conf
  nginx -t
  systemctl reload nginx
fi
if ! grep -q '^XUI_APPLICATION_ACCEL_PREFIX=' /etc/default/x-ui; then
  printf '%s\n' 'XUI_APPLICATION_ACCEL_PREFIX=/_nova_internal/client-applications/' >>/etc/default/x-ui
fi
systemctl restart x-ui

healthy=0
for _ in $(seq 1 40); do
  user_bootstrap_status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/api/v1/user/bootstrap" || true)"
  portal_status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/portal/" || true)"
  subscription_status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_SUB_PORT${NOVA_SUB_PATH}health-probe" || true)"
  enabled_inbounds="$(
    PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" \
      -tAc 'SELECT COUNT(*) FROM inbounds WHERE enable = true' 2>/dev/null | tr -d '[:space:]'
  )"
  xray_ready=0
  if pgrep -u nova -f "/usr/local/x-ui/bin/xray-linux-$arch" >/dev/null; then
    xray_ready=1
  elif [[ $enabled_inbounds == 0 ]] &&
       runuser -u nova -- "/usr/local/x-ui/bin/xray-linux-$arch" run -test \
         -c /usr/local/x-ui/bin/config.json >/dev/null 2>&1; then
    xray_ready=1
  fi
  if curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/$NOVA_ADMIN_PATH/" >/dev/null &&
     curl -fsS --max-time 3 -H "Host: $NOVA_DOMAIN" "http://127.0.0.1:$NOVA_PANEL_PORT/api/v1/guest/auth-config" | jq -e '.success == true' >/dev/null &&
     [[ $user_bootstrap_status == 401 && $portal_status == 308 && $subscription_status != 000 && $subscription_status -lt 500 ]] &&
     PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" -tAc 'SELECT 1' | grep -qx 1 &&
     (( xray_ready == 1 )); then
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
if find /etc/x-ui /var/lib/x-ui -maxdepth 1 -type f -name '*.db' -print -quit 2>/dev/null | grep -q .; then
  echo "更新后检测到 SQLite 文件。" >&2
  false
fi

sed -i "s|^NOVA_RELEASE_TAG=.*$|NOVA_RELEASE_TAG=$requested_tag|" /etc/nova/deploy.env
for script in update rollback backup rotate-admin-path uninstall finalize-domain sync-line-cert; do
  [[ -f /usr/local/x-ui/deploy/ubuntu/$script.sh ]] &&
    install -m 755 "/usr/local/x-ui/deploy/ubuntu/$script.sh" "/usr/local/sbin/nova-$script"
done
for diagnostic in diagnose-active-subscription diagnose-managed-ingress; do
  [[ -f /usr/local/x-ui/deploy/ubuntu/$diagnostic.py ]] &&
    install -m 755 "/usr/local/x-ui/deploy/ubuntu/$diagnostic.py" "/usr/local/sbin/nova-$diagnostic"
done
if [[ $NOVA_ADMIN_PATH =~ ^[0-9]{18}$ ]]; then
  echo "检测到旧版 18 位数字管理员入口，正在自动轮换为高强度随机入口。"
  /usr/local/sbin/nova-rotate-admin-path
  # shellcheck disable=SC1091
  source /etc/nova/deploy.env
fi
rollback_armed=0
echo "管理员后台：https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/"
echo "更新完成：$NOVA_RELEASE_TAG -> $requested_tag"
