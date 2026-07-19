import { screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import LoginPage from '@/pages/login/LoginPage';
import { renderWithProviders } from '@/test/test-utils';
import { HttpUtil } from '@/utils';

describe('administrator login two-factor challenge', () => {
  beforeEach(() => {
    vi.mocked(HttpUtil.get).mockResolvedValue({
      success: true,
      msg: '',
      obj: { site: { siteName: 'NOVA' } },
    });
  });

  it('shows a plain verification-code field only when two-factor authentication is enabled', async () => {
    vi.mocked(HttpUtil.post).mockResolvedValue({ success: true, msg: '', obj: true });

    renderWithProviders(<LoginPage />);

    const code = await screen.findByLabelText('验证码');
    expect(code.getAttribute('inputmode')).toBe('numeric');
    expect(code.getAttribute('maxlength')).toBe('6');
    expect(screen.queryByText(/TOTP|2FA/i)).toBeNull();
  });

  it('does not ask for a verification code when two-factor authentication is disabled', async () => {
    vi.mocked(HttpUtil.post).mockResolvedValue({ success: true, msg: '', obj: false });

    renderWithProviders(<LoginPage />);

    expect(await screen.findByRole('heading', { name: '登录管理员后台' })).toBeTruthy();
    expect(screen.queryByLabelText('验证码')).toBeNull();
  });
});
