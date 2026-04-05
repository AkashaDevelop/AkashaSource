import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, motion } from 'framer-motion'
import { X } from 'lucide-react'
import React from 'react'

export interface DrawerProps {
  isOpen: boolean
  onOpenChange?: (open: boolean) => void
  onClose?: () => void
  children?: React.ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl'
  isDismissable?: boolean
}

const sizeMap = {
  sm: 'w-80',
  md: 'w-96',
  lg: 'w-[32rem]',
  xl: 'w-[40rem]',
  '2xl': 'w-[48rem]',
}

export function Drawer({ isOpen, onOpenChange, onClose, children, size = 'md', isDismissable = true }: DrawerProps) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => { document.body.style.overflow = '' }
  }, [isOpen])

  function handleClose() {
    onClose?.()
    onOpenChange?.(false)
  }

  const sizeClass = sizeMap[size] ?? sizeMap.md

  return createPortal(
    <AnimatePresence>
      {isOpen && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-[9990] bg-black/50 backdrop-blur-sm"
            onClick={isDismissable ? handleClose : undefined}
          />
          <motion.div
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ duration: 0.3, ease: [0.32, 0.72, 0, 1] }}
            className={`fixed right-0 top-0 bottom-0 z-[9991] ${sizeClass} bg-[var(--bg-surface)] border-l border-[var(--border-color)] backdrop-blur-xl shadow-[0_0_40px_rgba(0,0,0,0.3)] overflow-hidden flex flex-col`}
            onClick={e => e.stopPropagation()}
          >
            {typeof children === 'function'
              ? (children as (onClose: () => void) => React.ReactNode)(handleClose)
              : React.isValidElement(children)
                ? React.cloneElement(children as React.ReactElement<{ onClose?: () => void }>, { onClose: handleClose })
                : children}
          </motion.div>
        </>
      )}
    </AnimatePresence>,
    document.body
  )
}

export interface DrawerContentProps {
  children: React.ReactNode | ((onClose: () => void) => React.ReactNode)
  onClose?: () => void
}

export function DrawerContent({ children, onClose }: DrawerContentProps) {
  if (typeof children === 'function') {
    return <>{children(onClose ?? (() => {}))}</>
  }
  return <>{children}</>
}

export interface DrawerHeaderProps {
  children?: React.ReactNode
  className?: string
  onClose?: () => void
}

export function DrawerHeader({ children, className = '', onClose }: DrawerHeaderProps) {
  return (
    <div className={`flex items-center justify-between px-6 pt-5 pb-3 border-b border-[var(--border-color)] flex-shrink-0 ${className}`}>
      <h3 className="text-base font-bold text-[var(--text-primary)]">{children}</h3>
      {onClose && (
        <button
          onClick={onClose}
          className="p-1 rounded-lg text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--nav-hover-bg)] transition-all duration-150"
        >
          <X size={16} />
        </button>
      )}
    </div>
  )
}

export function DrawerBody({ children, className = '' }: { children?: React.ReactNode; className?: string }) {
  return (
    <div className={`flex-1 overflow-y-auto px-6 py-4 text-[var(--text-primary)] ${className}`}>
      {children}
    </div>
  )
}

export function DrawerFooter({ children, className = '' }: { children?: React.ReactNode; className?: string }) {
  return (
    <div className={`px-6 pt-3 pb-5 border-t border-[var(--border-color)] flex items-center justify-end gap-2 flex-shrink-0 ${className}`}>
      {children}
    </div>
  )
}
