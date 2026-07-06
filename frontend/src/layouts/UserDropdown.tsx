import { useState, useEffect, useRef } from 'react';
import { ChevronDown } from 'lucide-react';
import ThemeToggle from '../components/ThemeToggle';

export interface HeaderMenuItem {
  icon: React.ComponentType<{ size?: number; style?: React.CSSProperties }>;
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

const variantColor = (v?: string) => {
  if (v === 'danger') return { color: 'var(--color-danger-fg)' };
  if (v === 'primary') return { color: 'var(--accent-primary)' };
  return { color: 'var(--text-secondary)' };
};

/* ── 顶栏用户头像下拉菜单 ── */
export default function UserDropdown({ menu }: { menu: HeaderMenuProps }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

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
                    ...variantColor(item.variant),
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
                    (e.currentTarget as HTMLElement).style.color = variantColor(item.variant).color;
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
