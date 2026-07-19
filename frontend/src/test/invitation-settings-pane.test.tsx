import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';

import InvitationSettingsPane from '@/pages/commercial/InvitationSettingsPane';
import { MarketingView } from '@/pages/commercial/CommercialPage';
import { DEFAULT_INVITATION_SETTINGS } from '@/pages/commercial/useInvitationSettings';
import { HttpUtil } from '@/utils';

describe('invitation and commission settings', () => {
  it('renders the non-withdrawable account-credit policy and emits scoped changes', () => {
    const onChange = vi.fn();
    const onSave = vi.fn();
    render(
      <InvitationSettingsPane
        settings={DEFAULT_INVITATION_SETTINGS}
        onChange={onChange}
        onSave={onSave}
      />,
    );

    expect(screen.getByRole('switch', { name: '开启强制邀请' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: '邀请佣金百分比' }).getAttribute('value')).toBe('10');
    expect(screen.getByRole('spinbutton', { name: '用户可创建邀请码上限' }).getAttribute('value')).toBe('5');
    expect(screen.getByText('佣金仅结算为站内余额')).toBeTruthy();
    expect(screen.getByText('余额不支持提现，只能用于购买、续费或升级本站套餐；下单时可由用户选择是否抵扣。')).toBeTruthy();
    expect(screen.queryByText('提现方式')).toBeNull();
    expect(screen.getByRole('switch', { name: '三级分销' })).toBeTruthy();

    fireEvent.click(screen.getByRole('switch', { name: '开启强制邀请' }));
    expect(onChange).toHaveBeenCalledWith({ forcedInvitation: true });
    fireEvent.click(screen.getByRole('button', { name: '保存设置' }));
    expect(onSave).toHaveBeenCalledOnce();
  });

  it('places the new settings tab first and keeps all three existing marketing tabs', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue({ success: true, msg: '', obj: DEFAULT_INVITATION_SETTINGS });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MarketingView
          coupons={[]}
          giftCards={[]}
          commissions={[]}
          onCreateCoupon={vi.fn()}
          onEditCoupon={vi.fn()}
          onIssueGiftCards={vi.fn()}
          onSettle={vi.fn()}
        />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('invitation-settings-pane')).toBeTruthy());
    const tabs = screen.getAllByRole('tab');
    expect(tabs).toHaveLength(4);
    expect(tabs[0]?.textContent).toContain('邀请&佣金设置');
    expect(screen.getByRole('tab', { name: '优惠券 (0)' })).toBeTruthy();
    expect(screen.getByRole('tab', { name: '礼品卡 (0)' })).toBeTruthy();
    expect(screen.getByRole('tab', { name: '邀请佣金 (0)' })).toBeTruthy();
  });
});
