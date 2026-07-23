#!/usr/bin/env bash
set -Eeuo pipefail

readonly DEPLOY_FILE="/etc/nova/deploy.env"
readonly TARGET_DIR="/var/lib/x-ui/certs"

die() { printf '[NOVA] 线路证书同步失败：%s\n' "$*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "请使用 root 执行。"
[[ -f $DEPLOY_FILE ]] || die "没有找到 $DEPLOY_FILE。"
# shellcheck disable=SC1090,SC1091
source "$DEPLOY_FILE"
[[ ${NOVA_DOMAIN:-} =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$ ]] || die "部署域名无效。"

source_dir="/etc/letsencrypt/live/$NOVA_DOMAIN"
[[ -r $source_dir/fullchain.pem && -r $source_dir/privkey.pem ]] || die "Let's Encrypt 证书尚不可用：$source_dir"

install -d -m 750 -o nova -g nova "$TARGET_DIR"
install -m 640 -o root -g nova "$source_dir/fullchain.pem" "$TARGET_DIR/fullchain.pem.new"
install -m 640 -o root -g nova "$source_dir/privkey.pem" "$TARGET_DIR/privkey.pem.new"
mv -f "$TARGET_DIR/fullchain.pem.new" "$TARGET_DIR/fullchain.pem"
mv -f "$TARGET_DIR/privkey.pem.new" "$TARGET_DIR/privkey.pem"

if [[ ${NOVA_SYNC_NO_RESTART:-false} != true ]] && systemctl is-active --quiet x-ui; then
  systemctl restart x-ui
fi
printf '[NOVA] 本站线路 TLS 证书已同步。\n'
