import { useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { LayoutDashboard, Key, LogOut, History, Wallet, User as UserIcon, Share2, FlaskConical, Tag, Shield, Crown, Files, MessageSquare, ListTodo } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useSystemStore } from '../store/system';
import ThemeToggle from '../components/ThemeToggle';

const navGroups = [
  {
    label: '概览',
    items: [
      { key: '/', icon: LayoutDashboard, label: '仪表盘' },
    ],
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
      { key: '/topup', icon: Wallet, label: '充值' },
      { key: '/subscription', icon: Crown, label: '订阅套餐' },
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
];

export default function UserLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { systemName, logoUrl, chatLink, chatLink2, fetch: fetchSystem } = useSystemStore();

  useEffect(() => { fetchSystem(); }, []);

  const handleLogout = () => { logout(); navigate('/login'); };

  const externalItems = [
    ...(chatLink ? [{ key: chatLink, icon: MessageSquare, label: '对话', external: true }] : []),
    ...(chatLink2 ? [{ key: chatLink2, icon: MessageSquare, label: '对话2', external: true }] : []),
  ];

  return (
    <div className="flex min-h-screen w-full" style={{ background: 'var(--bg-base)' }}>
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
              : <span className="text-xl" style={{ color: 'var(--accent-primary)' }}>✿</span>
            }
            <h1 className="text-base font-bold gradient-text truncate">{systemName}</h1>
          </div>
          <p className="text-xs mt-0.5 ml-9" style={{ color: 'var(--text-muted, var(--text-secondary))' }}>用户控制台</p>
        </div>

        {/* 用户信息卡片 */}
        <div
          className="mx-3 mb-3 p-3 rounded-xl cursor-pointer transition-all duration-200"
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
              ${((user?.quota ?? 0) / 500000).toFixed(4)}
            </span>
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

          {/* 外部链接 */}
          {externalItems.length > 0 && (
            <div>
              <p
                className="px-2 mb-1 text-xs font-semibold uppercase tracking-wider"
                style={{ color: 'var(--text-secondary)', opacity: 0.5 }}
              >
                外部
              </p>
              <div className="space-y-0.5">
                {externalItems.map(({ key, icon: Icon, label }) => (
                  <button
                    key={key}
                    onClick={() => window.open(key, '_blank')}
                    className="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm font-medium transition-all duration-150"
                    style={{ color: 'var(--text-secondary)' }}
                    onMouseEnter={e => {
                      (e.currentTarget as HTMLElement).style.background = 'var(--nav-hover-bg)';
                      (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)';
                    }}
                    onMouseLeave={e => {
                      (e.currentTarget as HTMLElement).style.background = 'transparent';
                      (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)';
                    }}
                  >
                    <span className="flex items-center justify-center w-6 h-6 rounded-md flex-shrink-0">
                      <Icon size={15} />
                    </span>
                    <span>{label}</span>
                    <span className="ml-auto text-xs opacity-40">↗</span>
                  </button>
                ))}
              </div>
            </div>
          )}
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
          {user && user.role >= 10 && (
            <button
              className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150"
              style={{ background: 'rgba(124,58,237,0.1)', color: 'var(--accent-primary)' }}
              onClick={() => navigate('/admin')}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(124,58,237,0.18)'; }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(124,58,237,0.1)'; }}
            >
              <Shield size={15} />
              <span>管理后台</span>
            </button>
          )}
          <button
            className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150"
            style={{ background: 'rgba(248,113,113,0.1)', color: '#f87171' }}
            onClick={handleLogout}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(248,113,113,0.18)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(248,113,113,0.1)'; }}
          >
            <LogOut size={15} />
            <span>退出登录</span>
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-6 animate-fade-in-up">
        <Outlet />
      </div>
    </div>
  );
}
