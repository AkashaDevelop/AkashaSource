import { type ReactNode } from 'react';

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  variant?: 'default' | 'search';
}

export default function EmptyState({
  icon = '✦',
  title,
  description,
  action,
  variant = 'default',
}: EmptyStateProps) {
  const defaultDesc = variant === 'search' ? '没有找到匹配的结果，请尝试其他关键词' : undefined;
  const displayDesc = description ?? defaultDesc;

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '48px 24px',
      gap: '10px',
      textAlign: 'center',
    }}>
      <div style={{
        width: '56px',
        height: '56px',
        borderRadius: 'var(--radius-xl)',
        background: 'var(--nav-active-bg)',
        border: '1px solid var(--border-color)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: typeof icon === 'string' ? '22px' : undefined,
        marginBottom: '6px',
        boxShadow: '0 4px 16px var(--accent-glow)',
        color: 'var(--accent-primary)',
        flexShrink: 0,
      }}>
        {icon}
      </div>
      <p style={{
        fontSize: '14px',
        fontWeight: 600,
        color: 'var(--text-primary)',
        margin: 0,
      }}>
        {title}
      </p>
      {displayDesc && (
        <p style={{
          fontSize: '13px',
          color: 'var(--text-muted)',
          margin: 0,
          maxWidth: '280px',
          lineHeight: 1.6,
        }}>
          {displayDesc}
        </p>
      )}
      {action && <div style={{ marginTop: '12px' }}>{action}</div>}
    </div>
  );
}
