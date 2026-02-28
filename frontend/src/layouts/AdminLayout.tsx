import { useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { LayoutDashboard, Server, Settings, Users, Gift, ScrollText, Layers, Box, ArrowLeft, Crown, Database, ListTodo } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useSystemStore } from '../store/system';
import ThemeToggle from '../components/ThemeToggle';

const navGroups = [
  {
    label: '概览',
    items: [
      { key: '/admin', icon: LayoutDashboard, label: '系统概览' },
    ],
  },
  {
    label: '用户',
    items: [
      { key: '/admin/user', icon: Users, label: '用户管理' },
      { key: '/admin/redemption', icon: Gift, label: '兑换码' },
      { key: '/admin/invitation', icon: Users, label: '邀请码管理' },
    ],
  },
  {
    label: '配置',
    items: [
      { key: '/admin/channel', icon: Server, label: '渠道管理' },
      { key: '/admin/group', icon: Layers, label: '分组管理' },
      { key: '/admin/model', icon: Box, label: '模型管理' },
      { key: '/admin/subscription', icon: Crown, label: '订阅套餐' },
    ],
  },
  {
    label: '运维',
    items: [
      { key: '/admin/tasks', icon: ListTodo, label: '任务管理' },
      { key: '/admin/log', icon: ScrollText, label: '日志管理' },
      { key: '/admin/migration', icon: Database, label: '数据库迁移' },
      { key: '/admin/setting', icon: Settings, label: '系统设置' },
    ],
  },
];

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { systemName, logoUrl, fetch: fetchSystem } = useSystemStore();

  useEffect(() => { fetchSystem(); }, []);

  const handleLogout = () => { logout(); navigate('/login'); };

  return (
    <div className="flex min-h-screen w-full" style={{ background: 'var(--bg-base)' }}>
      {/* 侧边栏 */}
      <div
        className="w-60 flex flex-col flex-shrink-0"
        style={{
          background: 'var(--bg-sidebar)',
          borderRight: '1px solid var(--border-color)',
          backdropFilter: 'blur(12px)',
        }}
      >
        {/* Logo */}
        <div className="px-5 pt-5 pb-3">
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
        </div>

        {/* 用户信息卡片 */}
        <div
          className="mx-3 mb-3 p-3 rounded-xl flex items-center gap-2.5 cursor-pointer transition-all duration-200"
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
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>超级管理员</p>
          </div>
        </div>

        {/* 导航分组 */}
        <nav className="flex-1 px-3 overflow-y-auto space-y-3 pb-2">
          {navGroups.map(group => (
            <div key={group.label}>
              <p
                className="px-2 mb-1 text-xs font-semibold uppercase tracking-wider"
                style={{ color: 'var(--text-secondary)', opacity: 0.5 }}
              >
                {group.label}
              </p>
              <div className="space-y-0.5">
                {group.items.map(({ key, icon: Icon, label }) => {
                  const isActive = location.pathname === key;
                  return (
                    <button
                      key={key}
                      onClick={() => navigate(key)}
                      className="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm font-medium transition-all duration-150 relative"
                      style={isActive ? {
                        background: 'var(--nav-active-bg)',
                        color: 'var(--accent-primary)',
                      } : {
                        color: 'var(--text-secondary)',
                      }}
                      onMouseEnter={e => {
                        if (!isActive) {
                          (e.currentTarget as HTMLElement).style.background = 'var(--nav-hover-bg)';
                          (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)';
                        }
                      }}
                      onMouseLeave={e => {
                        if (!isActive) {
                          (e.currentTarget as HTMLElement).style.background = 'transparent';
                          (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)';
                        }
                      }}
                    >
                      {isActive && (
                        <span
                          className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-4 rounded-full"
                          style={{ background: 'var(--accent-primary)' }}
                        />
                      )}
                      <span
                        className="flex items-center justify-center w-6 h-6 rounded-md flex-shrink-0"
                        style={isActive ? { background: 'rgba(124,58,237,0.15)' } : {}}
                      >
                        <Icon size={15} />
                      </span>
                      <span>{label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        {/* 底部操作区 */}
        <div className="p-3 space-y-1.5" style={{ borderTop: '1px solid var(--border-color)' }}>
          <div
            className="flex items-center justify-between px-2 py-1.5 rounded-lg"
            style={{ background: 'var(--bg-elevated)' }}
          >
            <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>主题</span>
            <ThemeToggle />
          </div>
          <button
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150"
            style={{ background: 'rgba(124,58,237,0.1)', color: 'var(--accent-primary)' }}
            onClick={() => navigate('/')}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(124,58,237,0.18)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(124,58,237,0.1)'; }}
          >
            <ArrowLeft size={15} />
            <span>用户控制台</span>
          </button>
          <button
            className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150"
            style={{ background: 'rgba(248,113,113,0.1)', color: '#f87171' }}
            onClick={handleLogout}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(248,113,113,0.18)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(248,113,113,0.1)'; }}
          >
            <span>退出登录</span>
          </button>
        </div>
      </div>

      {/* 主内容区 */}
      <div className="flex-1 overflow-auto p-6 animate-fade-in-up">
        <Outlet />
      </div>
    </div>
  );
}
