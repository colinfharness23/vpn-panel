import { describe, expect, it } from 'vitest';

import { buildPortalNavigation } from '../portal/navigation';
import { residentialRelayCopies } from '../portal/PortalApp';
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

  it('presents client packages as direct site downloads in every locale', () => {
    expect(portalCopies['zh-CN'].officialDownload).toBe('下载');
    for (const copy of Object.values(portalCopies)) {
      expect(copy.officialDownload).not.toMatch(/official|github|官方/i);
      expect(copy.clientsDescription).not.toMatch(/github/i);
    }
  });

  it('only shows the private subscription entry after sign-in', () => {
    expect(buildPortalNavigation(portalCopies['zh-CN'], false)).toEqual([]);
    expect(buildPortalNavigation(portalCopies['zh-CN'], true).map((item) => item.key)).toContain('subscription');
  });

  it('presents a redeem code instead of a coupon code at checkout', () => {
    expect(portalCopies['zh-CN'].couponCode).toBe('兑换码（可选）');
    for (const copy of Object.values(portalCopies)) {
      expect(copy.couponCode).not.toMatch(/coupon code|优惠码/i);
    }
  });

  it('localizes residential relay controls without exposing implementation details', () => {
    expect(Object.keys(residentialRelayCopies)).toHaveLength(localeOptions.length);
    for (const copy of Object.values(residentialRelayCopies)) {
      expect(copy.title).toBeTruthy();
      expect(copy.add).toBeTruthy();
      expect(`${copy.title} ${copy.description} ${copy.security}`).not.toMatch(/3x-ui|xray/i);
    }
  });
});
