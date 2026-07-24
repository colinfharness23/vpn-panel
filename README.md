# NOVA 商业 VPN 面板

NOVA 是面向商业订阅业务的 VPN 管理面板。生产部署使用 PostgreSQL、Nginx、Let's Encrypt 与 Xray，包含用户门户、套餐与订单、线路中心、客户端下载、工单、审计日志和服务器迁移。

> 生产安装仅支持 Ubuntu 22.04/24.04，架构为 amd64 或 arm64。安装器不会修改 UFW 或云安全组。

## FinalShell 一键安装

安装前，把域名的 A/AAAA 记录指向服务器，并放行 TCP 80、443；使用 Hysteria2 时同时放行 UDP 443。以 `root` 身份执行：

```bash
curl -fsSL https://raw.githubusercontent.com/colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev/main/deploy/ubuntu/install.sh | env NOVA_GITHUB_REPO=colinfharness23/r8eH6Z6rpQpAi2UI2gkZ0lteagev bash
```

安装过程依次询问：

1. 需要绑定的域名；
2. 管理员登录账号；
3. 管理员密码及确认。

内部面板端口、订阅端口、订阅路径和 18 位隐藏后台地址均自动随机生成。HTTPS 证书自动申请并由系统定时续期，不要求填写通知邮箱。只有 PostgreSQL、Nginx、Xray、根门户、隐藏后台、订阅服务与证书续期检查全部通过，脚本才会报告成功。

安装结果保存在仅 root 可读的 `/root/nova-install-result.txt`，管理员密码不会写入文件。

仓库为 Private 时，GitHub 的 raw 文件和 Release 无法匿名下载。安装或更新前临时改成 Public，完成并验证后再改回 Private；服务器不保存 GitHub Token。

## 日常操作

```bash
sudo nova-update                 # 更新到最新稳定 Release
sudo nova-backup                 # 立即备份数据库、配置和上传文件
sudo nova-rollback               # 回滚最近一次应用更新
sudo nova-rotate-admin-path      # 更换隐藏管理员入口
sudo nova-uninstall              # 先备份，再卸载 NOVA
sudo journalctl -u x-ui -n 200   # 查看服务日志
sudo nova-diagnose-active-subscription  # 用实际用户凭证逐条验证订阅链路
sudo nova-diagnose-managed-ingress      # 检查托管入站、路由与监听状态
```

安装器不会修改防火墙或云安全组。VMess、VLESS 与 Trojan 默认共用本站 TCP 443，首条 Hysteria2 默认使用 UDP 443；AnyTLS、Shadowsocks、WireGuard、额外 Hysteria2 线路及无法共用标准端口的原生传输，会在“线路中心”显示实际随机端口，此时只需放行页面显示的对应 TCP/UDP 端口（范围 `20000-59999`），不必整段全部开放。

## 线路中心怎么用

管理员登录后进入“商业管理 → 线路中心”：

- 订阅 URL：输入上游订阅地址，选择线路组和套餐；以后刷新发现的新节点自动继承分组关系。
- 协议链接：批量粘贴 VMess、VLESS、Trojan、Shadowsocks、Hysteria2、WireGuard 或 AnyTLS 链接，预览成功、重复和失败项目，再提交并分组。AnyTLS 使用随 Release 一同校验和安装的独立运行时，仍会经过本站托管入口、套餐授权和用户流量累计。
- 套餐：绑定一个或多个线路组。用户购买或管理员授权后，系统自动创建客户端、挂载健康线路并生成本站订阅，不需要手工添加入站或客户端。

上游订阅 URL、凭证和原始协议内容会加密保存，只在后台显示脱敏信息。客户端得到的是本站域名上的同类型托管线路（例如 Trojan 仍为 Trojan、AnyTLS 仍为 AnyTLS），节点名称严格使用你设置的别名；订阅标题、支持地址和资料页也只使用本站配置，不会显示上游机场名称、域名或凭证。

