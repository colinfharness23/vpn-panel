import { act, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ThemeProvider } from '@/hooks/useTheme';
import CommercialPage from '@/pages/commercial/CommercialPage';
import LineCenterPane from '@/pages/commercial/LineCenterPane';
import { HttpUtil } from '@/utils';

const lineResponses: Record<string, unknown[]> = {
  '/panel/api/commercial/line-sources': [
    {
      id: 'source-1',
      name: '稳定来源',
      kind: 'url',
      refreshInterval: 1800,
      enabled: true,
      status: 'healthy',
      groupIds: null,
      nodeCount: 1,
      healthyCount: 1,
    },
  ],
  '/panel/api/commercial/line-groups': [],
  '/panel/api/commercial/line-nodes': [],
  '/panel/api/commercial/plans': [],
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

    expect(await screen.findByText('稳定来源')).toBeTruthy();

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
    type SourceResponse = {
      success: boolean;
      msg: string;
      obj: unknown[];
    };
    const deferred = () => {
      let resolve!: (value: SourceResponse) => void;
      const promise = new Promise<SourceResponse>((done) => { resolve = done; });
      return { promise, resolve };
    };
    const older = deferred();
    const newer = deferred();
    let sourceRequest = 0;
    vi.mocked(HttpUtil.get).mockImplementation(async (url) => {
      if (String(url) === '/panel/api/commercial/line-sources') {
        sourceRequest += 1;
        return sourceRequest === 1 ? older.promise : newer.promise;
      }
      return { success: true, msg: '', obj: [] };
    });

    const view = render(<LineCenterPane refreshToken={0} />);
    await waitFor(() => expect(sourceRequest).toBe(1));
    view.rerender(<LineCenterPane refreshToken={1} />);
    await waitFor(() => expect(sourceRequest).toBe(2));

    await act(async () => {
      newer.resolve({
        success: true,
        msg: '',
        obj: [{
          id: 'new', name: '最新来源', kind: 'url', refreshInterval: 1800,
          enabled: true, status: 'healthy', groupIds: [], nodeCount: 1, healthyCount: 1,
        }],
      });
      await newer.promise;
    });
    expect(await screen.findByText('最新来源')).toBeTruthy();

    await act(async () => {
      older.resolve({
        success: true,
        msg: '',
        obj: [{
          id: 'old', name: '过期来源', kind: 'url', refreshInterval: 1800,
          enabled: true, status: 'healthy', groupIds: [], nodeCount: 1, healthyCount: 1,
        }],
      });
      await older.promise;
    });
    expect(screen.queryByText('过期来源')).toBeNull();
    expect(screen.getByText('最新来源')).toBeTruthy();
  });
});
