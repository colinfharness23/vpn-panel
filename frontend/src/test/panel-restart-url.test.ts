import { readFileSync } from 'node:fs';

import { describe, expect, it } from 'vitest';

import { buildPanelRestartURL, isIPAddress } from '@/pages/settings/panelRestartUrl';

describe('panel restart URL', () => {
  it('keeps the public HTTPS origin, hidden admin path, and active tab behind Nginx', () => {
    const target = buildPanelRestartURL(
      'https://vpn.pheero.com/437886227642434313/panel/settings#security',
      {
        webBasePath: '/437886227642434313/',
        webPort: 15821,
        webCertFile: '',
        webKeyFile: '',
      },
    );

    expect(target).toBe(
      'https://vpn.pheero.com/437886227642434313/panel/settings#security',
    );
  });

  it('uses a newly saved hidden path without leaving the administrator settings page', () => {
    const target = buildPanelRestartURL(
      'https://vpn.example.com/old-secret/panel/settings#general',
      { webBasePath: 'new-secret' },
    );

    expect(target).toBe(
      'https://vpn.example.com/new-secret/panel/settings#general',
    );
  });

  it('applies direct IP port and TLS changes', () => {
    const target = buildPanelRestartURL(
      'http://192.0.2.10:12000/secret/panel/settings',
      {
        webBasePath: '/secret/',
        webDomain: '192.0.2.11',
        webPort: 15432,
        webCertFile: '/etc/nova/panel.crt',
        webKeyFile: '/etc/nova/panel.key',
      },
    );

    expect(target).toBe(
      'https://192.0.2.11:15432/secret/panel/settings#general',
    );
  });

  it('recognizes IPv4 and IPv6 addresses without treating domains as IPs', () => {
    expect(isIPAddress('203.0.113.8')).toBe(true);
    expect(isIPAddress('[2001:db8::1]')).toBe(true);
    expect(isIPAddress('vpn.example.com')).toBe(false);
  });

  it('applies a bracketed IPv6 host to direct-IP deployments', () => {
    const target = buildPanelRestartURL(
      'http://[2001:db8::1]:12000/secret/panel/settings',
      {
        webBasePath: '/secret/',
        webDomain: '2001:db8::2',
        webPort: 15432,
      },
    );

    expect(target).toBe(
      'http://[2001:db8::2]:15432/secret/panel/settings#general',
    );
  });
});

describe('administrator HTML translation isolation', () => {
  for (const entry of ['index.html', 'login.html']) {
    it(`${entry} prevents browser translation from mutating React-owned DOM`, () => {
      const html = readFileSync(new URL(`../../${entry}`, import.meta.url), 'utf8');
      expect(html).toContain('<meta name="google" content="notranslate" />');
      expect(html).toContain('translate="no" class="notranslate"');
      expect(html).toContain('id="app" translate="no" class="notranslate"');
    });
  }
});
