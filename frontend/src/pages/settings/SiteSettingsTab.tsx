import { useState, type ReactNode } from 'react';
import { DeleteOutlined, UploadOutlined } from '@ant-design/icons';
import { Alert, Button, Divider, Image, Input, Select, Space, Switch, Typography, Upload } from 'antd';
import type { UploadProps } from 'antd';

import type { SiteSettings, TrialPlanOption } from './useSiteSettings';

const { Paragraph, Title, Text } = Typography;

interface SiteSettingsTabProps {
  settings: SiteSettings;
  plans: TrialPlanOption[];
  error?: string;
  onChange: (patch: Partial<SiteSettings>) => void;
  onLogoSave: (logoUrl: string) => Promise<unknown>;
  logoSaving?: boolean;
}

interface SiteSettingFieldProps {
  title: string;
  description: string;
  children: ReactNode;
}

function SiteSettingField({ title, description, children }: SiteSettingFieldProps) {
  return (
    <div className="site-setting-field">
      <Text strong>{title}</Text>
      <div className="site-setting-control">{children}</div>
      <Text type="secondary" className="site-setting-description">{description}</Text>
    </div>
  );
}

const MAX_LOGO_FILE_SIZE = 5 * 1024 * 1024;
const allowedLogoTypes = new Set(['image/png', 'image/jpeg', 'image/webp', 'image/gif']);

const currencyPresets = [
  { value: 'CNY', label: 'CNY — 人民币', symbol: '¥' },
  { value: 'USD', label: 'USD — 美元', symbol: '$' },
  { value: 'EUR', label: 'EUR — 欧元', symbol: '€' },
  { value: 'GBP', label: 'GBP — 英镑', symbol: '£' },
  { value: 'JPY', label: 'JPY — 日元', symbol: '¥' },
  { value: 'HKD', label: 'HKD — 港币', symbol: 'HK$' },
  { value: 'TWD', label: 'TWD — 新台币', symbol: 'NT$' },
  { value: 'KRW', label: 'KRW — 韩元', symbol: '₩' },
  { value: 'SGD', label: 'SGD — 新加坡元', symbol: 'S$' },
  { value: 'AUD', label: 'AUD — 澳元', symbol: 'A$' },
  { value: 'CAD', label: 'CAD — 加元', symbol: 'C$' },
  { value: 'CHF', label: 'CHF — 瑞士法郎', symbol: 'CHF' },
  { value: 'INR', label: 'INR — 印度卢比', symbol: '₹' },
  { value: 'RUB', label: 'RUB — 俄罗斯卢布', symbol: '₽' },
  { value: 'THB', label: 'THB — 泰铢', symbol: '฿' },
  { value: 'MYR', label: 'MYR — 马来西亚林吉特', symbol: 'RM' },
  { value: 'AED', label: 'AED — 阿联酋迪拉姆', symbol: 'د.إ' },
  { value: 'BRL', label: 'BRL — 巴西雷亚尔', symbol: 'R$' },
  { value: 'TRY', label: 'TRY — 土耳其里拉', symbol: '₺' },
  { value: 'IDR', label: 'IDR — 印尼盾', symbol: 'Rp' },
] as const;

const currencyPresetMap = new Map<string, (typeof currencyPresets)[number]>(currencyPresets.map((item) => [item.value, item]));
const currencySymbolOptions = [...new Set(currencyPresets.map((item) => item.symbol))].map((symbol) => ({
  value: symbol,
  label: symbol,
}));

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => typeof reader.result === 'string' ? resolve(reader.result) : reject(new Error('图片读取失败'));
    reader.onerror = () => reject(reader.error || new Error('图片读取失败'));
    reader.readAsDataURL(file);
  });
}

