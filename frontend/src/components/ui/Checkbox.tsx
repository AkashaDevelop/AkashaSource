import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { Check } from 'lucide-react'
import React from 'react'

export interface CheckboxProps {
  isSelected?: boolean
  defaultSelected?: boolean
  onValueChange?: (v: boolean) => void
  isDisabled?: boolean
  children?: React.ReactNode
  className?: string
}

export function Checkbox({
  isSelected,
  defaultSelected = false,
  onValueChange,
  isDisabled,
  children,
  className = '',
}: CheckboxProps) {
  const [internal, setInternal] = useState(defaultSelected)
  const controlled = isSelected !== undefined
  const checked = controlled ? isSelected : internal

  function toggle() {
    if (isDisabled) return
    const next = !checked
    if (!controlled) setInternal(next)
    onValueChange?.(next)
  }

  return (
    <label className={`inline-flex items-center gap-2 cursor-pointer select-none ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''} ${className}`}>
      <div
        onClick={toggle}
        className={`w-4 h-4 rounded flex items-center justify-center transition-all duration-150 flex-shrink-0
          ${checked
            ? 'bg-gradient-to-br from-[var(--accent-primary)] to-[var(--accent-cosmic)] border-transparent'
            : 'bg-[var(--bg-elevated)] border border-[var(--border-color)] hover:border-[var(--border-strong)]'
          }
        `}
      >
        <AnimatePresence>
          {checked && (
            <motion.div
              initial={{ scale: 0, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0, opacity: 0 }}
              transition={{ type: 'spring', stiffness: 600, damping: 25 }}
            >
              <Check size={10} className="text-white" strokeWidth={3} />
            </motion.div>
          )}
        </AnimatePresence>
      </div>
      {children && <span className="text-sm text-[var(--text-primary)]">{children}</span>}
    </label>
  )
}
