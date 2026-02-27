import React from 'react'

export interface DividerProps {
  orientation?: 'horizontal' | 'vertical'
  className?: string
}

export function Divider({ orientation = 'horizontal', className = '' }: DividerProps) {
  if (orientation === 'vertical') {
    return <div className={`w-px bg-[var(--border-color)] self-stretch mx-2 ${className}`} />
  }
  return <hr className={`border-0 h-px bg-[var(--border-color)] my-2 ${className}`} />
}
