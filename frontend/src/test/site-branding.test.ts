// @vitest-environment jsdom

import { afterEach, describe, expect, it } from 'vitest';

import { applySiteBranding } from '@/hooks/useSiteBranding';

describe('site document branding', () => {
  afterEach(() => {
    document.title = '';
    document.head.querySelector('link[data-site-favicon="true"]')?.remove();
  });

  it('applies the configured site name and logo to the browser tab', () => {
    applySiteBranding('PHEERO', 'data:image/png;base64,cG5n', '线路中心');

    expect(document.title).toBe('PHEERO - 线路中心');
    expect(document.head.querySelector<HTMLLinkElement>('link[data-site-favicon="true"]')?.href)
      .toContain('data:image/png;base64,cG5n');
  });

  it('removes the managed favicon when the logo is cleared', () => {
    applySiteBranding('PHEERO', 'data:image/png;base64,cG5n');
    applySiteBranding('PHEERO', '');

    expect(document.title).toBe('PHEERO');
    expect(document.head.querySelector('link[data-site-favicon="true"]')).toBeNull();
  });
});
