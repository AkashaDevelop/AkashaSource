import { createPortal } from 'react-dom';
import { useToastStore } from '../store/toast';
import type { ToastType } from '../store/toast';

const icons: Record<ToastType, string> = {
  success: '✓',
  error: '✕',
  warning: '⚠',
  info: 'ℹ',
};

const colors: Record<ToastType, { bg: string; border: string; icon: string; title: string }> = {
  success: {
    bg: 'rgba(16,185,129,0.12)',
    border: 'rgba(16,185,129,0.35)',
    icon: '#34d399',
    title: '#34d399',
  },
  error: {
    bg: 'rgba(239,68,68,0.12)',
    border: 'rgba(239,68,68,0.35)',
    icon: '#f87171',
    title: '#f87171',
  },
  warning: {
    bg: 'rgba(245,158,11,0.12)',
    border: 'rgba(245,158,11,0.35)',
    icon: '#fbbf24',
    title: '#fbbf24',
  },
  info: {
    bg: 'rgba(8,145,178,0.12)',
    border: 'rgba(8,145,178,0.35)',
    icon: '#22d3ee',
    title: '#22d3ee',
  },
};

export default function ToastProvider() {
  const { toasts, remove } = useToastStore();

  return createPortal(
    <div style={{
      position: 'fixed',
      top: '20px',
      right: '20px',
      zIndex: 10000,
      display: 'flex',
      flexDirection: 'column',
      gap: '10px',
      pointerEvents: 'none',
    }}>
      {toasts.map((t) => {
        const c = colors[t.type];
        return (
          <div
            key={t.id}
            className="toast-item"
            style={{
              pointerEvents: 'all',
              minWidth: '280px',
              maxWidth: '380px',
              background: 'var(--bg-surface)',
              border: `1px solid ${c.border}`,
              borderLeft: `4px solid ${c.icon}`,
              borderRadius: '14px',
              padding: '14px 16px',
              backdropFilter: 'blur(16px)',
              boxShadow: `0 8px 32px rgba(0,0,0,0.25), 0 0 0 1px ${c.border}`,
              display: 'flex',
              alignItems: 'flex-start',
              gap: '12px',
            }}
          >
            {/* Icon */}
            <div style={{
              width: '24px',
              height: '24px',
              borderRadius: '50%',
              background: c.bg,
              border: `1px solid ${c.border}`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '12px',
              fontWeight: 700,
              color: c.icon,
              flexShrink: 0,
              marginTop: '1px',
            }}>
              {icons[t.type]}
            </div>

            {/* Content */}
            <div style={{ flex: 1, minWidth: 0 }}>
              {t.title && (
                <p style={{ fontSize: '13px', fontWeight: 700, color: c.title, marginBottom: '2px' }}>
                  {t.title}
                </p>
              )}
              <p style={{ fontSize: '13px', color: 'var(--text-primary)', lineHeight: 1.5, wordBreak: 'break-word' }}>
                {t.message}
              </p>
            </div>

            {/* Close */}
            <button
              onClick={() => remove(t.id)}
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-muted)',
                fontSize: '16px',
                lineHeight: 1,
                padding: '0 2px',
                flexShrink: 0,
                marginTop: '-1px',
              }}
            >
              ×
            </button>
          </div>
        );
      })}
    </div>,
    document.body
  );
}
