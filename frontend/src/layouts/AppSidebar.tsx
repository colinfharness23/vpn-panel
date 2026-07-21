import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ComponentType } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Drawer, Layout, Menu } from 'antd';
import type { MenuProps } from 'antd';
import {
  ApiOutlined,
  CloseOutlined,
  CloudServerOutlined,
  CloudSyncOutlined,
  ClusterOutlined,
  CodeOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  ExportOutlined,
  FileTextOutlined,
  GiftOutlined,
  GithubOutlined,
  GlobalOutlined,
  HomeOutlined,
  ImportOutlined,
  LogoutOutlined,
  MailOutlined,
  MenuOutlined,
  MessageOutlined,
  SafetyOutlined,
  ShopOutlined,
  SettingOutlined,
  SwapOutlined,
  TagsOutlined,
  TeamOutlined,
  ToolOutlined,
} from '@ant-design/icons';

import { HttpUtil } from '@/utils';
import { formatPanelVersion } from '@/lib/panel-version';
import { useTheme } from '@/hooks/useTheme';
import './AppSidebar.css';

const SIDEBAR_COLLAPSED_KEY = 'isSidebarCollapsed';
const REPO_URL = 'https://github.com/colinfharness23/vpn-panel';
const LOGOUT_KEY = '__logout__';
const PORTAL_KEY = '__portal__';
const INVITATION_COMMISSION_KEY = '__invitation_commission__';
const PANEL_BRAND = 'NOVA';

type SidebarSection = 'xui' | 'system' | 'utility';
type IconName = 'dashboard' | 'inbound' | 'team' | 'groups' | 'setting' | 'tool' | 'cluster' | 'hosts' | 'logout' | 'website' | 'apidocs' | 'outbound' | 'routing' | 'commercial' | 'migration' | 'siteSettings' | 'securitySettings' | 'subscriptionSettings' | 'invitationSettings' | 'emailSettings' | 'telegramSettings' | 'subscriptionTemplate' | 'subscriptionFormats';

const iconByName: Record<IconName, ComponentType> = {
  dashboard: DashboardOutlined,
  inbound: ImportOutlined,
  team: TeamOutlined,
  groups: TagsOutlined,
  setting: SettingOutlined,
  tool: ToolOutlined,
  cluster: ClusterOutlined,
  hosts: GlobalOutlined,
  logout: LogoutOutlined,
  website: HomeOutlined,
  apidocs: ApiOutlined,
  outbound: ExportOutlined,
  routing: SwapOutlined,
  commercial: ShopOutlined,
  migration: CloudSyncOutlined,
  siteSettings: GlobalOutlined,
  securitySettings: SafetyOutlined,
  subscriptionSettings: CloudServerOutlined,
  invitationSettings: GiftOutlined,
  emailSettings: MailOutlined,
  telegramSettings: MessageOutlined,
  subscriptionTemplate: FileTextOutlined,
  subscriptionFormats: CodeOutlined,
};

function readCollapsed(): boolean {
  try {
    return JSON.parse(localStorage.getItem(SIDEBAR_COLLAPSED_KEY) || 'false');
  } catch {
    return false;
  }
}

function VersionBadge({ version, collapsed }: { version: string; collapsed?: boolean }) {
  if (!version) return null;
  const label = formatPanelVersion(version);
  return (
    <a
      href={REPO_URL}
      target="_blank"
      rel="noopener noreferrer"
      className={`sider-version${collapsed ? ' is-collapsed' : ''}`}
      aria-label={`GitHub ${label}`}
      title={label}
    >
      <GithubOutlined />
      {!collapsed && <span className="sider-version-text">{label}</span>}
    </a>
  );
}

