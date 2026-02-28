import { useState, useRef, useEffect } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import React from 'react'

export interface TooltipProps {
  content: React.ReactNode
  children: React.ReactElement
  placement?: 'top' | 'bottom' | 'left' | 'right'
  delay?: number
}

export function Tooltip({ content, children, placement = 'top', delay = 0 }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  function show() {
    timer.current = setTimeout(() => setVisible(true), delay)
  }
  function hide() {
    clearTimeout(timer.current)
    setVisible(false)
  }

  useEffect(() => () => clearTimeout(timer.current), [])

  const placementClass = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
  }[placement]

  return (
    <div className="relative inline-flex" onMouseEnter={show} onMouseLeave={hide}>
      {children}
      <AnimatePresence>
        {visible && (
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            transition={{ duration: 0.12 }}
            className={`absolute z-[9980] pointer-events-none whitespace-nowrap ${placementClass}`}
          >
            <div className="bg-[var(--bg-elevated)] border border-[var(--border-color)] text-[var(--text-primary)] text-xs rounded-xl px-3 py-1.5 shadow-[var(--shadow-hover)]">
              {content}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
