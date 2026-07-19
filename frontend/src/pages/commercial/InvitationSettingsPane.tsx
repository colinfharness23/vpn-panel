import type { ReactNode } from 'react';
import { Alert, Button, InputNumber, Space, Switch, Typography } from 'antd';

import type { InvitationSettings } from './useInvitationSettings';

const { Text } = Typography;

interface InvitationSettingsPaneProps {
  settings: InvitationSettings;
  error?: string;
  spinning?: boolean;
  saveDisabled?: boolean;
  onChange: (patch: Partial<InvitationSettings>) => void;
  onSave: () => void;
}

interface InvitationSettingFieldProps {
  title: string;
  description: string;
  children: ReactNode;
  switchField?: boolean;
}

function InvitationSettingField({ title, description, children, switchField = false }: InvitationSettingFieldProps) {
  return (
    <div className={`invitation-setting-field${switchField ? ' is-switch' : ''}`}>
      <Text strong>{title}</Text>
      {switchField && <Text type="secondary" className="invitation-setting-description">{description}</Text>}
      <div className="invitation-setting-control">{children}</div>
      {!switchField && <Text type="secondary" className="invitation-setting-description">{description}</Text>}
    </div>
  );
}

export default function InvitationSettingsPane({ settings, error, spinning = false, saveDisabled = false, onChange, onSave }: InvitationSettingsPaneProps) {
  return (
    <section className="invitation-settings-pane" data-testid="invitation-settings-pane">
      <div className="invitation-settings-actions">
        <Text type="secondary">保存后立即应用到注册、佣金计算和结算规则。</Text>
        <Button type="primary" loading={spinning} disabled={saveDisabled} onClick={onSave}>保存设置</Button>
      </div>
      {error && <Alert type="error" showIcon title="邀请与佣金设置加载失败" description={error} />}
      <Alert type="info" showIcon title="佣金仅结算为站内余额" description="余额不支持提现，只能用于购买、续费或升级本站套餐；下单时可由用户选择是否抵扣。" />

      <InvitationSettingField title="开启强制邀请" description="开启后只有被邀请的用户才可以进行注册。" switchField>
        <Switch aria-label="开启强制邀请" checked={settings.forcedInvitation} onChange={(checked) => onChange({ forcedInvitation: checked })} />
      </InvitationSettingField>

      <InvitationSettingField title="邀请佣金百分比" description="默认全局的佣金分配比例；开启三级分销时，各级按此比例计算。">
        <Space.Compact block>
          <InputNumber aria-label="邀请佣金百分比" value={settings.commissionPercent} min={0} max={100} precision={0} style={{ width: '100%' }} onChange={(value) => onChange({ commissionPercent: Number(value) || 0 })} />
          <Button disabled>%</Button>
        </Space.Compact>
      </InvitationSettingField>

      <InvitationSettingField title="用户可创建邀请码上限" description="每个用户允许创建的邀请码数量上限。">
        <InputNumber aria-label="用户可创建邀请码上限" value={settings.maxInviteCodes} min={1} max={100} precision={0} onChange={(value) => onChange({ maxInviteCodes: Number(value) || 1 })} />
      </InvitationSettingField>

      <InvitationSettingField title="邀请码永不失效" description="开启后邀请码使用后不会失效；关闭后邀请码仅可成功邀请一位用户。" switchField>
        <Switch aria-label="邀请码永不失效" checked={settings.inviteCodesNeverExpire} onChange={(checked) => onChange({ inviteCodesNeverExpire: checked })} />
      </InvitationSettingField>

      <InvitationSettingField title="佣金仅首次发放" description="开启后仅在被邀请人首次支付成功时产生佣金。" switchField>
        <Switch aria-label="佣金仅首次发放" checked={settings.commissionFirstPaymentOnly} onChange={(checked) => onChange({ commissionFirstPaymentOnly: checked })} />
      </InvitationSettingField>

      <InvitationSettingField title="佣金自动确认" description="开启后佣金将在订单完成 3 日后自动确认。" switchField>
        <Switch aria-label="佣金自动确认" checked={settings.commissionAutoConfirm} onChange={(checked) => onChange({ commissionAutoConfirm: checked })} />
      </InvitationSettingField>

      <InvitationSettingField title="三级分销" description="开启后最多沿邀请关系发放三级佣金，三级比例合计不得大于 100%。" switchField>
        <Switch aria-label="三级分销" checked={settings.multiLevelEnabled} onChange={(checked) => onChange({ multiLevelEnabled: checked })} />
      </InvitationSettingField>
    </section>
  );
}
