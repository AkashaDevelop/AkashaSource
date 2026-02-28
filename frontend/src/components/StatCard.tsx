import { type ReactNode } from 'react';

interface StatCardProps {
  title: string;
  value: ReactNode;
  icon: ReactNode;
  iconBg?: string;
  footer?: ReactNode;
}

export default function StatCard({ title, value, icon, iconBg, footer }: StatCardProps) {
  return (
    <div className="akasha-card p-5 hover-lift">
      <div className="flex justify-between items-start">
        <div>
          <p className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>{title}</p>
          <p className="text-2xl font-bold mt-1" style={{ color: 'var(--text-primary)' }}>{value}</p>
        </div>
        <div
          className="p-2.5 rounded-xl flex-shrink-0"
          style={{ background: iconBg || 'var(--bg-elevated)' }}
        >
          {icon}
        </div>
      </div>
      {footer && (
        <div className="mt-3 text-xs" style={{ color: 'var(--text-secondary)' }}>
          {footer}
        </div>
      )}
    </div>
  );
}
