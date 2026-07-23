#!/usr/bin/env bash
set -Eeuo pipefail

readonly DEPLOY_FILE="/etc/nova/deploy.env"
readonly RESULT_FILE="/root/nova-install-result.txt"
headers_file=""

log() { printf '[NOVA] %s\n' "$*"; }
die() { printf '[NOVA] 域名切换失败：%s\n' "$*" >&2; exit 1; }
cleanup() { [[ -z $headers_file || ! -f $headers_file ]] || rm -f -- "$headers_file"; }
trap cleanup EXIT

[[ $EUID -eq 0 ]] || die "请使用 root 执行，或在命令前使用 sudo。"
[[ -f $DEPLOY_FILE ]] || die "没有找到 $DEPLOY_FILE。"
# shellcheck disable=SC1091
source "$DEPLOY_FILE"

[[ ${NOVA_DOMAIN:-} =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$ ]] || die "部署域名无效。"
[[ ${NOVA_EXPECTED_PUBLIC_IP:-} != "" ]] || die "缺少迁移目标公网 IP，请重新执行一键迁移。"
python3 - "$NOVA_EXPECTED_PUBLIC_IP" <<'PY' || die "迁移目标公网 IP 无效。"
import ipaddress
import sys
ipaddress.ip_address(sys.argv[1])
PY

if [[ ${NOVA_TLS_READY:-false} == true ]]; then
  log "HTTPS 已经完成配置，无需重复执行。"
  exit 0
fi

log "正在等待 $NOVA_DOMAIN 的 DNS 解析到本机迁移地址 $NOVA_EXPECTED_PUBLIC_IP。"
log "现在可以在域名服务商处把原 A/AAAA 记录从旧服务器改为新服务器；不要同时保留两个源站 IP。"
dns_ready=0
for _ in $(seq 1 180); do
  mapfile -t resolved_ips < <(getent ahosts "$NOVA_DOMAIN" 2>/dev/null | awk '{print $1}' | sort -u)
  target_seen=0
  old_seen=0
  IFS=',' read -r -a previous_ips <<<"${NOVA_PREVIOUS_DNS_IPS:-}"
  for resolved_ip in "${resolved_ips[@]}"; do
    [[ $resolved_ip == "$NOVA_EXPECTED_PUBLIC_IP" ]] && target_seen=1
    for previous_ip in "${previous_ips[@]}"; do
      [[ -n $previous_ip && $previous_ip != "$NOVA_EXPECTED_PUBLIC_IP" && $resolved_ip == "$previous_ip" ]] && old_seen=1
    done
  done
  if ((target_seen == 1 && old_seen == 0)); then
    dns_ready=1
    break
  fi
  sleep 10
done
((dns_ready == 1)) || die "30 分钟内未检测到 DNS 指向 $NOVA_EXPECTED_PUBLIC_IP。旧服务器仍可继续使用，请检查 A/AAAA 记录后重试。"

systemctl is-active --quiet nginx || die "Nginx 未运行。"
systemctl is-active --quiet x-ui || die "面板服务未运行。"
nginx -t
log "DNS 已切换，正在自动申请 Let's Encrypt 证书并启用 HTTPS。"
certbot --nginx --non-interactive --agree-tos --redirect --register-unsafely-without-email -d "$NOVA_DOMAIN"
systemctl enable --now certbot.timer
certbot renew --cert-name "$NOVA_DOMAIN" --dry-run --no-random-sleep-on-renew
/usr/local/sbin/nova-sync-line-cert

xray_ready=0
for _ in $(seq 1 60); do
  if pgrep -u nova -f '/usr/local/x-ui/bin/xray-linux-' >/dev/null; then
    xray_ready=1
    break
  fi
  sleep 1
done
if ((xray_ready == 0)); then
  systemctl status x-ui --no-pager -l >&2 2>/dev/null || true
  journalctl -u x-ui -n 160 --no-pager >&2 2>/dev/null || true
  die "证书同步后 Xray 未在 60 秒内恢复。"
fi

curl -fsS --max-time 20 "https://$NOVA_DOMAIN/" | grep -qi '<div id="portal"></div>' || die "HTTPS 根门户验证失败。"
curl -fsS --max-time 20 "https://$NOVA_DOMAIN/$NOVA_ADMIN_PATH/" >/dev/null || die "HTTPS 管理后台验证失败。"
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 20 "https://$NOVA_DOMAIN/api/v1/user/bootstrap")"
[[ $status == 401 ]] || die "未登录用户接口验证失败（HTTP $status）。"
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 20 "https://$NOVA_DOMAIN/portal/")"
[[ $status == 308 ]] || die "/portal/ 重定向验证失败（HTTP $status）。"
status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 20 "https://$NOVA_DOMAIN/panel/")"
[[ $status == 404 ]] || die "公开后台路径未隐藏（HTTP $status）。"
headers_file="$(mktemp /tmp/nova-finalize.XXXXXX)"
status="$(curl -sS -D "$headers_file" -o /dev/null -w '%{http_code}' --max-time 20 "https://$NOVA_DOMAIN${NOVA_SUB_PATH}health-probe")"
[[ $status != 000 && $status -lt 500 ]] || die "订阅服务验证失败（HTTP $status）。"
tr -d '\r' <"$headers_file" | grep -qi '^X-Nova-Service: subscription$' || die "订阅路径没有进入独立订阅服务。"
PGPASSWORD="$NOVA_DB_PASSWORD" psql -h 127.0.0.1 -U "$NOVA_DB_USER" -d "$NOVA_DB_NAME" -tAc 'SELECT 1' | grep -qx 1 || die "PostgreSQL 验证失败。"
systemctl is-enabled --quiet certbot.timer || die "证书自动续期定时器未启用。"

if grep -q '^NOVA_TLS_READY=' "$DEPLOY_FILE"; then
  sed -i 's/^NOVA_TLS_READY=.*/NOVA_TLS_READY=true/' "$DEPLOY_FILE"
else
  printf 'NOVA_TLS_READY=true\n' >>"$DEPLOY_FILE"
fi
chmod 600 "$DEPLOY_FILE"

if [[ -f $RESULT_FILE ]]; then
  sed -i '/^迁移最后一步:/d;/^迁移目标 IP:/d' "$RESULT_FILE"
fi
log "同域名迁移切换完成。请从不同网络验证网站和订阅，确认无误后再关闭旧服务器。"
printf '网站地址: https://%s/\n管理员后台: https://%s/%s/\n' "$NOVA_DOMAIN" "$NOVA_DOMAIN" "$NOVA_ADMIN_PATH"
