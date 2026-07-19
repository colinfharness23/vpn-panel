import { createRoot } from 'react-dom/client';
import 'antd/dist/reset.css';
import '@/portal/portal.css';

import PortalApp from '@/portal/PortalApp';

const root = document.getElementById('portal');
if (root) {
  createRoot(root).render(<PortalApp />);
}
