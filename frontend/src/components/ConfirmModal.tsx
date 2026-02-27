import { useConfirmStore } from '../store/confirm';
import { Button } from './ui';

export default function ConfirmModal() {
  const { open, options, close } = useConfirmStore();
  if (!open || !options) return null;

  const isDanger = options.danger ?? false;

  return (
    <div
      className="confirm-backdrop"
      onClick={(e) => { if (e.target === e.currentTarget) close(false); }}
      style={{
        position: 'fixed', inset: 0, zIndex: 9998,
        background: 'rgba(0,0,0,0.45)',
        backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: '16px',
      }}
    >
      <div
        className="confirm-dialog"
        style={{
          background: 'var(--bg-surface)',
          border: `1px solid ${isDanger ? 'rgba(239,68,68,0.3)' : 'var(--border-color)'}`,
          borderRadius: '20px',
          padding: '28px 28px 24px',
          width: '100%',
          maxWidth: '400px',
          backdropFilter: 'blur(20px)',
          boxShadow: isDanger
            ? '0 24px 64px rgba(239,68,68,0.2), 0 4px 24px rgba(0,0,0,0.4)'
            : '0 24px 64px rgba(0,0,0,0.35), 0 4px 24px rgba(0,0,0,0.3)',
        }}
      >
        {/* Icon */}
        <div style={{
          width: '48px', height: '48px', borderRadius: '14px',
          background: isDanger ? 'rgba(239,68,68,0.12)' : 'rgba(124,58,237,0.12)',
          border: `1px solid ${isDanger ? 'rgba(239,68,68,0.3)' : 'var(--border-color)'}`,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: '22px', marginBottom: '16px',
        }}>
          {isDanger ? '⚠️' : '💬'}
        </div>

        {/* Title */}
        {options.title && (
          <h3 style={{
            fontSize: '17px', fontWeight: 700,
            color: isDanger ? '#f87171' : 'var(--text-primary)',
            marginBottom: '8px',
          }}>
            {options.title}
          </h3>
        )}

        {/* Message */}
        <p style={{
          fontSize: '14px', color: 'var(--text-secondary)',
          lineHeight: 1.6, marginBottom: '24px',
        }}>
          {options.message}
        </p>

        {/* Actions */}
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
          <Button
            variant="flat"
            onPress={() => close(false)}
            style={{
              background: 'var(--bg-elevated)',
              color: 'var(--text-secondary)',
              borderRadius: '12px',
              border: '1px solid var(--border-color)',
            }}
          >
            {options.cancelText ?? '取消'}
          </Button>
          <Button
            onPress={() => close(true)}
            style={{
              background: isDanger
                ? 'linear-gradient(135deg, #ef4444, #dc2626)'
                : 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
              color: 'white',
              borderRadius: '12px',
              fontWeight: 600,
            }}
          >
            {options.confirmText ?? '确认'}
          </Button>
        </div>
      </div>
    </div>
  );
}
