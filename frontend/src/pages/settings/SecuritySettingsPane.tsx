import type { ReactNode } from 'react';
import { Alert, Divider, Input, InputNumber, Switch, Typography } from 'antd';

import type { SecuritySettings } from './useSecuritySettings';
import { normalizePath, sanitizePath } from './uriPath';

const { Paragraph, Title, Text } = Typography;

interface SecuritySettingsPaneProps {
  settings: SecuritySettings;
  backendPath: string;
  error?: string;
  onChange: (patch: Partial<SecuritySettings>) => void;
  onBackendPathChange: (path: string) => void;
}

interface SecuritySettingFieldProps {
  title: string;
  description: string;
  children: ReactNode;
}

function SecuritySettingField({ title, description, children }: SecuritySettingFieldProps) {
  return (
    <div className="site-setting-field">
      <Text strong>{title}</Text>
      <div className="site-setting-control">{children}</div>
      <Text type="secondary" className="site-setting-description">{description}</Text>
    </div>
  );
}

function displayBackendPath(path = '/'): string {
  return path === '/' ? '' : path.replace(/^\/+|\/+$/g, '');
}

export default function SecuritySettingsPane({ settings, backendPath, error, onChange, onBackendPathChange }: SecuritySettingsPaneProps) {
  return (
    <div className="site-settings-pane security-settings-pane">
      <div className="site-settings-heading">
        <Title level={4}>安全设置</Title>
        <Paragraph>配置系统安全相关选项，包括登录验证、密码策略、API 访问与注册安全设置。</Paragraph>
      </div>
      <Divider />
      {error && <Alert type="error" showIcon title="安全设置加载失败" description={error} />}

      <SecuritySettingField title="邮箱验证" description="开启后将会强制要求用户进行邮箱验证。">
        <Switch aria-label="邮箱验证" checked={settings.emailVerification} onChange={(checked) => onChange({ emailVerification: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="禁止使用 Gmail 多别名" description="开启后 Gmail 点号或加号别名将无法注册。">
        <Switch aria-label="禁止使用 Gmail 多别名" checked={settings.disallowGmailAliases} onChange={(checked) => onChange({ disallowGmailAliases: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="安全模式" description="开启后，除站点 URL 对应域名外，使用其他域名访问前台网站与公开接口将返回 403。">
        <Switch aria-label="安全模式" checked={settings.safeMode} onChange={(checked) => onChange({ safeMode: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="后台路径" description="后台管理路径，修改并保存后需要重启面板才会生效。">
        <Input
          aria-label="后台路径"
          value={displayBackendPath(backendPath)}
          placeholder="例如：76f6c86b"
          onChange={(event) => onBackendPathChange(normalizePath(sanitizePath(event.target.value)))}
        />
      </SecuritySettingField>

      <SecuritySettingField title="邮箱后缀白名单" description="开启后，仅白名单中的邮箱后缀才允许注册。">
        <Switch aria-label="邮箱后缀白名单" checked={settings.emailSuffixWhitelistEnabled} onChange={(checked) => onChange({ emailSuffixWhitelistEnabled: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="邮箱后缀" description="输入允许的邮箱后缀，每行一个。">
        <Input.TextArea
          aria-label="邮箱后缀"
          value={settings.allowedEmailSuffixes}
          rows={3}
          disabled={!settings.emailSuffixWhitelistEnabled}
          placeholder={'gmail.com\nqq.com'}
          onChange={(event) => onChange({ allowedEmailSuffixes: event.target.value })}
        />
      </SecuritySettingField>

      <SecuritySettingField title="启用验证码" description="开启后，用户注册发送邮箱验证码前需要通过 Turnstile 人机验证。">
        <Switch aria-label="启用验证码" checked={settings.registrationCaptchaEnabled} onChange={(checked) => onChange({ registrationCaptchaEnabled: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="IP 注册限制" description="开启后，同一 IP 在 24 小时内最多注册 3 个账号。">
        <Switch aria-label="IP 注册限制" checked={settings.ipRegistrationLimitEnabled} onChange={(checked) => onChange({ ipRegistrationLimitEnabled: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="密码尝试限制" description="开启后将限制连续密码尝试次数，并在达到上限时临时锁定账号。">
        <Switch aria-label="密码尝试限制" checked={settings.passwordAttemptLimitEnabled} onChange={(checked) => onChange({ passwordAttemptLimitEnabled: checked })} />
      </SecuritySettingField>

      <SecuritySettingField title="尝试次数" description="允许的最大密码尝试次数。">
        <InputNumber
          aria-label="尝试次数"
          value={settings.maxPasswordAttempts}
          min={1}
          max={20}
          precision={0}
          disabled={!settings.passwordAttemptLimitEnabled}
          onChange={(value) => onChange({ maxPasswordAttempts: Number(value) || 1 })}
        />
      </SecuritySettingField>

      <SecuritySettingField title="锁定时长" description="账户锁定的持续时间（分钟）。">
        <InputNumber
          aria-label="锁定时长"
          value={settings.passwordLockDurationMinutes}
          min={1}
          max={1440}
          precision={0}
          disabled={!settings.passwordAttemptLimitEnabled}
          onChange={(value) => onChange({ passwordLockDurationMinutes: Number(value) || 1 })}
        />
      </SecuritySettingField>
    </div>
  );
}
