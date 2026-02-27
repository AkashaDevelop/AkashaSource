import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon?: string;
  title: string;
  description?: string;
  action?: ReactNode;
}

export default function EmptyState({ icon = '✦', title, description, action }: EmptyStateProps) {
  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '56px 24px',
      gap: '12px',
      textAlign: 'center',
    }}>
      <div style={{
        width: '64px',
        height: '64px',
        borderRadius: '20px',
        background: 'var(--nav-active-bg)',
        border: '1px solid var(--border-color)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '28px',
        marginBottom: '4px',
        boxShadow: '0 4px 16px var(--accent-glow)',
      }}>
        {icon}
      </div>
      <p style={{
        fontSize: '15px',
        fontWeight: 600,
        color: 'var(--text-primary)',
        margin: 0,
      }}>
        {title}
      </p>
      {description && (
        <p style={{
          fontSize: '13px',
          color: 'var(--text-muted)',
          margin: 0,
          maxWidth: '280px',
          lineHeight: 1.6,
        }}>
          {description}
        </p>
      )}
      {action && <div style={{ marginTop: '8px' }}>{action}</div>}
    </div>
  );
}
