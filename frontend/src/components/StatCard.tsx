import { type ReactNode } from 'react';

interface TrendInfo {
  value: number;
  label?: string;
}

interface StatCardProps {
  title: string;
  value: ReactNode;
  icon: ReactNode;
  iconBg?: string;
  footer?: ReactNode;
  trend?: TrendInfo;
}

export default function StatCard({ title, value, icon, iconBg, footer, trend }: StatCardProps) {
  return (
    <div className="akasha-card p-5 hover-lift">
      <div className="flex justify-between items-start">
        <div className="min-w-0">
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
      {(footer || trend) && (
        <div className="mt-3 flex items-center justify-between gap-2">
          <div className="text-xs flex-1" style={{ color: 'var(--text-secondary)' }}>
            {footer}
          </div>
          {trend && (
            <span className={trend.value >= 0 ? 'trend-up' : 'trend-down'}>
              {trend.value >= 0 ? '↑' : '↓'} {Math.abs(trend.value)}%
              {trend.label && <span className="trend-label">{trend.label}</span>}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
