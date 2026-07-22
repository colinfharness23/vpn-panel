import { useEffect, useState } from 'react';

export interface SiteBranding {
  siteName: string;
  logoUrl: string;
}

interface GuestAuthConfigEnvelope {
  success?: boolean;
  obj?: { site?: Record<string, string> };
}

const DEFAULT_SITE_BRANDING: SiteBranding = {
  siteName: 'NOVA',
  logoUrl: '',
};

let cachedBranding: SiteBranding | null = null;
let brandingRequest: Promise<SiteBranding> | null = null;
const brandingListeners = new Set<(branding: SiteBranding) => void>();

async function fetchSiteBranding(): Promise<SiteBranding> {
  const response = await fetch('/api/v1/guest/auth-config', {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) throw new Error('站点品牌信息加载失败');
  const envelope = (await response.json()) as GuestAuthConfigEnvelope;
  const site = envelope.success ? envelope.obj?.site : undefined;
  return {
    siteName: site?.siteName?.trim() || DEFAULT_SITE_BRANDING.siteName,
    logoUrl: site?.logoUrl?.trim() || '',
  };
}

function loadSiteBranding(): Promise<SiteBranding> {
  if (cachedBranding) return Promise.resolve(cachedBranding);
  if (!brandingRequest) {
    brandingRequest = fetchSiteBranding()
      .then((branding) => {
        updateSiteBranding(branding);
        return branding;
      })
      .finally(() => {
        brandingRequest = null;
      });
  }
  return brandingRequest;
}

export function updateSiteBranding(branding: SiteBranding): void {
  cachedBranding = {
    siteName: branding.siteName.trim() || DEFAULT_SITE_BRANDING.siteName,
    logoUrl: branding.logoUrl.trim(),
  };
  brandingListeners.forEach((listener) => listener(cachedBranding as SiteBranding));
}

export function applySiteBranding(siteName: string, logoUrl: string, pageTitle = ''): void {
  const brand = siteName.trim() || DEFAULT_SITE_BRANDING.siteName;
  document.title = pageTitle.trim() ? `${brand} - ${pageTitle.trim()}` : brand;

  let favicon = document.head.querySelector<HTMLLinkElement>('link[data-site-favicon="true"]');
  if (!logoUrl.trim()) {
    favicon?.remove();
    return;
  }
  if (!favicon) {
    favicon = document.createElement('link');
    favicon.rel = 'icon';
    favicon.dataset.siteFavicon = 'true';
    document.head.appendChild(favicon);
  }
  favicon.href = logoUrl;
}

export function useSiteBranding(): SiteBranding {
  const [branding, setBranding] = useState<SiteBranding>(() => cachedBranding || DEFAULT_SITE_BRANDING);
  useEffect(() => {
    brandingListeners.add(setBranding);
    void loadSiteBranding().catch(() => undefined);
    return () => {
      brandingListeners.delete(setBranding);
    };
  }, []);
  return branding;
}
