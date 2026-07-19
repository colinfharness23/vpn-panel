import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import SecuritySettingsPane from '@/pages/settings/SecuritySettingsPane';
import type { SecuritySettings } from '@/pages/settings/useSecuritySettings';

const settings: SecuritySettings = {
  emailVerification: true,
  disallowGmailAliases: true,
  safeMode: false,
  emailSuffixWhitelistEnabled: true,
  allowedEmailSuffixes: 'gmail.com',
  registrationCaptchaEnabled: false,
  ipRegistrationLimitEnabled: false,
  passwordAttemptLimitEnabled: true,
  maxPasswordAttempts: 5,
  passwordLockDurationMinutes: 60,
};

describe('security settings pane', () => {
  it('renders the reference controls and emits controlled changes', () => {
    const onChange = vi.fn();
    const onBackendPathChange = vi.fn();
    render(
      <SecuritySettingsPane
        settings={settings}
        backendPath="/admin-secure/"
        onChange={onChange}
        onBackendPathChange={onBackendPathChange}
      />,
    );

    expect(screen.getByRole('heading', { name: '安全设置' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: '后台路径' }).getAttribute('value')).toBe('admin-secure');
    expect(screen.getByRole('spinbutton', { name: '尝试次数' }).getAttribute('value')).toBe('5');
    expect(screen.getByRole('spinbutton', { name: '锁定时长' }).getAttribute('value')).toBe('60');

    fireEvent.click(screen.getByRole('switch', { name: '安全模式' }));
    expect(onChange).toHaveBeenCalledWith({ safeMode: true });

    fireEvent.change(screen.getByRole('textbox', { name: '后台路径' }), { target: { value: 'new-admin' } });
    expect(onBackendPathChange).toHaveBeenCalledWith('/new-admin/');
  });
});
