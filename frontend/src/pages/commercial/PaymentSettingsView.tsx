import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Col, Form, Input, Row, Segmented, Select, Space, Switch, Tag, Typography } from 'antd';
import { CreditCardOutlined, SafetyCertificateOutlined } from '@ant-design/icons';

const { Text } = Typography;

export type PaymentProviderName = 'epay' | 'alipay_f2f' | 'codepay';

export interface PaymentSettingsFormValues {
  epayEnabled: boolean;
  epayGatewayUrl: string;
  epayMerchantId: string;
  epayMerchantKey?: string;
  epayPaymentType: 'alipay' | 'wxpay' | 'qqpay';
  epayNotifyUrl: string;
  epayReturnUrl: string;
  alipayEnabled: boolean;
  alipayMode: 'sandbox' | 'production';
  alipayAppId: string;
  alipaySellerId: string;
  alipayNotifyUrl: string;
  alipayPrivateKey?: string;
  alipayPublicKey?: string;
  codepayEnabled: boolean;
  codepayGatewayUrl: string;
  codepayMerchantId: string;
  codepayKey?: string;
  codepayPaymentType: '1' | '2' | '3';
  codepayNotifyUrl: string;
  codepayReturnUrl: string;
  password: string;
  twoFactorCode?: string;
}

interface PaymentSettingsViewProps {
  settings: Record<string, string | boolean>;
  onSave: (values: PaymentSettingsFormValues) => Promise<boolean>;
}

const providerOptions = [
  { value: 'epay', label: 'Epay' },
  { value: 'alipay_f2f', label: 'AlipayF2F' },
  { value: 'codepay', label: '码支付' },
];

const epayTypeOptions = [
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
  { value: 'qqpay', label: 'QQ 钱包' },
];

const codepayTypeOptions = [
  { value: '1', label: '支付宝' },
  { value: '3', label: '微信支付' },
  { value: '2', label: 'QQ 钱包' },
];

const providerDescriptions: Record<PaymentProviderName, { type: 'info' | 'success' | 'warning'; title: string; description: string }> = {
  epay: { type: 'info', title: 'Epay（易支付 MAPI）', description: '兼容常见 Epay MAPI/MD5 协议，使用网关返回的支付链接生成二维码。' },
  alipay_f2f: { type: 'success', title: 'AlipayF2F（支付宝当面付）', description: '使用支付宝官方 alipay.trade.precreate 接口生成当面付二维码，并进行 RSA2 回调验签。' },
  codepay: { type: 'warning', title: '码支付（CodePay）', description: '兼容常见码支付 MD5 跳转协议，支持支付宝、微信支付和 QQ 钱包通道；网关地址可按服务商配置。' },
};

function configuredTag(configured: boolean) {
  return <Tag color={configured ? 'success' : 'warning'}>{configured ? '已配置，留空不修改' : '尚未配置'}</Tag>;
}

function EnabledSwitch({ name }: { name: keyof Pick<PaymentSettingsFormValues, 'epayEnabled' | 'alipayEnabled' | 'codepayEnabled'> }) {
  return <Form.Item name={name} label="启用此支付方式" valuePropName="checked" extra="启用并保存后，用户付款时可以选择此方式。"><Switch checkedChildren="已启用" unCheckedChildren="未启用" /></Form.Item>;
}

