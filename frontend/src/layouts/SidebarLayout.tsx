import { useState, useEffect, type ReactNode } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Menu, X, LogOut, User as UserIcon } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useSystemStore } from '../store/system';
import { formatQuota } from '../lib/quota';
import UserDropdown, { type HeaderMenuProps, type HeaderMenuItem } from './UserDropdown';

export type { HeaderMenuProps, HeaderMenuItem };

export interface NavItem {
  key: string;
  icon: React.ComponentType<{ size?: number }>;
  label: string;
  external?: boolean;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

interface SidebarLayoutProps {
  navGroups: NavGroup[];
  logoContent: ReactNode;
  userCard: ReactNode;
  headerMenu: HeaderMenuProps;
  children: ReactNode;
}

/* ── 共享布局数据 hook：封装 logo / 用户卡片 / 头像菜单的构建逻辑 ── */
export interface LayoutCommonOptions {
  defaultInitial: string;
  logoIcon: ReactNode;
  logoIconColor: string;
  subtitle: string;
  badge?: { label: string };
  roleLabel: string;
  showBalance?: boolean;
  switchTarget?: { path: string; label: string; icon: React.ComponentType<{ size?: number }>; minRole?: number };
}

export function useLayoutCommon(options: LayoutCommonOptions) {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { systemName, logoUrl, fetch: fetchSystem } = useSystemStore();

  useEffect(() => { fetchSystem(); }, []);

  const initials = (user?.username?.[0] ?? options.defaultInitial).toUpperCase();

  const logoContent = (
    <>
      <div className="flex items-center gap-2.5">
        {logoUrl
          ? <img src={logoUrl} alt="logo" className="w-7 h-7 rounded-full" />
          : <span className="text-xl" style={{ color: options.logoIconColor }}>{options.logoIcon}</span>
        }
        <h1 className="text-base font-bold gradient-text truncate">{systemName}</h1>
      </div>
      {options.badge ? (
        <div className="flex items-center gap-1.5 mt-1 ml-9">
          <span
            className="px-1.5 py-0.5 text-xs font-semibold rounded-md"
            style={{
              background: 'rgba(251,191,36,0.15)',
              border: '1px solid rgba(251,191,36,0.25)',
              color: 'var(--accent-star)',
            }}
          >
            {options.badge.label}
          </span>
          <p className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.6 }}>{options.subtitle}</p>
        </div>
      ) : (
        <p className="text-xs mt-0.5 ml-9" style={{ color: 'var(--text-muted, var(--text-secondary))' }}>{options.subtitle}</p>
      )}
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
          {initials}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>{user?.username}</p>
          <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>{options.roleLabel}</p>
        </div>
      </div>
      {options.showBalance && (
        <div
          className="mt-2 flex items-center justify-between px-2 py-1.5 rounded-lg"
          style={{ background: 'var(--bg-base)', border: '1px solid var(--border-color)' }}
        >
          <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>余额</span>
          <span style={{ fontSize: '12px', fontWeight: 700, color: 'var(--accent-cosmic)' }}>
            ${formatQuota(user?.quota ?? 0, 4)}
          </span>
        </div>
      )}
    </div>
  );

  const showSwitch = options.switchTarget && (!options.switchTarget.minRole || (user?.role ?? 0) >= options.switchTarget.minRole);

  const headerMenu: HeaderMenuProps = {
    initials,
    name: user?.username ?? '',
    subtitle: options.roleLabel,
    items: [
      { icon: UserIcon, label: '个人设置', onClick: () => navigate('/profile') },
      ...(showSwitch && options.switchTarget ? [{
        icon: options.switchTarget.icon,
        label: options.switchTarget.label,
        onClick: () => navigate(options.switchTarget!.path),
        variant: 'primary' as const,
      }] : []),
      { icon: LogOut, label: '退出登录', onClick: () => { logout(); navigate('/login'); }, variant: 'danger' as const },
    ],
  };

  return { logoContent, userCard, headerMenu, user };
}

