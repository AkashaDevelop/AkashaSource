import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import React from 'react'

export interface TabProps {
  title: React.ReactNode
  children?: React.ReactNode
  key?: string
}

export function Tab(_props: TabProps) {
  return null
}

export interface TabsProps {
  selectedKey?: string
  defaultSelectedKey?: string
  onSelectionChange?: (key: string) => void
  children: React.ReactNode
  className?: string
}

export function Tabs({ selectedKey, defaultSelectedKey, onSelectionChange, children, className = '' }: TabsProps) {
  const tabs: { key: string; title: React.ReactNode; children: React.ReactNode }[] = []

  React.Children.forEach(children, (child) => {
    if (React.isValidElement(child)) {
      const props = child.props as TabProps
      const k = (child.key as string) ?? ''
      tabs.push({ key: k, title: props.title, children: props.children })
    }
  })

  const [internal, setInternal] = useState(defaultSelectedKey ?? tabs[0]?.key ?? '')
  const controlled = selectedKey !== undefined
  const active = controlled ? selectedKey : internal

  function select(key: string) {
    if (!controlled) setInternal(key)
    onSelectionChange?.(key)
  }

  const activeTab = tabs.find(t => t.key === active)

  return (
    <div className={className}>
      <div className="flex gap-1 p-1 rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-color)] w-fit mb-4">
        {tabs.map(tab => (
          <button
            key={tab.key}
            type="button"
            onClick={() => select(tab.key)}
            className={`px-4 py-1.5 rounded-lg text-sm font-medium transition-all duration-150
              ${tab.key === active
                ? 'bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                : 'text-[var(--text-secondary)] hover:bg-[var(--nav-hover-bg)] hover:text-[var(--text-primary)]'
              }
            `}
          >
            {tab.title}
          </button>
        ))}
      </div>
      <AnimatePresence mode="wait">
        <motion.div
          key={active}
          initial={{ opacity: 0, x: 8 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: -8 }}
          transition={{ duration: 0.15 }}
        >
          {activeTab?.children}
        </motion.div>
      </AnimatePresence>
    </div>
  )
}
