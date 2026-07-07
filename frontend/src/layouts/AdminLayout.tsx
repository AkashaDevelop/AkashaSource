import { Outlet, useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import {
  LayoutDashboard, Server, Settings, Users, Gift, ScrollText, Layers, Box,
  Crown, Building2, Ticket, ListTodo,
  ArrowLeft, CreditCard, Shield, Sparkles, ShieldAlert,
} from 'lucide-react';
import { useAuthStore } from '../store/auth';
import SidebarLayout, { type NavGroup, useLayoutCommon } from './SidebarLayout';

const navGroups: NavGroup[] = [
  {
    label: '工作台',
    items: [
      { key: '/admin', icon: LayoutDashboard, label: '系统概览' },
      { key: '/admin/log', icon: ScrollText, label: '日志管理' },
      { key: '/admin/payment',   icon: CreditCard,  label: '支付管理' },
      { key: '/admin/tasks', icon: ListTodo, label: '任务记录' },
    ],
  },
  {
    label: '用户运营',
    items: [
      { key: '/admin/user', icon: Users, label: '用户管理' },
      { key: '/admin/subscription', icon: Crown, label: '订阅套餐' },
      { key: '/admin/redemption', icon: Gift, label: '兑换码' },
      { key: '/admin/invitation', icon: Ticket, label: '邀请码管理' },
    ],
  },
  {
    label: '模型资源',
    items: [
      { key: '/admin/channel', icon: Server, label: '渠道中心' },
      { key: '/admin/model', icon: Box, label: '模型中心' },
      { key: '/admin/vendor', icon: Building2, label: '供应商与部署' },
      { key: '/admin/group', icon: Layers, label: '分组管理' },
    ],
  },
  {
    label: '系统配置',
    items: [
      { key: '/admin/setting', icon: Settings, label: '系统设置' },
      { key: '/admin/security', icon: Shield, label: '安全中心' },
      { key: '/admin/changelog', icon: Sparkles, label: '更新日志' },
    ],
  },
];

export default function AdminLayout() {
  const { user, token } = useAuthStore();
  const navigate = useNavigate();
  const isRoot = (user?.role ?? 0) >= 100;

  // ↓↓↓ REMOVABLE：系统授权门禁的提示横幅，整体移除时删掉这个 state + effect + JSX 块即可 ↓↓↓
  const [licenseWarning, setLicenseWarning] = useState(false);
  useEffect(() => {
    if (!isRoot) return;
    fetch('/api/system-license/status', { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        if (data.code === 0 && data.data?.feature_enabled && !data.data?.authorized) {
          setLicenseWarning(true);
        }
      })
      .catch(() => {});
  }, [isRoot, token]);
  // ↑↑↑ REMOVABLE ↑↑↑

  const { logoContent, userCard, headerMenu } = useLayoutCommon({
    defaultInitial: 'A',
    logoIcon: '✦',
    logoIconColor: 'var(--accent-star)',
    subtitle: '控制台',
    badge: { label: 'Admin' },
    roleLabel: (user?.role ?? 0) >= 100 ? '超级管理员' : '管理员',
    switchTarget: { path: '/', label: '用户控制台', icon: ArrowLeft },
  });

  const filteredGroups = navGroups.map(group => ({
    ...group,
    items: group.items.filter(
      item => !(['/admin/setting', '/admin/payment'].includes(item.key) && (user?.role ?? 0) < 100)
    ),
  }));

  return (
    <SidebarLayout
      navGroups={filteredGroups}
      logoContent={logoContent}
      userCard={userCard}
      headerMenu={headerMenu}
    >
      {/* ↓↓↓ REMOVABLE：系统授权门禁的提示横幅 ↓↓↓ */}
      {licenseWarning && (
        <div
          onClick={() => navigate('/admin/security')}
          className="flex items-center gap-2 mb-4 px-4 py-2.5 rounded-xl cursor-pointer"
          style={{ background: 'rgba(251,191,36,0.1)', border: '1px solid rgba(251,191,36,0.3)', color: 'var(--accent-star)' }}
        >
          <ShieldAlert size={16} className="flex-shrink-0" />
          <span className="text-sm">系统未完成 GitHub 组织授权，功能已受限，点击前往「安全中心 → 系统授权」完成授权</span>
        </div>
      )}
      {/* ↑↑↑ REMOVABLE ↑↑↑ */}
      <Outlet />
    </SidebarLayout>
  );
}
