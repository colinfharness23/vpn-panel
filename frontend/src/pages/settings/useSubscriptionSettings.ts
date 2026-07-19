import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { HttpUtil } from '@/utils';

export type SubscriptionEvent = 'none' | 'reset_traffic';
export type MonthlyResetMode = 'calendar_month' | 'billing_cycle' | 'never';

export interface SubscriptionSettings {
  allowUserChange: boolean;
  monthlyResetMode: MonthlyResetMode;
  offsetEnabled: boolean;
  purchaseEvent: SubscriptionEvent;
  renewalEvent: SubscriptionEvent;
  changeEvent: SubscriptionEvent;
  showSubscriptionInfo: boolean;
  showProtocolInNodeName: boolean;
}

const SUBSCRIPTION_SETTINGS_QUERY_KEY = ['commercial', 'subscription-settings'] as const;

export const DEFAULT_SUBSCRIPTION_SETTINGS: SubscriptionSettings = {
  allowUserChange: true,
  monthlyResetMode: 'calendar_month',
  offsetEnabled: false,
  purchaseEvent: 'none',
  renewalEvent: 'none',
  changeEvent: 'none',
  showSubscriptionInfo: false,
  showProtocolInNodeName: false,
};

async function fetchSubscriptionSettings(): Promise<SubscriptionSettings> {
  const msg = await HttpUtil.get<SubscriptionSettings>('/panel/api/commercial/subscription-settings', undefined, { silent: true });
  if (!msg.success || !msg.obj) throw new Error(msg.msg || '订阅设置加载失败');
  return msg.obj;
}

export function useSubscriptionSettings() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: SUBSCRIPTION_SETTINGS_QUERY_KEY,
    queryFn: fetchSubscriptionSettings,
    staleTime: Infinity,
  });
  const server = useMemo(() => ({ ...DEFAULT_SUBSCRIPTION_SETTINGS, ...(query.data || {}) }), [query.data]);
  const [draft, setDraft] = useState<SubscriptionSettings>(DEFAULT_SUBSCRIPTION_SETTINGS);

  useEffect(() => {
    if (query.data) setDraft({ ...DEFAULT_SUBSCRIPTION_SETTINGS, ...query.data });
  }, [query.data]);

  const updateSubscriptionSettings = useCallback((patch: Partial<SubscriptionSettings>) => {
    setDraft((current) => ({ ...current, ...patch }));
  }, []);

  const saveMutation = useMutation({
    mutationFn: async (next: SubscriptionSettings) => {
      const msg = await HttpUtil.put('/panel/api/commercial/subscription-settings', next, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!msg.success) throw new Error(msg.msg || '订阅设置保存失败');
      return msg;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: SUBSCRIPTION_SETTINGS_QUERY_KEY }),
  });

  const saveSubscriptionSettings = useCallback(() => saveMutation.mutateAsync(draft), [draft, saveMutation]);
  const saveDisabled = useMemo(() => JSON.stringify(server) === JSON.stringify(draft), [draft, server]);

  return {
    subscriptionSettings: draft,
    updateSubscriptionSettings,
    saveSubscriptionSettings,
    saveDisabled,
    fetched: query.isFetched,
    spinning: query.isFetching || saveMutation.isPending,
    error: query.error instanceof Error ? query.error.message : '',
  };
}
