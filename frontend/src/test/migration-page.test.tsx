import { fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import MigrationPage from '@/pages/migration/MigrationPage';
import { renderWithProviders } from '@/test/test-utils';
import { HttpUtil } from '@/utils';

vi.mock('@/utils', () => ({
  HttpUtil: {
    get: vi.fn().mockResolvedValue({
      success: true,
      msg: '',
      obj: {
        supported: false,
        platform: 'windows',
        database: 'postgres',
        domain: 'vpn.example.com',
        configured: false,
      },
    }),
    post: vi.fn(),
  },
}));

vi.mock('@/layouts/AppSidebar', () => ({
  default: () => <aside aria-label="管理员导航" />,
}));

describe('migration page', () => {
  beforeEach(() => {
    vi.mocked(HttpUtil.post).mockReset();
  });

  it('shows the minimal SSH connection form and safety boundaries', async () => {
    renderWithProviders(<MigrationPage />);

    expect(screen.getByRole('heading', { name: '一键迁移' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: '服务器 IP 地址' })).toBeTruthy();
    expect(screen.getByRole('spinbutton', { name: 'SSH 端口' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: 'SSH 用户名' })).toBeTruthy();
    expect(screen.getByRole('button', { name: /检测连接与迁移环境/ })).toBeTruthy();
    expect(screen.getByText(/域名应继续指向旧服务器/)).toBeTruthy();
    expect(screen.getByText('旧服务器不自动删除')).toBeTruthy();
    expect(await screen.findByText('本地预览环境不能执行真实迁移')).toBeTruthy();
  });

  it('shows a persistent actionable error when SSH preflight fails', async () => {
    vi.mocked(HttpUtil.post).mockResolvedValueOnce({
      success: false,
      msg: 'SSH 连接失败：目标端没有返回 SSH 握手信息',
      obj: null,
    });
    renderWithProviders(<MigrationPage />);

    fireEvent.change(screen.getByRole('textbox', { name: '服务器 IP 地址' }), { target: { value: '203.0.113.10' } });
    fireEvent.change(screen.getByRole('textbox', { name: 'SSH 用户名' }), { target: { value: 'root' } });
    fireEvent.change(screen.getByLabelText('SSH 密码'), { target: { value: 'secret' } });
    fireEvent.click(screen.getByRole('button', { name: /检测连接与迁移环境/ }));

    expect(await screen.findByText('迁移环境检测失败')).toBeTruthy();
    expect(screen.getByText(/没有返回 SSH 握手信息/)).toBeTruthy();
    expect(screen.getByRole('button', { name: '重新检测' })).toBeTruthy();
  });
});
