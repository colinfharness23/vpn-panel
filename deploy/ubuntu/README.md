# NOVA Ubuntu 部署工具

本目录是 NOVA 商业生产部署的唯一受支持入口，适用于 Ubuntu 22.04/24.04 的 amd64 和 arm64 主机。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev/main/deploy/ubuntu/install.sh | env NOVA_GITHUB_REPO=colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev bash
```

安装器依次询问域名、管理员账号和强密码，随后自动安装 PostgreSQL、Nginx、Certbot、NOVA 与 Xray，随机生成内部端口、订阅路径和隐藏后台地址，并自动申请及续期 HTTPS。

## 已安装的运维命令

```bash
nova-update
nova-backup
nova-rollback
nova-rotate-admin-path
nova-finalize-domain
nova-uninstall
```

`nova-finalize-domain` 只用于“同域名一键迁移”的最后 DNS 切换阶段。普通全新安装不需要运行它。

## 同域名迁移

迁移期间让 DNS 保持指向旧服务器。后台一键迁移会以 `NOVA_DEFER_TLS=true` 在目标服务器预部署并恢复数据。完成后在目标服务器运行：

```bash
sudo nova-finalize-domain
```

按提示把 A/AAAA 记录从旧 IP 改为新 IP。工具确认 DNS 已包含目标地址且不再包含迁移前的旧地址后，才申请证书并进行 HTTPS 健康检查。

## 发布约束

Release 必须包含 amd64/arm64 压缩包和 `.sha256` 文件。安装、更新脚本均先校验摘要，再替换程序；商业发行包不包含或调用上游网页更新脚本。