/* ── 导航按钮 ── */
function NavButton({ item, isActive, onClick }: {
  item: NavItem;
  isActive: boolean;
  onClick: () => void;
}) {
  const navigate = useNavigate();
  const Icon = item.icon;
  return (
    <button
      onClick={() => {
        if (item.external) window.open(item.key, '_blank');
        else navigate(item.key);
        onClick();
      }}
      className={`nav-item w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-sm font-medium relative${isActive ? ' active' : ''}`}
      aria-current={isActive ? 'page' : undefined}
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
      <span>{item.label}</span>
      {item.external && <span className="ml-auto text-xs opacity-40">↗</span>}
    </button>
  );
}

/* ── 主布局组件 ── */
export default function SidebarLayout({ navGroups, logoContent, userCard, headerMenu, children }: SidebarLayoutProps) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const location = useLocation();

  useEffect(() => { setMobileOpen(false); }, [location.pathname]);

  const sidebar = (
    <div
      className="w-60 flex flex-col flex-shrink-0 h-full"
      style={{
        background: 'var(--bg-sidebar)',
        borderRight: '1px solid var(--border-color)',
        backdropFilter: 'blur(12px)',
      }}
    >
      {/* Logo */}
      <div className="px-5 pt-5 pb-3">
        {logoContent}
      </div>

      {/* 用户信息卡片 */}
      <div className="px-3 mb-3">
        {userCard}
      </div>

      {/* 导航分组 */}
      <nav className="flex-1 px-3 overflow-y-auto space-y-3 pb-4">
        {navGroups.map(group => (
          <div key={group.label}>
            <p
              className="px-2 mb-1 text-xs font-semibold uppercase tracking-wider"
              style={{ color: 'var(--text-secondary)', opacity: 0.5 }}
            >
              {group.label}
            </p>
            <div className="space-y-0.5">
              {group.items.map(item => (
                <NavButton
                  key={item.key}
                  item={item}
                  isActive={!item.external && location.pathname === item.key}
                  onClick={() => setMobileOpen(false)}
                />
              ))}
            </div>
          </div>
        ))}
      </nav>
    </div>
  );

  /* 顶栏（桌面 + 移动端共用右侧用户菜单） */
  const topBar = (
    <div
      className="sticky top-0 z-30 flex items-center justify-between px-4 md:px-6"
      style={{
        height: '52px',
        background: 'var(--bg-sidebar)',
        borderBottom: '1px solid var(--border-color)',
        backdropFilter: 'blur(12px)',
      }}
    >
      {/* 左侧：移动端汉堡菜单，桌面端为空 */}
      <div className="flex items-center gap-3">
        <button
          className="md:hidden p-1.5 rounded-lg"
          onClick={() => setMobileOpen(v => !v)}
          style={{ color: 'var(--text-secondary)', background: 'none', border: 'none', cursor: 'pointer' }}
          aria-label={mobileOpen ? '关闭菜单' : '打开菜单'}
        >
          {mobileOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
        {/* 移动端 Logo 文字 */}
        <span className="md:hidden text-sm font-bold gradient-text">Akasha</span>
      </div>

      {/* 右侧：用户头像下拉 */}
      <UserDropdown menu={headerMenu} />
    </div>
  );

  return (
    <div className="flex min-h-screen w-full" style={{ background: 'var(--bg-base)' }}>
      {/* 桌面端侧边栏 */}
      <div className="hidden md:flex flex-col flex-shrink-0 w-60 min-h-screen">
        {sidebar}
      </div>

      {/* 移动端遮罩 */}
      {mobileOpen && (
        <div
          className="sidebar-mobile-overlay md:hidden"
          onClick={() => setMobileOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* 移动端抽屉 */}
      <div className={`sidebar-mobile-drawer md:hidden${mobileOpen ? ' open' : ''}`} aria-label="导航菜单">
        {sidebar}
      </div>

      {/* 主内容区 */}
      <div className="flex-1 flex flex-col min-w-0">
        {topBar}
        <div className="flex-1 overflow-auto p-4 md:p-6 animate-fade-in-up">
          <div className="max-w-[1600px] mx-auto">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
