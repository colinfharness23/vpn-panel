import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Input, InputNumber, Select, Space, Switch, Tabs } from 'antd';
import { FileTextOutlined, NotificationOutlined, SendOutlined, SettingOutlined } from '@ant-design/icons';
import { HttpUtil } from '@/utils';
import type { AllSetting } from '@/models/setting';
import { SettingListItem } from '@/components/ui';
import { EmailNotifications } from '@/components/ui/notifications/EmailNotifications';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useSiteBranding } from '@/hooks/useSiteBranding';
import { catTabLabel } from './catTabLabel';
import SecretInput from './SecretInput';
import {
  CustomerEmailSendPane,
  EmailTemplatesPane,
  useCustomerEmailTemplates,
} from './CustomerEmailManager';

interface EmailTabProps {
  allSetting: AllSetting;
  updateSetting: (patch: Partial<AllSetting>) => void;
}

interface SmtpTestResult {
  success: boolean;
  stage?: string;
  msg: string;
}

const smtpPortPresets = [
  { port: 25, encryptionType: 'starttls', label: '25 / STARTTLS' },
  { port: 465, encryptionType: 'tls', label: '465 / SSL/TLS' },
  { port: 587, encryptionType: 'starttls', label: '587 / STARTTLS' },
] as const;

const smtpProviderPresets = [
  { key: 'gmail', label: 'Gmail', host: 'smtp.gmail.com', port: 587, encryptionType: 'starttls' },
  { key: 'outlook', label: 'Outlook.com', host: 'smtp-mail.outlook.com', port: 587, encryptionType: 'starttls' },
  { key: 'microsoft365', label: 'Microsoft 365', host: 'smtp.office365.com', port: 587, encryptionType: 'starttls' },
  { key: 'qq', label: 'QQ 邮箱', host: 'smtp.qq.com', port: 465, encryptionType: 'tls' },
  { key: '163', label: '163 邮箱', host: 'smtp.163.com', port: 465, encryptionType: 'tls' },
  { key: 'yahoo', label: 'Yahoo Mail', host: 'smtp.mail.yahoo.com', port: 465, encryptionType: 'tls' },
  { key: 'icloud', label: 'iCloud Mail', host: 'smtp.mail.me.com', port: 587, encryptionType: 'starttls' },
] as const;

