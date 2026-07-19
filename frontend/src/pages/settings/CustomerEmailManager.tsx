import { useEffect, useMemo, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Input,
  Row,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
} from 'antd';
import { SaveOutlined, SendOutlined } from '@ant-design/icons';

import { HttpUtil } from '@/utils';

const { Paragraph, Text } = Typography;

export interface CustomerEmailTemplate {
  key: string;
  name: string;
  subject: string;
  bodyHtml: string;
  active: boolean;
  system: boolean;
  sortOrder: number;
}

interface CustomerRow {
  id: string;
  email: string;
  displayName?: string;
  status: string;
}

interface CustomerPage {
  items: CustomerRow[];
  total: number;
}

interface QueueResult {
  campaignId: string;
  queued: number;
}

interface TemplateFormValues {
  name: string;
  subject: string;
  bodyHtml: string;
  active: boolean;
}

interface SendFormValues {
  audience: 'selected' | 'active' | 'subscribed';
  customerIds?: string[];
  templateKey?: string;
  subject: string;
  bodyHtml: string;
  password: string;
  twoFactorCode?: string;
  confirmed: boolean;
}

const variableTags = (
  <Space size={[4, 4]} wrap>
    <Text type="secondary">可用变量：</Text>
    <Tag>{'{{display_name}}'}</Tag>
    <Tag>{'{{email}}'}</Tag>
    <Tag>{'{{site_name}}'}</Tag>
  </Space>
);

export function useCustomerEmailTemplates() {
  const [templates, setTemplates] = useState<CustomerEmailTemplate[]>([]);
  const [loading, setLoading] = useState(true);

  async function reload() {
    setLoading(true);
    const result = await HttpUtil.get<CustomerEmailTemplate[]>('/panel/api/commercial/email-templates', undefined, { silent: true });
    if (result.success && result.obj) setTemplates(result.obj);
    setLoading(false);
  }

  useEffect(() => {
    void reload();
  }, []);

  return { templates, setTemplates, loading, reload };
}

interface EmailTemplatesPaneProps {
  templates: CustomerEmailTemplate[];
  setTemplates: Dispatch<SetStateAction<CustomerEmailTemplate[]>>;
  loading: boolean;
}

export function EmailTemplatesPane({ templates, setTemplates, loading }: EmailTemplatesPaneProps) {
  const [form] = Form.useForm<TemplateFormValues>();
  const [selectedKey, setSelectedKey] = useState<string>('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!selectedKey && templates.length > 0) setSelectedKey(templates[0].key);
  }, [selectedKey, templates]);

  useEffect(() => {
    const selected = templates.find((template) => template.key === selectedKey);
    if (selected) {
      form.setFieldsValue({
        name: selected.name,
        subject: selected.subject,
        bodyHtml: selected.bodyHtml,
        active: selected.active,
      });
    }
  }, [form, selectedKey, templates]);

  async function saveTemplate(values: TemplateFormValues) {
    if (!selectedKey) return;
    setSaving(true);
    const result = await HttpUtil.put<CustomerEmailTemplate>(
      `/panel/api/commercial/email-templates/${encodeURIComponent(selectedKey)}`,
      values,
      { headers: { 'Content-Type': 'application/json' } },
    );
    if (result.success && result.obj) {
      setTemplates((current) => current.map((template) => template.key === result.obj?.key ? result.obj : template));
    }
    setSaving(false);
  }

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        title="用户邮件模板"
        description="这里的模板用于发送给已注册用户。修改后会保存到数据库，不会影响 3X-UI 的系统告警邮件。"
      />
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={7}>
          <Card size="small" title="模板">
            <Select
              value={selectedKey || undefined}
              loading={loading}
              onChange={setSelectedKey}
              options={templates.map((template) => ({
                value: template.key,
                label: template.name,
              }))}
              style={{ width: '100%' }}
            />
            <Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
              已内置运营公告、订阅开通、到期提醒、流量提醒和工单回复模板。
            </Paragraph>
          </Card>
        </Col>
        <Col xs={24} lg={17}>
          <Card size="small" title="编辑模板">
            <Form form={form} layout="vertical" onFinish={saveTemplate}>
              <Row gutter={12}>
                <Col xs={24} md={18}>
                  <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}>
                    <Input maxLength={120} />
                  </Form.Item>
                </Col>
                <Col xs={24} md={6}>
                  <Form.Item name="active" label="启用" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
              </Row>
              <Form.Item name="subject" label="邮件主题" rules={[{ required: true, message: '请输入邮件主题' }]}>
                <Input maxLength={200} />
              </Form.Item>
              <Form.Item label="模板变量">{variableTags}</Form.Item>
              <Form.Item name="bodyHtml" label="邮件正文（支持 HTML）" rules={[{ required: true, message: '请输入邮件正文' }]}>
                <Input.TextArea rows={12} showCount maxLength={200000} />
              </Form.Item>
              <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
                保存模板
              </Button>
            </Form>
          </Card>
        </Col>
      </Row>
    </Space>
  );
}