const termsTemplates = [
  {
    value: 'standard',
    label: '标准服务条款（推荐）',
    title: '服务使用条款',
    content: `欢迎使用本站服务。注册或购买前，请完整阅读以下条款：

1. 账户与安全
用户应提供真实、有效的注册信息，并妥善保管账户、密码及订阅链接。因主动分享或保管不当造成的损失由用户自行承担。

2. 服务使用范围
本站提供网络连接订阅及相关技术支持。用户不得利用服务从事违法活动、网络攻击、垃圾信息发送、欺诈、侵犯他人权益或其他可能影响平台及节点安全的行为。

3. 套餐、流量与设备
套餐流量、有效期、设备限制及重置周期以购买页面显示为准。超出套餐限制或套餐到期后，相关服务可能自动暂停。

4. 支付与款项确认
数字订阅在支付确认后自动开通。订单支付成功后不可撤销或退回款项，请在付款前确认套餐、周期及设备限制。

5. 服务调整与账号处置
为保障节点及其他用户的正常使用，本站可对异常流量、滥用、攻击或违反本条款的账户采取限制、暂停或终止服务等措施。

6. 隐私与数据
本站仅处理提供账户、订阅、支付、安全审计及客户支持所必要的信息，不记录用户具体浏览内容。

继续注册即表示您已阅读、理解并同意遵守本条款。`,
  },
  {
    value: 'concise',
    label: '精简服务条款',
    title: '用户服务协议',
    content: `注册并使用本站服务前，请确认以下事项：

1. 请妥善保管账户密码和订阅链接，不得转售、共享或公开订阅。
2. 不得利用本站服务从事违法活动、攻击、欺诈、垃圾信息发送或其他滥用行为。
3. 套餐流量、有效期、设备限制和重置周期以购买页面为准。
4. 数字订阅支付成功后自动开通，款项一经确认不可撤销或退回，请在支付前确认订单内容。
5. 对于异常流量、滥用或违反本协议的账户，本站有权限制、暂停或终止服务。
6. 本站仅保存提供服务及保障账户安全所必要的信息，不记录具体浏览内容。

勾选同意并继续注册，即表示您理解并接受本协议。`,
  },
  {
    value: 'acceptable-use',
    label: '严格使用规范',
    title: '服务条款与可接受使用规范',
    content: `使用本站服务即表示您同意遵守以下规范：

1. 严禁网络攻击、端口扫描、漏洞利用、恶意爬取、垃圾邮件、欺诈、盗版分发及任何违法活动。
2. 严禁共享、转售、泄露订阅链接，或以自动化方式规避设备、流量及并发限制。
3. 发现异常登录、异常流量或节点安全风险时，本站可立即限制或暂停相关账户并进行安全核查。
4. 套餐权益以订单页面为准；数字订阅支付成功后自动交付，款项一经确认不可撤销或退回。
5. 用户应自行确保其使用行为符合所在地适用的法律法规。
6. 本站为提供服务可保留账户安全、聚合流量、订单、支付及管理审计记录，但不记录具体浏览内容。

勾选同意并继续注册，即表示您接受本条款及使用规范。`,
  },
] as const;

