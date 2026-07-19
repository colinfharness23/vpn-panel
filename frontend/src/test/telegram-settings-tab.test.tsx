import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AllSetting } from '@/models/setting';
import TelegramTab from '@/pages/settings/TelegramTab';
import { HttpUtil } from '@/utils';

describe('Telegram settings', () => {
  afterEach(() => vi.restoreAllMocks());

  it('persists the customer support link from the dedicated support tab', () => {
    const updateSetting = vi.fn();
    render(<TelegramTab allSetting={new AllSetting()} updateSetting={updateSetting} />);

    fireEvent.change(screen.getByPlaceholderText('https://t.me/your_username'), {
      target: { value: 'https://t.me/my_support' },
    });

    expect(updateSetting).toHaveBeenLastCalledWith({ tgGroupLink: 'https://t.me/my_support' });
  });

  it('only enables the test link for a direct Telegram destination', () => {
    const updateSetting = vi.fn();
    const { rerender } = render(<TelegramTab allSetting={new AllSetting({ tgGroupLink: 'https://t.me/' })} updateSetting={updateSetting} />);
    expect((screen.getByRole('button', { name: /Test Telegram link/ }) as HTMLButtonElement).disabled).toBe(true);

    rerender(<TelegramTab allSetting={new AllSetting({ tgGroupLink: 'https://t.me/my_support' })} updateSetting={updateSetting} />);
    expect((screen.getByRole('link', { name: /Test Telegram link/ }) as HTMLAnchorElement).getAttribute('href')).toBe('https://t.me/my_support');
  });

  it('configures the fixed webhook receiver and reports the result', async () => {
    const post = vi.spyOn(HttpUtil, 'post').mockResolvedValue({
      success: true,
      msg: 'Telegram webhook is active',
      obj: { url: 'https://vpn.example.com/telegram/webhook', pendingUpdateCount: 0 },
    });
    const updateSetting = vi.fn();
    const settings = new AllSetting({ tgWebhookURL: 'https://vpn.example.com/telegram/webhook' });
    render(<TelegramTab allSetting={settings} updateSetting={updateSetting} />);

    fireEvent.click(screen.getByRole('tab', { name: /Bot settings/ }));
    fireEvent.click(screen.getByRole('button', { name: /Set Webhook/ }));

    await waitFor(() => expect(post).toHaveBeenCalledWith(
      '/panel/api/setting/configureTgWebhook',
      { url: 'https://vpn.example.com/telegram/webhook' },
    ));
    expect(await screen.findByText('Telegram webhook is active')).toBeTruthy();
    expect(updateSetting).toHaveBeenCalledWith({ tgWebhookURL: 'https://vpn.example.com/telegram/webhook' });
  });
});