export default function PaymentSettingsView({ settings, onSave }: PaymentSettingsViewProps) {
  const [form] = Form.useForm<PaymentSettingsFormValues>();
  const [saving, setSaving] = useState(false);
  const [provider, setProvider] = useState<PaymentProviderName>('epay');
  const epayEnabled = Form.useWatch('epayEnabled', form) || false;
  const alipayEnabled = Form.useWatch('alipayEnabled', form) || false;
  const codepayEnabled = Form.useWatch('codepayEnabled', form) || false;
  const epayKeyConfigured = settings['epay.key.configured'] === true;
  const alipayPrivateKeyConfigured = settings['alipay.private_key.configured'] === true;
  const alipayPublicKeyConfigured = settings['alipay.public_key.configured'] === true;
  const codepayKeyConfigured = settings['codepay.key.configured'] === true;
  const initialValues = useMemo<Partial<PaymentSettingsFormValues>>(() => ({
    epayEnabled: settings['payment.epay.enabled'] === true,
    epayGatewayUrl: String(settings['epay.gateway_url'] || ''),
    epayMerchantId: String(settings['epay.pid'] || ''),
    epayPaymentType: (['alipay', 'wxpay', 'qqpay'].includes(String(settings['epay.type'])) ? settings['epay.type'] : 'alipay') as PaymentSettingsFormValues['epayPaymentType'],
    epayNotifyUrl: String(settings['epay.notify_url'] || ''),
    epayReturnUrl: String(settings['epay.return_url'] || ''),
    alipayEnabled: settings['payment.alipay_f2f.enabled'] === true,
    alipayMode: settings['alipay.mode'] === 'production' ? 'production' : 'sandbox',
    alipayAppId: String(settings['alipay.app_id'] || ''),
    alipaySellerId: String(settings['alipay.seller_id'] || ''),
    alipayNotifyUrl: String(settings['alipay.notify_url'] || ''),
    codepayEnabled: settings['payment.codepay.enabled'] === true,
    codepayGatewayUrl: String(settings['codepay.gateway_url'] || ''),
    codepayMerchantId: String(settings['codepay.id'] || ''),
    codepayPaymentType: (['1', '2', '3'].includes(String(settings['codepay.type'])) ? settings['codepay.type'] : '1') as PaymentSettingsFormValues['codepayPaymentType'],
    codepayNotifyUrl: String(settings['codepay.notify_url'] || ''),
    codepayReturnUrl: String(settings['codepay.return_url'] || ''),
  }), [settings]);

  useEffect(() => {
    form.setFieldsValue(initialValues);
  }, [form, initialValues]);

  const submit = async (values: PaymentSettingsFormValues) => {
    setSaving(true);
    try {
      if (await onSave(values)) {
        form.resetFields(['epayMerchantKey', 'alipayPrivateKey', 'alipayPublicKey', 'codepayKey', 'password', 'twoFactorCode']);
      }
    } finally {
      setSaving(false);
    }
  };

  const description = providerDescriptions[provider];
  return <Space orientation="vertical" size={16} style={{ width: '100%' }}>
    <Alert type="info" showIcon title="SMTP 与邮件通知使用面板统一邮件设置" description="注册验证码、系统通知与测试发送都从同一处配置。" action={<Button href="settings#email">打开 SMTP 设置</Button>} />
    <Form form={form} layout="vertical" onFinish={submit} requiredMark={false}>
      <Card title={<Space><CreditCardOutlined />支付接口</Space>} extra={<Text type="secondary">可同时启用多个接口</Text>}>
        <Form.Item label="选择接口进行配置">
          <Segmented size="large" block value={provider} onChange={(value) => setProvider(value as PaymentProviderName)} options={providerOptions} />
        </Form.Item>
        <Alert type={description.type} showIcon title={description.title} description={description.description} style={{ marginBottom: 20 }} />

        {provider === 'epay' && <>
          <EnabledSwitch name="epayEnabled" />
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item name="epayGatewayUrl" label="Epay 网关地址" extra="填写站点根地址或 mapi.php 完整地址。" rules={epayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://pay.example.com" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="epayMerchantId" label="商户 ID（PID）" rules={epayEnabled ? [{ required: true }] : []}><Input placeholder="1000" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="epayMerchantKey" label={<Space>商户密钥{configuredTag(epayKeyConfigured)}</Space>} rules={epayEnabled && !epayKeyConfigured ? [{ required: true }] : []}><Input.Password placeholder={epayKeyConfigured ? '留空表示不修改' : '请输入 Epay 商户密钥'} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="epayPaymentType" label="支付通道" rules={epayEnabled ? [{ required: true }] : []}><Select options={epayTypeOptions} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="epayNotifyUrl" label="异步通知地址" rules={epayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://你的域名/api/v1/guest/payments/epay/notify" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="epayReturnUrl" label="支付后跳转地址" rules={epayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://你的域名/" /></Form.Item></Col>
          </Row>
        </>}

        {provider === 'alipay_f2f' && <>
          <EnabledSwitch name="alipayEnabled" />
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item name="alipayMode" label="运行模式" rules={alipayEnabled ? [{ required: true }] : []}><Select options={[{ value: 'sandbox', label: '沙箱' }, { value: 'production', label: '生产' }]} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="alipayAppId" label="支付宝 App ID" rules={alipayEnabled ? [{ required: true }] : []}><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="alipaySellerId" label="支付宝商户 ID"><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="alipayNotifyUrl" label="异步通知地址" rules={alipayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://你的域名/api/v1/guest/payments/alipay/notify" /></Form.Item></Col>
            <Col span={24}><Form.Item name="alipayPrivateKey" label={<Space>支付宝应用私钥{configuredTag(alipayPrivateKeyConfigured)}</Space>} rules={alipayEnabled && !alipayPrivateKeyConfigured ? [{ required: true }] : []}><Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} placeholder={alipayPrivateKeyConfigured ? '留空表示不修改' : '粘贴 PKCS#8 或 PKCS#1 私钥'} /></Form.Item></Col>
            <Col span={24}><Form.Item name="alipayPublicKey" label={<Space>支付宝公钥{configuredTag(alipayPublicKeyConfigured)}</Space>} rules={alipayEnabled && !alipayPublicKeyConfigured ? [{ required: true }] : []}><Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} placeholder={alipayPublicKeyConfigured ? '留空表示不修改' : '粘贴支付宝公钥'} /></Form.Item></Col>
          </Row>
        </>}

        {provider === 'codepay' && <>
          <EnabledSwitch name="codepayEnabled" />
          <Row gutter={16}>
            <Col xs={24} md={12}><Form.Item name="codepayGatewayUrl" label="码支付创建订单地址" extra="填写服务商提供的 creat_order 完整地址。" rules={codepayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://codepay.example.com/creat_order/" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="codepayMerchantId" label="码支付 ID" rules={codepayEnabled ? [{ required: true }] : []}><Input placeholder="10041" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="codepayKey" label={<Space>通信密钥{configuredTag(codepayKeyConfigured)}</Space>} rules={codepayEnabled && !codepayKeyConfigured ? [{ required: true }] : []}><Input.Password placeholder={codepayKeyConfigured ? '留空表示不修改' : '请输入码支付通信密钥'} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="codepayPaymentType" label="支付通道" rules={codepayEnabled ? [{ required: true }] : []}><Select options={codepayTypeOptions} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="codepayNotifyUrl" label="异步通知地址" rules={codepayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://你的域名/api/v1/guest/payments/codepay/notify" /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name="codepayReturnUrl" label="支付后跳转地址" rules={codepayEnabled ? [{ required: true }, { type: 'url' }] : [{ type: 'url', warningOnly: true }]}><Input placeholder="https://你的域名/" /></Form.Item></Col>
          </Row>
        </>}
      </Card>

      <Card title={<Space><SafetyCertificateOutlined />管理员重新验证</Space>} style={{ marginTop: 16 }}>
        <Row gutter={16}>
          <Col xs={24} md={12}><Form.Item name="password" label="管理员密码" rules={[{ required: true }]}><Input.Password autoComplete="current-password" /></Form.Item></Col>
          <Col xs={24} md={12}><Form.Item name="twoFactorCode" label="验证码（如已启用 2FA）"><Input maxLength={8} inputMode="numeric" /></Form.Item></Col>
        </Row>
        <Button type="primary" htmlType="submit" loading={saving}>保存支付设置</Button>
      </Card>
    </Form>
  </Space>;
}
