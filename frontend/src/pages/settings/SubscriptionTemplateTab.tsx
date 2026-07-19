import { Alert, Input, Space } from 'antd';
import { useTranslation } from 'react-i18next';

import { SettingListItem } from '@/components/ui';
import type { AllSetting } from '@/models/setting';

interface SubscriptionTemplateTabProps {
  allSetting: AllSetting;
  updateSetting: (patch: Partial<AllSetting>) => void;
}

export default function SubscriptionTemplateTab({ allSetting, updateSetting }: SubscriptionTemplateTabProps) {
  const { t } = useTranslation();

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="info"
        showIcon
        title="订阅模板"
        description="设置订阅服务读取的静态模板目录。目录必须位于服务器本地，保存并重启面板后生效。"
      />
      <SettingListItem
        paddings="small"
        title={t('pages.settings.subThemeDir')}
        description={t('pages.settings.subThemeDirDesc')}
      >
        <Input
          value={allSetting.subThemeDir}
          placeholder="/etc/3x-ui/sub_templates/my-theme/"
          onChange={(event) => updateSetting({ subThemeDir: event.target.value })}
        />
      </SettingListItem>
    </Space>
  );
}
