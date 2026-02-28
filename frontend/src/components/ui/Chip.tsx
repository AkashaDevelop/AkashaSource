import React from 'react'
import { X } from 'lucide-react'

export interface ChipProps {
  color?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger'
  variant?: 'solid' | 'flat' | 'bordered' | 'dot'
  size?: 'sm' | 'md' | 'lg'
  children?: React.ReactNode
  className?: string
  style?: React.CSSProperties
  onClick?: React.MouseEventHandler<HTMLSpanElement>
  onClose?: () => void
  startContent?: React.ReactNode
  endContent?: React.ReactNode
}

const colorMap: Record<string, string> = {
  primary: 'bg-[var(--accent-primary)]/10 text-[var(--accent-primary)] border border-[var(--accent-primary)]/20',
  secondary: 'bg-[var(--accent-cosmic)]/10 text-[var(--accent-cosmic)] border border-[var(--accent-cosmic)]/20',
  success: 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20',
  warning: 'bg-amber-500/10 text-amber-400 border border-amber-500/20',
  danger: 'bg-red-500/10 text-red-400 border border-red-500/20',
  default: 'bg-[var(--bg-elevated)] text-[var(--text-muted)] border border-[var(--border-color)]',
}

const solidMap: Record<string, string> = {
  primary: 'bg-[var(--accent-primary)] text-white',
  secondary: 'bg-[var(--accent-cosmic)] text-white',
  success: 'bg-emerald-500 text-white',
  warning: 'bg-amber-500 text-white',
  danger: 'bg-red-500 text-white',
  default: 'bg-[var(--bg-elevated)] text-[var(--text-primary)]',
}

const sizeMap = {
  sm: 'px-1.5 py-0.5 text-[11px]',
  md: 'px-2 py-0.5 text-xs',
  lg: 'px-2.5 py-1 text-sm',
}

export function Chip({ color = 'default', variant = 'flat', size = 'md', children, className = '', style, onClick, onClose, startContent, endContent }: ChipProps) {
  const colorClass = variant === 'solid' ? solidMap[color] : colorMap[color]
  return (
    <span className={`inline-flex items-center gap-1 rounded-lg font-medium ${colorClass} ${sizeMap[size]} ${className}`} style={style} onClick={onClick}>
      {startContent}
      {children}
      {endContent}
      {onClose && (
        <button type="button" onClick={e => { e.stopPropagation(); onClose(); }} className="ml-0.5 hover:opacity-70">
          <X size={10} />
        </button>
      )}
    </span>
  )
}
