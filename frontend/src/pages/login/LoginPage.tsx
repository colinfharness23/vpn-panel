import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  ConfigProvider,
  Form,
  Input,
  Layout,
  Spin,
  message,
} from 'antd';
import {
  KeyOutlined,
  LockOutlined,
  SafetyCertificateFilled,
  UserOutlined,
} from '@ant-design/icons';
import { FormProvider, useForm } from 'react-hook-form';

import { FormField, rhfZodValidate } from '@/components/form/rhf';
import { LoginFormSchema, TotpCodeSchema, type LoginFormValues } from '@/schemas/login';
import { HttpUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';
import { applySiteBranding } from '@/hooks/useSiteBranding';
import './LoginPage.css';

type LoginForm = LoginFormValues;
type GuestAuthConfig = { site?: Record<string, string> };
type GuestAuthConfigEnvelope = { success?: boolean; obj?: GuestAuthConfig };

const basePath = window.X_UI_BASE_PATH || '';
const rawPublicBasePath = window.X_UI_PUBLIC_BASE_PATH || '/';
const publicBasePath = rawPublicBasePath.endsWith('/') ? rawPublicBasePath : `${rawPublicBasePath}/`;
const loginTheme = {
  token: {
    colorPrimary: '#4f95f5',
    colorText: '#17233d',
    colorTextSecondary: '#68758b',
    colorBorder: '#d7e4f0',
    borderRadius: 10,
    controlHeightLG: 48,
    fontFamily: 'Inter, "PingFang SC", "Microsoft YaHei", system-ui, sans-serif',
  },
  components: {
    Button: { primaryShadow: '0 8px 20px rgba(79, 149, 245, 0.2)' },
    Input: { activeShadow: '0 0 0 3px rgba(79, 149, 245, 0.12)' },
  },
};

export default function LoginPage() {
  const [messageApi, messageContextHolder] = message.useMessage();
  const [fetched, setFetched] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [twoFactorEnable, setTwoFactorEnable] = useState(false);
  const [siteName, setSiteName] = useState('NOVA');
  const [logoUrl, setLogoUrl] = useState('');
  const methods = useForm<LoginForm>({ defaultValues: { username: '', password: '', twoFactorCode: '' } });

  useEffect(() => { setMessageInstance(messageApi); }, [messageApi]);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      HttpUtil.post<boolean>('/getTwoFactorEnable', undefined, { silent: true }),
      fetch('/api/v1/guest/auth-config', { credentials: 'same-origin' })
        .then(async (response) => response.ok ? (await response.json()) as GuestAuthConfigEnvelope : null)
        .catch(() => null),
    ]).then(([twoFactorResult, bootstrapResult]) => {
      if (cancelled) return;
      if (twoFactorResult.success) setTwoFactorEnable(Boolean(twoFactorResult.obj));
      const configuredName = bootstrapResult?.obj?.site?.siteName?.trim();
      const configuredLogo = bootstrapResult?.obj?.site?.logoUrl?.trim();
      if (configuredName) setSiteName(configuredName);
      if (configuredLogo) setLogoUrl(configuredLogo);
      setFetched(true);
    });
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    applySiteBranding(siteName, logoUrl, '管理员登录');
  }, [logoUrl, siteName]);

  const onSubmit = useCallback(async (values: LoginForm) => {
    setSubmitting(true);
    try {
      const result = await HttpUtil.post('/login', values);
      if (result.success) window.location.href = basePath + 'panel/';
    } finally {
      setSubmitting(false);
    }
  }, []);

  return (
    <ConfigProvider theme={loginTheme}>
      {messageContextHolder}
      <Layout className="login-app">
        <Layout.Content className="login-content">
          <header className="login-header">
            <a className="login-brand" href={publicBasePath} aria-label={`${siteName} 用户前台`}>
              <span className="login-brand-mark">
                {logoUrl ? <img src={logoUrl} alt="" /> : <SafetyCertificateFilled />}
              </span>
              <span>{siteName}</span>
            </a>
            <span className="login-console-label">管理员控制台</span>
          </header>

          <main className="login-wrapper">
            <section className="login-stage" aria-label="管理员登录">
              <div className="login-intro">
                <span className="login-kicker">安全运营后台</span>
                <h1>管理客户、套餐与节点</h1>
                <p>订单、订阅、节点与平台设置集中在一个清晰的工作空间中。</p>
                <span className="login-access-note"><SafetyCertificateFilled /> 仅限授权管理员访问</span>
              </div>

              <div className="login-panel">
                {!fetched ? (
                  <div className="login-loading"><Spin size="large" /></div>
                ) : (
                  <div className="login-card">
                    <div className="login-card-heading">
                      <span className="login-card-icon"><SafetyCertificateFilled /></span>
                      <div>
                        <h2>登录管理员后台</h2>
                        <p>请输入管理员账号和密码</p>
                      </div>
                    </div>

                    <FormProvider {...methods}>
                      <Form layout="vertical" className="login-form" onFinish={methods.handleSubmit(onSubmit)}>
                        <FormField name="username" label="用户名" rules={{ validate: rhfZodValidate(LoginFormSchema.shape.username) }}>
                          <Input prefix={<UserOutlined />} autoComplete="username" size="large" placeholder="请输入管理员用户名" autoFocus />
                        </FormField>

                        <FormField name="password" label="密码" rules={{ validate: rhfZodValidate(LoginFormSchema.shape.password) }}>
                          <Input.Password prefix={<LockOutlined />} autoComplete="current-password" size="large" placeholder="请输入管理员密码" />
                        </FormField>

                        {twoFactorEnable ? (
                          <FormField name="twoFactorCode" label="验证码" rules={{ validate: rhfZodValidate(TotpCodeSchema) }}>
                            <Input
                              prefix={<KeyOutlined />}
                              autoComplete="one-time-code"
                              inputMode="numeric"
                              maxLength={6}
                              size="large"
                              placeholder="请输入验证码"
                            />
                          </FormField>
                        ) : null}

                        <Form.Item className="submit-row">
                          <Button type="primary" htmlType="submit" loading={submitting} size="large" block>登录后台</Button>
                        </Form.Item>
                      </Form>
                    </FormProvider>
                  </div>
                )}
              </div>
            </section>
          </main>
        </Layout.Content>
      </Layout>
    </ConfigProvider>
  );
}
