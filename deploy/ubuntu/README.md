# Ubuntu 裸机一键部署

本目录用于将机场面板直接安装到 Ubuntu 22.04 或 24.04，不依赖 Docker。安装脚本通过 `/dev/tty` 运行交互式向导，可直接粘贴到 FinalShell 执行。

## 发布安装包

给版本创建标签后，GitHub Actions 会生成以下 Release 资产：

- `x-ui-linux-amd64.tar.gz` 与对应的 `.sha256`
- `x-ui-linux-arm64.tar.gz` 与对应的 `.sha256`

安装和更新脚本会强制校验 SHA-256，并检查压缩包必须包含面板程序、Xray 程序及 `bin/config.json`。

## FinalShell 一行安装

先将域名的 A/AAAA 记录解析到服务器，并确认安全组允许 80、443 以及之后创建的节点业务端口。然后以 root 用户运行：

```bash
curl -fsSL https://raw.githubusercontent.com/OWNER/REPOSITORY/main/deploy/ubuntu/install.sh | env NOVA_GITHUB_REPO=OWNER/REPOSITORY bash
```

将 `OWNER/REPOSITORY` 替换为实际 GitHub 仓库。向导会依次询问管理员账号、管理员密码及确认、域名和 Let's Encrypt 通知邮箱；密码输入不回显。

脚本会安装 PostgreSQL、Nginx、Certbot 和运行依赖，创建最小权限用户，自动签发证书，并生成只保存于服务器的 18 位数字后台入口。只有数据库、门户、后台、Xray、订阅路由和证书续期检查全部通过时才会显示安装成功。

安装结果仅写入 root 可读的 `/root/nova-install-result.txt`，权限为 `600`。其中不保存管理员密码。

## 地址与安全约束

- 用户门户固定为 `https://域名/`；未登录时只显示登录/注册窗口。
- `/portal` 与 `/portal/` 永久重定向到根地址。
- 管理后台为安装时生成的 `https://域名/18位数字/`。
- `/admin`、`/panel`、错误数字路径均返回 404。
- 后台路径保存在 `/etc/nova/deploy.env`，更新、备份恢复和迁移时保持不变。
- 需要主动更换后台入口时运行 `nova-rotate-admin-path`；命令会在健康检查失败时自动回滚。

随机路径只是附加防护，仍应启用强密码、登录限流与 2FA。

## 运维命令

```bash
nova-update                 # 更新到最新稳定 Release
nova-update v1.1.0          # 更新到指定标签
nova-backup                 # 立即备份数据库和部署配置
nova-rollback               # 回滚到最近的应用备份
nova-rotate-admin-path      # 安全轮换 18 位后台入口
systemctl status x-ui       # 查看服务状态
journalctl -u x-ui -n 200   # 查看最近日志
```

PostgreSQL、部署配置和持久化安装包每天自动备份到 `/var/backups/nova/database`，默认保留 14 天并附带 SHA-256 清单。应用更新前也会备份；新版本健康检查失败时自动回滚。

## 正式收款前

- 在后台完成 SMTP、支付接口、异步回调、Turnstile、站点条款和 Telegram 客服配置。
- 在线路中心导入订阅 URL 或六类协议链接，完成探测、分组和套餐绑定；普通开通流程无需手工创建入站或客户端。
- 在真实 Ubuntu 主机完成“导入线路 → 绑定套餐 → 支付或手动开通 → 获取订阅 → 客户端连接 → 流量统计 → 到期、重置和续费”闭环验收。
- 支付回调必须使用 HTTPS；Cloudflare 或其他代理开启后仍要保证源站证书有效。
- 不要把内部监听端口暴露到公网；只放行 Nginx 的 80/443 和线路业务 TCP `20000-59999`。安装脚本不会修改 UFW 或云安全组。
