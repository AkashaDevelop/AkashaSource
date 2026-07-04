import { Tabs, Tab } from '../../components/ui';
import { Server, UserCheck, Network } from 'lucide-react';
import ChannelManagement from './Channel';
import ChannelAccountManagement from './ChannelAccount';
import ChannelAffinity from './ChannelAffinity';

// ～渠道中心：把"渠道管理/渠道运维/路由亲和性"这三种看渠道的视角收进一个 Hub～
// 三个 Tab 内容直接复用原页面组件，不动它们内部的业务逻辑，
// 只是换了个"家"，侧边栏因此能精简掉两个入口啦 (｡•ᴗ•｡)

export default function ChannelHub() {
  return (
    <div className="space-y-6">
      <Tabs aria-label="渠道中心">
        <Tab
          key="manage"
          title={
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4" />
              <span>渠道管理</span>
            </div>
          }
        >
          <ChannelManagement />
        </Tab>
        <Tab
          key="ops"
          title={
            <div className="flex items-center gap-2">
              <UserCheck className="w-4 h-4" />
              <span>渠道运维</span>
            </div>
          }
        >
          <ChannelAccountManagement />
        </Tab>
        <Tab
          key="affinity"
          title={
            <div className="flex items-center gap-2">
              <Network className="w-4 h-4" />
              <span>路由亲和性</span>
            </div>
          }
        >
          <ChannelAffinity />
        </Tab>
      </Tabs>
    </div>
  );
}
