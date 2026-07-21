import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  ConfigProvider,
  Descriptions,
  Form,
  Input,
  InputNumber,
  List,
  Layout,
  Modal,
  Progress,
  Result,
  Row,
  Space,
  Spin,
  Steps,
  Tag,
  Typography,
} from 'antd';
import {
  CheckCircleFilled,
  CloudServerOutlined,
  ExclamationCircleFilled,
  GlobalOutlined,
  LoadingOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  SendOutlined,
  WarningFilled,
} from '@ant-design/icons';

import { HttpUtil } from '@/utils';
import { useTheme } from '@/hooks/useTheme';
import AppSidebar from '@/layouts/AppSidebar';
import './MigrationPage.css';

type MigrationSource = {
  supported: boolean;
  platform: string;
  database: string;
  domain: string;
  configured: boolean;
};

type MigrationCredentials = {
  host: string;
  port: number;
  username: string;
  password: string;
};

type MigrationCheck = {
  key: string;
  label: string;
  status: 'success' | 'warning' | 'error';
  detail: string;
};

type MigrationPreflight = {
  ready: boolean;
  targetIp: string;
  targetOs: string;
  targetArch: string;
  targetDiskFree: number;
  fingerprint: string;
  domain: string;
  dnsAddresses: string[];
  dnsReady: boolean;
  existingInstall: boolean;
  checks: MigrationCheck[];
};

type MigrationJob = {
  id: string;
  status: 'running' | 'completed' | 'failed';
  progress: number;
  step: string;
  logs: string[];
  portalUrl?: string;
  adminUrl?: string;
  dnsCutoverRequired?: boolean;
  finalizeCommand?: string;
  error?: string;
  startedAt: string;
  endedAt?: string;
};

type ReauthValues = {
  adminPassword: string;
  twoFactorCode?: string;
  confirmed: boolean;
};

const MIGRATION_JOB_KEY = 'nova-active-migration-job';

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '—';
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

function checkIcon(status: MigrationCheck['status']) {
  if (status === 'success') return <CheckCircleFilled className="migration-check-success" />;
  if (status === 'warning') return <WarningFilled className="migration-check-warning" />;
  return <ExclamationCircleFilled className="migration-check-error" />;
}

