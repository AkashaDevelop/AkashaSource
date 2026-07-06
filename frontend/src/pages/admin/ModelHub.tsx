import { Tabs, Tab } from '../../components/ui';
import { Box, Boxes, DollarSign } from 'lucide-react';
import ModelManagement from './Model';
import ModelMetaManagement from './ModelMeta';
import ModelPricing from './ModelPricing';

// ～模型中心：计费配置和供应商模型目录，都装进这一个 Hub 里啦～

export default function ModelHub() {
  return (
    <div className="space-y-6">
      <Tabs aria-label="模型中心">
        <Tab
          key="config"
          title={
            <div className="flex items-center gap-2">
              <Box className="w-4 h-4" />
              <span>模型配置</span>
            </div>
          }
        >
          <ModelManagement />
        </Tab>
        <Tab
          key="pricing"
          title={
            <div className="flex items-center gap-2">
              <DollarSign className="w-4 h-4" />
              <span>模型定价</span>
            </div>
          }
        >
          <ModelPricing />
        </Tab>
        <Tab
          key="meta"
          title={
            <div className="flex items-center gap-2">
              <Boxes className="w-4 h-4" />
              <span>模型元数据</span>
            </div>
          }
        >
          <ModelMetaManagement />
        </Tab>
      </Tabs>
    </div>
  );
}
