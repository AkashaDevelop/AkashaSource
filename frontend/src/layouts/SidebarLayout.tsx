import { useState, useEffect, useRef, type ReactNode } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Menu, X, ChevronDown } from 'lucide-react';
import ThemeToggle from '../components/ThemeToggle';

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

export interface HeaderMenuItem {
  icon: React.ComponentType<{ size?: number }>;
  label: string;
  onClick: () => void;
  variant?: 'default' | 'primary' | 'danger';
}

export interface HeaderMenuProps {
  initials: string;
  name: string;
  subtitle: string;
  items: HeaderMenuItem[];
}

interface SidebarLayoutProps {
  navGroups: NavGroup[];
  logoContent: ReactNode;
  userCard: ReactNode;
  headerMenu: HeaderMenuProps;
  children: ReactNode;
}

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

/* ── 顶栏用户头像下拉菜单 ── */
function UserDropdown({ menu }: { menu: HeaderMenuProps }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const variantStyle = (v?: string) => {
    if (v === 'danger')   return { color: 'var(--color-danger-fg)' };
    if (v === 'primary')  return { color: 'var(--accent-primary)' };
    return { color: 'var(--text-secondary)' };
  };

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      {/* 触发按钮 */}
      <button
        onClick={() => setOpen(v => !v)}
        className="flex items-center gap-2 px-2 py-1.5 rounded-xl transition-all duration-150"
        style={{
          background: open ? 'var(--nav-hover-bg)' : 'transparent',
          border: '1px solid transparent',
          cursor: 'pointer',
        }}
        onMouseEnter={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'var(--nav-hover-bg)'; }}
        onMouseLeave={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
        aria-label="用户菜单"
        aria-expanded={open}
      >
        {/* 头像 */}
        <div
          className="flex items-center justify-center text-white font-bold"
          style={{
            width: '30px', height: '30px', borderRadius: '50%', fontSize: '12px',
            background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
            boxShadow: open ? '0 0 0 2px var(--accent-primary)' : 'none',
            transition: 'box-shadow 0.15s',
            flexShrink: 0,
          }}
        >
          {menu.initials}
        </div>
        {/* 姓名（md+ 显示） */}
        <div className="hidden md:block text-left" style={{ maxWidth: '120px' }}>
          <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {menu.name}
          </p>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>{menu.subtitle}</p>
        </div>
        <ChevronDown
          size={14}
          className="hidden md:block"
          style={{
            color: 'var(--text-muted)',
            transform: open ? 'rotate(180deg)' : 'rotate(0)',
            transition: 'transform 0.2s',
          }}
        />
      </button>

      {/* 下拉面板 */}
      {open && (
        <div
          className="header-dropdown"
          style={{
            position: 'absolute', top: 'calc(100% + 8px)', right: 0,
            width: '220px', zIndex: 200,
            background: 'var(--bg-elevated)',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-xl)',
            boxShadow: 'var(--shadow-hover)',
            overflow: 'hidden',
          }}
        >
          {/* 用户信息头 */}
          <div style={{ padding: '14px 16px 12px', borderBottom: '1px solid var(--border-color)' }}>
            <div className="flex items-center gap-2.5">
              <div
                className="flex items-center justify-center text-white font-bold flex-shrink-0"
                style={{
                  width: '36px', height: '36px', borderRadius: '50%', fontSize: '13px',
                  background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
                }}
              >
                {menu.initials}
              </div>
              <div className="min-w-0">
                <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {menu.name}
                </p>
                <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>{menu.subtitle}</p>
              </div>
            </div>
          </div>

          {/* 主题切换 */}
          <div
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '10px 16px',
              borderBottom: '1px solid var(--border-color)',
            }}
          >
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>界面主题</span>
            <ThemeToggle />
          </div>

          {/* 操作项 */}
          <div style={{ padding: '6px' }}>
            {menu.items.map((item, i) => {
              const Icon = item.icon;
              const isDanger = item.variant === 'danger';
              return (
                <button
                  key={i}
                  onClick={() => { item.onClick(); setOpen(false); }}
                  className="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-all duration-100"
                  style={{
                    ...variantStyle(item.variant),
                    background: 'transparent',
                    border: 'none',
                    cursor: 'pointer',
                    textAlign: 'left',
                  }}
                  onMouseEnter={e => {
                    (e.currentTarget as HTMLElement).style.background = isDanger ? 'var(--color-danger-bg)' : 'var(--nav-hover-bg)';
                    (e.currentTarget as HTMLElement).style.color = isDanger ? 'var(--color-danger-fg)' : 'var(--text-primary)';
                  }}
                  onMouseLeave={e => {
                    (e.currentTarget as HTMLElement).style.background = 'transparent';
                    (e.currentTarget as HTMLElement).style.color = variantStyle(item.variant).color;
                  }}
                >
                  <Icon size={15} style={{ flexShrink: 0 }} />
                  <span>{item.label}</span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
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
