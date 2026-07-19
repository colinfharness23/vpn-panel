import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { HttpUtil } from '@/utils';

export interface InvitationSettings {
  forcedInvitation: boolean;
  commissionPercent: number;
  maxInviteCodes: number;
  inviteCodesNeverExpire: boolean;
  commissionFirstPaymentOnly: boolean;
  commissionAutoConfirm: boolean;
  multiLevelEnabled: boolean;
}

export const DEFAULT_INVITATION_SETTINGS: InvitationSettings = {
  forcedInvitation: false,
  commissionPercent: 10,
  maxInviteCodes: 5,
  inviteCodesNeverExpire: true,
  commissionFirstPaymentOnly: true,
  commissionAutoConfirm: true,
  multiLevelEnabled: false,
};

const INVITATION_SETTINGS_QUERY_KEY = ['commercial', 'invitation-settings'] as const;

async function fetchInvitationSettings(): Promise<InvitationSettings> {
  const msg = await HttpUtil.get<InvitationSettings>('/panel/api/commercial/invitation-settings', undefined, { silent: true });
  if (!msg.success || !msg.obj) throw new Error(msg.msg || '邀请与佣金设置加载失败');
  return msg.obj;
}

export function useInvitationSettings() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: INVITATION_SETTINGS_QUERY_KEY,
    queryFn: fetchInvitationSettings,
    staleTime: Infinity,
  });
  const server = useMemo(() => ({ ...DEFAULT_INVITATION_SETTINGS, ...(query.data || {}) }), [query.data]);
  const [draft, setDraft] = useState<InvitationSettings>(DEFAULT_INVITATION_SETTINGS);

  useEffect(() => {
    if (query.data) setDraft({ ...DEFAULT_INVITATION_SETTINGS, ...query.data });
  }, [query.data]);

  const updateInvitationSettings = useCallback((patch: Partial<InvitationSettings>) => {
    setDraft((current) => ({ ...current, ...patch }));
  }, []);

  const saveMutation = useMutation({
    mutationFn: async (next: InvitationSettings) => {
      const msg = await HttpUtil.put('/panel/api/commercial/invitation-settings', next, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!msg.success) throw new Error(msg.msg || '邀请与佣金设置保存失败');
      return msg;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: INVITATION_SETTINGS_QUERY_KEY }),
  });

  const saveInvitationSettings = useCallback(() => saveMutation.mutateAsync(draft), [draft, saveMutation]);
  const saveDisabled = useMemo(() => JSON.stringify(server) === JSON.stringify(draft), [draft, server]);

  return {
    invitationSettings: draft,
    updateInvitationSettings,
    saveInvitationSettings,
    saveDisabled,
    spinning: query.isFetching || saveMutation.isPending,
    error: query.error instanceof Error ? query.error.message : '',
  };
}
