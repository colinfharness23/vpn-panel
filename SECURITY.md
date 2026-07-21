# NOVA 安全说明

## 支持范围

安全加固和商业功能仅保证在 `deploy/ubuntu/install.sh` 部署的 Ubuntu 22.04/24.04、PostgreSQL、Nginx 和稳定 Release 组合上生效。上游兼容脚本、Docker 示例和 SQLite 测试路径不属于 NOVA 商业生产部署。

## 报告漏洞

请不要在公开 Issue 中提交可利用细节、账号、订阅链接、数据库备份或服务器地址。使用 GitHub Security Advisory 的私密漏洞报告功能，并包含：受影响版本、复现条件、影响范围和建议修复方式。

## 已执行的审计边界

发布前检查包含：

- 搜索硬编码凭据、SSH 持久化、反向 Shell、隐蔽下载执行和异常定时任务；
- 审查登录、会话、CSRF、Cookie、请求体限制、文件上传、SSRF、SMTP、加密存储和管理员权限；
- 审查安装、更新、备份、回滚、卸载、迁移、systemd 与 Nginx 配置；
- 执行 Go/前端测试、race、静态分析、ShellCheck、生产构建和 Release 安装演练；
- 使用 GitHub CodeQL、依赖漏洞告警和密钥扫描作为持续检查。

一次审计不能数学意义上证明软件永远没有漏洞。发现问题时应升级到修复版本，并轮换可能已经暴露的管理员密码、订阅标识、支付密钥、SMTP 密钥和主加密密钥。

## 生产建议

- 只开放 TCP 80、443 和实际需要的线路端口；不要公开 PostgreSQL、内部面板端口或内部订阅端口。
- 为管理员启用强密码和 2FA，定期执行 `nova-rotate-admin-path`。
- 在云平台开启 DDoS 防护、异常流量告警和不可变快照；Nginx 限流不能替代上游网络清洗。
- 定期执行 `nova-backup` 并在隔离环境验证恢复。
- 仓库设为 Private 时，更新前短暂设为 Public，运行 `sudo nova-update`，验证后立即恢复 Private。
