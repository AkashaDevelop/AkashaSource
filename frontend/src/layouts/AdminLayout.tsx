import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Button } from '../components/ui';
import { LayoutDashboard, Server, Settings, LogOut, Users, Gift, ScrollText, Layers, Box, ArrowLeft } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import ThemeToggle from '../components/ThemeToggle';

const navItems = [
  { key: '/admin', icon: LayoutDashboard, label: '系统概览' },
  { key: '/admin/channel', icon: Server, label: '渠道管理' },
  { key: '/admin/user', icon: Users, label: '用户管理' },
  { key: '/admin/redemption', icon: Gift, label: '兑换码' },
  { key: '/admin/group', icon: Layers, label: '分组管理' },
  { key: '/admin/model', icon: Box, label: '模型管理' },
  { key: '/admin/log', icon: ScrollText, label: '日志管理' },
  { key: '/admin/setting', icon: Settings, label: '系统设置' },
];

export default function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen w-full" style={{ background: 'var(--bg-base)' }}>
      {/* 侧边栏 */}
      <div
        className="w-64 flex flex-col flex-shrink-0"
        style={{
          background: 'var(--bg-sidebar)',
          borderRight: '1px solid var(--border-color)',
          backdropFilter: 'blur(12px)',
        }}
      >
        {/* Logo */}
        <div className="p-6 pb-4">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xl" style={{ color: 'var(--accent-star)' }}>✦</span>
            <h1 className="text-xl font-bold gradient-text">Akasha</h1>
          </div>
          <div className="flex items-center gap-2 ml-7 mt-1">
            <span
              className="px-2 py-0.5 text-xs font-semibold rounded-full"
              style={{
                background: 'rgba(251,191,36,0.15)',
                border: '1px solid rgba(251,191,36,0.3)',
                color: 'var(--accent-star)',
              }}
            >
              管理员
            </span>
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>控制台</p>
          </div>
        </div>

        {/* 用户信息 */}
        <div
          className="mx-4 mb-4 p-3 rounded-2xl flex items-center gap-3 cursor-pointer transition-all duration-200"
          style={{
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-color)',
          }}
          onClick={() => navigate('/profile')}
        >
          <div
            className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-bold flex-shrink-0"
            style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}
          >
            {user?.username?.[0]?.toUpperCase() || 'A'}
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>{user?.username}</p>
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>超级管理员</p>
          </div>
        </div>

        {/* 导航 */}
        <nav className="flex-1 px-3 space-y-0.5 overflow-y-auto">
          {navItems.map(({ key, icon: Icon, label }) => {
            const isActive = location.pathname === key;
            return (
              <button
                key={key}
                onClick={() => navigate(key)}
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm font-medium transition-all duration-200"
                style={isActive ? {
                  background: 'var(--nav-active-bg)',
                  color: 'var(--accent-primary)',
                  boxShadow: 'var(--shadow-card)',
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
                <Icon size={17} />
                <span>{label}</span>
                {isActive && <span className="ml-auto text-xs" style={{ color: 'var(--accent-primary)' }}>◆</span>}
              </button>
            );
          })}
        </nav>

        {/* 底部 */}
        <div className="p-4 space-y-2" style={{ borderTop: '1px solid var(--border-color)' }}>
          <div className="flex items-center justify-between">
            <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>主题切换</span>
            <ThemeToggle />
          </div>
          <Button
            variant="flat"
            startContent={<ArrowLeft size={16} />}
            className="w-full"
            style={{
              background: 'rgba(124,58,237,0.12)',
              color: 'var(--accent-primary)',
              borderRadius: '12px',
            }}
            onPress={() => navigate('/')}
          >
            用户控制台
          </Button>
          <Button
            className="w-full"
            style={{
              background: 'rgba(248,113,113,0.12)',
              color: '#f87171',
              borderRadius: '12px',
            }}
            onPress={handleLogout}
          >
            退出登录
          </Button>
        </div>
      </div>

      {/* 主内容区 */}
      <div className="flex-1 overflow-auto p-6 animate-fade-in-up">
        <Outlet />
      </div>
    </div>
  );
}
