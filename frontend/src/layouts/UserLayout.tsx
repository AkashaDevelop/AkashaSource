import { Outlet } from 'react-router-dom';
import {
  LayoutDashboard, Key, History, User as UserIcon,
  Share2, FlaskConical, Tag, Shield, Files, MessageSquare, ListTodo, CreditCard,
} from 'lucide-react';
import { useSystemStore } from '../store/system';
import SidebarLayout, { type NavGroup, useLayoutCommon } from './SidebarLayout';

export default function UserLayout() {
  const { chatLink, chatLink2 } = useSystemStore();

  const { logoContent, userCard, headerMenu } = useLayoutCommon({
    defaultInitial: 'U',
    logoIcon: '✿',
    logoIconColor: 'var(--accent-primary)',
    subtitle: '用户控制台',
    roleLabel: '普通用户',
    showBalance: true,
    switchTarget: { path: '/admin', label: '管理后台', icon: Shield, minRole: 10 },
  });

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
