import { useState, useRef, useEffect, cloneElement, isValidElement } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import React from 'react'

/* ───── DropdownItem ───── */
export interface DropdownItemProps {
  key?: string
  children: React.ReactNode
  description?: string
  startContent?: React.ReactNode
  endContent?: React.ReactNode
  color?: 'default' | 'danger' | 'primary' | 'success' | 'warning'
  isDisabled?: boolean
  showDivider?: boolean
  onPress?: () => void
}

export function DropdownItem(_props: DropdownItemProps) {
  // 渲染由 DropdownMenu 接管，这里只作类型载体
  return null
}

/* ───── DropdownMenu ───── */
export interface DropdownMenuProps {
  children: React.ReactNode
  ariaLabel?: string
  onAction?: (key: string) => void
  /** 内部使用 */
  _close?: () => void
}

const colorMap: Record<string, string> = {
  default: 'text-[var(--text-primary)]',
  primary: 'text-[var(--accent-primary)]',
  success: 'text-[var(--color-success-fg)]',
  warning: 'text-[var(--color-warning-fg)]',
  danger:  'text-[var(--color-danger-fg)]',
}
const hoverMap: Record<string, string> = {
  default: 'hover:bg-[var(--nav-hover-bg)]',
  primary: 'hover:bg-[var(--nav-hover-bg)]',
  success: 'hover:bg-[var(--color-success-bg)]',
  warning: 'hover:bg-[var(--color-warning-bg)]',
  danger:  'hover:bg-[var(--color-danger-bg)]',
}

export function DropdownMenu({ children, ariaLabel, onAction, _close }: DropdownMenuProps) {
  const items = React.Children.toArray(children).filter(Boolean)
  return (
    <div role="menu" aria-label={ariaLabel} className="p-1.5">
      {items.map((child, i) => {
        if (!isValidElement(child)) return null
        const p = child.props as DropdownItemProps
        const key = (child.key ?? String(i)).replace(/^\.\$/, '')
        const color = p.color ?? 'default'
        return (
          <div key={key}>
            <button
              role="menuitem"
              type="button"
              disabled={p.isDisabled}
              onClick={() => {
                if (p.isDisabled) return
                p.onPress?.()
                onAction?.(key)
                _close?.()
              }}
              className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm text-left transition-colors duration-100
                ${p.isDisabled
                  ? 'opacity-40 cursor-not-allowed text-[var(--text-muted)]'
                  : `cursor-pointer ${colorMap[color]} ${hoverMap[color]}`
                }`}
            >
              {p.startContent && <span className="flex-shrink-0 flex items-center">{p.startContent}</span>}
              <span className="flex-1 min-w-0">
                <span className="block truncate">{p.children}</span>
                {p.description && (
                  <span className="block text-xs truncate" style={{ color: 'var(--text-muted)' }}>{p.description}</span>
                )}
              </span>
              {p.endContent && <span className="flex-shrink-0 flex items-center">{p.endContent}</span>}
            </button>
            {p.showDivider && <div className="my-1 mx-2" style={{ borderTop: '1px solid var(--border-color)' }} />}
          </div>
        )
      })}
    </div>
  )
}

/* ───── DropdownTrigger ───── */
export function DropdownTrigger({ children }: { children: React.ReactNode }) {
  return <>{children}</>
}

/* ───── Dropdown（容器） ───── */
export interface DropdownProps {
  children: React.ReactNode
  placement?: 'bottom-end' | 'bottom-start' | 'bottom'
}

export function Dropdown({ children, placement = 'bottom-end' }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handle(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    if (open) document.addEventListener('mousedown', handle)
    return () => document.removeEventListener('mousedown', handle)
  }, [open])

  const childArr = React.Children.toArray(children)
  const trigger = childArr.find(c => isValidElement(c) && (c.type === DropdownTrigger))
  const menu    = childArr.find(c => isValidElement(c) && (c.type === DropdownMenu))

  const triggerNode = isValidElement(trigger)
    ? (trigger.props as { children: React.ReactNode }).children
    : null

  const posClass =
    placement === 'bottom-start' ? 'left-0'
    : placement === 'bottom' ? 'left-1/2 -translate-x-1/2'
    : 'right-0'

  return (
    <div className="relative inline-block" ref={ref}>
      {isValidElement(triggerNode)
        ? cloneElement(triggerNode as React.ReactElement<any>, { onClick: () => setOpen(v => !v) })
        : <span onClick={() => setOpen(v => !v)}>{triggerNode}</span>}

      <AnimatePresence>
        {open && isValidElement(menu) && (
          <motion.div
            initial={{ opacity: 0, y: -6, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -6, scale: 0.97 }}
            transition={{ duration: 0.15 }}
            className={`absolute z-[100] mt-1.5 min-w-[180px] rounded-2xl border border-[var(--border-color)] bg-[var(--bg-surface)] backdrop-blur-xl shadow-[var(--shadow-hover)] overflow-hidden ${posClass}`}
          >
            {cloneElement(menu as React.ReactElement<DropdownMenuProps>, { _close: () => setOpen(false) })}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
