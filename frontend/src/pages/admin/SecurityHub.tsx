import { Tabs, Tab } from '../../components/ui';
import { ShieldCheck, Eye, ScrollText, KeyRound } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import ContextSanitization from './ContextSanitization';
import XuanJian from './XuanJian';
import AdminAuditLog from './AdminAuditLog';
import SystemLicense from './SystemLicense'; // ⚠️ REMOVABLE MODULE 依赖

// ～安全中心：宸汐清源看内容、宸汐玄鉴看行为、操作审计看管理员自己～
// 后两个 Tab 只有超级管理员才看得到，普通管理员进来只有宸汐清源一个入口哦

export default function SecurityHub() {
  const { user } = useAuthStore();
  const isRoot = (user?.role ?? 0) >= 100;

  return (
    <div className="space-y-6">
      <Tabs aria-label="安全中心">
        <Tab
          key="qingyuan"
          title={
            <div className="flex items-center gap-2">
              <ShieldCheck className="w-4 h-4" />
              <span>宸汐清源</span>
            </div>
          }
        >
          <ContextSanitization />
        </Tab>
        {isRoot && (
          <Tab
            key="xuanjian"
            title={
              <div className="flex items-center gap-2">
                <Eye className="w-4 h-4" />
                <span>宸汐玄鉴</span>
              </div>
            }
          >
            <XuanJian />
          </Tab>
        )}
        {isRoot && (
          <Tab
            key="audit"
            title={
              <div className="flex items-center gap-2">
                <ScrollText className="w-4 h-4" />
                <span>操作审计</span>
              </div>
            }
          >
            <AdminAuditLog />
          </Tab>
        )}
        {/* ↓↓↓ REMOVABLE：系统授权门禁的入口 Tab，整体移除时删掉这一块即可 ↓↓↓ */}
        {isRoot && (
          <Tab
            key="license"
            title={
              <div className="flex items-center gap-2">
                <KeyRound className="w-4 h-4" />
                <span>系统授权</span>
              </div>
            }
          >
            <SystemLicense />
          </Tab>
        )}
        {/* ↑↑↑ REMOVABLE ↑↑↑ */}
      </Tabs>
    </div>
  );
}
