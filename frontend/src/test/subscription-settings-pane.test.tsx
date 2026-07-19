import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { AllSetting } from '@/models/setting';
import SubscriptionGeneralTab from '@/pages/settings/SubscriptionGeneralTab';
import { DEFAULT_SUBSCRIPTION_SETTINGS } from '@/pages/settings/useSubscriptionSettings';

describe('subscription settings tab', () => {
  it('adds the new reference tab before all seven existing subscription tabs', () => {
    const updateSetting = vi.fn();
    const updateSubscriptionSettings = vi.fn();
    render(
      <SubscriptionGeneralTab
        allSetting={new AllSetting({ subPath: '/sub/' })}
        updateSetting={updateSetting}
        subscriptionSettings={DEFAULT_SUBSCRIPTION_SETTINGS}
        updateSubscriptionSettings={updateSubscriptionSettings}
      />,
    );

    const tabs = screen.getAllByRole('tab');
    expect(tabs).toHaveLength(8);
    expect(tabs[0]?.textContent).toContain('订阅设置');
    expect(screen.getByRole('tab', { name: /Clash \/ Mihomo/ })).toBeTruthy();
    expect(screen.getByRole('tab', { name: /Happ/ })).toBeTruthy();
    expect(screen.getByRole('tab', { name: /Incy/ })).toBeTruthy();

    expect(screen.getByRole('combobox', { name: '月流量重置方式' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: '订阅路径' }).getAttribute('value')).toBe('/sub/');
    fireEvent.click(screen.getByRole('switch', { name: '允许用户更改订阅' }));
    expect(updateSubscriptionSettings).toHaveBeenCalledWith({ allowUserChange: false });
    fireEvent.change(screen.getByRole('textbox', { name: '订阅路径' }), { target: { value: 'private-sub' } });
    expect(updateSetting).toHaveBeenCalledWith({ subPath: 'private-sub' });
  });
});
