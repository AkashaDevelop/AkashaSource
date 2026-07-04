import { useEffect } from 'react';
import { Outlet, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard, Server, Settings, Users, Gift, ScrollText, Layers, Box,
  Crown, Database, Building2, Ticket, ListTodo,
  ArrowLeft, LogOut, User as UserIcon, Activity, Shield, CreditCard,
} from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useSystemStore } from '../store/system';
import SidebarLayout, { type NavGroup, type HeaderMenuProps } from './SidebarLayout';

const navGroups: NavGroup[] = [
  {
    label: '工作台',
    items: [
      { key: '/admin', icon: LayoutDashboard, label: '系统概览' },
      { key: '/admin/system-monitor', icon: Activity, label: '系统监控' },
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
      { key: '/admin/migration', icon: Database, label: '数据迁移' },
      { key: '/admin/security', icon: Shield, label: '安全中心' },
    ],
  },
];

export default function AdminLayout() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { systemName, logoUrl, fetch: fetchSystem } = useSystemStore();

  useEffect(() => { fetchSystem(); }, []);

  const filteredGroups = navGroups.map(group => ({
    ...group,
    items: group.items.filter(
      item => !(['/admin/setting', '/admin/payment'].includes(item.key) && (user?.role ?? 0) < 100)
    ),
  }));

  const logoContent = (
    <>
      <div className="flex items-center gap-2.5">
        {logoUrl
          ? <img src={logoUrl} alt="logo" className="w-7 h-7 rounded-full" />
          : <span className="text-xl" style={{ color: 'var(--accent-star)' }}>✦</span>
        }
        <h1 className="text-base font-bold gradient-text truncate">{systemName}</h1>
      </div>
      <div className="flex items-center gap-1.5 mt-1 ml-9">
        <span
          className="px-1.5 py-0.5 text-xs font-semibold rounded-md"
          style={{
            background: 'rgba(251,191,36,0.15)',
            border: '1px solid rgba(251,191,36,0.25)',
            color: 'var(--accent-star)',
          }}
        >
          Admin
        </span>
        <p className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.6 }}>控制台</p>
      </div>
    </>
  );

  const userCard = (
    <div
      className="p-3 rounded-xl flex items-center gap-2.5 cursor-pointer transition-all duration-200"
      style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
      onClick={() => navigate('/profile')}
    >
      <div
        className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold flex-shrink-0"
        style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}
      >
        {user?.username?.[0]?.toUpperCase() || 'A'}
      </div>
      <div className="min-w-0">
        <p className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>{user?.username}</p>
        <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>
          {user?.role && user.role >= 100 ? '超级管理员' : '管理员'}
        </p>
      </div>
    </div>
  );

  const headerMenu: HeaderMenuProps = {
    initials: (user?.username?.[0] ?? 'A').toUpperCase(),
    name: user?.username ?? '',
    subtitle: (user?.role ?? 0) >= 100 ? '超级管理员' : '管理员',
    items: [
      {
        icon: UserIcon,
        label: '个人设置',
        onClick: () => navigate('/profile'),
      },
      {
        icon: ArrowLeft,
        label: '用户控制台',
        onClick: () => navigate('/'),
        variant: 'primary',
      },
      {
        icon: LogOut,
        label: '退出登录',
        onClick: () => { logout(); navigate('/login'); },
        variant: 'danger',
      },
    ],
  };

  return (
    <SidebarLayout
      navGroups={filteredGroups}
      logoContent={logoContent}
      userCard={userCard}
      headerMenu={headerMenu}
    >
      <Outlet />
    </SidebarLayout>
  );
}
