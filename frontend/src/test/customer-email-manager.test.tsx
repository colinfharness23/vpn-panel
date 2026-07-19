import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AllSetting } from '@/models/setting';
import EmailTab from '@/pages/settings/EmailTab';
import { HttpUtil } from '@/utils';

const templates = [
  {
    key: 'announcement',
    name: '运营公告',
    subject: '[{{site_name}}] 服务公告',
    bodyHtml: '<p>{{display_name}}，您好</p>',
    active: true,
    system: true,
    sortOrder: 10,
  },
  {
    key: 'subscription_activated',
    name: '订阅开通成功',
    subject: '[{{site_name}}] 您的订阅已开通',
    bodyHtml: '<p>订阅已经开通</p>',
    active: true,
    system: true,
    sortOrder: 20,
  },
];

describe('customer email settings', () => {
  afterEach(() => vi.restoreAllMocks());

  it('offers common SMTP ports while preserving custom port input', () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue({ success: true, msg: '', obj: templates });
    const updateSetting = vi.fn();

    render(<EmailTab allSetting={new AllSetting()} updateSetting={updateSetting} />);

    fireEvent.click(screen.getByRole('button', { name: '25 / STARTTLS' }));
    expect(updateSetting).toHaveBeenLastCalledWith({ smtpPort: 25, smtpEncryptionType: 'starttls' });

    fireEvent.click(screen.getByRole('button', { name: '465 / SSL/TLS' }));
    expect(updateSetting).toHaveBeenLastCalledWith({ smtpPort: 465, smtpEncryptionType: 'tls' });

    fireEvent.click(screen.getByRole('button', { name: '587 / STARTTLS' }));
    expect(updateSetting).toHaveBeenLastCalledWith({ smtpPort: 587, smtpEncryptionType: 'starttls' });

    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '2525' } });
    expect(updateSetting).toHaveBeenLastCalledWith({ smtpPort: 2525 });
  });

  it('fills common SMTP provider settings and labels implicit SSL/TLS clearly', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue({ success: true, msg: '', obj: templates });
    const updateSetting = vi.fn();

    render(<EmailTab allSetting={new AllSetting()} updateSetting={updateSetting} />);

    expect(screen.getByPlaceholderText('user@example.com')).toBeTruthy();
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Common email provider' }));
    fireEvent.click(await screen.findByText('QQ 邮箱'));
    expect(updateSetting).toHaveBeenLastCalledWith({
      smtpHost: 'smtp.qq.com',
      smtpPort: 465,
      smtpEncryptionType: 'tls',
    });

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Encryption' }));
    expect(await screen.findByText('SSL/TLS (implicit, usually port 465)')).toBeTruthy();
  });

  it('keeps system alerts and adds template and customer-send tabs', async () => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(async (url: string) => {
      if (url.includes('email-templates')) return { success: true, msg: '', obj: templates } as never;
      return { success: true, msg: '', obj: { items: [], total: 0 } } as never;
    });

    render(<EmailTab allSetting={new AllSetting()} updateSetting={vi.fn()} />);

    const tabs = screen.getAllByRole('tab');
    expect(tabs).toHaveLength(4);
    expect(screen.getByRole('tab', { name: /SMTP Settings/ })).toBeTruthy();
    expect(screen.getByRole('tab', { name: /系统告警/ })).toBeTruthy();
    expect(screen.getByRole('tab', { name: /邮件模板/ })).toBeTruthy();
    expect(screen.getByRole('tab', { name: /发送邮件/ })).toBeTruthy();

    fireEvent.click(screen.getByRole('tab', { name: /系统告警/ }));
    expect(await screen.findByText('发送给管理员的运行告警')).toBeTruthy();
    expect(screen.getByText(/不会发送给机场用户/)).toBeTruthy();

    fireEvent.click(screen.getByRole('tab', { name: /邮件模板/ }));
    expect(await screen.findByText('用户邮件模板')).toBeTruthy();
    expect(screen.getByRole('button', { name: /保存模板/ })).toBeTruthy();

    fireEvent.click(screen.getByRole('tab', { name: /发送邮件/ }));
    expect(await screen.findByText('发送给机场用户')).toBeTruthy();
    expect(screen.getByRole('button', { name: /加入发送队列/ })).toBeTruthy();
  });

  it('loads and persists an editable template', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue({ success: true, msg: '', obj: templates });
    const put = vi.spyOn(HttpUtil, 'put').mockResolvedValue({
      success: true,
      msg: '模板已保存',
      obj: { ...templates[0], name: '重要运营公告' },
    });

    render(<EmailTab allSetting={new AllSetting()} updateSetting={vi.fn()} />);
    fireEvent.click(screen.getByRole('tab', { name: /邮件模板/ }));

    const nameInput = await screen.findByRole('textbox', { name: '模板名称' });
    await waitFor(() => expect(nameInput.getAttribute('value')).toBe('运营公告'));
    fireEvent.change(nameInput, { target: { value: '重要运营公告' } });
    fireEvent.click(screen.getByRole('button', { name: /保存模板/ }));

    await waitFor(() => expect(put).toHaveBeenCalledWith(
      '/panel/api/commercial/email-templates/announcement',
      expect.objectContaining({ name: '重要运营公告' }),
      { headers: { 'Content-Type': 'application/json' } },
    ));
  });
});
