import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'react-router-dom';
import {
  Alert,
  Button,
  Card,
  Col,
  ConfigProvider,
  FloatButton,
  Layout,
  Modal,
  Row,
  Space,
  Spin,
  message,
} from 'antd';

import { HttpUtil, PromiseUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';
import { useTheme } from '@/hooks/useTheme';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useAllSettings } from '@/api/queries/useAllSettings';
import { AllSettingSchema } from '@/schemas/setting';
import AppSidebar from '@/layouts/AppSidebar';
import GeneralTab from './GeneralTab';
import SecurityTab from './SecurityTab';
import TelegramTab from './TelegramTab';
import EmailTab from './EmailTab';
import SubscriptionGeneralTab from './SubscriptionGeneralTab';
import SubscriptionFormatsTab from './SubscriptionFormatsTab';
import SubscriptionTemplateTab from './SubscriptionTemplateTab';
import { useSiteSettings } from './useSiteSettings';
import { useSecuritySettings } from './useSecuritySettings';
import { useSubscriptionSettings } from './useSubscriptionSettings';
import './SettingsPage.css';

interface ApiMsg {
  success?: boolean;
}

const tabSlugs = ['general', 'security', 'telegram', 'email', 'subscription', 'subscription-formats', 'subscription-template'];

function isIp(h: string): boolean {
  if (typeof h !== 'string') return false;
  const v4 = h.split('.');
  if (v4.length === 4 && v4.every((p) => /^\d{1,3}$/.test(p) && Number(p) <= 255)) return true;
  if (!h.includes(':') || h.includes(':::')) return false;
  const parts = h.split('::');
  if (parts.length > 2) return false;
  const split = (s: string) => (s ? s.split(':').filter(Boolean) : []);
  const head = split(parts[0]);
  const tail = split(parts[1]);
  const valid = (seg: string) => /^[0-9a-fA-F]{1,4}$/.test(seg);
  if (![...head, ...tail].every(valid)) return false;
  const groups = head.length + tail.length;
  return parts.length === 2 ? groups < 8 : groups === 8;
}

function scrollTarget() {
  return document.getElementById('content-layout') as HTMLElement;
}