export default function MigrationPage() {
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const [connectionForm] = Form.useForm<MigrationCredentials>();
  const [reauthForm] = Form.useForm<ReauthValues>();
  const [source, setSource] = useState<MigrationSource | null>(null);
  const [preflight, setPreflight] = useState<MigrationPreflight | null>(null);
  const [credentials, setCredentials] = useState<MigrationCredentials | null>(null);
  const [checking, setChecking] = useState(false);
  const [startOpen, setStartOpen] = useState(false);
  const [starting, setStarting] = useState(false);
  const [job, setJob] = useState<MigrationJob | null>(null);

  const loadJob = useCallback(async (jobId: string) => {
    const response = await HttpUtil.get<MigrationJob>(`/panel/api/server/migration/status/${jobId}`, undefined, {
      silent: true,
    });
    if (!response.success || !response.obj) {
      if (response.msg.includes('不存在') || response.msg.includes('重启')) sessionStorage.removeItem(MIGRATION_JOB_KEY);
      return;
    }
    setJob(response.obj);
    if (response.obj.status !== 'running') sessionStorage.removeItem(MIGRATION_JOB_KEY);
  }, []);

  useEffect(() => {
    HttpUtil.get<MigrationSource>('/panel/api/server/migration/source', undefined, { silent: true }).then((response) => {
      if (response.success && response.obj) setSource(response.obj);
    });
    const existingJob = sessionStorage.getItem(MIGRATION_JOB_KEY);
    if (existingJob) void loadJob(existingJob);
  }, [loadJob]);

  useEffect(() => {
    if (!job || job.status !== 'running') return;
    const timer = window.setInterval(() => void loadJob(job.id), 2000);
    return () => window.clearInterval(timer);
  }, [job, loadJob]);

  const step = useMemo(() => {
    if (job) return 2;
    if (preflight?.ready) return 1;
    return 0;
  }, [job, preflight]);

  async function runPreflight(values: MigrationCredentials) {
    setChecking(true);
    setPreflight(null);
    setCredentials(values);
    try {
      const response = await HttpUtil.post<MigrationPreflight>('/panel/api/server/migration/preflight', values);
      if (response.success && response.obj) setPreflight(response.obj);
    } finally {
      setChecking(false);
    }
  }

  async function startMigration(values: ReauthValues) {
    if (!credentials || !preflight) return;
    setStarting(true);
    try {
      const response = await HttpUtil.post<MigrationJob>(
        '/panel/api/server/migration/start',
        { ...credentials, fingerprint: preflight.fingerprint, confirmed: values.confirmed },
        {
          headers: {
            'X-Admin-Password': values.adminPassword,
            'X-Admin-2FA': values.twoFactorCode || '',
          },
        },
      );
      if (!response.success || !response.obj) return;
      setJob(response.obj);
      sessionStorage.setItem(MIGRATION_JOB_KEY, response.obj.id);
      setStartOpen(false);
      reauthForm.resetFields();
    } finally {
      setStarting(false);
    }
  }

  function resetConnection() {
    setPreflight(null);
    setCredentials(null);
    setJob(null);
    sessionStorage.removeItem(MIGRATION_JOB_KEY);
  }

  const pageClass = `migration-page-shell ${isDark ? 'is-dark' : ''} ${isUltra ? 'is-ultra' : ''}`.trim();

  return (
    <ConfigProvider theme={antdThemeConfig}>
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content className="content-area">
            <div className="migration-page">
      <div className="migration-page-heading">
        <div>
          <Typography.Title level={2}>一键迁移</Typography.Title>
          <Typography.Paragraph type="secondary">
            将当前机场面板、用户、套餐、订单、订阅、节点及系统设置安全迁移到新的 Ubuntu 服务器。
          </Typography.Paragraph>
        </div>
        <Tag color="blue" icon={<SafetyCertificateOutlined />}>SSH 加密传输</Tag>
      </div>

      <Alert
        type="info"
        showIcon
        title="推荐迁移顺序"
        description="先填写目标服务器并完成连接检测。域名应继续指向旧服务器；系统会通过 SSH 预部署和恢复数据，完成后再提示你切换 DNS 与自动申请 HTTPS。"
      />

      <Card className="migration-steps-card">
        <Steps
          current={step}
          items={[
            { title: '连接目标服务器', content: '填写 SSH 信息并完成预检' },
            { title: '确认迁移', content: '核对域名、主机指纹和覆盖范围' },
            { title: '传输与验证', content: '自动安装、恢复数据并健康检查' },
          ]}
        />
      </Card>

      {source && !source.supported && (
        <Alert
          type="warning"
          showIcon
          title="本地预览环境不能执行真实迁移"
          description="页面功能可以查看；上传到 Ubuntu 并使用项目的一键部署脚本安装后，连接检测与迁移按钮才会实际工作。"
        />
      )}

      {!job && (
        <Row gutter={[18, 18]} align="stretch">
          <Col xs={24} xl={10}>
            <Card title={<Space><CloudServerOutlined />目标服务器</Space>} className="migration-card migration-connection-card">
              <Form<MigrationCredentials>
                form={connectionForm}
                layout="vertical"
                initialValues={{ port: 22, username: 'root' }}
                onFinish={runPreflight}
              >
                <Form.Item name="host" label="服务器 IP 地址" rules={[{ required: true, message: '请输入目标服务器 IP 地址' }]}>
                  <Input placeholder="例如 203.0.113.10" autoComplete="off" />
                </Form.Item>
                <Row gutter={12}>
                  <Col span={10}>
                    <Form.Item name="port" label="SSH 端口" rules={[{ required: true }]}>
                      <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col span={14}>
                    <Form.Item name="username" label="SSH 用户名" rules={[{ required: true, message: '请输入 SSH 用户名' }]}>
                      <Input autoComplete="username" />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.Item name="password" label="SSH 密码" rules={[{ required: true, message: '请输入 SSH 密码' }]}>
                  <Input.Password autoComplete="new-password" placeholder="仅用于本次迁移，不会保存" />
                </Form.Item>
                <Alert
                  type="warning"
                  showIcon
                  title="目标账号需要 root 或 sudo 权限"
                  description="密码只保存在本次迁移任务内存中，不会写入数据库、配置文件或操作日志。"
                />
                <Button type="primary" htmlType="submit" block loading={checking} icon={<SafetyCertificateOutlined />}>
                  检测连接与迁移环境
                </Button>
              </Form>
            </Card>
          </Col>

          <Col xs={24} xl={14}>
            <Card title={<Space><GlobalOutlined />迁移前检查</Space>} className="migration-card migration-preflight-card">
              {!preflight && !checking && (
                <div className="migration-empty">
                  <SafetyCertificateOutlined />
                  <Typography.Title level={4}>等待检测目标服务器</Typography.Title>
                  <Typography.Text type="secondary">检测不会修改目标服务器，只读取系统版本、架构、磁盘、DNS 与现有安装状态。</Typography.Text>
                </div>
              )}
              {checking && <div className="migration-empty"><Spin size="large" /><Typography.Text>正在建立 SSH 连接并检查环境…</Typography.Text></div>}
              {preflight && (
                <>
                  <Descriptions size="small" column={{ xs: 1, sm: 2 }} className="migration-target-summary">
                    <Descriptions.Item label="目标 IP">{preflight.targetIp}</Descriptions.Item>
                    <Descriptions.Item label="系统">{preflight.targetOs}</Descriptions.Item>
                    <Descriptions.Item label="架构">{preflight.targetArch}</Descriptions.Item>
                    <Descriptions.Item label="可用空间">{formatBytes(preflight.targetDiskFree)}</Descriptions.Item>
                    <Descriptions.Item label="域名">{preflight.domain || '未配置'}</Descriptions.Item>
                    <Descriptions.Item label="主机指纹">
                      <Typography.Text copyable ellipsis className="migration-fingerprint">{preflight.fingerprint}</Typography.Text>
                    </Descriptions.Item>
                  </Descriptions>
                  <List
                    className="migration-check-list"
                    dataSource={preflight.checks}
                    renderItem={(item) => (
                      <List.Item>
                        <List.Item.Meta avatar={checkIcon(item.status)} title={item.label} description={item.detail} />
                        <Tag color={item.status === 'success' ? 'success' : item.status === 'warning' ? 'warning' : 'error'}>
                          {item.status === 'success' ? '通过' : item.status === 'warning' ? '注意' : '未通过'}
                        </Tag>
                      </List.Item>
                    )}
                  />
                  <Space className="migration-preflight-actions" wrap>
                    <Button onClick={() => credentials && void runPreflight(credentials)} loading={checking}>重新检测</Button>
                    <Button type="primary" disabled={!preflight.ready} icon={<SendOutlined />} onClick={() => setStartOpen(true)}>
                      确认并开始迁移
                    </Button>
                  </Space>
                </>
              )}
            </Card>
          </Col>
        </Row>
      )}

      {job && (
        <Card className="migration-job-card">
          {job.status === 'running' && (
            <div className="migration-job-running">
              <LoadingOutlined />
              <Typography.Title level={3}>{job.step}</Typography.Title>
              <Typography.Text type="secondary">请保持旧服务器在线。关闭浏览器不会中断迁移，再次进入本页仍可查看进度。</Typography.Text>
              <Progress percent={job.progress} status="active" />
            </div>
          )}
          {job.status === 'completed' && (
            <Result
              status="success"
              title={job.dnsCutoverRequired ? '数据迁移完成，等待域名切换' : '服务器迁移完成'}
              subTitle={job.dnsCutoverRequired
                ? '旧服务器仍在对外服务。请先在新服务器启动下面的收尾命令，再修改域名 A/AAAA 记录；脚本检测到新 IP 后会自动申请 HTTPS。'
                : '目标服务器健康检查已通过。请分别检查用户网站和管理员后台，确认无误后再停用旧服务器。'}
              extra={job.dnsCutoverRequired ? (
                <Space direction="vertical" size="middle">
                  <Typography.Text code copyable>{job.finalizeCommand || 'sudo nova-finalize-domain'}</Typography.Text>
                  <Typography.Text type="warning">不要同时保留新旧两个源站 IP，也不要在确认新站前关闭旧服务器。</Typography.Text>
                </Space>
              ) : [
                <Button key="portal" type="primary" href={job.portalUrl} target="_blank">打开用户网站</Button>,
                <Button key="panel" href={job.adminUrl} target="_blank">打开管理员后台</Button>,
              ]}
            />
          )}
          {job.status === 'failed' && (
            <Result
              status="error"
              title="迁移未完成"
              subTitle={job.error || '目标服务器未通过迁移或健康检查，旧服务器没有被关闭。'}
              extra={<Button type="primary" onClick={resetConnection}>返回重新检测</Button>}
            />
          )}
          <div className="migration-log-panel" aria-label="迁移进度记录">
            <strong>迁移记录</strong>
            {job.logs.map((line, index) => <div key={`${index}-${line}`}>{line}</div>)}
          </div>
        </Card>
      )}

      <Card title="迁移范围与安全说明" className="migration-scope-card">
        <Row gutter={[18, 18]}>
          <Col xs={24} md={8}><Space align="start"><CheckCircleFilled /><div><strong>完整业务数据</strong><p>管理员、客户、套餐、订单、订阅、节点、工单、公告、邮件模板和系统设置。</p></div></Space></Col>
          <Col xs={24} md={8}><Space align="start"><LockOutlined /><div><strong>凭据不落盘</strong><p>SSH 密码不会保存；执行迁移时还需要重新验证当前管理员密码与验证码。</p></div></Space></Col>
          <Col xs={24} md={8}><Space align="start"><SafetyCertificateOutlined /><div><strong>旧服务器不自动删除</strong><p>目标端验证通过后仍保留旧服务器，便于人工核对和必要时回退。</p></div></Space></Col>
        </Row>
      </Card>

      <Modal
        open={startOpen}
        title="最后确认迁移"
        okText="开始迁移"
        cancelText="取消"
        confirmLoading={starting}
        okButtonProps={{ danger: preflight?.existingInstall }}
        onCancel={() => setStartOpen(false)}
        onOk={() => reauthForm.submit()}
        destroyOnHidden
      >
        <Alert
          type={preflight?.existingInstall ? 'warning' : 'info'}
          showIcon
          title={preflight?.existingInstall ? '目标服务器检测到现有安装' : '即将开始传输并安装目标服务'}
          description={preflight?.existingInstall
            ? '目标端会先保留应用备份，但同名数据库将由当前服务器数据完整覆盖。'
            : '迁移期间请勿关闭当前服务器，也不要在旧站继续修改套餐或处理订单。'}
        />
        <Form<ReauthValues> form={reauthForm} layout="vertical" onFinish={startMigration}>
          <Form.Item name="adminPassword" label="当前管理员密码" rules={[{ required: true, message: '请输入当前管理员密码' }]}>
            <Input.Password autoComplete="current-password" />
          </Form.Item>
          <Form.Item name="twoFactorCode" label="验证码（如已启用）">
            <Input inputMode="numeric" maxLength={6} autoComplete="one-time-code" />
          </Form.Item>
          <Form.Item name="confirmed" valuePropName="checked" rules={[{ validator: (_, checked) => checked ? Promise.resolve() : Promise.reject(new Error('请确认迁移覆盖范围')) }]}>
            <Checkbox>我已确认目标服务器无误，并同意覆盖目标端同名服务及数据库；如域名仍指向旧机，将在迁移完成后再切换。</Checkbox>
          </Form.Item>
        </Form>
              </Modal>
            </div>
          </Layout.Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}
