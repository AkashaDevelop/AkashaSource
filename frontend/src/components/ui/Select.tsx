import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Check } from 'lucide-react'
import { AnimatePresence, motion } from 'framer-motion'
import React from 'react'

export interface SelectItemProps {
  key?: string
  value?: string
  children: React.ReactNode
  textValue?: string
}

export function SelectItem({ children }: SelectItemProps) {
  return <>{children}</>
}

export interface SelectProps {
  label?: string
  placeholder?: string
  selectedKeys?: Set<string> | string[]
  defaultSelectedKeys?: Set<string> | string[]
  onSelectionChange?: (keys: Set<string>) => void
  children: React.ReactNode
  isDisabled?: boolean
  isRequired?: boolean
  errorMessage?: string
  className?: string
  size?: 'sm' | 'md' | 'lg'
  selectionMode?: 'single' | 'multiple' // 🎀 多选模式支持～
}

function getKeys(keys?: Set<string> | string[]): string[] {
  if (!keys) return []
  if (keys instanceof Set) return [...keys]
  return keys
}

export function Select({
  label,
  placeholder = '请选择',
  selectedKeys,
  defaultSelectedKeys,
  onSelectionChange,
  children,
  isDisabled,
  isRequired,
  errorMessage,
  className = '',
  size = 'md',
  selectionMode = 'single',
}: SelectProps) {
  const [open, setOpen] = useState(false)
  const [internalSelected, setInternalSelected] = useState<string[]>(
    getKeys(defaultSelectedKeys)
  )
  const ref = useRef<HTMLDivElement>(null)

  const controlled = selectedKeys !== undefined
  const selected = controlled ? getKeys(selectedKeys) : internalSelected

  // Build options from children
  const options: { key: string; label: string }[] = []
  React.Children.forEach(children, (child) => {
    if (React.isValidElement(child)) {
      const k = child.key ?? (child.props as SelectItemProps).value ?? ''
      const label = (child.props as SelectItemProps).textValue ?? String((child.props as SelectItemProps).children ?? k)
      options.push({ key: k, label })
    }
  })

  const selectedLabel = selected.length > 0
    ? selectionMode === 'multiple'
      ? selected.length === 1
        ? options.find(o => o.key === selected[0])?.label ?? selected[0]
        : `已选 ${selected.length} 项`
      : options.find(o => o.key === selected[0])?.label ?? selected[0]
    : null

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  function select(key: string) {
    if (selectionMode === 'multiple') {
      // 🎀 多选：点一下切换选中状态，面板保持打开～
      const next = new Set(selected)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      if (!controlled) setInternalSelected([...next])
      onSelectionChange?.(next)
      return
    }
    const next = new Set([key])
    if (!controlled) setInternalSelected([key])
    onSelectionChange?.(next)
    setOpen(false)
  }

  const py = size === 'sm' ? 'py-1.5' : size === 'lg' ? 'py-3' : 'py-2.5'
  const hasError = !!errorMessage

  return (
    <div className={`flex flex-col gap-1 ${className}`} ref={ref}>
      {label && <label className="text-xs font-medium text-[var(--text-secondary)]">{label}{isRequired && <span className="text-red-400 ml-0.5">*</span>}</label>}
      <div className="relative">
        <button
          type="button"
          disabled={isDisabled}
          onClick={() => !isDisabled && setOpen(v => !v)}
          className={`w-full flex items-center justify-between gap-2 px-3 ${py} rounded-xl border text-sm transition-all duration-200
            bg-[var(--bg-elevated)] text-[var(--text-primary)]
            ${hasError
              ? 'border-red-400'
              : open
                ? 'border-[var(--accent-primary)] ring-2 ring-[var(--accent-glow)]'
                : 'border-[var(--border-color)] hover:border-[var(--border-strong)]'
            }
            ${isDisabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
          `}
        >
          <span className={selectedLabel ? 'text-[var(--text-primary)]' : 'text-[var(--text-faint)]'}>
            {selectedLabel ?? placeholder}
          </span>
          <ChevronDown
            size={14}
            className={`text-[var(--text-muted)] transition-transform duration-200 flex-shrink-0 ${open ? 'rotate-180' : ''}`}
          />
        </button>

        <AnimatePresence>
          {open && (
            <motion.div
              initial={{ opacity: 0, y: -6, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -6, scale: 0.97 }}
              transition={{ duration: 0.15 }}
              className="absolute z-50 w-full mt-1 rounded-2xl border border-[var(--border-color)] bg-[var(--bg-surface)] backdrop-blur-xl shadow-[var(--shadow-hover)] overflow-hidden"
            >
              <div className="p-1 max-h-56 overflow-y-auto">
                {options.map(opt => (
                  <button
                    key={opt.key}
                    type="button"
                    onClick={() => select(opt.key)}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm transition-all duration-150 cursor-pointer
                      ${selected.includes(opt.key)
                        ? 'bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                        : 'text-[var(--text-primary)] hover:bg-[var(--nav-hover-bg)] hover:text-[var(--accent-primary)]'
                      }
                    `}
                  >
                    {opt.label}
                    {selected.includes(opt.key) && <Check size={12} />}
                  </button>
                ))}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
      {errorMessage && <p className="text-xs text-red-400">{errorMessage}</p>}
    </div>
  )
}
