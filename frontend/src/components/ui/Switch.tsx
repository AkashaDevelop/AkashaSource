import { useState } from 'react'
import { motion } from 'framer-motion'
import React from 'react'

export interface SwitchProps {
  isSelected?: boolean
  defaultSelected?: boolean
  onValueChange?: (v: boolean) => void
  onChange?: React.ChangeEventHandler<HTMLInputElement>
  isDisabled?: boolean
  size?: 'sm' | 'md' | 'lg'
  children?: React.ReactNode
  className?: string
}

export function Switch({
  isSelected,
  defaultSelected = false,
  onValueChange,
  onChange,
  isDisabled,
  size = 'md',
  children,
  className = '',
}: SwitchProps) {
  const [internal, setInternal] = useState(defaultSelected)
  const controlled = isSelected !== undefined
  const checked = controlled ? isSelected : internal

  const trackSize = size === 'sm' ? 'w-9 h-5' : size === 'lg' ? 'w-14 h-7' : 'w-11 h-6'
  const thumbSize = size === 'sm' ? 'w-3.5 h-3.5' : size === 'lg' ? 'w-5 h-5' : 'w-4 h-4'
  const thumbOff = size === 'sm' ? 2 : 2
  const thumbOn = size === 'sm' ? 18 : size === 'lg' ? 30 : 22

  function toggle() {
    if (isDisabled) return
    const next = !checked
    if (!controlled) setInternal(next)
    onValueChange?.(next)
  }

  return (
    <label className={`inline-flex items-center gap-2 cursor-pointer select-none ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''} ${className}`}>
      <div
        className={`relative ${trackSize} rounded-full transition-all duration-200 ${
          checked
            ? 'bg-gradient-to-r from-[var(--accent-primary)] to-[var(--accent-cosmic)]'
            : 'bg-[var(--bg-elevated)] border border-[var(--border-color)]'
        }`}
      >
        <input
          type="checkbox"
          className="sr-only"
          checked={checked}
          disabled={isDisabled}
          onChange={(e) => {
            toggle()
            onChange?.(e)
          }}
        />
        <motion.div
          className={`absolute top-1/2 -translate-y-1/2 ${thumbSize} rounded-full bg-white shadow-md`}
          animate={{ x: checked ? thumbOn : thumbOff }}
          transition={{ type: 'spring', stiffness: 500, damping: 30 }}
        />
      </div>
      {children && <span className="text-sm text-[var(--text-primary)]">{children}</span>}
    </label>
  )
}