export default function AppSidebar() {
  const { t } = useTranslation();
  const { isDark } = useTheme();
  const navigate = useNavigate();
  const { pathname, hash } = useLocation();

  const [collapsed, setCollapsed] = useState<boolean>(() => readCollapsed());
  const [drawerOpen, setDrawerOpen] = useState(false);

  const currentTheme: 'light' | 'dark' = isDark ? 'dark' : 'light';
  const panelVersion = window.X_UI_CUR_VER || '';

  const tabs = useMemo<{ key: string; icon: IconName; title: string; section: SidebarSection }[]>(() => [
    { key: '/inbounds', icon: 'inbound', title: t('menu.inbounds'), section: 'xui' },
    { key: '/clients', icon: 'team', title: t('menu.clients'), section: 'xui' },
    { key: '/groups', icon: 'groups', title: t('menu.groups'), section: 'xui' },
    { key: '/nodes', icon: 'cluster', title: t('menu.nodes'), section: 'xui' },
    { key: '/hosts', icon: 'hosts', title: t('menu.hosts'), section: 'xui' },
    { key: '/outbound', icon: 'outbound', title: t('menu.outbounds'), section: 'xui' },
    { key: '/routing', icon: 'routing', title: t('menu.routing'), section: 'xui' },
    { key: '/xray', icon: 'tool', title: t('menu.xray'), section: 'xui' },
    { key: '/', icon: 'dashboard', title: t('menu.dashboard'), section: 'system' },
    { key: '/migration', icon: 'migration', title: '一键迁移', section: 'system' },
    { key: '/commercial', icon: 'commercial', title: t('menu.commercial'), section: 'system' },
    { key: '/settings#general', icon: 'siteSettings', title: '站点设置', section: 'system' },
    { key: '/settings#security', icon: 'securitySettings', title: '安全设置', section: 'system' },
    { key: '/settings#subscription', icon: 'subscriptionSettings', title: '订阅设置', section: 'system' },
    { key: INVITATION_COMMISSION_KEY, icon: 'invitationSettings', title: '邀请&佣金设置', section: 'system' },
    { key: '/settings#email', icon: 'emailSettings', title: '邮件设置', section: 'system' },
    { key: '/settings#telegram', icon: 'telegramSettings', title: 'Telegram设置', section: 'system' },
    { key: '/settings#subscription-template', icon: 'subscriptionTemplate', title: '订阅模板', section: 'system' },
    { key: '/settings#subscription-formats', icon: 'subscriptionFormats', title: '订阅格式', section: 'system' },
    { key: '/api-docs', icon: 'apidocs', title: t('menu.apiDocs'), section: 'system' },
    { key: PORTAL_KEY, icon: 'website', title: t('menu.enterSite'), section: 'utility' },
    { key: LOGOUT_KEY, icon: 'logout', title: t('logout'), section: 'utility' },
  ], [t]);

  const xuiItems = useMemo(() => tabs.filter((tab) => tab.section === 'xui'), [tabs]);
  const systemItems = useMemo(() => tabs.filter((tab) => tab.section === 'system'), [tabs]);
  const utilItems = useMemo(() => tabs.filter((tab) => tab.section === 'utility'), [tabs]);

  const xrayChildren = useMemo<NonNullable<MenuProps['items']>>(() => [
    { key: '/xray#basic', icon: <SettingOutlined />, label: t('pages.xray.basicTemplate') },
    { key: '/xray#balancer', icon: <ClusterOutlined />, label: t('pages.xray.Balancers') },
    { key: '/xray#dns', icon: <DatabaseOutlined />, label: 'DNS' },
    { key: '/xray#advanced', icon: <CodeOutlined />, label: t('pages.xray.advancedTemplate') },
  ], [t]);

  const settingsActive = pathname === '/settings';
  const xrayActive = pathname === '/xray';
  const invitationCommissionActive = pathname === '/commercial' && hash === '#marketing';
  const selectedKey = invitationCommissionActive
    ? INVITATION_COMMISSION_KEY
    : settingsActive
    ? `/settings${hash || '#general'}`
    : xrayActive
      ? `/xray${hash || '#basic'}`
      : (pathname === '' ? '/' : pathname);

  const requiredOpenKeys = useMemo(() => xrayActive ? ['/xray'] : [], [xrayActive]);
  const [openKeys, setOpenKeys] = useState<string[]>(() => requiredOpenKeys);
  useEffect(() => {
    if (requiredOpenKeys.length > 0) {
      setOpenKeys((keys) => Array.from(new Set([...keys, ...requiredOpenKeys])));
    }
  }, [requiredOpenKeys]);

  const toMenuItems = useCallback((items: typeof tabs): MenuProps['items'] =>
    items.map((tab) => {
      const Icon = iconByName[tab.icon];
      if (tab.key === '/xray') {
        return { key: tab.key, icon: <Icon />, label: tab.title, children: xrayChildren };
      }
      return { key: tab.key, icon: <Icon />, label: tab.title };
    }),
  [xrayChildren]);

  const groupedNavItems = useMemo<MenuProps['items']>(() => [
    {
      type: 'group',
      key: '__xui_group__',
      label: collapsed ? null : PANEL_BRAND,
      children: toMenuItems(xuiItems),
    },
    {
      type: 'group',
      key: '__system_group__',
      label: collapsed ? null : t('menu.systemManagement'),
      children: toMenuItems(systemItems),
    },
  ], [collapsed, systemItems, t, toMenuItems, xuiItems]);

  const drawerNavItems = useMemo<MenuProps['items']>(() => [
    {
      type: 'group',
      key: '__drawer_xui_group__',
      label: PANEL_BRAND,
      children: toMenuItems(xuiItems),
    },
    {
      type: 'group',
      key: '__drawer_system_group__',
      label: t('menu.systemManagement'),
      children: toMenuItems(systemItems),
    },
  ], [systemItems, t, toMenuItems, xuiItems]);

  const openLink = useCallback(async (key: string) => {
    if (key === PORTAL_KEY) {
      const basePath = window.X_UI_PUBLIC_BASE_PATH || '/';
      const normalizedBasePath = basePath.endsWith('/') ? basePath : `${basePath}/`;
      window.location.href = normalizedBasePath;
      return;
    }
    if (key === LOGOUT_KEY) {
      await HttpUtil.post('/logout');
      window.location.href = window.X_UI_BASE_PATH || '/';
      return;
    }
    if (key === INVITATION_COMMISSION_KEY) {
      navigate('/commercial#marketing');
      return;
    }
    navigate(key);
  }, [navigate]);

  const onMenuClick = useCallback<NonNullable<MenuProps['onClick']>>(({ key }) => {
    openLink(String(key));
  }, [openLink]);

  const onSiderCollapse = useCallback((isCollapsed: boolean, type: 'clickTrigger' | 'responsive') => {
    if (type === 'clickTrigger') {
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(isCollapsed));
      setCollapsed(isCollapsed);
    }
  }, []);

  return (
    <div className="ant-sidebar">
      <Layout.Sider
        theme={currentTheme}
        width={220}
        collapsible
        collapsed={collapsed}
        breakpoint="md"
        trigger={null}
        onCollapse={onSiderCollapse}
      >
        <Menu
          theme={currentTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          openKeys={collapsed ? undefined : openKeys}
          onOpenChange={(keys) => setOpenKeys(keys as string[])}
          className="sider-nav"
          items={groupedNavItems}
          onClick={onMenuClick}
        />
        <Menu
          theme={currentTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          className="sider-utility"
          items={toMenuItems(utilItems)}
          onClick={onMenuClick}
        />
        <div className="sider-footer">
          <VersionBadge version={panelVersion} collapsed={collapsed} />
        </div>
      </Layout.Sider>

      <Drawer
        placement="left"
        closable={false}
        open={drawerOpen}
        rootClassName={currentTheme}
        size="min(82vw, 320px)"
        styles={{
          wrapper: { padding: 0 },
          body: { padding: 0, display: 'flex', flexDirection: 'column', height: '100%' },
          header: { display: 'none' },
        }}
        onClose={() => setDrawerOpen(false)}
      >
        <div className="drawer-header">
          <div className="brand-block">
            <span className="drawer-brand">{PANEL_BRAND}</span>
          </div>
          <div className="drawer-header-actions">
            <button
              className="drawer-close"
              type="button"
              aria-label={t('close')}
              onClick={() => setDrawerOpen(false)}
            >
              <CloseOutlined />
            </button>
          </div>
        </div>
        <Menu
          theme={currentTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          openKeys={openKeys}
          onOpenChange={(keys) => setOpenKeys(keys as string[])}
          className="drawer-menu drawer-nav"
          items={drawerNavItems}
          onClick={(info) => { onMenuClick(info); setDrawerOpen(false); }}
        />
        <Menu
          theme={currentTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          className="drawer-menu drawer-utility"
          items={toMenuItems(utilItems)}
          onClick={(info) => { onMenuClick(info); setDrawerOpen(false); }}
        />
        <div className="drawer-footer">
          <VersionBadge version={panelVersion} />
        </div>
      </Drawer>

      {!drawerOpen && (
        <button
          className="drawer-handle"
          type="button"
          aria-label={t('menu.openMenu')}
          onClick={() => setDrawerOpen(true)}
        >
          <MenuOutlined />
        </button>
      )}
    </div>
  );
}
