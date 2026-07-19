import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import SiteSettingsTab from '@/pages/settings/SiteSettingsTab';
import type { SiteSettings } from '@/pages/settings/useSiteSettings';

const settings: SiteSettings = {
  siteName: 'NOVA',
  siteDescription: '安全连接，简单开始',
  siteUrl: 'https://nova.example',
  forceHttps: false,
  logoUrl: '',
  subscriptionUrls: '',
  termsUrl: '',
  termsTemplate: 'standard',
  termsTitle: '服务使用条款',
  termsContent: '默认条款正文',
  registrationClosed: false,
  trialPlanId: '',
  currency: 'CNY',
  currencySymbol: '¥',
};

describe('site settings tab', () => {
  it('renders the reference fields and emits controlled updates', () => {
    const onChange = vi.fn();
    const onLogoSave = vi.fn().mockResolvedValue(undefined);
    render(<SiteSettingsTab settings={settings} plans={[{ id: 'starter', name: '轻量套餐' }]} onChange={onChange} onLogoSave={onLogoSave} />);

    expect(screen.getByRole('heading', { name: '站点设置' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: '站点网址' }).getAttribute('value')).toBe('https://nova.example');
    expect(screen.getByRole('combobox', { name: '注册试用' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: '使用条款模板' })).toBeTruthy();
    expect(screen.getByRole('button', { name: /上传并应用/ })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: '货币单位' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: '货币符号' })).toBeTruthy();
    expect((screen.getByRole('textbox', { name: '使用条款正文' }) as HTMLTextAreaElement).value).toBe('默认条款正文');

    fireEvent.change(screen.getByRole('textbox', { name: '站点名称' }), { target: { value: '机场面板' } });
    expect(onChange).toHaveBeenCalledWith({ siteName: '机场面板' });

    fireEvent.click(screen.getByRole('switch', { name: '停止新用户注册' }));
    expect(onChange).toHaveBeenCalledWith({ registrationClosed: true });

    fireEvent.change(screen.getByRole('textbox', { name: '使用条款正文' }), { target: { value: '管理员修改后的条款' } });
    expect(onChange).toHaveBeenCalledWith({ termsContent: '管理员修改后的条款' });

    fireEvent.mouseDown(screen.getByRole('combobox', { name: '货币单位' }));
    fireEvent.click(screen.getByText('USD — 美元'));
    expect(onChange).toHaveBeenCalledWith({ currency: 'USD', currencySymbol: '$' });
  });

  it('loads an uploaded image into the controlled logo setting', async () => {
    const onChange = vi.fn();
    const onLogoSave = vi.fn().mockResolvedValue(undefined);
    const { container } = render(<SiteSettingsTab settings={settings} plans={[]} onChange={onChange} onLogoSave={onLogoSave} />);
    const fileInput = container.querySelector<HTMLInputElement>('input[type="file"]');
    expect(fileInput).toBeTruthy();

    const file = new File([new Uint8Array([137, 80, 78, 71])], 'logo.png', { type: 'image/png' });
    fireEvent.change(fileInput!, { target: { files: [file] } });

    await vi.waitFor(() => expect(onLogoSave).toHaveBeenCalledWith(expect.stringMatching(/^data:image\/png;base64,/)));
    expect(screen.getByText('已保存并应用到用户网站。')).toBeTruthy();
  });

  it('accepts logo files larger than the previous 1 MB limit', async () => {
    const onLogoSave = vi.fn().mockResolvedValue(undefined);
    const { container } = render(<SiteSettingsTab settings={settings} plans={[]} onChange={vi.fn()} onLogoSave={onLogoSave} />);
    const fileInput = container.querySelector<HTMLInputElement>('input[type="file"]');
    const file = new File([new Uint8Array(2 * 1024 * 1024)], 'large-logo.png', { type: 'image/png' });

    fireEvent.change(fileInput!, { target: { files: [file] } });

    await vi.waitFor(() => expect(onLogoSave).toHaveBeenCalledOnce());
  });
});
