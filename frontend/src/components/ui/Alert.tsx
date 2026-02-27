import { CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-react'
import React from 'react'

export interface AlertProps {
  color?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  children?: React.ReactNode
  className?: string
  title?: string
  description?: string
}

const colorMap = {
  success: {
    wrap: 'bg-emerald-500/10 border border-emerald-500/30 text-emerald-400',
    icon: <CheckCircle2 size={16} className="flex-shrink-0 mt-0.5" />,
  },
  danger: {
    wrap: 'bg-red-500/10 border border-red-500/30 text-red-400',
    icon: <XCircle size={16} className="flex-shrink-0 mt-0.5" />,
  },
  warning: {
    wrap: 'bg-amber-500/10 border border-amber-500/30 text-amber-400',
    icon: <AlertTriangle size={16} className="flex-shrink-0 mt-0.5" />,
  },
  primary: {
    wrap: 'bg-[var(--accent-primary)]/10 border border-[var(--accent-primary)]/25 text-[var(--accent-primary)]',
    icon: <Info size={16} className="flex-shrink-0 mt-0.5" />,
  },
  default: {
    wrap: 'bg-[var(--bg-elevated)] border border-[var(--border-color)] text-[var(--text-primary)]',
    icon: <Info size={16} className="flex-shrink-0 mt-0.5" />,
  },
}

export function Alert({ color = 'default', children, className = '', title, description }: AlertProps) {
  const { wrap, icon } = colorMap[color] ?? colorMap.default
  return (
    <div className={`rounded-xl px-4 py-3 text-sm backdrop-blur-sm flex items-start gap-3 ${wrap} ${className}`}>
      {icon}
      <div className="flex-1 min-w-0">
        {title && <p className="font-semibold mb-0.5">{title}</p>}
        {description && <p className="opacity-90">{description}</p>}
        {children}
      </div>
    </div>
  )
}
