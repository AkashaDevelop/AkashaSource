import { Tabs, Tab } from '../../components/ui';
import { Building2, Rocket } from 'lucide-react';
import VendorManagement from './Vendor';
import DeploymentManagement from './Deployment';

// ～供应商与部署：算力供给侧的两块配置放一起，逛起来更顺手～

export default function VendorHub() {
  return (
    <div className="space-y-6">
      <Tabs aria-label="供应商与部署">
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
        <Tab
          key="deployment"
          title={
            <div className="flex items-center gap-2">
              <Rocket className="w-4 h-4" />
              <span>部署管理</span>
            </div>
          }
        >
          <DeploymentManagement />
        </Tab>
      </Tabs>
    </div>
  );
}