interface CustomerEmailSendPaneProps {
  templates: CustomerEmailTemplate[];
  templatesLoading: boolean;
}

export function CustomerEmailSendPane({ templates, templatesLoading }: CustomerEmailSendPaneProps) {
  const [form] = Form.useForm<SendFormValues>();
  const audience = Form.useWatch('audience', form) || 'selected';
  const selectedCustomerIDs = Form.useWatch('customerIds', form) || [];
  const [customers, setCustomers] = useState<CustomerRow[]>([]);
  const [customersLoading, setCustomersLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [queued, setQueued] = useState<QueueResult | null>(null);

  useEffect(() => {
    async function loadCustomers() {
      setCustomersLoading(true);
      const result = await HttpUtil.get<CustomerPage>('/panel/api/commercial/customers', { page: 1, pageSize: 200 }, { silent: true });
      if (result.success && result.obj) setCustomers(result.obj.items);
      setCustomersLoading(false);
    }
    void loadCustomers();
  }, []);

  const customerOptions = useMemo(() => customers.map((customer) => ({
    value: customer.id,
    label: `${customer.displayName ? `${customer.displayName} · ` : ''}${customer.email}`,
  })), [customers]);

  function selectTemplate(key: string) {
    const template = templates.find((item) => item.key === key);
    if (!template) return;
    form.setFieldsValue({ templateKey: key, subject: template.subject, bodyHtml: template.bodyHtml });
  }

  async function queueEmail(values: SendFormValues) {
    setSending(true);
    setQueued(null);
    const result = await HttpUtil.post<QueueResult>('/panel/api/commercial/emails/send', {
      audience: values.audience,
      customerIds: values.customerIds || [],
      templateKey: values.templateKey || '',
      subject: values.subject,
      bodyHtml: values.bodyHtml,
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-Admin-Password': values.password,
        'X-Admin-2FA': values.twoFactorCode || '',
      },
    });
    if (result.success && result.obj) {
      setQueued(result.obj);
      form.setFieldsValue({ password: '', twoFactorCode: '', confirmed: false });
    }
    setSending(false);
  }

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="warning"
        showIcon
        title="发送给机场用户"
        description="邮件会逐个进入持久化发送队列，不会公开其他用户邮箱。批量发送前需要重新验证管理员密码与验证码。"
      />
      {queued && (
        <Alert
          type="success"
          showIcon
          closable
          onClose={() => setQueued(null)}
          title={`已将 ${queued.queued} 封邮件加入发送队列`}
          description={`任务编号：${queued.campaignId}`}
        />
      )}
      <Card size="small" title="新建用户邮件">
        <Form
          form={form}
          layout="vertical"
          initialValues={{ audience: 'selected', customerIds: [], confirmed: false }}
          onFinish={queueEmail}
        >
          <Row gutter={12}>
            <Col xs={24} md={12}>
              <Form.Item name="audience" label="收件用户" rules={[{ required: true }]}>
                <Select options={[
                  { value: 'selected', label: '指定用户' },
                  { value: 'active', label: '全部正常用户' },
                  { value: 'subscribed', label: '有效订阅用户' },
                ]} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="templateKey" label="使用模板（可选）">
                <Select
                  allowClear
                  loading={templatesLoading}
                  onChange={(value) => value && selectTemplate(value)}
                  options={templates.filter((template) => template.active).map((template) => ({ value: template.key, label: template.name }))}
                />
              </Form.Item>
            </Col>
          </Row>
          {audience === 'selected' && (
            <Form.Item
              name="customerIds"
              label="选择用户"
              rules={[{ required: true, type: 'array', min: 1, message: '请选择至少一个用户' }]}
              extra={`已选择 ${selectedCustomerIDs.length} 个用户；列表显示最近的 200 个账号。`}
            >
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                loading={customersLoading}
                options={customerOptions}
                placeholder="按姓名或 Gmail 地址选择用户"
              />
            </Form.Item>
          )}
          <Form.Item name="subject" label="邮件主题" rules={[{ required: true, message: '请输入邮件主题' }]}>
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item label="模板变量">{variableTags}</Form.Item>
          <Form.Item name="bodyHtml" label="邮件正文（支持 HTML）" rules={[{ required: true, message: '请输入邮件正文' }]}>
            <Input.TextArea rows={10} showCount maxLength={200000} />
          </Form.Item>
          <Row gutter={12}>
            <Col xs={24} md={12}>
              <Form.Item name="password" label="管理员密码" rules={[{ required: true, message: '请输入管理员密码' }]}>
                <Input.Password autoComplete="current-password" />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="twoFactorCode" label="验证码（已开启 2FA 时必填）">
                <Input maxLength={8} inputMode="numeric" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="confirmed" valuePropName="checked" rules={[{
            validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('请确认收件范围和邮件内容')),
          }]}>
            <Checkbox>我已确认收件用户范围和邮件内容</Checkbox>
          </Form.Item>
          <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={sending}>
            加入发送队列
          </Button>
        </Form>
      </Card>
    </Space>
  );
}
