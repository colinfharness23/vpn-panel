import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { applySiteBranding, useSiteBranding } from './useSiteBranding';

const TITLE_KEYS: Record<string, string> = {
  '/': 'menu.dashboard',
  '/inbounds': 'menu.inbounds',
  '/clients': 'menu.clients',
  '/groups': 'menu.groups',
  '/nodes': 'menu.nodes',
  '/hosts': 'menu.hosts',
  '/settings': 'menu.settings',
  '/xray': 'menu.xray',
  '/outbound': 'menu.outbounds',
  '/routing': 'menu.routing',
  '/api-docs': 'menu.apiDocs',
};

export function usePageTitle() {
  const { pathname } = useLocation();
  const { t } = useTranslation();
  const { siteName, logoUrl } = useSiteBranding();

  useEffect(() => {
    const key = TITLE_KEYS[pathname];
    const title = key ? t(key) : '';
    applySiteBranding(siteName, logoUrl, title);
  }, [logoUrl, pathname, siteName, t]);
}