## 同域名迁移到新服务器

同一个域名不需要同时绑定两台服务器。正确流程是“先搬数据，最后切 DNS”：

1. 提前把域名 DNS TTL 调低，并等待旧 TTL 过期；此时域名仍指向旧服务器。
2. 在旧服务器后台进入“服务器设置 → 一键迁移”，填写新服务器公网 IP 和 SSH 凭据。
3. NOVA 在新服务器完成 PostgreSQL、程序、配置和上传文件恢复，但暂不申请证书。
4. 迁移完成后，在新服务器运行 `sudo nova-finalize-domain`。
5. 工具提示后，把域名 A/AAAA 记录从旧 IP 改到新 IP。不要同时保留两个源站 IP。
6. 工具检测到 DNS 已切换后，会自动签发 HTTPS、启用续期并检查门户、后台和登录隔离。
7. 从不同网络验证网站与订阅，确认正常后再关闭旧服务器。

迁移快照开始后不要继续在旧站新增订单、用户或配置，否则这些新写入不会包含在已生成的快照中。

## 域名发信与 SMTP

域名本身不会自动附带邮箱或 SMTP。只需要让网站发送验证码、通知和账单时，可以使用 Resend 等 SMTP 发信服务，不必先购买收件邮箱：

1. 在 Resend 添加你的域名，把它显示的 SPF、DKIM 记录复制到域名 DNS。
2. 等待域名验证通过并创建 API Key。
3. 在“设置 → 邮件”选择 `Resend` 预设；SMTP 用户名保持 `resend`，SMTP 密码填写 API Key。
4. “公开发件地址”填写已经验证域名下的地址，例如 `no-reply@pheero.com`；发件显示名会自动使用后台当前站点名称。
5. 保存后发送测试邮件。若还要接收用户回复，再向域名服务商或企业邮箱服务商购买收件邮箱。

SPF、DKIM、DMARC 和正常的发信信誉能降低进入垃圾箱的概率，但任何应用或 SMTP 服务商都无法保证每封邮件一定进入收件箱。后台邮件设置页也内置了同样的五步说明和 Resend 自动填充预设。

## 安全基线

- 商业生产环境强制 PostgreSQL，检测到 SQLite 会拒绝启动。
- 会话 Cookie 强制 `Secure`、`HttpOnly` 和 `SameSite=Lax`；写接口有 CSRF 防护。
- 默认启用 HSTS、CSP、点击劫持防护、MIME 嗅探防护和浏览器权限限制。
- 登录、注册、验证码和密码重置在 Nginx 与应用层双重限流。
- 普通请求体限制为 10 MiB；只有已认证的安装包上传路由允许 1 GiB。
- 订阅 URL 拒绝私网、回环、链路本地与元数据地址，并在 DNS 连接及重定向阶段重复检查。
- 商业生产模式禁止网页下载并执行更新脚本；`nova-update` 只安装带 SHA-256 校验文件的本站 Release。
- 服务以无登录权限的 `nova` 用户运行，并通过 systemd 限制文件系统、内核与提权能力。
- 生产密钥和数据库密码仅保存在服务器 root 限权文件中，敏感业务配置使用主密钥加密。

安全问题与审计边界见 [SECURITY.md](/SECURITY.md)。

## 发布资产

稳定 Release 必须同时包含：

- `x-ui-linux-amd64.tar.gz` 与对应 `.sha256`
- `x-ui-linux-arm64.tar.gz` 与对应 `.sha256`

安装器按服务器架构选择资产，并先验证 SHA-256，再替换程序。Release 未通过自动测试或缺少必要文件时不得发布。

## 开源说明

本项目基于 GPL-3.0 许可的 3x-ui/Xray 管理能力继续开发，保留原项目版权和许可证文件。NOVA 的商业门户、线路编排、PostgreSQL 部署、迁移和安全加固由本仓库维护。
