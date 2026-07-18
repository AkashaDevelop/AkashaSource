import { Tabs, Tab } from '../../components/ui';
import { Boxes, DollarSign, Building2 } from 'lucide-react';
import ModelMetaManagement from './ModelMeta';
import ModelPricing from './ModelPricing';
import VendorManagement from './Vendor';

// ～模型中心：元数据、定价、供应商，模型相关的一切都在这一个 Hub 里啦～

export default function ModelHub() {
  return (
    <div className="space-y-6">
      <Tabs aria-label="模型中心">
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
          key="vendor"
          title={
            <div className="flex items-center gap-2">
              <Building2 className="w-4 h-4" />
              <span>供应商管理</span>
            </div>
          }
        >
          <VendorManagement />
        </Tab>
      </Tabs>
    </div>
  );
}