export default function SiteSettingsTab({ settings, plans, error, onChange, onLogoSave, logoSaving = false }: SiteSettingsTabProps) {
  const [logoError, setLogoError] = useState('');
  const [logoStatus, setLogoStatus] = useState('');
  const trialOptions = [
    { value: '', label: '关闭' },
    ...plans.map((plan) => ({ value: plan.id, label: plan.name })),
  ];
  const handleLogoUpload: UploadProps['beforeUpload'] = async (file) => {
    setLogoError('');
    setLogoStatus('');
    if (!allowedLogoTypes.has(file.type)) {
      setLogoError('仅支持 PNG、JPG、WebP 或 GIF 图片，不支持 SVG。');
      return Upload.LIST_IGNORE;
    }
    if (file.size > MAX_LOGO_FILE_SIZE) {
      setLogoError('LOGO 图片不能超过 5 MB。');
      return Upload.LIST_IGNORE;
    }
    try {
      await onLogoSave(await readFileAsDataURL(file));
      setLogoStatus('已保存并应用到用户网站。');
    } catch (uploadError) {
      setLogoError(uploadError instanceof Error ? uploadError.message : '图片保存失败，请重新选择文件。');
    }
    return Upload.LIST_IGNORE;
  };

  return (
    <div className="site-settings-pane">
      <div className="site-settings-heading">
        <Title level={4}>站点设置</Title>
        <Paragraph>配置站点基本信息，包括站点名称、描述、网址、注册规则与货币单位等核心设置。</Paragraph>
      </div>
      <Divider />
      {error && <Alert type="error" showIcon title="站点设置加载失败" description={error} />}

      <SiteSettingField title="站点名称" description="用于显示需要站点名称的地方。">
        <Input aria-label="站点名称" value={settings.siteName} maxLength={120} onChange={(event) => onChange({ siteName: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="站点描述" description="用于显示需要站点描述的地方。">
        <Input aria-label="站点描述" value={settings.siteDescription} maxLength={500} onChange={(event) => onChange({ siteDescription: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="站点网址" description="当前网站最新网址，将会在邮件等需要用于网址处体现。">
        <Input aria-label="站点网址" value={settings.siteUrl} placeholder="https://example.com" onChange={(event) => onChange({ siteUrl: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="强制 HTTPS" description="当站点没有直接使用 HTTPS，且通过 CDN 或反代提供 HTTPS 时需要开启。">
        <Switch aria-label="强制 HTTPS" checked={settings.forceHttps} onChange={(checked) => onChange({ forceHttps: checked })} />
      </SiteSettingField>

      <SiteSettingField title="LOGO" description="选择图片后会立即保存并应用到用户网站与登录窗口；支持 PNG、JPG、WebP、GIF，最大 5 MB。">
        <div className="logo-upload-panel">
          {settings.logoUrl ? (
            <div className="logo-preview" aria-label="LOGO 预览">
              <Image src={settings.logoUrl} alt="当前站点 LOGO" preview={false} />
            </div>
          ) : (
            <div className="logo-preview logo-preview-empty" aria-label="暂无 LOGO">暂无 LOGO</div>
          )}
          <div className="logo-upload-actions">
            <Space wrap>
              <Upload
                accept=".png,.jpg,.jpeg,.webp,.gif,image/png,image/jpeg,image/webp,image/gif"
                beforeUpload={handleLogoUpload}
                maxCount={1}
                showUploadList={false}
              >
                <Button loading={logoSaving} icon={<UploadOutlined />}>上传并应用</Button>
              </Upload>
              <Button
                danger
                disabled={!settings.logoUrl || logoSaving}
                icon={<DeleteOutlined />}
                onClick={async () => {
                  setLogoError('');
                  setLogoStatus('');
                  try {
                    await onLogoSave('');
                    setLogoStatus('LOGO 已移除并同步到用户网站。');
                  } catch (removeError) {
                    setLogoError(removeError instanceof Error ? removeError.message : 'LOGO 移除失败，请重试。');
                  }
                }}
              >
                移除 LOGO
              </Button>
            </Space>
            {logoError ? <Text type="danger" role="alert">{logoError}</Text> : null}
            {logoStatus ? <Text type="success" role="status">{logoStatus}</Text> : null}
          </div>
        </div>
      </SiteSettingField>

      <SiteSettingField title="订阅 URL" description="用于订阅所使用；留空则使用站点 URL。多个订阅地址可使用分号或换行分隔。">
        <Input.TextArea aria-label="订阅 URL" value={settings.subscriptionUrls} rows={2} placeholder="https://subscribe.example.com" onChange={(event) => onChange({ subscriptionUrls: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="使用条款模板" description="选择预制模板后仍可继续修改标题和正文；保存后新注册用户会看到最新内容。">
        <Select
          aria-label="使用条款模板"
          value={settings.termsTemplate || 'standard'}
          options={termsTemplates.map(({ value, label }) => ({ value, label }))}
          onChange={(value) => {
            const template = termsTemplates.find((item) => item.value === value);
            if (template) onChange({ termsTemplate: value, termsTitle: template.title, termsContent: template.content });
          }}
        />
      </SiteSettingField>

      <SiteSettingField title="使用条款标题" description="显示在用户注册时的条款窗口顶部。">
        <Input aria-label="使用条款标题" value={settings.termsTitle} maxLength={120} onChange={(event) => onChange({ termsTitle: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="使用条款正文" description="支持换行，用户注册时以纯文本展示；修改保存后条款版本会自动更新。">
        <Input.TextArea aria-label="使用条款正文" value={settings.termsContent} rows={14} maxLength={50000} showCount onChange={(event) => onChange({ termsContent: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="外部条款链接（可选）" description="可填写完整的 HTTP/HTTPS 地址，用户可从条款窗口继续打开外部页面。">
        <Input aria-label="用户条款 URL" value={settings.termsUrl} placeholder="https://example.com/terms" onChange={(event) => onChange({ termsUrl: event.target.value })} />
      </SiteSettingField>

      <SiteSettingField title="停止新用户注册" description="开启后，公开注册入口将被关闭，新的注册验证码请求也会被拒绝。">
        <Switch aria-label="停止新用户注册" checked={settings.registrationClosed} onChange={(checked) => onChange({ registrationClosed: checked })} />
      </SiteSettingField>

      <SiteSettingField title="注册试用" description="选择新用户注册后自动获得的试用套餐；如没有选项，请先到套餐管理添加有效套餐和计费周期。">
        <Select aria-label="注册试用" value={settings.trialPlanId} options={trialOptions} onChange={(value) => onChange({ trialPlanId: value })} />
      </SiteSettingField>

      <SiteSettingField title="货币单位" description="仅用于展示；更改后网站中的货币单位将同步变化。">
        <Select
          aria-label="货币单位"
          showSearch
          optionFilterProp="label"
          value={settings.currency}
          options={currencyPresets.map(({ value, label }) => ({ value, label }))}
          onChange={(value) => {
            const preset = currencyPresetMap.get(value);
            onChange({ currency: value, currencySymbol: preset?.symbol || settings.currencySymbol });
          }}
        />
      </SiteSettingField>

      <SiteSettingField title="货币符号" description="仅用于展示；更改后网站中的货币符号将同步变化。">
        <Select
          aria-label="货币符号"
          showSearch
          optionFilterProp="label"
          value={settings.currencySymbol}
          options={currencySymbolOptions}
          onChange={(value) => onChange({ currencySymbol: value })}
        />
      </SiteSettingField>
    </div>
  );
}
