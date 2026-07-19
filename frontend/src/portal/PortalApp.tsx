import { useCallback, useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import {
  Alert,
  App as AntApp,
  Avatar,
  Badge,
  Button,
  Card,
  Checkbox,
  Col,
  ConfigProvider,
  Descriptions,
  Divider,
  Dropdown,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Progress,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { FormInstance, MenuProps, TableProps } from 'antd';
import arEG from 'antd/locale/ar_EG';
import enUS from 'antd/locale/en_US';
import esES from 'antd/locale/es_ES';
import faIR from 'antd/locale/fa_IR';
import idID from 'antd/locale/id_ID';
import jaJP from 'antd/locale/ja_JP';
import ptBR from 'antd/locale/pt_BR';
import ruRU from 'antd/locale/ru_RU';
import trTR from 'antd/locale/tr_TR';
import ukUA from 'antd/locale/uk_UA';
import viVN from 'antd/locale/vi_VN';
import zhCN from 'antd/locale/zh_CN';
import zhTW from 'antd/locale/zh_TW';
import {
  BellOutlined,
  ArrowRightOutlined,
  AppstoreOutlined,
  CheckCircleFilled,
  CloudDownloadOutlined,
  CopyOutlined,
  CreditCardOutlined,
  CustomerServiceOutlined,
  GlobalOutlined,
  GiftOutlined,
  ImportOutlined,
  LaptopOutlined,
  LinkOutlined,
  LockOutlined,
  LoginOutlined,
  LogoutOutlined,
  MailOutlined,
  MobileOutlined,
  QrcodeOutlined,
  SafetyCertificateFilled,
  SendOutlined,
  ShoppingCartOutlined,
  ReloadOutlined,
  SafetyOutlined,
  TeamOutlined,
  ThunderboltFilled,
  UserOutlined,
  WalletOutlined,
  WifiOutlined,
} from '@ant-design/icons';

import { portalAsset, PortalApiError, portalRequest } from './api';
import { buildPortalNavigation } from './navigation';
import { localeOptions, portalCopies } from './translations';
import type { PortalCopy, PortalLocale } from './translations';
import type {
  Dashboard,
  GuestBootstrap,
  Order,
  PaymentPayload,
  PlanCatalogItem,
  SubscriptionOverview,
  Ticket,
  TicketMessage,
} from './types';

const { Title, Text, Paragraph } = Typography;
type Section = 'home' | 'subscription' | 'plans' | 'guides' | 'tickets' | 'orders' | 'account';
type AccountTab = 'overview' | 'invitation' | 'security';
type AuthMode = 'login' | 'register' | 'reset';
type PurchaseAction = 'purchase' | 'renewal' | 'upgrade';

declare global {
  interface Window {
    turnstile?: {
      render: (element: HTMLElement, options: { sitekey: string; callback: (token: string) => void; 'expired-callback': () => void; theme: string }) => string;
      remove: (widgetID: string) => void;
    };
  }
}

const GB = 1024 ** 3;

const antdLocales: Record<PortalLocale, typeof enUS> = {
  'ar-EG': arEG,
  'en-US': enUS,
  'fa-IR': faIR,
  'zh-CN': zhCN,
  'zh-TW': zhTW,
  'ja-JP': jaJP,
  'ru-RU': ruRU,
  'vi-VN': viVN,
  'es-ES': esES,
  'id-ID': idID,
  'uk-UA': ukUA,
  'tr-TR': trTR,
  'pt-BR': ptBR,
};

const fallbackBootstrapZH: GuestBootstrap = {
  site: { siteName: 'NOVA', siteTagline: '稳定连接，清晰可控', emailVerification: 'true', emailSuffixWhitelist: 'true', allowedEmailSuffixes: 'gmail.com', forcedInvitation: 'false' },
  plans: [
    { plan: { id: 'starter', slug: 'starter', name: '轻量套餐', description: '适合轻度浏览与临时使用', trafficBytes: 100 * GB, deviceLimit: 3, resetCycle: 'monthly', nodeGroup: 'default', visibility: 'public', renewable: true, upgradable: true, active: true, sortOrder: 1 }, prices: [{ id: 'starter-month', planId: 'starter', billingPeriod: 'monthly', months: 1, amountFen: 1000, active: true }] },
    { plan: { id: 'pro', slug: 'pro', name: '专业套餐', description: '适合多设备的日常稳定使用', trafficBytes: 300 * GB, deviceLimit: 5, resetCycle: 'monthly', nodeGroup: 'default', visibility: 'public', renewable: true, upgradable: true, active: true, sortOrder: 2 }, prices: [{ id: 'pro-month', planId: 'pro', billingPeriod: 'monthly', months: 1, amountFen: 2000, active: true }] },
    { plan: { id: 'ultimate', slug: 'ultimate', name: '旗舰套餐', description: '适合多设备与高流量场景', trafficBytes: 1024 * GB, deviceLimit: 10, resetCycle: 'monthly', nodeGroup: 'default', visibility: 'public', renewable: true, upgradable: true, active: true, sortOrder: 3 }, prices: [{ id: 'ultimate-month', planId: 'ultimate', billingPeriod: 'monthly', months: 1, amountFen: 4500, active: true }] },
  ],
  paymentMethods: [{ code: 'alipay-demo', name: '演示支付' }],
  notices: [{ id: 'welcome', slug: 'welcome', level: 'info', title: '服务公告', content: '欢迎使用 NOVA。选购前可先查看套餐说明与使用文档，开通后重要信息会集中显示在账户中。' }],
  articles: [
    { id: 'windows', slug: 'windows-import', category: 'Windows', title: 'Windows 导入教程', content: '安装 v2rayN 或 Clash Verge Rev，复制订阅链接，在客户端中选择从剪贴板导入并更新订阅。' },
    { id: 'macos', slug: 'macos-import', category: 'macOS', title: 'macOS 导入教程', content: '安装 Clash Verge Rev，添加远程配置并粘贴订阅链接。' },
    { id: 'mobile', slug: 'mobile-import', category: 'Android / iOS', title: '移动端导入教程', content: '在官方客户端中添加远程配置，粘贴订阅链接或扫描二维码。' },
  ],
  applications: [
    { id: 'v2rayn', slug: 'v2rayn', name: 'v2rayN', platform: 'Windows / macOS / Linux', officialUrl: 'https://github.com/2dust/v2rayN/releases', sourceUrl: 'https://github.com/2dust/v2rayN', description: '适合桌面设备使用，支持订阅导入与更新' },
    { id: 'clash-verge', slug: 'clash-verge-rev', name: 'Clash Verge Rev', platform: 'Windows / macOS / Linux', officialUrl: 'https://github.com/clash-verge-rev/clash-verge-rev/releases', sourceUrl: 'https://github.com/clash-verge-rev/clash-verge-rev', description: '界面清晰的跨平台桌面客户端' },
    { id: 'shadowrocket', slug: 'shadowrocket', name: 'Shadowrocket', platform: 'iOS', officialUrl: 'https://apps.apple.com/us/app/shadowrocket/id932747118', description: '从 App Store 获取正版客户端' },
  ],
};

const fallbackBootstrapEN: GuestBootstrap = {
  site: { siteName: 'NOVA', siteTagline: 'Secure access, simple setup' },
  plans: [
    { plan: { ...fallbackBootstrapZH.plans[0].plan, name: 'Starter', description: 'For light browsing and occasional use' }, prices: fallbackBootstrapZH.plans[0].prices },
    { plan: { ...fallbackBootstrapZH.plans[1].plan, name: 'Professional', description: '300 GB monthly traffic with high-speed nodes' }, prices: fallbackBootstrapZH.plans[1].prices },
    { plan: { ...fallbackBootstrapZH.plans[2].plan, name: 'Ultimate', description: 'For multiple devices and high-traffic use' }, prices: fallbackBootstrapZH.plans[2].prices },
  ],
  paymentMethods: [{ code: 'alipay-demo', name: 'Demo payment' }],
  notices: [{ id: 'welcome', slug: 'welcome', level: 'info', title: 'Service notice', content: 'New high-speed nodes are now available. Refresh your subscription in the app to use them.' }],
  articles: [
    { id: 'windows', slug: 'windows-import', category: 'Windows', title: 'Windows setup guide', content: 'Install v2rayN or Clash Verge Rev, copy the subscription link, import it from the clipboard, then refresh the profile.' },
    { id: 'macos', slug: 'macos-import', category: 'macOS', title: 'macOS setup guide', content: 'Install Clash Verge Rev, add a remote profile, and paste your subscription link.' },
    { id: 'mobile', slug: 'mobile-import', category: 'Android / iOS', title: 'Mobile setup guide', content: 'Add a remote profile in a supported app, then paste the subscription link or scan the QR code.' },
  ],
  applications: [
    { ...fallbackBootstrapZH.applications[0], description: 'Desktop app with support for Xray and sing-box' },
    { ...fallbackBootstrapZH.applications[1], description: 'Cross-platform desktop app powered by Mihomo' },
    { ...fallbackBootstrapZH.applications[2], description: 'Get the official app from the App Store' },
  ],
};

const fallbackTermsContentZH = `1. 请妥善保管账户密码和专属订阅链接。
2. 不得利用服务从事违法活动、攻击、欺诈、垃圾信息发送或其他滥用行为。
3. 套餐流量、有效期、设备限制和重置周期以购买页面为准。
4. 数字订阅支付成功后自动开通，款项一经确认不可撤销或退回。
5. 违反使用条款的账户可能被限制、暂停或关闭。`;

const fallbackTermsContentEN = `1. Keep your account password and private subscription link secure.
2. Do not use the service for unlawful activity, attacks, fraud, spam, or other abuse.
3. Plan traffic, validity, device limits, and reset periods follow the purchase page.
4. Digital subscriptions are activated after payment; confirmed payments are final and cannot be reversed.
5. Accounts that violate these terms may be limited, suspended, or closed.`;

Object.assign(fallbackBootstrapZH.site, {
  termsTitle: '服务使用条款',
  termsContent: fallbackTermsContentZH,
  termsVersion: 'preview-terms-v1',
});
Object.assign(fallbackBootstrapEN.site, {
  termsTitle: 'Terms of Service',
  termsContent: fallbackTermsContentEN,
  termsVersion: 'preview-terms-v1',
});

function fallbackBootstrapForLocale(locale: PortalLocale): GuestBootstrap {
  return locale === 'zh-CN' ? fallbackBootstrapZH : fallbackBootstrapEN;
}

function previewDashboardForLocale(locale: PortalLocale): Dashboard {
  const localizedBootstrap = fallbackBootstrapForLocale(locale);
  return {
    customer: { id: 'preview', email: 'member@gmail.com', displayName: locale === 'zh-CN' ? 'NOVA 用户' : 'NOVA Member', locale, status: 'active', balanceFen: 2860, inviteCode: 'NOVA2026' },
    subscription: {
      entitlement: { id: 'preview-entitlement', planId: localizedBootstrap.plans[1].plan.id, status: 'active', trafficQuota: 300 * GB, trafficUsed: 68.4 * GB, deviceLimit: 5, startsAt: '2026-07-01T00:00:00Z', expiresAt: '2026-08-17T00:00:00Z' },
      plan: localizedBootstrap.plans[1].plan,
      usedBytes: 68.4 * GB,
      links: { raw: 'https://subscribe.example.com/sub/preview-private-token', clash: 'https://subscribe.example.com/clash/preview-private-token', json: 'https://subscribe.example.com/json/preview-private-token' },
    },
    invitation: { enabled: true, inviteCode: 'NOVA2026', directInviteCount: 3, commissionPercent: 10, pendingFen: 860, confirmedFen: 0, settledFen: 2000, commissionFirstPaymentOnly: true, inviteCodesNeverExpire: true },
    notices: localizedBootstrap.notices,
    orders: [],
  };
}

function initialLocale(): PortalLocale {
  const saved = localStorage.getItem('nova-locale') as PortalLocale | null;
  if (saved && localeOptions.some((item) => item.value === saved)) return saved;
  const candidate = navigator.language.toLowerCase();
  const exact = localeOptions.find((item) => item.value.toLowerCase() === candidate);
  if (exact) return exact.value;
  const language = candidate.split('-')[0];
  return localeOptions.find((item) => item.value.toLowerCase().startsWith(`${language}-`))?.value || 'zh-CN';
}

function formatBytes(value: number): string {
  if (value >= 1024 * GB) return `${(value / (1024 * GB)).toFixed(1)} TB`;
  if (value >= GB) return `${(value / GB).toFixed(1)} GB`;
  return `${(value / 1024 ** 2).toFixed(0)} MB`;
}

function formatDate(value?: string, locale: PortalLocale = 'zh-CN'): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value));
}