export default function EmailTab({ allSetting, updateSetting }: EmailTabProps) {
  const { t } = useTranslation();
  const { siteName } = useSiteBranding();
  const { isMobile } = useMediaQuery();
  const [testLoading, setTestLoading] = useState(false);
  const [testResult, setTestResult] = useState<SmtpTestResult | null>(null);
  const emailTemplates = useCustomerEmailTemplates();
  const activeSmtpProvider = smtpProviderPresets.find(({ host, port, encryptionType }) => (
    host === allSetting.smtpHost
    && port === allSetting.smtpPort
    && encryptionType === allSetting.smtpEncryptionType
  ))?.key;

  const stageLabel: Record<string, string> = {
    connect: t('pages.settings.smtpStageConnect'),
    auth: t('pages.settings.smtpStageAuth'),
    send: t('pages.settings.smtpStageSend'),
  };

  async function handleTestSmtp() {
    setTestLoading(true);
    setTestResult(null);
    try {
      const res = await HttpUtil.post('/panel/api/setting/testSmtp') as SmtpTestResult;
      setTestResult(res);
    } catch (e: unknown) {
      setTestResult({ success: false, msg: e instanceof Error ? e.message : t('pages.settings.requestFailed') });
    } finally {
      setTestLoading(false);
    }
  }

  return (
    <Tabs defaultActiveKey="1" items={[
      {
        key: '1',
        label: catTabLabel(<SettingOutlined />, t('pages.settings.smtpSettings'), isMobile),
        children: (
          <>
            <Alert
              type="info"
              showIcon
              title={t('pages.settings.smtpSenderIdentity', {
                siteName,
                address: allSetting.smtpUsername || 'user@example.com',
              })}
              description={t('pages.settings.smtpDeliverabilityHint')}
              style={{ marginBottom: 16 }}
            />
            <SettingListItem paddings="small" title={t('pages.settings.smtpEnable')} description={t('pages.settings.smtpEnableDesc')}>
              <Switch checked={allSetting.smtpEnable} onChange={(v) => updateSetting({ smtpEnable: v })} />
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpHost')} description={t('pages.settings.smtpHostDesc')}>
              <Space orientation="vertical" size={8} style={{ width: '100%' }}>
                <Input value={allSetting.smtpHost} placeholder="smtp.example.com"
                  onChange={(e) => updateSetting({ smtpHost: e.target.value })} />
                <Select
                  aria-label={t('pages.settings.smtpProviderPreset')}
                  value={activeSmtpProvider}
                  placeholder={t('pages.settings.smtpProviderPresetPlaceholder')}
                  options={smtpProviderPresets.map(({ key, label }) => ({ value: key, label }))}
                  onChange={(key) => {
                    const preset = smtpProviderPresets.find((item) => item.key === key);
                    if (preset) {
                      updateSetting({
                        smtpHost: preset.host,
                        smtpPort: preset.port,
                        smtpEncryptionType: preset.encryptionType,
                      });
                    }
                  }}
                  style={{ width: '100%' }}
                />
              </Space>
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpPort')} description={t('pages.settings.smtpPortDesc')}>
              <Space orientation="vertical" size={8} style={{ width: '100%' }}>
                <InputNumber value={allSetting.smtpPort} min={1} max={65535} style={{ width: '100%' }}
                  onChange={(v) => updateSetting({ smtpPort: Number(v) || 587 })} />
                <Space size={[8, 8]} wrap>
                  {smtpPortPresets.map(({ port, encryptionType, label }) => (
                    <Button
                      key={port}
                      size="small"
                      type={allSetting.smtpPort === port ? 'primary' : 'default'}
                      onClick={() => updateSetting({ smtpPort: port, smtpEncryptionType: encryptionType })}
                    >
                      {label}
                    </Button>
                  ))}
                </Space>
              </Space>
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpUsername')} description={t('pages.settings.smtpUsernameDesc')}>
              <Input value={allSetting.smtpUsername} placeholder="user@example.com"
                onChange={(e) => updateSetting({ smtpUsername: e.target.value })} />
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpPassword')}
              description={allSetting.hasSmtpPassword && !allSetting.clearSmtpPassword ? t('pages.settings.smtpPasswordConfigured') : t('pages.settings.smtpPasswordDesc')}>
              <SecretInput value={allSetting.smtpPassword}
                configured={allSetting.hasSmtpPassword}
                clearArmed={allSetting.clearSmtpPassword}
                placeholder={t('pages.settings.smtpPasswordPlaceholder')}
                onChange={(v) => updateSetting({ smtpPassword: v })}
                onClearArmedChange={(armed) => updateSetting({ clearSmtpPassword: armed })} />
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpTo')} description={t('pages.settings.smtpToDesc')}>
              <Input value={allSetting.smtpTo} placeholder="admin@example.com, ops@example.com"
                onChange={(e) => updateSetting({ smtpTo: e.target.value })} />
            </SettingListItem>

            <SettingListItem paddings="small" title={t('pages.settings.smtpEncryption')} description={t('pages.settings.smtpEncryptionDesc')}>
              <Select
                value={allSetting.smtpEncryptionType}
                onChange={(v) => updateSetting({ smtpEncryptionType: v })}
                options={[
                  { value: 'none', label: t('pages.settings.smtpEncryptionNone') },
                  { value: 'starttls', label: t('pages.settings.smtpEncryptionStartTLS') },
                  { value: 'tls', label: t('pages.settings.smtpEncryptionTLS') },
                ]}
                style={{ width: '100%' }}
              />
            </SettingListItem>

            <Space orientation="vertical" size={8} style={{ width: '100%', marginTop: 16 }}>
              <Button type="primary" icon={<SendOutlined />} loading={testLoading} onClick={handleTestSmtp}>
                {t('pages.settings.testSmtp')}
              </Button>
              {testResult && (
                <Alert
                  type={testResult.success ? 'success' : 'error'}
                  title={
                    testResult.success
                      ? t('pages.settings.' + testResult.msg)
                      : <span><b>{stageLabel[testResult.stage || ''] || testResult.stage}:</b> {t('pages.settings.' + testResult.msg)}</span>
                  }
                  showIcon
                  closable
                  onClose={() => setTestResult(null)}
                />
              )}
            </Space>
          </>
        ),
      },
      {
        key: '2',
        label: catTabLabel(<NotificationOutlined />, '系统告警', isMobile),
        children: (
          <Space orientation="vertical" size={16} style={{ width: '100%' }}>
            <Alert
              type="info"
              showIcon
              title="发送给管理员的运行告警"
              description="节点离线、Xray 崩溃、资源过高等事件会发送到 SMTP 设置中的管理员收件人，不会发送给机场用户。"
            />
            <SettingListItem paddings="small" title={t('pages.settings.smtpEventBusNotify')} description={t('pages.settings.smtpEventBusNotifyDesc')}>
              <EmailNotifications allSetting={allSetting} updateSetting={updateSetting} />
            </SettingListItem>
          </Space>
        ),
      },
      {
        key: '3',
        label: catTabLabel(<FileTextOutlined />, '邮件模板', isMobile),
        children: (
          <EmailTemplatesPane
            templates={emailTemplates.templates}
            setTemplates={emailTemplates.setTemplates}
            loading={emailTemplates.loading}
          />
        ),
      },
      {
        key: '4',
        label: catTabLabel(<SendOutlined />, '发送邮件', isMobile),
        children: (
          <CustomerEmailSendPane
            templates={emailTemplates.templates}
            templatesLoading={emailTemplates.loading}
          />
        ),
      },
    ]} />
  );
}
