import type { PortalCopy } from './translations';

export function buildPortalNavigation<SectionKey extends string>(copy: PortalCopy, authenticated: boolean): Array<{ key: SectionKey; label: string }> {
  if (!authenticated) return [];
  const items = [
    { key: 'home', label: copy.home },
    { key: 'subscription', label: copy.subscription },
    { key: 'plans', label: copy.plans },
    { key: 'guides', label: copy.guides },
    { key: 'tickets', label: copy.tickets },
  ];

  return items as Array<{ key: SectionKey; label: string }>;
}
