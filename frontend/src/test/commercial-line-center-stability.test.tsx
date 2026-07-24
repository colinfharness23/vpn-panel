import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ThemeProvider } from '@/hooks/useTheme';
import CommercialPage from '@/pages/commercial/CommercialPage';
import LineCenterPane from '@/pages/commercial/LineCenterPane';
import { HttpUtil } from '@/utils';

const lineResponses: Record<string, unknown[]> = {
  '/panel/api/commercial/line-groups': [
    {
      id: 'group-1',
      name: '稳定线路组',
      description: '',
      active: true,
      planIds: null,
      nodeCount: 1,
      healthyCount: 1,
      publishedCount: 1,
    },
  ],
  '/panel/api/commercial/line-nodes': [],
};

describe('commercial line center stability', () => {
  beforeEach(() => {
    vi.mocked(HttpUtil.get).mockClear();
    vi.mocked(HttpUtil.get).mockImplementation(async (url) => ({
      success: true,
      msg: '',
      obj: lineResponses[String(url)] ?? [],
    }));
  });

  it('loads each line endpoint once when opening #lines and renders the result', async () => {
    render(
      <MemoryRouter initialEntries={['/commercial#lines']}>
        <ThemeProvider>
          <CommercialPage />
        </ThemeProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText('协议链接批量导入')).toBeTruthy();
    expect(screen.queryByText('订阅 URL 自动分配')).toBeNull();
    fireEvent.click(screen.getByText('分组与套餐线路'));
    expect(await screen.findByText('稳定线路组')).toBeTruthy();

    await waitFor(() => {
      for (const endpoint of Object.keys(lineResponses)) {
        const calls = vi.mocked(HttpUtil.get).mock.calls.filter(
          ([url]) => String(url) === endpoint,
        );
        expect(calls).toHaveLength(1);
      }
    });
  });

  it('ignores an older response that finishes after a manual refresh', async () => {
    type GroupResponse = {
      success: boolean;
      msg: string;
      obj: unknown[];
    };
    const deferred = () => {
      let resolve!: (value: GroupResponse) => void;
      const promise = new Promise<GroupResponse>((done) => { resolve = done; });
      return { promise, resolve };
    };
    const older = deferred();
    const newer = deferred();
    let groupRequest = 0;
    vi.mocked(HttpUtil.get).mockImplementation(async (url) => {
      if (String(url) === '/panel/api/commercial/line-groups') {
        groupRequest += 1;
        return groupRequest === 1 ? older.promise : newer.promise;
      }
      return { success: true, msg: '', obj: [] };
    });

    const view = render(<LineCenterPane refreshToken={0} />);
    await waitFor(() => expect(groupRequest).toBe(1));
    view.rerender(<LineCenterPane refreshToken={1} />);
    await waitFor(() => expect(groupRequest).toBe(2));

    await act(async () => {
      newer.resolve({
        success: true,
        msg: '',
        obj: [{
          id: 'new', name: '最新线路组', description: '', active: true,
          planIds: [], nodeCount: 1, healthyCount: 1, publishedCount: 1,
        }],
      });
      await newer.promise;
    });
    fireEvent.click(screen.getByText('分组与套餐线路'));
    expect(await screen.findByText('最新线路组')).toBeTruthy();

    await act(async () => {
      older.resolve({
        success: true,
        msg: '',
        obj: [{
          id: 'old', name: '过期线路组', description: '', active: true,
          planIds: [], nodeCount: 1, healthyCount: 1, publishedCount: 1,
        }],
      });
      await older.promise;
    });
    expect(screen.queryByText('过期线路组')).toBeNull();
    expect(screen.getByText('最新线路组')).toBeTruthy();
  });
});
