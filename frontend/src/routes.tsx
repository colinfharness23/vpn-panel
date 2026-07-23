import { lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate, type RouteObject } from 'react-router-dom';

import PanelLayout from '@/layouts/PanelLayout';

const IndexPage = lazy(() => import('@/pages/index/IndexPage'));
const InboundsPage = lazy(() => import('@/pages/inbounds/InboundsPage'));
const ClientsPage = lazy(() => import('@/pages/clients/ClientsPage'));
const GroupsPage = lazy(() => import('@/pages/groups/GroupsPage'));
const NodesPage = lazy(() => import('@/pages/nodes/NodesPage'));
const HostsPage = lazy(() => import('@/pages/hosts/HostsPage'));
const SettingsPage = lazy(() => import('@/pages/settings/SettingsPage'));
const XrayPage = lazy(() => import('@/pages/xray/XrayPage'));
const ApiDocsPage = lazy(() => import('@/pages/api-docs/ApiDocsPage'));
const CommercialPage = lazy(() => import('@/pages/commercial/CommercialPage'));
const MigrationPage = lazy(() => import('@/pages/migration/MigrationPage'));

function withSuspense(node: React.ReactNode) {
  return <Suspense fallback={null}>{node}</Suspense>;
}

const commercialMode =
  typeof window !== 'undefined' && window.X_UI_COMMERCIAL_MODE === true;
const legacyEngineRoute = (node: React.ReactNode) =>
  commercialMode ? <Navigate to="/commercial" replace /> : withSuspense(node);

const routes: RouteObject[] = [
  {
    path: '/',
    element: <PanelLayout />,
    children: [
      { index: true, element: withSuspense(<IndexPage />) },
      { path: 'inbounds', element: legacyEngineRoute(<InboundsPage />) },
      { path: 'clients', element: legacyEngineRoute(<ClientsPage />) },
      { path: 'groups', element: legacyEngineRoute(<GroupsPage />) },
      { path: 'nodes', element: legacyEngineRoute(<NodesPage />) },
      { path: 'hosts', element: legacyEngineRoute(<HostsPage />) },
      { path: 'settings', element: withSuspense(<SettingsPage />) },
      { path: 'xray', element: legacyEngineRoute(<XrayPage />) },
      { path: 'outbound', element: legacyEngineRoute(<XrayPage />) },
      { path: 'routing', element: legacyEngineRoute(<XrayPage />) },
      { path: 'api-docs', element: withSuspense(<ApiDocsPage />) },
      { path: 'commercial', element: withSuspense(<CommercialPage />) },
      { path: 'migration', element: withSuspense(<MigrationPage />) },
    ],
  },
];

function computeBasename() {
  const raw = (typeof window !== 'undefined' && window.X_UI_BASE_PATH) || '/';
  const trimmed = raw.replace(/\/+$/, '');
  return `${trimmed}/panel`;
}

export const router = createBrowserRouter(routes, {
  basename: computeBasename(),
});
