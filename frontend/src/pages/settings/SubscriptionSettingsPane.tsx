import type { ReactNode } from 'react';
import { Alert, Input, Select, Switch, Typography } from 'antd';

import type { AllSetting } from '@/models/setting';
import type { SubscriptionSettings } from './useSubscriptionSettings';
import { normalizePath, sanitizePath } from './uriPath';

const { Text } = Typography;

interface SubscriptionSettingsPaneProps {
  settings: SubscriptionSettings;
  allSetting: AllSetting;
  error?: string;
  onChange: (patch: Partial<SubscriptionSettings>) => void;
  onAllSettingChange: (patch: Partial<AllSetting>) => void;
}

interface SubscriptionSettingFieldProps {
  title: string;
  description: string;
  children: ReactNode;
}

const eventOptions = [
  { value: 'none', label: '不执行任何动作' },
  { value: 'reset_traffic', label: '重置用户流量' },
];

function SubscriptionSettingField({ title, description, children }: SubscriptionSettingFieldProps) {
  return (
    <div className="site-setting-field">
      <Text strong>{title}</Text>
      <div className="site-setting-control">{children}</div>
      <Text type="secondary" className="site-setting-description">{description}</Text>
    </div>
  );
}

export default function SubscriptionSettingsPane({ settings, allSetting, error, onChange, onAllSettingChange }: SubscriptionSettingsPaneProps) {
  return (
    <div className="site-settings-pane subscription-settings-pane">
      {error && <Alert type="error" showIcon title="订阅设置加载失败" description={error} />}

      <SubscriptionSettingField title="允许用户更改订阅" description="开启后用户可以对当前订阅进行续费或升级。关闭后仅管理员可以调整用户套餐。">
        <Switch aria-label="允许用户更改订阅" checked={settings.allowUserChange} onChange={(checked) => onChange({ allowUserChange: checked })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="月流量重置方式" description="全局月流量重置方式；套餐设置为按月重置时使用此规则。">
        <Select
          aria-label="月流量重置方式"
          value={settings.monthlyResetMode}
          options={[
            { value: 'calendar_month', label: '按月重置' },
            { value: 'billing_cycle', label: '按订阅周期重置' },
            { value: 'never', label: '不自动重置' },
          ]}
          onChange={(monthlyResetMode) => onChange({ monthlyResetMode })}
        />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="开启折抵方案" description="开启后用户升级套餐时，系统会按原套餐剩余有效期计算折抵金额。">
        <Switch aria-label="开启折抵方案" checked={settings.offsetEnabled} onChange={(checked) => onChange({ offsetEnabled: checked })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="当订阅新购时触发事件" description="新购订阅完成开通后触发该任务。">
        <Select aria-label="当订阅新购时触发事件" value={settings.purchaseEvent} options={eventOptions} onChange={(purchaseEvent) => onChange({ purchaseEvent })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="当订阅续费时触发事件" description="续费订阅完成开通后触发该任务。">
        <Select aria-label="当订阅续费时触发事件" value={settings.renewalEvent} options={eventOptions} onChange={(renewalEvent) => onChange({ renewalEvent })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="当订阅变更时触发事件" description="升级或变更订阅完成开通后触发该任务。">
        <Select aria-label="当订阅变更时触发事件" value={settings.changeEvent} options={eventOptions} onChange={(changeEvent) => onChange({ changeEvent })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="订阅路径" description="订阅服务使用的路径；修改并保存后需要重启面板才能生效。">
        <Input
          aria-label="订阅路径"
          value={allSetting.subPath}
          placeholder="/sub/"
          onChange={(event) => onAllSettingChange({ subPath: sanitizePath(event.target.value) })}
          onBlur={() => onAllSettingChange({ subPath: normalizePath(allSetting.subPath) })}
        />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="在订阅中展示订阅信息" description="开启后将在客户端订阅响应中输出流量、到期时间等订阅信息。">
        <Switch aria-label="在订阅中展示订阅信息" checked={settings.showSubscriptionInfo} onChange={(checked) => onChange({ showSubscriptionInfo: checked })} />
      </SubscriptionSettingField>

      <SubscriptionSettingField title="在订阅中线路名称中显示协议名称" description="开启后订阅线路名称会显示 VLESS、VMESS、TROJAN 等协议名称。">
        <Switch aria-label="在订阅中线路名称中显示协议名称" checked={settings.showProtocolInNodeName} onChange={(checked) => onChange({ showProtocolInNodeName: checked })} />
      </SubscriptionSettingField>
    </div>
  );
}
