export interface PanelRestartSettings {
  webBasePath?: string;
  webDomain?: string;
  webPort?: number | string;
  webCertFile?: string;
  webKeyFile?: string;
}

function stripIPv6Brackets(hostname: string): string {
  return hostname.replace(/^\[|\]$/g, '');
}

function hostnameForURL(hostname: string): string {
  const stripped = stripIPv6Brackets(hostname);
  return stripped.includes(':') ? `[${stripped}]` : stripped;
}

export function isIPAddress(hostname: string): boolean {
  const host = stripIPv6Brackets(hostname);
  const v4 = host.split('.');
  if (v4.length === 4 && v4.every((part) => /^\d{1,3}$/.test(part) && Number(part) <= 255)) {
    return true;
  }
  if (!host.includes(':') || host.includes(':::')) return false;
  const parts = host.split('::');
  if (parts.length > 2) return false;
  const split = (value: string) => (value ? value.split(':').filter(Boolean) : []);
  const groups = [...split(parts[0]), ...split(parts[1])];
  if (!groups.every((segment) => /^[0-9a-fA-F]{1,4}$/.test(segment))) return false;
  return parts.length === 2 ? groups.length < 8 : groups.length === 8;
}

function basePathFromCurrentPath(pathname: string): string {
  const panelIndex = pathname.search(/\/panel(?:\/|$)/);
  return panelIndex >= 0 ? pathname.slice(0, panelIndex + 1) : '/';
}

function normalizeBasePath(value: string | undefined, currentPathname: string): string {
  let base = value?.trim() || basePathFromCurrentPath(currentPathname);
  if (!base.startsWith('/')) base = `/${base}`;
  if (!base.endsWith('/')) base += '/';
  return base.replace(/\/{2,}/g, '/');
}

/**
 * Build the public URL used after a panel restart.
 *
 * Domain deployments normally terminate TLS and ports at Nginx, so the
 * browser's public origin is authoritative. Direct IP deployments still need
 * to follow panel TLS/port changes. In both cases the hidden base path and the
 * active settings tab are retained.
 */
export function buildPanelRestartURL(
  currentHref: string,
  settings: PanelRestartSettings,
): string {
  const current = new URL(currentHref);
  const target = new URL(currentHref);
  const directIP = isIPAddress(current.hostname);

  if (directIP) {
    const configuredHost = settings.webDomain?.trim();
    if (configuredHost && isIPAddress(configuredHost)) {
      target.hostname = hostnameForURL(configuredHost);
    }
    const configuredPort = Number(settings.webPort || 0);
    if (configuredPort > 0 && configuredPort <= 65535) {
      target.port = String(configuredPort);
    }
    target.protocol = settings.webCertFile || settings.webKeyFile ? 'https:' : 'http:';
  }

  const basePath = normalizeBasePath(settings.webBasePath, current.pathname);
  target.pathname = `${basePath}panel/settings`;
  target.search = '';
  target.hash = current.hash || '#general';
  return target.toString();
}
