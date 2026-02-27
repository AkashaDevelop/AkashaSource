import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Button } from '../components/ui';
import { LayoutDashboard, Key, LogOut, History, Wallet, User as UserIcon, Share2, FlaskConical, Tag, Shield } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import ThemeToggle from '../components/ThemeToggle';

const navItems = [
  { key: '/', icon: LayoutDashboard, label: '仪表盘' },
  { key: '/token', icon: Key, label: '令牌管理' },
  { key: '/log', icon: History, label: '调用日志' },
  { key: '/topup', icon: Wallet, label: '充值' },
  { key: '/invitation', icon: Share2, label: '邀请管理' },
  { key: '/pricing', icon: Tag, label: '模型定价' },
  { key: '/profile', icon: UserIcon, label: '个人设置' },
  { key: '/playground', icon: FlaskConical, label: 'Playground' },
];

export default function UserLayout() {
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
            <span className="text-2xl" style={{ color: 'var(--accent-primary)' }}>✿</span>
            <h1 className="text-xl font-bold gradient-text">Akasha</h1>
          </div>
          <p className="text-xs ml-8" style={{ color: 'var(--text-secondary)' }}>用户控制台</p>
        </div>

        {/* 用户信息 */}
        <div
          className="mx-4 mb-4 p-3 rounded-2xl cursor-pointer transition-all duration-200"
          style={{
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-color)',
          }}
          onClick={() => navigate('/profile')}
        >
          <div className="flex items-center gap-3 mb-2">
            <div
              className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-bold flex-shrink-0"
              style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}
            >
              {user?.username?.[0]?.toUpperCase() || 'U'}
            </div>
            <div className="min-w-0">
              <p className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>{user?.username}</p>
              <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>普通用户</p>
            </div>
          </div>
          {/* 余额显示 */}
          <div style={{
            padding: '6px 10px', borderRadius: '8px',
            background: 'var(--bg-base)', border: '1px solid var(--border-color)',
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          }}>
            <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>余额</span>
            <span style={{ fontSize: '12px', fontWeight: 700, color: 'var(--accent-cosmic)' }}>
              ${((user?.quota ?? 0) / 500000).toFixed(4)}
            </span>
          </div>
        </div>

        {/* 导航 */}
        <nav className="flex-1 px-3 space-y-1">
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
                <Icon size={18} />
                <span>{label}</span>
                {isActive && <span className="ml-auto text-xs" style={{ color: 'var(--accent-primary)' }}>✦</span>}
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
          {user && user.role >= 10 && (
            <Button
              variant="flat"
              startContent={<Shield size={16} />}
              className="w-full"
              style={{
                background: 'rgba(124,58,237,0.12)',
                color: 'var(--accent-primary)',
                borderRadius: '12px',
              }}
              onPress={() => navigate('/admin')}
            >
              管理后台
            </Button>
          )}
          <Button
            variant="flat"
            startContent={<LogOut size={16} />}
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
