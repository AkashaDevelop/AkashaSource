import { motion } from 'framer-motion'
import React from 'react'

export interface ProgressProps {
  value: number
  color?: 'primary' | 'success' | 'warning' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  label?: string
  showValueLabel?: boolean
  className?: string
}

const colorMap = {
  primary: 'from-[var(--accent-primary)] to-[var(--accent-cosmic)]',
  success: 'from-emerald-500 to-emerald-400',
  warning: 'from-amber-500 to-amber-400',
  danger: 'from-red-500 to-red-400',
}

const heightMap = { sm: 'h-1.5', md: 'h-2.5', lg: 'h-4' }

export function Progress({ value, color = 'primary', size = 'md', label, showValueLabel, className = '' }: ProgressProps) {
  const pct = Math.min(100, Math.max(0, value))
  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      {(label || showValueLabel) && (
        <div className="flex justify-between items-center">
          {label && <span className="text-xs text-[var(--text-muted)]">{label}</span>}
          {showValueLabel && <span className="text-xs font-medium text-[var(--text-secondary)]">{pct.toFixed(0)}%</span>}
        </div>
      )}
      <div className={`w-full rounded-full bg-[var(--bg-elevated)] overflow-hidden ${heightMap[size]}`}>
        <motion.div
          className={`h-full rounded-full bg-gradient-to-r ${colorMap[color]}`}
          initial={{ width: 0 }}
          animate={{ width: `${pct}%` }}
          transition={{ duration: 0.6, ease: 'easeOut' }}
        />
      </div>
    </div>
  )
}
