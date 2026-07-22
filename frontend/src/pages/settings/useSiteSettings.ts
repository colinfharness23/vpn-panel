import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { HttpUtil } from '@/utils';
import { updateSiteBranding } from '@/hooks/useSiteBranding';

export interface SiteSettings {
  siteName: string;
  siteDescription: string;
  siteUrl: string;
  forceHttps: boolean;
  logoUrl: string;
  subscriptionUrls: string;
  termsUrl: string;
  termsTemplate: string;
  termsTitle: string;
  termsContent: string;
  registrationClosed: boolean;
  trialPlanId: string;
  currency: string;
  currencySymbol: string;
}

export interface TrialPlanOption {
  id: string;
  name: string;
}

interface SiteSettingsResponse {
  settings: SiteSettings;
  plans: TrialPlanOption[];
}

const SITE_SETTINGS_QUERY_KEY = ['commercial', 'site-settings'] as const;

const DEFAULT_TERMS_CONTENT = `欢迎使用本站服务。注册或购买前，请完整阅读以下条款：

1. 账户与安全
用户应提供真实、有效的注册信息，并妥善保管账户、密码及订阅链接。因主动分享或保管不当造成的损失由用户自行承担。
2. 服务使用范围
本站提供网络连接订阅及相关技术支持。用户不得利用服务从事违法活动、网络攻击、垃圾信息发送、欺诈、侵犯他人权益或其他可能影响平台及节点安全的行为。
3. 套餐、流量与设备
套餐流量、有效期、设备限制及重置周期以购买页面显示为准。超出套餐限制或套餐到期后，相关服务可能自动暂停。
4. 支付与款项确认
数字订阅在支付确认后自动开通。订单支付成功后不可撤销或退回款项，请在付款前确认套餐、周期及设备限制。
5. 服务调整与账号处置
为保障节点及其他用户的正常使用，本站可对异常流量、滥用、攻击或违反本条款的账户采取限制、暂停或终止服务等措施。
6. 隐私与数据
本站仅处理提供账户、订阅、支付、安全审计及客户支持所必要的信息，不记录用户具体浏览内容。

继续注册即表示您已阅读、理解并同意遵守本条款。`;

const DEFAULT_SITE_SETTINGS: SiteSettings = {
  siteName: 'NOVA',
  siteDescription: '稳定连接，清晰可控',
  siteUrl: '',
  forceHttps: false,
  logoUrl: '',
  subscriptionUrls: '',
  termsUrl: '',
  termsTemplate: 'standard',
  termsTitle: '服务使用条款',
  termsContent: DEFAULT_TERMS_CONTENT,
  registrationClosed: false,
  trialPlanId: '',
  currency: 'CNY',
  currencySymbol: '¥',
};

async function fetchSiteSettings(): Promise<SiteSettingsResponse> {
  const msg = await HttpUtil.get<SiteSettingsResponse>('/panel/api/commercial/site-settings', undefined, { silent: true });
  if (!msg.success || !msg.obj) throw new Error(msg.msg || '站点设置加载失败');
  return msg.obj;
}

export function useSiteSettings() {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: SITE_SETTINGS_QUERY_KEY,
    queryFn: fetchSiteSettings,
    staleTime: Infinity,
  });
  const server = useMemo(() => ({ ...DEFAULT_SITE_SETTINGS, ...(query.data?.settings || {}) }), [query.data]);
  const [draft, setDraft] = useState<SiteSettings>(DEFAULT_SITE_SETTINGS);

  useEffect(() => {
    if (query.data?.settings) setDraft({ ...DEFAULT_SITE_SETTINGS, ...query.data.settings });
  }, [query.data]);

  const updateSiteSettings = useCallback((patch: Partial<SiteSettings>) => {
    setDraft((current) => ({ ...current, ...patch }));
  }, []);

  const saveMutation = useMutation({
    mutationFn: async (next: SiteSettings) => {
      const msg = await HttpUtil.put('/panel/api/commercial/site-settings', next, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!msg.success) throw new Error(msg.msg || '站点设置保存失败');
      return msg;
    },
    onSuccess: (_message, next) => {
      updateSiteBranding({
        siteName: next.siteName.trim() || 'NOVA',
        logoUrl: next.logoUrl.trim(),
      });
      return queryClient.invalidateQueries({ queryKey: SITE_SETTINGS_QUERY_KEY });
    },
  });

  const logoMutation = useMutation({
    mutationFn: async (logoUrl: string) => {
      const msg = await HttpUtil.put('/panel/api/commercial/site-settings/logo', { logoUrl }, {
        headers: { 'Content-Type': 'application/json' },
      });
      if (!msg.success) throw new Error(msg.msg || 'LOGO 保存失败');
      return logoUrl;
    },
    onSuccess: (logoUrl) => {
      setDraft((current) => ({ ...current, logoUrl }));
      updateSiteBranding({
        siteName: draft.siteName.trim() || 'NOVA',
        logoUrl,
      });
    },
  });

  const saveSiteSettings = useCallback(() => saveMutation.mutateAsync(draft), [draft, saveMutation]);
  const saveSiteLogo = useCallback((logoUrl: string) => logoMutation.mutateAsync(logoUrl), [logoMutation]);
  const saveDisabled = useMemo(
    () => JSON.stringify({ ...server, logoUrl: draft.logoUrl }) === JSON.stringify(draft),
    [draft, server],
  );

  return {
    siteSettings: draft,
    trialPlans: query.data?.plans || [],
    updateSiteSettings,
    saveSiteSettings,
    saveSiteLogo,
    logoSaving: logoMutation.isPending,
    saveDisabled,
    fetched: query.isFetched,
    spinning: query.isFetching || saveMutation.isPending || logoMutation.isPending,
    error: query.error instanceof Error ? query.error.message : '',
  };
}
