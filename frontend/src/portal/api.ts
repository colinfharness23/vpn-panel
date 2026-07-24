import type { Envelope } from './types';

const rawBase = window.X_UI_BASE_PATH || '/';
const basePath = rawBase.endsWith('/') ? rawBase.slice(0, -1) : rawBase;
let csrfToken = '';

export class PortalApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function ensureCSRF(): Promise<string> {
  if (csrfToken) return csrfToken;
  const response = await fetch(`${basePath}/api/v1/passport/csrf-token`, { credentials: 'include' });
  const payload = await response.json() as Envelope<string>;
  if (!response.ok || !payload.success) {
    throw new PortalApiError(payload.msg || 'Unable to start secure session', response.status);
  }
  csrfToken = payload.obj;
  return csrfToken;
}

export async function portalRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method || 'GET').toUpperCase();
  const headers = new Headers(init.headers);
  if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
    headers.set('X-CSRF-Token', await ensureCSRF());
    if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  }
  headers.set('X-Requested-With', 'XMLHttpRequest');
  const response = await fetch(`${basePath}${path}`, {
    ...init,
    headers,
    credentials: 'include',
    cache: 'no-store',
  });
  const payload = await response.json() as Envelope<T>;
  if (!response.ok || !payload.success) {
    throw new PortalApiError(payload.msg || `Request failed (${response.status})`, response.status);
  }
  return payload.obj;
}

export function portalAsset(path: string): string {
  return `${basePath}${path}`;
}