export default function SettingsPage() {
  const { t } = useTranslation();
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const { isMobile } = useMediaQuery();
  const [modal, modalContextHolder] = Modal.useModal();
  const [messageApi, messageContextHolder] = message.useMessage();

  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const {
    allSetting,
    updateSetting,
    fetched: settingsFetched,
    spinning: settingsSpinning,
    setSpinning,
    saveDisabled: settingsSaveDisabled,
    saveAll,
    savePayload,
  } = useAllSettings();
  const {
    siteSettings,
    trialPlans,
    updateSiteSettings,
    saveSiteSettings,
    saveSiteLogo,
    logoSaving,
    saveDisabled: siteSettingsSaveDisabled,
    fetched: siteSettingsFetched,
    spinning: siteSettingsSpinning,
    error: siteSettingsError,
  } = useSiteSettings();
  const {
    securitySettings,
    updateSecuritySettings,
    saveSecuritySettings,
    saveDisabled: securitySettingsSaveDisabled,
    fetched: securitySettingsFetched,
    spinning: securitySettingsSpinning,
    error: securitySettingsError,
  } = useSecuritySettings();
  const {
    subscriptionSettings,
    updateSubscriptionSettings,
    saveSubscriptionSettings,
    saveDisabled: subscriptionSettingsSaveDisabled,
    fetched: subscriptionSettingsFetched,
    spinning: subscriptionSettingsSpinning,
    error: subscriptionSettingsError,
  } = useSubscriptionSettings();

  const [entryHost, setEntryHost] = useState('');
  const [entryPort, setEntryPort] = useState('');
  const [entryIsIP, setEntryIsIP] = useState(false);

  useEffect(() => {
     
    const host = window.location.hostname;
    setEntryHost(host);
    setEntryPort(window.location.port);
    setEntryIsIP(isIp(host));
     
  }, []);

  const [alertVisible, setAlertVisible] = useState(true);
  const location = useLocation();
  const slug = location.hash.replace(/^#/, '');
  const activeSlug = tabSlugs.includes(slug) ? slug : 'general';

  function rebuildUrlAfterRestart(): string {
    const { webDomain, webPort, webBasePath, webCertFile, webKeyFile } = allSetting;
    const newProtocol = (webCertFile || webKeyFile) ? 'https:' : 'http:';

    let base = webBasePath ? webBasePath.replace(/^\//, '') : '';
    if (base && !base.endsWith('/')) base += '/';

    if (!entryIsIP) {
      const url = new URL(window.location.href);
      url.pathname = `/${base}panel/settings`;
      url.protocol = newProtocol;
      return url.toString();
    }

    let finalHost = entryHost;
    let finalPort = entryPort || '';
    if (webDomain && isIp(webDomain)) finalHost = webDomain;
    if (webPort && Number(webPort) !== Number(entryPort)) finalPort = String(webPort);

    const url = new URL(`${newProtocol}//${finalHost}`);
    if (finalPort) url.port = finalPort;
    url.pathname = `/${base}panel/settings`;
    return url.toString();
  }

  async function onSave() {
    const saves: Promise<unknown>[] = [];
    if (!settingsSaveDisabled) {
      const result = AllSettingSchema.safeParse(allSetting);
      if (!result.success) {
        const issue = result.error.issues[0];
        const fieldPath = issue?.path.join('.') ?? 'value';
        const msgKey = issue?.message ?? 'somethingWentWrong';
        messageApi.error(`${fieldPath}: ${t(msgKey, { defaultValue: msgKey })}`);
        return;
      }
      saves.push(saveAll());
    }
    if (!siteSettingsSaveDisabled) saves.push(saveSiteSettings());
    if (!securitySettingsSaveDisabled) saves.push(saveSecuritySettings());
    if (!subscriptionSettingsSaveDisabled) saves.push(saveSubscriptionSettings());
    await Promise.all(saves);
    messageApi.success('设置已保存');
  }

  function restartPanel() {
    modal.confirm({
      title: t('pages.settings.restartPanel'),
      content: t('pages.settings.restartPanelDesc'),
      okText: t('pages.settings.restartPanel'),
      okButtonProps: { danger: true },
      cancelText: t('cancel'),
      onOk: async () => {
        setSpinning(true);
        try {
          const msg = await HttpUtil.post('/panel/api/setting/restartPanel') as ApiMsg;
          if (!msg?.success) return;
          await PromiseUtil.sleep(5000);
          window.location.replace(rebuildUrlAfterRestart());
        } finally {
          setSpinning(false);
        }
      },
    });
  }

  const confAlerts = useMemo<string[]>(() => {
    const out: string[] = [];
    if (window.location.protocol !== 'https:') {
      out.push(t('pages.settings.warnHttp'));
    }
    if (allSetting.webPort === 2053) {
      out.push(t('pages.settings.warnDefaultPort'));
    }
    const segs = window.location.pathname.split('/').length < 4;
    if (segs && allSetting.webBasePath === '/') {
      out.push(t('pages.settings.warnDefaultBasePath'));
    }
    if (allSetting.subEnable) {
      let subPath = allSetting.subPath;
      if (allSetting.subURI) {
        try { subPath = new URL(allSetting.subURI).pathname; } catch { /* noop */ }
      }
      if (subPath === '/sub/') {
        out.push(t('pages.settings.warnDefaultSubPath'));
      }
    }
    if (allSetting.subJsonEnable) {
      let p = allSetting.subJsonPath;
      if (allSetting.subJsonURI) {
        try { p = new URL(allSetting.subJsonURI).pathname; } catch { /* noop */ }
      }
      if (p === '/json/') {
        out.push(t('pages.settings.warnDefaultJsonPath'));
      }
    }
    return out;
  }, [allSetting, t]);

  const pageClass = useMemo(() => {
    const classes = ['settings-page'];
    if (isDark) classes.push('is-dark');
    if (isUltra) classes.push('is-ultra');
    return classes.join(' ');
  }, [isDark, isUltra]);

  const categoryBody = useMemo(() => {
    switch (activeSlug) {
      case 'security': return (
        <SecurityTab
          allSetting={allSetting}
          updateSetting={updateSetting}
          saveSetting={savePayload}
          securitySettings={securitySettings}
          securitySettingsError={securitySettingsError}
          updateSecuritySettings={updateSecuritySettings}
        />
      );
      case 'telegram': return <TelegramTab allSetting={allSetting} updateSetting={updateSetting} />;
      case 'email': return <EmailTab allSetting={allSetting} updateSetting={updateSetting} />;
      case 'subscription': return (
        <SubscriptionGeneralTab
          allSetting={allSetting}
          updateSetting={updateSetting}
          subscriptionSettings={subscriptionSettings}
          subscriptionSettingsError={subscriptionSettingsError}
          updateSubscriptionSettings={updateSubscriptionSettings}
        />
      );
      case 'subscription-formats': return <SubscriptionFormatsTab allSetting={allSetting} updateSetting={updateSetting} />;
      case 'subscription-template': return <SubscriptionTemplateTab allSetting={allSetting} updateSetting={updateSetting} />;
      default: return (
        <GeneralTab
          allSetting={allSetting}
          updateSetting={updateSetting}
          siteSettings={siteSettings}
          trialPlans={trialPlans}
          siteSettingsError={siteSettingsError}
          updateSiteSettings={updateSiteSettings}
          saveSiteLogo={saveSiteLogo}
          logoSaving={logoSaving}
        />
      );
    }
  }, [activeSlug, allSetting, siteSettings, trialPlans, siteSettingsError, securitySettings, securitySettingsError, subscriptionSettings, subscriptionSettingsError, updateSetting, updateSiteSettings, updateSecuritySettings, updateSubscriptionSettings, savePayload, saveSiteLogo, logoSaving]);

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      {modalContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />

        <Layout className="content-shell">
          <Layout.Content id="content-layout" className="content-area">
            <Spin spinning={settingsSpinning || siteSettingsSpinning || securitySettingsSpinning || subscriptionSettingsSpinning || !settingsFetched || !siteSettingsFetched || !securitySettingsFetched || !subscriptionSettingsFetched} delay={200} description={t('loading')} size="large">
              {!settingsFetched || !siteSettingsFetched || !securitySettingsFetched || !subscriptionSettingsFetched ? (
                <div className="loading-spacer" />
              ) : (
                <>
                  {confAlerts.length > 0 && alertVisible && (
                    <Alert
                      type="error"
                      showIcon
                      closable={{ onClose: () => setAlertVisible(false) }}
                      className="conf-alert"
                      title={t('pages.settings.securityWarnings')}
                      description={(
                        <>
                          <b>{t('pages.settings.panelExposed')}</b>
                          <ul>
                            {confAlerts.map((msg, i) => <li key={i}>{msg}</li>)}
                          </ul>
                        </>
                      )}
                    />
                  )}

                  <Row gutter={[isMobile ? 8 : 16, isMobile ? 0 : 12]}>
                    <Col span={24}>
                      <Card hoverable>
                        <Row className="header-row">
                          <Col xs={24} sm={10} className="header-actions">
                            <Space>
                              <Button type="primary" disabled={settingsSaveDisabled && siteSettingsSaveDisabled && securitySettingsSaveDisabled && subscriptionSettingsSaveDisabled} onClick={onSave}>
                                {t('pages.settings.save')}
                              </Button>
                              <Button type="primary" danger disabled={!settingsSaveDisabled} onClick={restartPanel}>
                                {t('pages.settings.restartPanel')}
                              </Button>
                            </Space>
                          </Col>
                          <Col xs={24} sm={14} className="header-info">
                            <FloatButton.BackTop target={scrollTarget} visibilityHeight={200} />
                            <Alert type="warning" showIcon title={t('pages.settings.infoDesc')} />
                          </Col>
                        </Row>
                      </Card>
                    </Col>

                    <Col span={24}>
                      <Card hoverable>
                        {categoryBody}
                      </Card>
                    </Col>
                  </Row>
                </>
              )}
            </Spin>
          </Layout.Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}