function formatDateTime(value?: string, locale: PortalLocale = 'zh-CN'): string {
  if (!value) return '—';
  return new Intl.DateTimeFormat(locale, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

function billingLabel(period: string, months: number, locale: PortalLocale): string {
  const zh: Record<string, string> = { monthly: '月付', quarterly: '季付', half_yearly: '半年付', yearly: '年付', multi_year: `${months} 个月`, one_time: '一次性' };
  const en: Record<string, string> = { monthly: 'Monthly', quarterly: 'Quarterly', half_yearly: 'Half-year', yearly: 'Yearly', multi_year: `${months} months`, one_time: 'One-time' };
  return (locale === 'zh-CN' || locale === 'zh-TW' ? zh : en)[period] || period;
}

function buildInvitationLink(inviteCode: string): string {
  const url = new URL(window.location.href);
  url.hash = '';
  url.search = '';
  url.searchParams.set('invite', inviteCode);
  return url.toString();
}

function PortalContent() {
  const { message } = AntApp.useApp();
  const [locale, setLocale] = useState<PortalLocale>(() => initialLocale());
  const [section, setSection] = useState<Section>('home');
  const [accountTab, setAccountTab] = useState<AccountTab>('overview');
  const [bootstrap, setBootstrap] = useState<GuestBootstrap>(() => fallbackBootstrapForLocale(initialLocale()));
  const [bootstrapError, setBootstrapError] = useState('');
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [authOpen, setAuthOpen] = useState(false);
  const [authMode, setAuthMode] = useState<AuthMode>('login');
  const [authBusy, setAuthBusy] = useState(false);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [termsError, setTermsError] = useState(false);
  const [termsOpen, setTermsOpen] = useState(false);
  const [turnstileToken, setTurnstileToken] = useState('');
  const [payment, setPayment] = useState<PaymentPayload | null>(null);
  const [paymentOrderID, setPaymentOrderID] = useState('');
  const [paymentBusy, setPaymentBusy] = useState(false);
  const [qrOpen, setQROpen] = useState(false);
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selectedPurchase, setSelectedPurchase] = useState<{ priceID: string; couponCode: string; action: PurchaseAction; entitlementID: string } | null>(null);
  const [purchaseOpen, setPurchaseOpen] = useState(false);
  const [useBalance, setUseBalance] = useState(true);
  const [selectedNotice, setSelectedNotice] = useState<GuestBootstrap['notices'][number] | null>(null);
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [ticketMessages, setTicketMessages] = useState<TicketMessage[]>([]);
  const [authForm] = Form.useForm();
  const [ticketForm] = Form.useForm<{ subject: string; body: string }>();
  const [replyForm] = Form.useForm<{ body: string }>();
  const [giftForm] = Form.useForm<{ code: string }>();
  const [passwordForm] = Form.useForm<{ currentPassword: string; newPassword: string; confirmPassword: string }>();
  const [passwordBusy, setPasswordBusy] = useState(false);
  const copy = portalCopies[locale];
  const fallbackSite = fallbackBootstrapForLocale(locale).site;
  const currentTermsTitle = bootstrap.site.termsTitle || fallbackSite.termsTitle || copy.termsOfService;
  const currentTermsContent = bootstrap.site.termsContent || fallbackSite.termsContent || copy.termsOfService;
  const currentTermsVersion = bootstrap.site.termsVersion || fallbackSite.termsVersion || '';
  const rtl = locale === 'ar-EG' || locale === 'fa-IR';
  const preview = new URLSearchParams(window.location.search).get('preview') === 'design';
  const invitedByURL = new URLSearchParams(window.location.search).get('invite')?.trim().toUpperCase() || '';
  const currencySymbol = bootstrap.site.currencySymbol || '¥';
  const registrationClosed = bootstrap.site.registrationClosed === 'true';
  const emailVerification = bootstrap.site.emailVerification !== 'false';
  const forcedInvitation = bootstrap.site.forcedInvitation === 'true';
  const allowUserSubscriptionChange = bootstrap.site.allowUserSubscriptionChange !== 'false';
  const allowedEmailSuffixes = (bootstrap.site.allowedEmailSuffixes || 'gmail.com').split(',').map((value) => value.trim()).filter(Boolean);
  const availablePaymentMethods = bootstrap.paymentMethods || [];
  const currentPaymentMethodName = payment
    ? availablePaymentMethods.find((method) => method.code === payment.intent.provider || (method.code === 'alipay_f2f' && payment.intent.provider === 'alipay'))?.name || copy.paymentTitle
    : copy.paymentTitle;

  const changeAuthMode = useCallback((mode: AuthMode) => {
    setAuthMode(mode);
    setTurnstileToken('');
    setTermsAccepted(false);
    setTermsError(false);
    authForm.resetFields(['password', 'newPassword', 'confirmPassword', 'code', 'inviteCode']);
  }, [authForm]);

  useEffect(() => {
    if (preview || bootstrap.site.forceHttps !== 'true' || window.location.protocol === 'https:') return;
    if (['localhost', '127.0.0.1', '::1'].includes(window.location.hostname)) return;
    const configured = bootstrap.site.siteUrl;
    if (configured?.startsWith('https://')) {
      window.location.replace(configured.replace(/\/$/, '') + window.location.pathname + window.location.search + window.location.hash);
      return;
    }
    const target = new URL(window.location.href);
    target.protocol = 'https:';
    window.location.replace(target.toString());
  }, [bootstrap.site.forceHttps, bootstrap.site.siteUrl, preview]);

  useEffect(() => {
    if (registrationClosed && authMode === 'register') setAuthMode('login');
  }, [authMode, registrationClosed]);

  useEffect(() => {
    if (!invitedByURL || registrationClosed || dashboard) return;
    authForm.setFieldValue('inviteCode', invitedByURL);
    setAuthMode('register');
    setAuthOpen(true);
  }, [authForm, dashboard, invitedByURL, registrationClosed]);

  const refreshDashboard = useCallback(async () => {
    if (preview) {
      setDashboard(previewDashboardForLocale(locale));
      return;
    }
    try {
      setDashboard(await portalRequest<Dashboard>('/api/v1/user/dashboard'));
    } catch (error) {
      if (!(error instanceof PortalApiError) || error.status !== 401) {
        message.error(error instanceof Error ? error.message : 'Unable to load account');
      }
      setDashboard(null);
    }
  }, [locale, message, preview]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setBootstrapError('');
    portalRequest<GuestBootstrap>(`/api/v1/guest/bootstrap?locale=${encodeURIComponent(locale)}`)
      .then((data) => {
        if (active) setBootstrap(data);
      })
      .catch((error) => {
        if (active) {
          const fallback = fallbackBootstrapForLocale(locale);
          setBootstrap(preview ? fallback : { ...fallback, plans: [], notices: [], articles: [], applications: [] });
          setBootstrapError(error instanceof Error ? error.message : 'Service unavailable');
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    refreshDashboard();
    return () => {
      active = false;
    };
  }, [locale, preview, refreshDashboard]);

  useEffect(() => {
    document.documentElement.lang = locale;
    document.documentElement.dir = rtl ? 'rtl' : 'ltr';
    localStorage.setItem('nova-locale', locale);
  }, [locale, rtl]);

  const nav = buildPortalNavigation<Section>(copy, Boolean(dashboard));

  const submitAuth = async (values: Record<string, string>) => {
    if (authMode === 'register' && !termsAccepted) {
      setTermsError(true);
      message.warning(copy.termsRequired);
      return;
    }
    setAuthBusy(true);
    try {
      if (authMode === 'login') {
        await portalRequest('/api/v1/passport/login', { method: 'POST', body: JSON.stringify({ email: values.email, password: values.password }) });
      } else if (authMode === 'register') {
        await portalRequest('/api/v1/passport/register', { method: 'POST', body: JSON.stringify({ email: values.email, password: values.password, code: values.code || '', inviteCode: values.inviteCode || '', locale, turnstileToken, acceptedTerms: termsAccepted, termsVersion: currentTermsVersion }) });
      } else {
        await portalRequest('/api/v1/passport/reset-password', { method: 'POST', body: JSON.stringify({ email: values.email, password: values.newPassword, code: values.code }) });
        setAuthMode('login');
        authForm.resetFields(['password', 'newPassword', 'confirmPassword', 'code']);
        message.success(copy.resetSuccess);
        return;
      }
      setAuthOpen(false);
      await refreshDashboard();
      message.success(authMode === 'register' ? copy.registrationSuccess : copy.loginSuccess);
      if (selectedPurchase) await purchase(selectedPurchase.priceID, selectedPurchase.couponCode, true, selectedPurchase.action, selectedPurchase.entitlementID);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Request failed');
    } finally {
      setAuthBusy(false);
    }
  };

  const sendCode = async () => {
    const email = authForm.getFieldValue('email') as string | undefined;
    if (!email) {
      message.warning(copy.email);
      return;
    }
    if (bootstrap.site.turnstileSiteKey && !turnstileToken) {
      message.warning(locale === 'zh-CN' ? '请先完成人机验证' : 'Please complete the security check');
      return;
    }
    setAuthBusy(true);
    try {
      const result = await portalRequest<{ sent: boolean; debugCode?: string }>('/api/v1/passport/send-code', { method: 'POST', body: JSON.stringify({ email, purpose: authMode === 'reset' ? 'reset' : 'register', turnstileToken }) });
      if (result.debugCode) authForm.setFieldValue('code', result.debugCode);
      message.success(result.debugCode ? `${copy.code}: ${result.debugCode}` : copy.codeSent);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to send code');
    } finally {
      setAuthBusy(false);
    }
  };

  const purchase = async (priceID: string, couponCode = '', authenticated = false, action: PurchaseAction = 'purchase', entitlementID = '') => {
    setSelectedPurchase({ priceID, couponCode, action, entitlementID });
    if (!dashboard && !authenticated) {
      setAuthMode('login');
      setAuthOpen(true);
      return;
    }
    setUseBalance(true);
    setPurchaseOpen(true);
  };

  const confirmPurchase = async () => {
    if (!selectedPurchase) return;
    setPaymentBusy(true);
    try {
      const order = await portalRequest<Order>('/api/v1/user/orders', { method: 'POST', body: JSON.stringify({ planPriceId: selectedPurchase.priceID, orderKind: selectedPurchase.action, entitlementId: selectedPurchase.entitlementID, couponCode: selectedPurchase.couponCode, useBalance }) });
      setPurchaseOpen(false);
      setSelectedPurchase(null);
      if (order.payableFen === 0 && order.status !== 'pending') {
        await refreshDashboard();
        message.success(locale === 'zh-CN' ? '订单已支付，系统正在自动开通。' : 'Order paid. Provisioning has started.');
      } else {
        setPaymentOrderID(order.id);
        await refreshDashboard();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to create order');
    } finally {
      setPaymentBusy(false);
    }
  };

  const payOrder = async (orderID: string) => {
    setPayment(null);
    setPaymentOrderID(orderID);
  };

  const beginPayment = async (provider: string) => {
    if (!paymentOrderID) return;
    setPaymentBusy(true);
    try {
      const payload = await portalRequest<PaymentPayload>(`/api/v1/user/orders/${paymentOrderID}/pay`, { method: 'POST', body: JSON.stringify({ provider }) });
      setPayment(payload);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to continue payment');
    } finally { setPaymentBusy(false); }
  };

  const cancelOrder = async (orderID: string) => {
    try {
      await portalRequest(`/api/v1/user/orders/${orderID}/cancel`, { method: 'POST', body: '{}' });
      await refreshDashboard();
      message.success(copy.orderCancelled);
    } catch (error) { message.error(error instanceof Error ? error.message : 'Unable to cancel order'); }
  };

  const confirmDemoPayment = async () => {
    if (!paymentOrderID) return;
    setPaymentBusy(true);
    try {
      await portalRequest(`/api/v1/user/orders/${paymentOrderID}/demo-pay`, { method: 'POST', body: '{}' });
      setPayment(null);
      setPaymentOrderID('');
      await refreshDashboard();
      message.success(copy.demoPaymentSuccess);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Payment is pending');
    } finally {
      setPaymentBusy(false);
    }
  };

  const copyLink = async () => {
    const value = dashboard?.subscription?.links.raw;
    if (!value) return;
    await navigator.clipboard.writeText(value);
    message.success(copy.linkCopied);
  };

  const rotateSubscription = async () => {
    try {
      await portalRequest('/api/v1/user/subscription/rotate', { method: 'POST', body: '{}' });
      await refreshDashboard();
      message.success(copy.linkRotated);
    } catch (error) { message.error(error instanceof Error ? error.message : 'Unable to rotate subscription'); }
  };

  const logout = async () => {
    try {
      await portalRequest('/api/v1/passport/logout', { method: 'POST', body: '{}' });
    } finally {
      setDashboard(null);
      setSection('home');
    }
  };

  const loadTickets = useCallback(async () => {
    if (!dashboard) return;
    try {
      setTickets(await portalRequest<Ticket[]>('/api/v1/user/tickets'));
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load tickets');
    }
  }, [dashboard, message]);

  useEffect(() => {
    if (section === 'tickets') loadTickets();
  }, [section, loadTickets]);

  const submitTicket = async (values: { subject: string; body: string }) => {
    try {
      await portalRequest('/api/v1/user/tickets', { method: 'POST', body: JSON.stringify(values) });
      ticketForm.resetFields();
      await loadTickets();
      message.success(copy.ticketSubmitted);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to create ticket');
    }
  };

  const openTicket = async (ticket: Ticket) => {
    setSelectedTicket(ticket);
    try { setTicketMessages(await portalRequest<TicketMessage[]>(`/api/v1/user/tickets/${ticket.id}/messages`)); }
    catch (error) { message.error(error instanceof Error ? error.message : 'Unable to load ticket'); }
  };

  const replyTicket = async ({ body }: { body: string }) => {
    if (!selectedTicket) return;
    try {
      await portalRequest(`/api/v1/user/tickets/${selectedTicket.id}/reply`, { method: 'POST', body: JSON.stringify({ body }) });
      replyForm.resetFields();
      setTicketMessages(await portalRequest<TicketMessage[]>(`/api/v1/user/tickets/${selectedTicket.id}/messages`));
      await loadTickets();
    } catch (error) { message.error(error instanceof Error ? error.message : 'Unable to reply'); }
  };

  const changePassword = async (values: { currentPassword: string; newPassword: string; confirmPassword: string }) => {
    setPasswordBusy(true);
    try {
      await portalRequest('/api/v1/user/account/password', {
        method: 'POST',
        body: JSON.stringify({ currentPassword: values.currentPassword, newPassword: values.newPassword }),
      });
      passwordForm.resetFields();
      message.success(copy.passwordChanged);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to change password');
    } finally {
      setPasswordBusy(false);
    }
  };

  const redeemGiftCard = async ({ code }: { code: string }) => {
    try {
      await portalRequest('/api/v1/user/gift-cards/redeem', { method: 'POST', body: JSON.stringify({ code }) });
      giftForm.resetFields();
      await refreshDashboard();
      message.success(copy.redeemedSuccess);
    } catch (error) { message.error(error instanceof Error ? error.message : 'Unable to redeem gift card'); }
  };

  const copyInvitationLink = async () => {
    const code = dashboard?.invitation?.inviteCode || dashboard?.customer.inviteCode;
    if (!code) return;
    await navigator.clipboard.writeText(buildInvitationLink(code));
    message.success(copy.invitationCopied);
  };

  const profileItems: MenuProps['items'] = dashboard ? [
    { key: 'account', icon: <UserOutlined />, label: copy.accountCenter },
    { key: 'invitation', icon: <TeamOutlined />, label: copy.invitationRewards },
    { key: 'orders', icon: <ShoppingCartOutlined />, label: copy.orders },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: copy.signOut, danger: true },
  ] : [{ key: 'login', icon: <LoginOutlined />, label: copy.signIn }];

  const profileAction: MenuProps['onClick'] = ({ key }) => {
    if (key === 'logout') logout();
    else if (key === 'orders') setSection('orders');
    else if (key === 'account' || key === 'invitation') {
      setAccountTab(key === 'invitation' ? 'invitation' : 'overview');
      setSection('account');
    }
    else setAuthOpen(true);
  };

  const page = (() => {
    if (section === 'home') return <HomeView copy={copy} locale={locale} bootstrap={bootstrap} dashboard={dashboard} currencySymbol={currencySymbol} onSection={setSection} onAccount={(tab) => { setAccountTab(tab); setSection('account'); }} onBuy={purchase} paymentBusy={paymentBusy} />;
    if (section === 'subscription') return <SubscriptionView copy={copy} locale={locale} dashboard={dashboard} allowUserSubscriptionChange={allowUserSubscriptionChange} onCopy={copyLink} onQR={() => setQROpen(true)} onRotate={rotateSubscription} onPlans={() => setSection('plans')} onSection={setSection} />;
    if (section === 'plans') return <PlansView copy={copy} locale={locale} plans={bootstrap.plans} subscription={dashboard?.subscription} allowUserSubscriptionChange={allowUserSubscriptionChange} currencySymbol={currencySymbol} onBuy={purchase} paymentBusy={paymentBusy} />;
    if (section === 'guides') return <GuidesView copy={copy} articles={bootstrap.articles} applications={bootstrap.applications} onSection={setSection} telegramGroupLink={bootstrap.site.telegramGroupLink} />;
    if (section === 'tickets') return <TicketsView copy={copy} locale={locale} authenticated={Boolean(dashboard)} tickets={tickets} form={ticketForm} telegramGroupLink={bootstrap.site.telegramGroupLink} onSubmit={submitTicket} onOpen={openTicket} onLogin={() => setAuthOpen(true)} />;
    if (section === 'account') return <AccountView copy={copy} activeTab={accountTab} onTabChange={setAccountTab} dashboard={dashboard} currencySymbol={currencySymbol} giftForm={giftForm} passwordForm={passwordForm} passwordBusy={passwordBusy} onCopyInvitation={copyInvitationLink} onRedeem={redeemGiftCard} onChangePassword={changePassword} onLogin={() => setAuthOpen(true)} />;
    return <OrdersView copy={copy} locale={locale} orders={dashboard?.orders || []} currencySymbol={currencySymbol} paymentBusy={paymentBusy} onPay={payOrder} onCancel={cancelOrder} />;
  })();

  const selectedPurchaseDetails = selectedPurchase
    ? bootstrap.plans.flatMap((item) => item.prices.map((price) => ({ plan: item.plan, price }))).find((item) => item.price.id === selectedPurchase.priceID)
    : undefined;
  const purchaseText = locale === 'zh-CN'
    ? { title: '确认购买', description: '创建订单前，请确认套餐与付款方式。', useBalance: '使用账户余额抵扣', available: '可用余额' }
    : locale === 'zh-TW'
      ? { title: '確認購買', description: '建立訂單前，請確認方案與付款方式。', useBalance: '使用帳戶餘額折抵', available: '可用餘額' }
      : { title: 'Confirm purchase', description: 'Review the plan and payment method before creating the order.', useBalance: 'Use account balance', available: 'Available balance' };
  const purchaseActionText = selectedPurchase?.action === 'renewal'
    ? { title: copy.renew, description: copy.renewalDescription, confirm: copy.renew }
    : selectedPurchase?.action === 'upgrade'
      ? { title: copy.upgrade, description: copy.upgradeDescription, confirm: copy.upgrade }
      : { title: purchaseText.title, description: purchaseText.description, confirm: copy.buyNow };
  const footerText = locale === 'zh-CN'
    ? { promise: '清晰的套餐信息、可靠的账户管理与持续可查的服务记录。', navigation: '服务导航', terms: '使用条款', privacy: '隐私政策', copyright: '服务信息以账户与订单页面显示为准' }
    : locale === 'zh-TW'
      ? { promise: '清楚的方案資訊、可靠的帳戶管理與持續可查的服務記錄。', navigation: '服務導覽', terms: '使用條款', privacy: '隱私政策', copyright: '服務資訊以帳戶與訂單頁面顯示為準' }
      : { promise: 'Clear plan details, dependable account controls and service records you can revisit.', navigation: 'Service navigation', terms: 'Terms of service', privacy: 'Privacy policy', copyright: 'Account and order pages show the current service details' };

  return (
    <ConfigProvider locale={antdLocales[locale]} direction={rtl ? 'rtl' : 'ltr'}>
    <div className="portal-shell" dir={rtl ? 'rtl' : 'ltr'}>
      <header className="portal-header">
        <button type="button" className="portal-brand" onClick={() => setSection('home')} aria-label={`${bootstrap.site.siteName || 'NOVA'} home`}>
          {bootstrap.site.logoUrl && <img className="portal-brand-logo" src={bootstrap.site.logoUrl} alt="" />}
          <span>{bootstrap.site.siteName || 'NOVA'}</span>
        </button>
        <nav className="portal-nav" aria-label="Primary navigation">
          {nav.map((item) => <button type="button" key={item.key} className={section === item.key ? 'is-active' : ''} onClick={() => setSection(item.key)}>{item.label}</button>)}
        </nav>
        <div className="portal-actions">
          {dashboard && <button type="button" className="portal-balance-quick" onClick={() => { setAccountTab('overview'); setSection('account'); }}><WalletOutlined /><span>{currencySymbol} {(dashboard.customer.balanceFen / 100).toFixed(2)}</span></button>}
          <Select aria-label="Language" className="portal-language" value={locale} onChange={setLocale} options={localeOptions} optionLabelProp="shortLabel" popupMatchSelectWidth={false} />
          <Dropdown menu={{ items: profileItems, onClick: profileAction }} trigger={['click']}>
            <button type="button" className="portal-profile" aria-label="Account menu">
              <Badge dot={Boolean(dashboard)} color="#35c48d"><Avatar size={32}>{(dashboard?.customer.displayName || dashboard?.customer.email || 'N').slice(0, 1).toUpperCase()}</Avatar></Badge>
              <span className="portal-profile-name">{dashboard?.customer.displayName || copy.signIn}</span>
            </button>
          </Dropdown>
        </div>
      </header>
      {bootstrap.notices[0] && (
        <button type="button" className="portal-announcement" onClick={() => setSelectedNotice(bootstrap.notices[0])}>
          <BellOutlined />
          <span><strong>{bootstrap.notices[0].title}</strong>：{bootstrap.notices[0].content}</span>
          <span className="portal-announcement-link">{copy.announcementDetails}</span>
        </button>
      )}
      <main className="portal-main">
        {bootstrapError && !preview && <Alert type="error" showIcon title={locale === 'zh-CN' ? '服务数据加载失败，暂时无法购买套餐' : 'Service data could not be loaded. Purchases are temporarily unavailable.'} description={bootstrapError} className="portal-service-error" />}
        {loading ? <div className="portal-loading"><Spin size="large" /></div> : page}
      </main>

      <footer className="portal-footer">
        <div className="portal-footer-inner">
          <div className="portal-footer-brand">
            <span>{footerText.promise}</span>
          </div>
          <div className="portal-footer-links" aria-label={footerText.navigation}>
            <div>
              <button type="button" onClick={() => setSection('plans')}>{copy.plans}</button>
              <button type="button" onClick={() => setSection('guides')}>{copy.guides}</button>
              <button type="button" onClick={() => setSection('tickets')}>{copy.tickets}</button>
              {bootstrap.site.termsUrl && <a href={bootstrap.site.termsUrl} target="_blank" rel="noopener noreferrer">{footerText.terms}</a>}
              {bootstrap.site.privacyUrl && <a href={bootstrap.site.privacyUrl} target="_blank" rel="noopener noreferrer">{footerText.privacy}</a>}
            </div>
          </div>
        </div>
        <div className="portal-footer-meta">© {new Date().getFullYear()} {bootstrap.site.siteName || 'NOVA'} · {footerText.copyright}</div>
      </footer>

      {bootstrap.site.telegramGroupLink && <a className="telegram-support-fab" href={bootstrap.site.telegramGroupLink} target="_blank" rel="noopener noreferrer" referrerPolicy="no-referrer" aria-label={copy.telegramSupport} title={copy.telegramSupport}><SendOutlined /></a>}

      <Modal className="portal-auth-modal" open={authOpen} onCancel={() => setAuthOpen(false)} footer={null} width={450} centered title={null} destroyOnHidden>
        <div className="auth-modal-header">
          {bootstrap.site.logoUrl
            ? <img className="auth-site-logo" src={bootstrap.site.logoUrl} alt="" />
            : <span className="portal-brand-icon"><SafetyCertificateFilled /></span>}
          <Title level={3}>{bootstrap.site.siteName || 'NOVA'}</Title>
        </div>
        {authMode === 'reset'
          ? <div className="auth-reset-heading"><Text strong>{copy.reset}</Text><Button type="link" onClick={() => changeAuthMode('login')}>{copy.login}</Button></div>
          : <Tabs activeKey={authMode} onChange={(key) => changeAuthMode(key as AuthMode)} centered items={[{ key: 'login', label: copy.login }, ...(registrationClosed ? [] : [{ key: 'register', label: copy.register }])]} />}
        {authMode === 'reset' && <Paragraph type="secondary" className="auth-mode-description">{copy.resetDescription}</Paragraph>}
        <Form form={authForm} layout="vertical" onFinish={submitAuth} requiredMark={false}>
          <Form.Item name="email" label={authMode === 'login' ? copy.emailOrAdmin : copy.email} rules={authMode === 'login' ? [{ required: true }] : [{ required: true }, { type: 'email' }]}><Input size="large" autoComplete="username" placeholder={authMode === 'login' ? copy.emailOrAdmin : `name@${allowedEmailSuffixes[0] || 'gmail.com'}`} /></Form.Item>
          {authMode !== 'reset' && <Form.Item name="password" label={copy.password} extra={authMode === 'register' ? copy.passwordRule : undefined} rules={authMode === 'login' ? [{ required: true }] : [{ required: true }, { min: 10 }, { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/, message: copy.passwordRule }]}><Input.Password size="large" autoComplete={authMode === 'login' ? 'current-password' : 'new-password'} /></Form.Item>}
          {authMode === 'login' && <div className="auth-forgot-row"><Button type="link" className="auth-forgot-link" onClick={() => changeAuthMode('reset')}>{copy.reset}</Button></div>}
          {authMode === 'reset' && <Form.Item name="newPassword" label={copy.newPassword} extra={copy.passwordRule} rules={[{ required: true }, { min: 10 }, { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/, message: copy.passwordRule }]}><Input.Password size="large" autoComplete="new-password" /></Form.Item>}
          {authMode !== 'login' && <Form.Item name="confirmPassword" label={copy.confirmPassword} dependencies={[authMode === 'reset' ? 'newPassword' : 'password']} rules={[{ required: true }, ({ getFieldValue }) => ({ validator(_, value) { const original = getFieldValue(authMode === 'reset' ? 'newPassword' : 'password'); return !value || original === value ? Promise.resolve() : Promise.reject(new Error(copy.passwordMismatch)); } })]}><Input.Password size="large" autoComplete="new-password" /></Form.Item>}
          {(authMode === 'reset' || (authMode === 'register' && emailVerification)) && <Form.Item label={copy.code} required><Space.Compact block><Form.Item name="code" noStyle rules={[{ required: true }, { len: 6 }]}><Input size="large" maxLength={6} inputMode="numeric" /></Form.Item><Button size="large" onClick={sendCode} loading={authBusy}>{copy.sendCode}</Button></Space.Compact></Form.Item>}
          {authMode !== 'login' && bootstrap.site.turnstileSiteKey && <TurnstileWidget siteKey={bootstrap.site.turnstileSiteKey} onToken={setTurnstileToken} />}
          {authMode === 'register' && <Form.Item name="inviteCode" label={forcedInvitation ? (locale === 'zh-CN' ? '邀请码（必填）' : 'Invitation code (required)') : copy.inviteOptional} rules={forcedInvitation ? [{ required: true }] : undefined}><Input size="large" /></Form.Item>}
          {authMode === 'register' && <div className={`auth-terms-consent${termsError ? ' has-error' : ''}`}><div><Checkbox checked={termsAccepted} onChange={(event) => { setTermsAccepted(event.target.checked); if (event.target.checked) setTermsError(false); }}>{copy.acceptTermsPrefix}</Checkbox><Button type="link" onClick={() => setTermsOpen(true)}>{copy.termsOfService}</Button></div>{termsError && <Text type="danger">{copy.termsRequired}</Text>}</div>}
          <Button
            type="primary"
            size="large"
            htmlType={authMode === 'register' ? 'button' : 'submit'}
            block
            loading={authBusy}
            onClick={authMode === 'register' ? () => {
              if (!termsAccepted) {
                setTermsError(true);
                message.warning(copy.termsRequired);
                return;
              }
              authForm.submit();
            } : undefined}
          >{copy.submit}</Button>
        </Form>
      </Modal>

      <Modal
        open={termsOpen}
        onCancel={() => setTermsOpen(false)}
        onOk={() => { setTermsAccepted(true); setTermsError(false); setTermsOpen(false); }}
        okText={copy.acceptTermsAction}
        cancelText={locale === 'zh-CN' ? '关闭' : 'Close'}
        title={currentTermsTitle}
        width={720}
        centered
      >
        <Paragraph className="terms-content">{currentTermsContent}</Paragraph>
        {bootstrap.site.termsUrl && <Button type="link" href={bootstrap.site.termsUrl} target="_blank" rel="noopener noreferrer">{copy.readTerms}</Button>}
      </Modal>

      <Modal open={Boolean(selectedNotice)} onCancel={() => setSelectedNotice(null)} footer={null} title={selectedNotice?.title} centered>
        <Paragraph className="notice-detail-content">{selectedNotice?.content}</Paragraph>
      </Modal>

      <Modal open={purchaseOpen} onCancel={() => { setPurchaseOpen(false); setSelectedPurchase(null); }} onOk={confirmPurchase} okText={purchaseActionText.confirm} confirmLoading={paymentBusy} title={purchaseActionText.title} centered>
        <Paragraph type="secondary">{purchaseActionText.description}</Paragraph>
        {selectedPurchaseDetails && <Descriptions bordered size="small" column={1} items={[{ key: 'plan', label: copy.plans, children: selectedPurchaseDetails.plan.name }, { key: 'period', label: copy.billingPeriod, children: billingLabel(selectedPurchaseDetails.price.billingPeriod, selectedPurchaseDetails.price.months, locale) }, { key: 'amount', label: copy.amount, children: `${currencySymbol} ${(selectedPurchaseDetails.price.amountFen / 100).toFixed(2)}` }, ...(selectedPurchase?.couponCode ? [{ key: 'coupon', label: copy.couponCode, children: selectedPurchase.couponCode }] : [])]} />}
        <Divider />
        <Checkbox checked={useBalance} disabled={!dashboard?.customer.balanceFen} onChange={(event) => setUseBalance(event.target.checked)}>{purchaseText.useBalance}</Checkbox>
        <Text type="secondary" className="purchase-balance">{purchaseText.available}：{currencySymbol} {((dashboard?.customer.balanceFen || 0) / 100).toFixed(2)}</Text>
        <Paragraph type="secondary" className="purchase-balance-rule">{copy.commissionBalanceHint}</Paragraph>
      </Modal>

      <Modal open={Boolean(paymentOrderID && !payment)} onCancel={() => setPaymentOrderID('')} footer={null} width={460} centered title={copy.choosePaymentMethod}>
        <Paragraph type="secondary">{copy.choosePaymentMethodDescription}</Paragraph>
        {availablePaymentMethods.length > 0
          ? <div className="payment-method-list">{availablePaymentMethods.map((method) => <Button key={method.code} size="large" block icon={<CreditCardOutlined />} loading={paymentBusy} onClick={() => beginPayment(method.code)}>{method.name}</Button>)}</div>
          : <Empty description={copy.noPaymentMethods} />}
      </Modal>

      <Modal open={Boolean(payment)} onCancel={() => { setPayment(null); setPaymentOrderID(''); }} footer={null} width={430} centered>
        {payment && <div className="payment-modal"><Title level={3}>{currentPaymentMethodName}</Title><Text type="secondary">{copy.orderLabel} {payment.intent.outTradeNo}</Text><img src={payment.qrImage} alt={`${currentPaymentMethodName} QR code`} /><div className="payment-amount">{currencySymbol} {(payment.intent.amountFen / 100).toFixed(2)}</div><Text type="secondary">{copy.paymentValidUntil} {formatDateTime(payment.intent.expiresAt, locale)}</Text>{payment.intent.provider === 'alipay-demo' && <Button type="primary" block size="large" loading={paymentBusy} onClick={confirmDemoPayment}>{copy.confirmDemoPayment}</Button>}</div>}
      </Modal>

      <Modal open={qrOpen} onCancel={() => setQROpen(false)} footer={null} width={400} centered>
        <div className="payment-modal"><Title level={3}>{copy.subscriptionQRTitle}</Title>{preview ? <QrcodeOutlined className="preview-qr-icon" /> : <img src={portalAsset('/api/v1/user/subscription/qr?format=raw')} alt="Subscription QR code" />}<Text type="secondary">{copy.subscriptionQRWarning}</Text></div>
      </Modal>

      <Modal open={Boolean(selectedTicket)} onCancel={() => setSelectedTicket(null)} footer={null} width={650} title={selectedTicket?.subject} centered>
        <div className="portal-ticket-thread">{ticketMessages.map((item) => <div key={item.id} className={`portal-ticket-message is-${item.senderType}`}><div><Tag color={item.senderType === 'customer' ? 'blue' : 'default'}>{item.senderType === 'customer' ? dashboard?.customer.displayName || dashboard?.customer.email : copy.tickets}</Tag><Text type="secondary">{formatDateTime(item.createdAt, locale)}</Text></div><Paragraph>{item.body}</Paragraph></div>)}</div>
        {selectedTicket?.status !== 'closed' && <Form form={replyForm} layout="vertical" onFinish={replyTicket}><Form.Item name="body" label={copy.reply} rules={[{ required: true }]}><Input.TextArea rows={4} /></Form.Item><Button type="primary" htmlType="submit" icon={<SendOutlined />}>{copy.reply}</Button></Form>}
      </Modal>
    </div>
    </ConfigProvider>
  );
}

function TurnstileWidget({ siteKey, onToken }: { siteKey: string; onToken: (token: string) => void }) {
  const host = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let widgetID = '';
    let cancelled = false;
    const render = () => {
      if (cancelled || !host.current || !window.turnstile) return;
      widgetID = window.turnstile.render(host.current, { sitekey: siteKey, callback: onToken, 'expired-callback': () => onToken(''), theme: 'light' });
    };
    const existing = document.querySelector<HTMLScriptElement>('script[data-nova-turnstile]');
    if (window.turnstile) render();
    else if (existing) existing.addEventListener('load', render, { once: true });
    else {
      const script = document.createElement('script');
      script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
      script.async = true;
      script.defer = true;
      script.dataset.novaTurnstile = 'true';
      script.addEventListener('load', render, { once: true });
      document.head.appendChild(script);
    }
    return () => { cancelled = true; if (widgetID && window.turnstile) window.turnstile.remove(widgetID); };
  }, [onToken, siteKey]);
  return <div ref={host} className="turnstile-host" />;
}

function HomeView({ copy, locale, bootstrap, dashboard, currencySymbol, onSection, onAccount, onBuy, paymentBusy }: { copy: PortalCopy; locale: PortalLocale; bootstrap: GuestBootstrap; dashboard: Dashboard | null; currencySymbol: string; onSection: (section: Section) => void; onAccount: (tab: AccountTab) => void; onBuy: (priceID: string, couponCode?: string) => void; paymentBusy: boolean }) {
  const serviceFacts = [
    { value: '5', label: copy.platformGuides },
    { value: '3', label: copy.subscriptionFormats },
    { value: '24/7', label: copy.selfService },
  ];
  const benefits = [
    { icon: <ThunderboltFilled />, title: copy.fastTitle, description: copy.fastDescription },
    { icon: <SafetyCertificateFilled />, title: copy.privacyTitle, description: copy.privacyDescription },
    { icon: <GlobalOutlined />, title: copy.simpleTitle, description: copy.simpleDescription },
  ];
  const servicePromises = [
    { icon: <CreditCardOutlined />, title: copy.paymentSecureTitle, description: copy.paymentSecureDescription },
    { icon: <ThunderboltFilled />, title: copy.activationTitle, description: copy.activationDescription },
    { icon: <CustomerServiceOutlined />, title: copy.supportCenterTitle, description: copy.responseTimeDescription },
  ];
  return (
    <div className="guest-home">
      {dashboard && <MemberOverview copy={copy} locale={locale} dashboard={dashboard} currencySymbol={currencySymbol} onSection={onSection} onAccount={onAccount} />}
      <section className="home-hero">
        <div className="home-hero-copy">
          <Tag color="blue" icon={<ThunderboltFilled />}>{copy.heroBadge}</Tag>
          <Title>{copy.heroTitle}</Title>
          <Paragraph>{copy.heroDescription}</Paragraph>
          <Space className="home-hero-actions" wrap>
            <Button type="primary" size="large" onClick={() => onSection('plans')}>{copy.browsePlans}</Button>
            <Button size="large" onClick={() => onSection('guides')}>{copy.readGuides}</Button>
          </Space>
        </div>
        <Card className="home-service-card" variant="borderless">
          <div className="home-service-heading">
            <span className="home-service-icon"><GlobalOutlined /></span>
            <div><Title level={3}>{copy.serviceTitle}</Title><Paragraph>{copy.serviceDescription}</Paragraph></div>
          </div>
          <div className="home-service-facts">
            {serviceFacts.map((fact) => <div key={fact.label}><strong>{fact.value}</strong><span>{fact.label}</span></div>)}
          </div>
          <Button type="link" onClick={() => onSection('subscription')}>{copy.manage} <ArrowRightOutlined /></Button>
        </Card>
      </section>

      <section className="home-promise-strip" aria-label={copy.benefitsTitle}>
        {servicePromises.map((promise) => <div key={promise.title}><span>{promise.icon}</span><p><strong>{promise.title}</strong><small>{promise.description}</small></p></div>)}
      </section>

      <section className="home-benefits">
        <div className="home-section-heading"><Title level={2}>{copy.benefitsTitle}</Title><Paragraph>{copy.benefitsDescription}</Paragraph></div>
        <Row gutter={[20, 20]}>
          {benefits.map((benefit) => <Col xs={24} md={8} key={benefit.title}><Card className="home-benefit-card" variant="borderless"><span className="home-benefit-icon">{benefit.icon}</span><Title level={4}>{benefit.title}</Title><Paragraph>{benefit.description}</Paragraph></Card></Col>)}
        </Row>
      </section>

      <PlansView copy={copy} locale={locale} plans={bootstrap.plans} currencySymbol={currencySymbol} onBuy={onBuy} paymentBusy={paymentBusy} compact />
      <QuickStart copy={copy} onSection={onSection} />
    </div>
  );
}

function MemberOverview({ copy, locale, dashboard, currencySymbol, onSection, onAccount }: { copy: PortalCopy; locale: PortalLocale; dashboard: Dashboard; currencySymbol: string; onSection: (section: Section) => void; onAccount: (tab: AccountTab) => void }) {
  const invitation = dashboard.invitation ?? { directInviteCount: 0, commissionPercent: 0 };
  const subscriptionLabel = dashboard.subscription
    ? `${dashboard.subscription.plan.name} · ${formatDate(dashboard.subscription.entitlement.expiresAt, locale)}`
    : copy.noSubscription;
  return <section className="member-overview" aria-label={copy.accountOverview}>
    <button type="button" className="member-summary-card" onClick={() => onAccount('overview')}><span className="summary-icon"><WalletOutlined /></span><span><small>{copy.balance}</small><strong>{currencySymbol} {(dashboard.customer.balanceFen / 100).toFixed(2)}</strong><em>{copy.accountCenter}</em></span></button>
    <button type="button" className="member-summary-card" onClick={() => onSection('subscription')}><span className="summary-icon"><WifiOutlined /></span><span><small>{copy.subscription}</small><strong>{subscriptionLabel}</strong><em>{dashboard.subscription ? copy.manage : copy.buyNow}</em></span></button>
    <button type="button" className="member-summary-card" onClick={() => onAccount('invitation')}><span className="summary-icon"><TeamOutlined /></span><span><small>{copy.invitationRewards}</small><strong>{invitation.directInviteCount} {copy.invitedUsers}</strong><em>{invitation.commissionPercent}% {copy.commissionRate}</em></span></button>
  </section>;
}

function QuickStart({ copy, onSection }: { copy: PortalCopy; onSection: (section: Section) => void }) {
  const items = [
    { icon: <CloudDownloadOutlined />, number: 1, title: copy.chooseClient, description: copy.clientStepDescription },
    { icon: <ImportOutlined />, number: 2, title: copy.importProfile, description: copy.importStepDescription },
    { icon: <WifiOutlined />, number: 3, title: copy.connect, description: copy.connectStepDescription },
  ];
  return (
    <Card className="quick-card" variant="borderless">
      <div className="quick-header"><Title level={3}>{copy.quickStart}</Title><Button type="link" onClick={() => onSection('guides')}>{copy.allGuides} <ArrowRightOutlined /></Button></div>
      <div className="quick-steps">
        {items.map((item, index) => <div className="quick-step-wrap" key={item.number}><button type="button" className="quick-step" onClick={() => onSection('guides')}><span className="quick-step-icon">{item.icon}</span><span className="quick-step-copy"><strong><i>{item.number}</i>{item.title}</strong><small>{item.description}</small></span></button>{index < items.length - 1 && <ArrowRightOutlined className="quick-arrow" />}</div>)}
      </div>
    </Card>
  );
}

function SubscriptionView({ copy, locale, dashboard, allowUserSubscriptionChange, onCopy, onQR, onRotate, onPlans, onSection }: { copy: PortalCopy; locale: PortalLocale; dashboard: Dashboard | null; allowUserSubscriptionChange: boolean; onCopy: () => void; onQR: () => void; onRotate: () => void; onPlans: () => void; onSection: (section: Section) => void }) {
  if (!dashboard?.subscription) return <div className="section-stack empty-subscription-page">
    <Card className="empty-card empty-subscription-hero" variant="borderless"><Empty description={<div className="empty-subscription-copy"><Title level={3}>{copy.noSubscription}</Title><Paragraph>{copy.emptySubscriptionDescription}</Paragraph></div>}><Space wrap><Button type="primary" size="large" onClick={onPlans}>{copy.buyNow}</Button><Button size="large" onClick={() => onSection('guides')}>{copy.readGuides}</Button></Space></Empty></Card>
    <Row gutter={[20, 20]}>
      <Col xs={24} md={8}><InfoCard icon={<ShoppingCartOutlined />} title={copy.browsePlans} description={copy.planHelpDescription} action={copy.buyNow} onClick={onPlans} /></Col>
      <Col xs={24} md={8}><InfoCard icon={<CloudDownloadOutlined />} title={copy.chooseClient} description={copy.clientStepDescription} action={copy.guides} onClick={() => onSection('guides')} /></Col>
      <Col xs={24} md={8}><InfoCard icon={<ImportOutlined />} title={copy.subscriptionReadyTitle} description={copy.subscriptionReadyDescription} action={copy.readGuides} onClick={() => onSection('guides')} /></Col>
    </Row>
    <QuickStart copy={copy} onSection={onSection} />
  </div>;
  const subscription = dashboard.subscription;
  const percent = Math.min(100, Math.round(subscription.usedBytes / subscription.entitlement.trafficQuota * 100));
  return (
    <div className="section-stack subscription-dashboard">
      <section className="dashboard-grid">
        <Card className="plan-card dashboard-card" variant="borderless">
          <div className="customer-greeting"><Title level={1}>{copy.greeting}，{dashboard.customer.displayName || dashboard.customer.email.split('@')[0]}</Title><div><MailOutlined /><span>{dashboard.customer.email}</span><CheckCircleFilled /><strong>{copy.verified}</strong></div></div>
          <Divider className="account-divider" />
          <div className="plan-summary"><div><Text type="secondary">{copy.currentPlan}</Text><strong>{subscription.plan.name}</strong></div><div><Text type="secondary">{copy.status}</Text><strong className="status-active"><CheckCircleFilled /> {copy.active}</strong></div><div><Text type="secondary">{copy.expires}</Text><strong>{formatDate(subscription.entitlement.expiresAt, locale)}</strong></div></div>
          <div className="usage-row">
            <div className="usage-chart"><Progress type="circle" percent={percent} size={300} strokeWidth={7} strokeColor="#176df3" railColor="#edf4fd" format={() => <div className="usage-progress-label"><strong>{formatBytes(subscription.usedBytes)}</strong><span>/ {formatBytes(subscription.entitlement.trafficQuota)}</span><em>{percent}% {copy.used}</em></div>} /><Text type="secondary">{copy.trafficReset}</Text></div>
            <div className="usage-details"><div><Text type="secondary"><i className="usage-dot is-used" />{copy.used}</Text><strong>{formatBytes(subscription.usedBytes)}</strong></div><div><Text type="secondary"><i className="usage-dot" />{copy.totalTraffic}</Text><strong>{formatBytes(subscription.entitlement.trafficQuota)}</strong></div></div>
          </div>
          {allowUserSubscriptionChange && (subscription.plan.renewable || subscription.plan.upgradable) && <Space className="subscription-actions" wrap>{subscription.plan.renewable && <Button type="primary" size="large" onClick={onPlans}>{copy.renew}</Button>}{subscription.plan.upgradable && <Button size="large" onClick={onPlans}>{copy.upgrade}</Button>}</Space>}
        </Card>
        <Card className="import-card dashboard-card" variant="borderless">
          <div className="import-heading"><div className="import-icon"><LinkOutlined /></div><div><Title level={2}>{copy.importToClient}</Title><Paragraph>{copy.importDescription}</Paragraph></div></div>
          <Divider />
          <Text type="secondary" className="subscription-label">{copy.subscriptionLinkLabel}</Text>
          <Space orientation="vertical" size={12} className="import-actions"><Button type="primary" size="large" block icon={<CopyOutlined />} onClick={onCopy}>{copy.copyLink}</Button><Button size="large" block icon={<QrcodeOutlined />} onClick={onQR}>{copy.showQR}</Button><Popconfirm title={copy.rotateConfirm} onConfirm={onRotate}><Button danger block icon={<ReloadOutlined />}>{copy.rotateLink}</Button></Popconfirm></Space>
          <div className="security-note"><SafetyCertificateFilled /><span>{copy.securityNote}</span></div>
        </Card>
      </section>
      <QuickStart copy={copy} onSection={onSection} />
    </div>
  );
}

function InfoCard({ icon, title, description, action, onClick }: { icon: ReactNode; title: string; description: string; action?: string; onClick?: () => void }) {
  return <Card className="portal-info-card" variant="borderless"><span className="portal-info-icon">{icon}</span><Title level={4}>{title}</Title><Paragraph>{description}</Paragraph>{action && onClick && <Button type="link" onClick={onClick}>{action} <ArrowRightOutlined /></Button>}</Card>;
}

function PlansView({ copy, locale, plans, subscription, allowUserSubscriptionChange = true, currencySymbol, onBuy, paymentBusy, compact = false }: { copy: PortalCopy; locale: PortalLocale; plans: PlanCatalogItem[]; subscription?: SubscriptionOverview; allowUserSubscriptionChange?: boolean; currencySymbol: string; onBuy: (priceID: string, couponCode?: string, authenticated?: boolean, action?: PurchaseAction, entitlementID?: string) => void; paymentBusy: boolean; compact?: boolean }) {
  const [selectedPrices, setSelectedPrices] = useState<Record<string, string>>({});
  const [couponCode, setCouponCode] = useState('');
  return (
    <section className={compact ? 'plans-section is-compact' : 'plans-section'}>
      <div className="section-title"><div><Title level={2}>{compact ? copy.plans : copy.planHelpTitle}</Title><Text type="secondary">{compact ? copy.billingNotice : copy.planHelpDescription}</Text></div></div>
      <Row gutter={[20, 20]}>
        {plans.map((item, index) => {
          const price = item.prices.find((candidate) => candidate.id === selectedPrices[item.plan.id]) || item.prices[0];
          const isCurrentPlan = subscription?.plan.id === item.plan.id || subscription?.plan.slug === item.plan.slug;
          const isHigherPlan = Boolean(subscription) && !isCurrentPlan && (
            (item.plan.sortOrder || 0) > (subscription?.plan.sortOrder || 0)
            || item.plan.trafficBytes > (subscription?.plan.trafficBytes || 0)
            || item.plan.deviceLimit > (subscription?.plan.deviceLimit || 0)
          );
          const action: PurchaseAction = !subscription ? 'purchase' : isCurrentPlan ? 'renewal' : 'upgrade';
          const actionAvailable = !subscription || (allowUserSubscriptionChange && (
            (isCurrentPlan && subscription.plan.renewable)
            || (isHigherPlan && subscription.plan.upgradable)
          ));
          const actionLabel = action === 'renewal' ? copy.renew : action === 'upgrade' ? copy.upgrade : copy.buyNow;
          return <Col xs={24} md={8} key={item.plan.id}><Card className={index === 1 ? 'product-plan is-featured' : 'product-plan'} variant="borderless">{index === 1 && <div className="recommended">{copy.recommended}</div>}<Title level={3}>{item.plan.name}{isCurrentPlan && <Tag color="blue">{copy.currentPlan}</Tag>}</Title><Paragraph>{item.plan.description}</Paragraph><div className="plan-price"><span>{currencySymbol}</span><strong>{price ? (price.amountFen / 100).toFixed(2) : '—'}</strong><small>{price ? billingLabel(price.billingPeriod, price.months, locale) : ''}</small></div>{item.prices.length > 1 && <Select aria-label={copy.billingPeriod} className="plan-period-select" value={price?.id} options={item.prices.map((candidate) => ({ value: candidate.id, label: `${billingLabel(candidate.billingPeriod, candidate.months, locale)} · ${currencySymbol}${(candidate.amountFen / 100).toFixed(2)}` }))} onChange={(value) => setSelectedPrices((current) => ({ ...current, [item.plan.id]: value }))} />}<Divider /><Space orientation="vertical"><Text><CheckCircleFilled /> {formatBytes(item.plan.trafficBytes)} {copy.traffic}</Text><Text><CheckCircleFilled /> {item.plan.deviceLimit} {copy.devices}</Text><Text><CheckCircleFilled /> {copy.automaticActivation}</Text></Space><Button type={index === 1 ? 'primary' : 'default'} size="large" block loading={paymentBusy} disabled={!price || !actionAvailable} onClick={() => price && actionAvailable && onBuy(price.id, compact ? '' : couponCode, false, action, subscription?.entitlement.id || '')}>{actionLabel}</Button></Card></Col>;
        })}
      </Row>
      {!compact && <div className="coupon-entry"><Input value={couponCode} onChange={(event) => setCouponCode(event.target.value.toUpperCase())} allowClear placeholder={copy.couponCode} /></div>}
      {!compact && <section className="plan-assurance-grid">
        <div><CreditCardOutlined /><span><strong>{copy.paymentSecureTitle}</strong><small>{copy.paymentSecureDescription}</small></span></div>
        <div><ThunderboltFilled /><span><strong>{copy.activationTitle}</strong><small>{copy.activationDescription}</small></span></div>
        <div><SafetyOutlined /><span><strong>{copy.privacyTitle}</strong><small>{copy.securityNote}</small></span></div>
      </section>}
    </section>
  );
}

function GuidesView({ copy, articles, applications, onSection, telegramGroupLink }: { copy: PortalCopy; articles: GuestBootstrap['articles']; applications: GuestBootstrap['applications']; onSection: (section: Section) => void; telegramGroupLink?: string }) {
  const scrollToClients = () => document.getElementById('official-clients')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  return <div className="section-stack"><div className="section-title"><div><Title level={2}>{copy.guides}</Title><Text type="secondary">{copy.guidesDescription}</Text></div></div>
    <Card className="guide-overview-card" variant="borderless"><div><span className="portal-info-icon"><ImportOutlined /></span><div><Title level={3}>{copy.guideStartTitle}</Title><Paragraph>{copy.guideStartDescription}</Paragraph></div></div><Button type="primary" onClick={() => onSection('subscription')}>{copy.subscription}</Button></Card>
    <section id="official-clients" className="merged-guide-section"><div className="section-title merged-guide-heading"><div><Title level={3}>{copy.clientPickerTitle}</Title><Text type="secondary">{copy.clientsDescription}</Text></div></div>
      <Card className="client-picker-card" variant="borderless"><div><span className="portal-info-icon"><AppstoreOutlined /></span><div><Title level={3}>{copy.chooseClient}</Title><Paragraph>{copy.clientPickerDescription}</Paragraph></div></div></Card>
      <Row gutter={[20, 20]}>{applications.map((application) => <Col xs={24} md={12} lg={12} xl={6} key={application.id}><Card className="client-card"><div className="client-icon">{application.platform.includes('iOS') || application.platform.includes('Android') ? <MobileOutlined /> : application.platform.includes('Linux') ? <GlobalOutlined /> : <LaptopOutlined />}</div><Title level={4}>{application.name}</Title><Text type="secondary">{application.platform}</Text><Paragraph>{application.description}</Paragraph><Button href={application.officialUrl} target="_blank" rel="noopener noreferrer" type="primary" ghost icon={<CloudDownloadOutlined />}>{copy.officialDownload}</Button></Card></Col>)}</Row>
    </section>
    <section className="merged-guide-section"><div className="section-title merged-guide-heading"><div><Title level={3}>{copy.readGuides}</Title><Text type="secondary">{copy.guidesDescription}</Text></div></div>
      <Row gutter={[20, 20]}>{articles.map((article, index) => <Col xs={24} md={articles.length % 2 === 1 && index === articles.length - 1 ? 24 : 12} key={article.id}><Card className="content-card guide-content-card"><div className="guide-card-index">{String(index + 1).padStart(2, '0')}</div><Tag color="blue">{article.category}</Tag><Title level={4}>{article.title}</Title><Paragraph>{article.content}</Paragraph><Button type="link" onClick={scrollToClients}>{copy.chooseClient} <ArrowRightOutlined /></Button></Card></Col>)}</Row>
    </section>
    <Card className="after-install-card" variant="borderless"><div><Title level={3}>{copy.afterInstallTitle}</Title><Paragraph>{copy.afterInstallDescription}</Paragraph></div><Button type="primary" onClick={() => onSection('subscription')}>{copy.subscription}</Button></Card>
    <Row gutter={[20, 20]}><Col xs={24} md={12}><InfoCard icon={<ReloadOutlined />} title={copy.troubleshooting} description={copy.troubleshootingDescription} action={copy.tickets} onClick={() => onSection('tickets')} /></Col><Col xs={24} md={12}><InfoCard icon={<CustomerServiceOutlined />} title={copy.supportCenterTitle} description={copy.supportCenterDescription} action={telegramGroupLink ? copy.telegramSupport : copy.tickets} onClick={() => telegramGroupLink ? window.open(telegramGroupLink, '_blank', 'noopener,noreferrer') : onSection('tickets')} /></Col></Row>
  </div>;
}

function TicketsView({ copy, locale, authenticated, tickets, form, telegramGroupLink, onSubmit, onOpen, onLogin }: { copy: PortalCopy; locale: PortalLocale; authenticated: boolean; tickets: Ticket[]; form: FormInstance<{ subject: string; body: string }>; telegramGroupLink?: string; onSubmit: (values: { subject: string; body: string }) => void; onOpen: (ticket: Ticket) => void; onLogin: () => void }) {
  if (!authenticated) return <div className="section-stack"><div className="section-title"><div><Title level={2}>{copy.supportCenterTitle}</Title><Text type="secondary">{copy.supportCenterDescription}</Text></div>{telegramGroupLink && <Button href={telegramGroupLink} target="_blank" rel="noopener noreferrer" icon={<SendOutlined />}>{copy.telegramSupport}</Button>}</div><Card className="empty-card support-login-card" variant="borderless"><Empty description={copy.signInForTickets}><Space wrap><Button type="primary" onClick={onLogin}>{copy.signIn}</Button></Space></Empty></Card><SupportGuidance copy={copy} telegramGroupLink={telegramGroupLink} /></div>;
  return <div className="section-stack"><div className="section-title"><div><Title level={2}>{copy.supportCenterTitle}</Title><Text type="secondary">{copy.supportCenterDescription}</Text></div>{telegramGroupLink && <Button href={telegramGroupLink} target="_blank" rel="noopener noreferrer" icon={<SendOutlined />}>{copy.telegramSupport}</Button>}</div><Row gutter={[20, 20]}><Col xs={24} lg={10}><Card title={copy.createTicket}><Form form={form} layout="vertical" onFinish={onSubmit}><Form.Item name="subject" label={copy.subject} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="body" label={copy.body} rules={[{ required: true }]}><Input.TextArea rows={6} /></Form.Item><Button type="primary" htmlType="submit" icon={<SendOutlined />}>{copy.createTicket}</Button></Form></Card></Col><Col xs={24} lg={14}><Card title={copy.tickets}>{tickets.length === 0 ? <Empty description={copy.emptyTickets} /> : <div className="portal-record-list">{tickets.map((ticket) => <div className="portal-record" key={ticket.id}><Avatar icon={<CustomerServiceOutlined />} /><div className="portal-record-copy"><strong>{ticket.subject}</strong><Text type="secondary">{formatDate(ticket.createdAt, locale)}</Text></div><Tag color={ticket.status === 'open' ? 'processing' : 'default'}>{ticket.status}</Tag><Button type="link" onClick={() => onOpen(ticket)}>{copy.ticketConversation}</Button></div>)}</div>}</Card></Col></Row><SupportGuidance copy={copy} telegramGroupLink={telegramGroupLink} /></div>;
}

function SupportGuidance({ copy, telegramGroupLink }: { copy: PortalCopy; telegramGroupLink?: string }) {
  return <Row gutter={[20, 20]}><Col xs={24} md={8}><InfoCard icon={<CustomerServiceOutlined />} title={copy.responseTimeTitle} description={copy.responseTimeDescription} /></Col><Col xs={24} md={8}><InfoCard icon={<ReloadOutlined />} title={copy.troubleshooting} description={copy.troubleshootingDescription} /></Col><Col xs={24} md={8}><InfoCard icon={<SafetyOutlined />} title={copy.faqTitle} description={copy.faqDescription} action={telegramGroupLink ? copy.telegramSupport : undefined} onClick={telegramGroupLink ? () => window.open(telegramGroupLink, '_blank', 'noopener,noreferrer') : undefined} /></Col></Row>;
}

function OrdersView({ copy, locale, orders, currencySymbol, paymentBusy, onPay, onCancel }: { copy: PortalCopy; locale: PortalLocale; orders: Order[]; currencySymbol: string; paymentBusy: boolean; onPay: (id: string) => void; onCancel: (id: string) => void }) {
  const columns: TableProps<Order>['columns'] = [
    { title: copy.orderNumber, dataIndex: 'outTradeNo' },
    { title: copy.amount, render: (_, row) => <Space orientation="vertical" size={0}><Text>{currencySymbol} {((row.originalFen - row.discountFen) / 100).toFixed(2)}</Text>{row.balancePaidFen > 0 && <Text type="secondary">{copy.balance} {currencySymbol} {(row.balancePaidFen / 100).toFixed(2)} · {copy.paymentTitle} {currencySymbol} {(row.payableFen / 100).toFixed(2)}</Text>}</Space> },
    { title: copy.status, dataIndex: 'status', render: (value: string) => <Tag color={value === 'completed' ? 'success' : value === 'pending' ? 'processing' : 'default'}>{value}</Tag> },
    { title: copy.createdAt, dataIndex: 'createdAt', render: (value: string) => formatDate(value, locale) },
    { title: '', render: (_, row) => row.status === 'pending' && <Space><Button type="primary" size="small" loading={paymentBusy} onClick={() => onPay(row.id)}>{copy.continuePayment}</Button><Popconfirm title={copy.cancelOrder} onConfirm={() => onCancel(row.id)}><Button size="small" danger>{copy.cancelOrder}</Button></Popconfirm></Space> },
  ];
  return <Card title={copy.orders}><Table rowKey="id" dataSource={orders} columns={columns} pagination={false} locale={{ emptyText: <Empty description={copy.emptyOrders} /> }} /></Card>;
}

function AccountView({ copy, activeTab, onTabChange, dashboard, currencySymbol, giftForm, passwordForm, passwordBusy, onCopyInvitation, onRedeem, onChangePassword, onLogin }: { copy: PortalCopy; activeTab: AccountTab; onTabChange: (tab: AccountTab) => void; dashboard: Dashboard | null; currencySymbol: string; giftForm: FormInstance<{ code: string }>; passwordForm: FormInstance<{ currentPassword: string; newPassword: string; confirmPassword: string }>; passwordBusy: boolean; onCopyInvitation: () => void; onRedeem: (values: { code: string }) => void; onChangePassword: (values: { currentPassword: string; newPassword: string; confirmPassword: string }) => void; onLogin: () => void }) {
  if (!dashboard) return <Card className="empty-card"><Empty description={copy.signIn}><Button type="primary" onClick={onLogin}>{copy.signIn}</Button></Empty></Card>;
  const invitation = dashboard.invitation ?? { enabled: false, inviteCode: dashboard.customer.inviteCode, directInviteCount: 0, commissionPercent: 0, pendingFen: 0, confirmedFen: 0, settledFen: 0, commissionFirstPaymentOnly: false, inviteCodesNeverExpire: false };
  const invitationLink = buildInvitationLink(invitation.inviteCode || dashboard.customer.inviteCode);
  const earnedFen = invitation.pendingFen + invitation.confirmedFen + invitation.settledFen;
  const overview = <div className="account-tab-content"><section className="account-stat-grid"><div><span><WalletOutlined /></span><small>{copy.balance}</small><strong>{currencySymbol} {(dashboard.customer.balanceFen / 100).toFixed(2)}</strong></div><div><span><WifiOutlined /></span><small>{copy.subscription}</small><strong>{dashboard.subscription?.plan.name || copy.noSubscription}</strong></div><div><span><TeamOutlined /></span><small>{copy.invitedUsers}</small><strong>{invitation.directInviteCount}</strong></div><div><span><GiftOutlined /></span><small>{copy.earnedCommission}</small><strong>{currencySymbol} {(earnedFen / 100).toFixed(2)}</strong></div></section><Row gutter={[20, 20]}><Col xs={24} lg={14}><Card title={copy.accountOverview}><Descriptions column={1} bordered size="small" items={[{ key: 'email', label: copy.email, children: dashboard.customer.email }, { key: 'status', label: copy.status, children: <Tag color="success">{dashboard.customer.status}</Tag> }, { key: 'invite', label: copy.invitationCode, children: <Text copyable>{invitation.inviteCode}</Text> }]} /></Card></Col><Col xs={24} lg={10}><Card title={copy.redeemGiftCard}><Paragraph type="secondary">{copy.commissionBalanceHint}</Paragraph><Form form={giftForm} layout="vertical" onFinish={onRedeem}><Form.Item name="code" label={copy.giftCardCode} rules={[{ required: true }]}><Input /></Form.Item><Button type="primary" htmlType="submit" block icon={<GiftOutlined />}>{copy.redeemGiftCard}</Button></Form></Card></Col></Row></div>;
  const invitationPanel = <div className="account-tab-content"><Card className="invitation-hero-card" variant="borderless"><div><span className="portal-info-icon"><TeamOutlined /></span><div><Title level={2}>{copy.inviteFriends}</Title><Paragraph>{copy.invitationDescription}</Paragraph></div></div><Button type="primary" size="large" icon={<CopyOutlined />} onClick={onCopyInvitation}>{copy.copyInvitation}</Button></Card><section className="account-stat-grid invitation-stats"><div><span><TeamOutlined /></span><small>{copy.invitedUsers}</small><strong>{invitation.directInviteCount}</strong></div><div><span><GiftOutlined /></span><small>{copy.commissionRate}</small><strong>{invitation.commissionPercent}%</strong></div><div><span><ReloadOutlined /></span><small>{copy.pendingCommission}</small><strong>{currencySymbol} {((invitation.pendingFen + invitation.confirmedFen) / 100).toFixed(2)}</strong></div><div><span><WalletOutlined /></span><small>{copy.earnedCommission}</small><strong>{currencySymbol} {(earnedFen / 100).toFixed(2)}</strong></div></section><Card title={copy.invitationLink}><Space.Compact block><Input value={invitationLink} readOnly aria-label={copy.invitationLink} /><Button type="primary" icon={<CopyOutlined />} onClick={onCopyInvitation}>{copy.copyInvitation}</Button></Space.Compact><div className="invitation-rules"><CheckCircleFilled /><span>{copy.commissionBalanceHint}</span>{invitation.commissionFirstPaymentOnly && <><CheckCircleFilled /><span>{copy.firstPaymentRule}</span></>}{invitation.inviteCodesNeverExpire && <><CheckCircleFilled /><span>{copy.inviteCodePermanent}</span></>}</div></Card></div>;
  const security = <div className="account-tab-content"><Card title={copy.changePassword} className="password-change-card"><Paragraph type="secondary">{copy.passwordChangeHint}</Paragraph><Form form={passwordForm} layout="vertical" requiredMark={false} onFinish={onChangePassword}><Form.Item name="currentPassword" label={copy.currentPassword} rules={[{ required: true }]}><Input.Password autoComplete="current-password" /></Form.Item><Form.Item name="newPassword" label={copy.newPassword} extra={copy.passwordRule} rules={[{ required: true }, { min: 10, max: 128, message: copy.passwordRule }, { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).+$/, message: copy.passwordRule }]}><Input.Password autoComplete="new-password" /></Form.Item><Form.Item name="confirmPassword" label={copy.confirmPassword} dependencies={['newPassword']} rules={[{ required: true }, ({ getFieldValue }) => ({ validator(_, value) { return !value || getFieldValue('newPassword') === value ? Promise.resolve() : Promise.reject(new Error(copy.passwordMismatch)); } })]}><Input.Password autoComplete="new-password" /></Form.Item><Button type="primary" htmlType="submit" block icon={<LockOutlined />} loading={passwordBusy}>{copy.changePassword}</Button></Form></Card></div>;
  return <div className="section-stack account-center"><div className="section-title"><div><Title level={2}>{copy.accountCenter}</Title><Text type="secondary">{dashboard.customer.displayName || dashboard.customer.email}</Text></div></div><Card className="account-center-shell" variant="borderless"><Tabs activeKey={activeTab} onChange={(key) => onTabChange(key as AccountTab)} items={[{ key: 'overview', label: copy.accountOverview, icon: <UserOutlined />, children: overview }, { key: 'invitation', label: copy.invitationRewards, icon: <TeamOutlined />, children: invitationPanel }, { key: 'security', label: copy.accountSecurity, icon: <SafetyOutlined />, children: security }]} /></Card></div>;
}

export default function PortalApp() {
  return <ConfigProvider theme={{ token: { colorPrimary: '#4f95f5', colorInfo: '#4f95f5', borderRadius: 10, borderRadiusLG: 14, fontFamily: 'Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif', colorText: '#17233d' } }}><AntApp><PortalContent /></AntApp></ConfigProvider>;
}
