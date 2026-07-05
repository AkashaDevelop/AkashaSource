import { useEffect } from 'react';
import { Outlet, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard, Key, LogOut, History, User as UserIcon,
  Share2, FlaskConical, Tag, Shield, Files, MessageSquare, ListTodo, CreditCard,
} from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useSystemStore } from '../store/system';
import SidebarLayout, { type NavGroup, type HeaderMenuProps } from './SidebarLayout';
import { formatQuota } from '../lib/quota';

export default function UserLayout() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { systemName, logoUrl, chatLink, chatLink2, fetch: fetchSystem } = useSystemStore();

  useEffect(() => { fetchSystem(); }, []);

  const navGroups: NavGroup[] = [
    {
      label: '概览',
      items: [{ key: '/', icon: LayoutDashboard, label: '仪表盘' }],
    },
    {
      label: '资源',
      items: [
        { key: '/token', icon: Key, label: '令牌管理' },
        { key: '/log', icon: History, label: '调用日志' },
        { key: '/files', icon: Files, label: '文件管理' },
        { key: '/tasks', icon: ListTodo, label: '任务记录' },
      ],
    },
    {
      label: '账户',
      items: [
        { key: '/billing', icon: CreditCard, label: '账单中心' },
        { key: '/invitation', icon: Share2, label: '邀请管理' },
      ],
    },
    {
      label: '工具',
      items: [
        { key: '/pricing', icon: Tag, label: '模型定价' },
        { key: '/playground', icon: FlaskConical, label: 'Playground' },
        { key: '/profile', icon: UserIcon, label: '个人设置' },
      ],
    },
    ...(chatLink || chatLink2 ? [{
      label: '外部',
      items: [
        ...(chatLink  ? [{ key: chatLink,  icon: MessageSquare, label: '对话',  external: true as const }] : []),
        ...(chatLink2 ? [{ key: chatLink2, icon: MessageSquare, label: '对话2', external: true as const }] : []),
      ],
    }] : []),
  ];

  const logoContent = (
    <>
      <div className="flex items-center gap-2.5">
        {logoUrl
          ? <img src={logoUrl} alt="logo" className="w-7 h-7 rounded-full" />
          : <span className="text-xl" style={{ color: 'var(--accent-primary)' }}>✿</span>
        }
        <h1 className="text-base font-bold gradient-text truncate">{systemName}</h1>
      </div>
      <p className="text-xs mt-0.5 ml-9" style={{ color: 'var(--text-muted, var(--text-secondary))' }}>用户控制台</p>
    </>
  );

  const userCard = (
    <div
      className="p-3 rounded-xl cursor-pointer transition-all duration-200"
      style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
      onClick={() => navigate('/profile')}
    >
      <div className="flex items-center gap-2.5">
        <div
          className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold flex-shrink-0"
          style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}
        >
          {user?.username?.[0]?.toUpperCase() || 'U'}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>{user?.username}</p>
          <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>普通用户</p>
        </div>
      </div>
      <div
        className="mt-2 flex items-center justify-between px-2 py-1.5 rounded-lg"
        style={{ background: 'var(--bg-base)', border: '1px solid var(--border-color)' }}
      >
        <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>余额</span>
        <span style={{ fontSize: '12px', fontWeight: 700, color: 'var(--accent-cosmic)' }}>
          ${formatQuota(user?.quota ?? 0, 4)}
        </span>
      </div>
    </div>
  );

  const headerMenu: HeaderMenuProps = {
    initials: (user?.username?.[0] ?? 'U').toUpperCase(),
    name: user?.username ?? '',
    subtitle: '普通用户',
    items: [
      {
        icon: UserIcon,
        label: '个人设置',
        onClick: () => navigate('/profile'),
      },
      ...(user && user.role >= 10 ? [{
        icon: Shield,
        label: '管理后台',
        onClick: () => navigate('/admin'),
        variant: 'primary' as const,
      }] : []),
      {
        icon: LogOut,
        label: '退出登录',
        onClick: () => { logout(); navigate('/login'); },
        variant: 'danger' as const,
      },
    ],
  };

  return (
    <SidebarLayout
      navGroups={navGroups}
      logoContent={logoContent}
      userCard={userCard}
      headerMenu={headerMenu}
    >
      <Outlet />
    </SidebarLayout>
  );
}
