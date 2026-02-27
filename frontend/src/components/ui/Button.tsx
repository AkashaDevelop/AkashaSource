import { motion } from 'framer-motion'
import { Loader2 } from 'lucide-react'
import React from 'react'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'solid' | 'flat' | 'ghost' | 'bordered' | 'light'
  color?: 'primary' | 'secondary' | 'danger' | 'success' | 'warning' | 'default'
  size?: 'sm' | 'md' | 'lg'
  isLoading?: boolean
  isDisabled?: boolean
  startContent?: React.ReactNode
  endContent?: React.ReactNode
  onPress?: () => void
  isIconOnly?: boolean
}

const colorMap: Record<string, Record<string, string>> = {
  solid: {
    primary: 'bg-gradient-to-r from-[var(--accent-primary)] to-[var(--accent-cosmic)] text-white hover:opacity-90',
    secondary: 'bg-[var(--accent-cosmic)] text-white hover:opacity-90',
    danger: 'bg-red-500 text-white hover:bg-red-600',
    success: 'bg-emerald-500 text-white hover:bg-emerald-600',
    warning: 'bg-amber-500 text-white hover:bg-amber-600',
    default: 'bg-[var(--bg-elevated)] text-[var(--text-primary)] border border-[var(--border-color)] hover:border-[var(--border-strong)]',
  },
  flat: {
    primary: 'bg-[var(--accent-primary)]/10 text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/20',
    secondary: 'bg-[var(--accent-cosmic)]/10 text-[var(--accent-cosmic)] hover:bg-[var(--accent-cosmic)]/20',
    danger: 'bg-red-500/10 text-red-400 hover:bg-red-500/20',
    success: 'bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20',
    warning: 'bg-amber-500/10 text-amber-400 hover:bg-amber-500/20',
    default: 'bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:bg-[var(--nav-hover-bg)]',
  },
  ghost: {
    primary: 'bg-transparent border border-[var(--accent-primary)]/40 text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/10',
    secondary: 'bg-transparent border border-[var(--accent-cosmic)]/40 text-[var(--accent-cosmic)] hover:bg-[var(--accent-cosmic)]/10',
    danger: 'bg-transparent border border-red-400/40 text-red-400 hover:bg-red-500/10',
    success: 'bg-transparent border border-emerald-400/40 text-emerald-400 hover:bg-emerald-500/10',
    warning: 'bg-transparent border border-amber-400/40 text-amber-400 hover:bg-amber-500/10',
    default: 'bg-transparent border border-[var(--border-color)] text-[var(--text-primary)] hover:bg-[var(--nav-hover-bg)]',
  },
  bordered: {
    primary: 'bg-transparent border border-[var(--accent-primary)] text-[var(--accent-primary)] hover:bg-[var(--accent-primary)]/10',
    secondary: 'bg-transparent border border-[var(--accent-cosmic)] text-[var(--accent-cosmic)] hover:bg-[var(--accent-cosmic)]/10',
    danger: 'bg-transparent border border-red-400 text-red-400 hover:bg-red-500/10',
    success: 'bg-transparent border border-emerald-400 text-emerald-400 hover:bg-emerald-500/10',
    warning: 'bg-transparent border border-amber-400 text-amber-400 hover:bg-amber-500/10',
    default: 'bg-transparent border border-[var(--border-color)] text-[var(--text-primary)] hover:border-[var(--border-strong)]',
  },
  light: {
    primary: 'bg-transparent text-[var(--accent-primary)] hover:bg-[var(--nav-hover-bg)]',
    secondary: 'bg-transparent text-[var(--accent-cosmic)] hover:bg-[var(--nav-hover-bg)]',
    danger: 'bg-transparent text-red-400 hover:bg-red-500/10',
    success: 'bg-transparent text-emerald-400 hover:bg-emerald-500/10',
    warning: 'bg-transparent text-amber-400 hover:bg-amber-500/10',
    default: 'bg-transparent text-[var(--text-secondary)] hover:bg-[var(--nav-hover-bg)]',
  },
}

const sizeMap = {
  sm: 'px-3 py-1.5 text-xs gap-1.5',
  md: 'px-4 py-2 text-sm gap-2',
  lg: 'px-5 py-2.5 text-base gap-2',
}

export function Button({
  variant = 'solid',
  color = 'primary',
  size = 'md',
  isLoading,
  isDisabled,
  startContent,
  endContent,
  onPress,
  onClick,
  children,
  className = '',
  ...rest
}: ButtonProps) {
  const colorClass = colorMap[variant]?.[color] ?? colorMap.solid.primary
  const sizeClass = sizeMap[size]
  const disabled = isDisabled || isLoading

  return (
    <motion.button
      whileTap={disabled ? {} : { scale: 0.97 }}
      className={`inline-flex items-center justify-center font-medium rounded-xl transition-all duration-200 cursor-pointer select-none disabled:opacity-50 disabled:cursor-not-allowed ${colorClass} ${sizeClass} ${className}`}
      disabled={disabled}
      onClick={(e) => {
        onPress?.()
        onClick?.(e)
      }}
      {...(rest as React.ComponentProps<typeof motion.button>)}
    >
      {isLoading ? <Loader2 className="animate-spin" size={14} /> : startContent}
      {children}
      {!isLoading && endContent}
    </motion.button>
  )
}
