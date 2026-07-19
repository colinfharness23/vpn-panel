import { describe, expect, it } from 'vitest';

import { buildPortalNavigation } from '../portal/navigation';
import { localeOptions, portalCopies } from '../portal/translations';

describe('portal presentation policy', () => {
  it('keeps the customer-facing hero copy product-neutral in every locale', () => {
    for (const copy of Object.values(portalCopies)) {
      expect(copy.heroBadge).not.toMatch(/3x-ui/i);
    }
  });

  it('uses compact language codes in the header selector', () => {
    expect(localeOptions.find((option) => option.value === 'zh-CN')?.shortLabel).toBe('CN');
    expect(localeOptions.find((option) => option.value === 'en-US')?.shortLabel).toBe('EN');
    expect(localeOptions.every((option) => /^[A-Z]{2}$/.test(option.shortLabel))).toBe(true);
  });

  it('keeps public email and reset copy compatible with the configured domain allowlist', () => {
    for (const copy of Object.values(portalCopies)) {
      expect(copy.email).not.toMatch(/gmail/i);
      expect(copy.emailOrAdmin).not.toMatch(/gmail/i);
      expect(copy.gmailOnly).not.toMatch(/gmail/i);
      expect(copy.resetDescription).not.toMatch(/gmail/i);
    }
  });

  it('only shows the private subscription entry after sign-in', () => {
    expect(buildPortalNavigation(portalCopies['zh-CN'], false).map((item) => item.key)).toEqual([
      'home',
      'plans',
      'guides',
      'tickets',
    ]);
    expect(buildPortalNavigation(portalCopies['zh-CN'], true).map((item) => item.key)).toContain('subscription');
  });
});
