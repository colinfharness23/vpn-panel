import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { HttpUtil } from '@/utils';

export interface SecuritySettings {
  emailVerification: boolean;
  disallowGmailAliases: boolean;
  safeMode: boolean;
  emailSuffixWhitelistEnabled: boolean;
  allowedEmailSuffixes: string;
  registrationCaptchaEnabled: boolean;
  ipRegistrationLimitEnabled: boolean;
  passwordAttemptLimitEnabled: boolean;
  maxPasswordAttempts: number;
  passwordLockDurationMinutes: number;
}

const SECURITY_SETTINGS_QUERY_KEY = ['commercial', 'security-settings'] as const;

export const DEFAULT_SECURITY_SETTINGS: SecuritySettings = {
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

async function fetchSecuritySettings(): Promise<SecuritySettings> {
  const msg = await HttpUtil.get<SecuritySettings>('/panel/api/commercial/security-settings', undefined, { silent: true });
  if (!msg.success || !msg.obj) throw new Error(msg.msg || '安全设置加载失败');
  return msg.obj;
}

export function useSecuritySettings() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: SECURITY_SETTINGS_QUERY_KEY,
    queryFn: fetchSecuritySettings,
    staleTime: Infinity,
  });
  const server = useMemo(() => ({ ...DEFAULT_SECURITY_SETTINGS, ...(query.data || {}) }), [query.data]);
  const [draft, setDraft] = useState<SecuritySettings>(DEFAULT_SECURITY_SETTINGS);

  useEffect(() => {
    if (query.data) setDraft({ ...DEFAULT_SECURITY_SETTINGS, ...query.data });
  }, [query.data]);

  const updateSecuritySettings = useCallback((patch: Partial<SecuritySettings>) => {
    setDraft((current) => ({ ...current, ...patch }));
  }, []);

  const saveMutation = useMutation({
    mutationFn: async (next: SecuritySettings) => {
      const msg = await HttpUtil.put('/panel/api/commercial/security-settings', next, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!msg.success) throw new Error(msg.msg || '安全设置保存失败');
      return msg;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: SECURITY_SETTINGS_QUERY_KEY }),
  });

  const saveSecuritySettings = useCallback(() => saveMutation.mutateAsync(draft), [draft, saveMutation]);
  const saveDisabled = useMemo(() => JSON.stringify(server) === JSON.stringify(draft), [draft, server]);

  return {
    securitySettings: draft,
    updateSecuritySettings,
    saveSecuritySettings,
    saveDisabled,
    fetched: query.isFetched,
    spinning: query.isFetching || saveMutation.isPending,
    error: query.error instanceof Error ? query.error.message : '',
  };
}
