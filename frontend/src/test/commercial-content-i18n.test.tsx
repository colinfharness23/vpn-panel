import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ThemeProvider } from '@/hooks/useTheme';
import CommercialPage from '@/pages/commercial/CommercialPage';
import { HttpUtil } from '@/utils';

const notice = {
  id: 'notice-1',
  slug: 'welcome',
  level: 'info',
  titleI18n: JSON.stringify({ 'zh-CN': '服务公告', 'en-US': 'Service notice' }),
  contentI18n: JSON.stringify({
    'zh-CN': '欢迎使用 PHEERO。',
    'en-US': 'Welcome to PHEERO.',
    'ja-JP': 'PHEERO へようこそ。',
  }),
  published: true,
  updatedAt: '2026-07-22T00:00:00Z',
};

describe('commercial localized content editor', () => {
  beforeEach(() => {
    vi.mocked(HttpUtil.get).mockImplementation(async (url) => ({
      success: true,
      msg: '',
      obj: String(url) === '/panel/api/commercial/notices' ? [notice] : [],
    }));
    vi.mocked(HttpUtil.post).mockClear();
    vi.mocked(HttpUtil.post).mockResolvedValue({ success: true, msg: '', obj: notice });
  });

  it('preserves unmounted locale tabs when only Chinese is edited', async () => {
    render(
      <MemoryRouter initialEntries={['/commercial#content']}>
        <ThemeProvider>
          <CommercialPage />
        </ThemeProvider>
      </MemoryRouter>,
    );

    await screen.findByText('服务公告');
    fireEvent.click(screen.getByText('编辑'));
    const chineseContent = await waitFor(() => {
      const textarea = document.querySelector<HTMLTextAreaElement>('textarea');
      expect(textarea).not.toBeNull();
      return textarea!;
    });
    expect(chineseContent).toBeTruthy();
    fireEvent.change(chineseContent, { target: { value: '欢迎使用新的 PHEERO。' } });
    const saveButton = document.querySelector<HTMLButtonElement>('.ant-modal-footer .ant-btn-primary');
    expect(saveButton).toBeTruthy();
    fireEvent.click(saveButton!);

    await waitFor(() => expect(HttpUtil.post).toHaveBeenCalled());
    const [, payload] = vi.mocked(HttpUtil.post).mock.calls[0];
    const saved = payload as { contentI18n: string; titleI18n: string };
    expect(JSON.parse(saved.contentI18n)).toEqual({
      'zh-CN': '欢迎使用新的 PHEERO。',
      'en-US': 'Welcome to PHEERO.',
      'ja-JP': 'PHEERO へようこそ。',
    });
    expect(JSON.parse(saved.titleI18n)['en-US']).toBe('Service notice');
  });
});
