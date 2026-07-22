// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { httpUpload } from '@/api/http-init';

class FakeXMLHttpRequest {
  static instance: FakeXMLHttpRequest | null = null;

  upload: { onprogress: ((event: ProgressEvent) => void) | null } = { onprogress: null };
  status = 200;
  statusText = 'OK';
  responseText = '{"success":true,"obj":{"id":"app-1"}}';
  timeout = 0;
  withCredentials = false;
  onerror: (() => void) | null = null;
  ontimeout: (() => void) | null = null;
  onabort: (() => void) | null = null;
  onload: (() => void) | null = null;
  onloadend: (() => void) | null = null;
  method = '';
  url = '';
  headers = new Map<string, string>();

  constructor() {
    FakeXMLHttpRequest.instance = this;
  }

  open(method: string, url: string): void {
    this.method = method;
    this.url = url;
  }

  setRequestHeader(key: string, value: string): void {
    this.headers.set(key, value);
  }

  getResponseHeader(key: string): string | null {
    return key.toLowerCase() === 'content-type' ? 'application/json' : null;
  }

  send(): void {
    this.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 } as ProgressEvent);
    this.onload?.();
    this.onloadend?.();
  }

  abort(): void {
    this.onabort?.();
  }
}

describe('large file upload transport', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    document.head.querySelector('meta[name="csrf-token"]')?.remove();
    FakeXMLHttpRequest.instance = null;
  });

  it('reports upload progress and sends the CSRF header', async () => {
    vi.stubGlobal('XMLHttpRequest', FakeXMLHttpRequest as unknown as typeof XMLHttpRequest);
    const meta = document.createElement('meta');
    meta.name = 'csrf-token';
    meta.content = 'csrf-test-token';
    document.head.appendChild(meta);
    const onProgress = vi.fn();
    const form = new FormData();
    form.append('package', new File(['0123456789'], 'client.zip', { type: 'application/zip' }));

    const response = await httpUpload('/panel/api/commercial/applications/app-1/package', form, { onProgress });

    expect(response.ok).toBe(true);
    expect(onProgress).toHaveBeenCalledWith(5, 10);
    expect(FakeXMLHttpRequest.instance?.method).toBe('POST');
    expect(FakeXMLHttpRequest.instance?.headers.get('X-CSRF-Token')).toBe('csrf-test-token');
    expect(FakeXMLHttpRequest.instance?.headers.has('Content-Type')).toBe(false);
  });
});
