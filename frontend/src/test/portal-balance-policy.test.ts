import { describe, expect, it } from 'vitest';

import { portalCopies } from '@/portal/translations';

describe('portal account-credit policy', () => {
  it('clearly states that account credit is non-withdrawable and reusable for plans', () => {
    expect(portalCopies['zh-CN'].commissionBalanceHint).toBe('账户余额不能提现，只能在购买、续费或升级本站套餐时抵扣。');
    expect(portalCopies['zh-TW'].commissionBalanceHint).toBe('帳戶餘額不可提領，只能在購買、續費或升級本站方案時折抵。');
    expect(portalCopies['en-US'].commissionBalanceHint).toContain('cannot be withdrawn');
  });
});
