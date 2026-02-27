import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, motion } from 'framer-motion'
import { X } from 'lucide-react'
import React from 'react'

export function useDisclosure(defaultOpen = false) {
  const [isOpen, setIsOpen] = useState(defaultOpen)
  return {
    isOpen,
    onOpen: () => setIsOpen(true),
    onClose: () => setIsOpen(false),
    onOpenChange: (open?: boolean) => setIsOpen(v => open !== undefined ? open : !v),
  }
}

export interface ModalProps {
  isOpen: boolean
  onOpenChange?: (open: boolean) => void
  onClose?: () => void
  children?: React.ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | 'full'
  isDismissable?: boolean
  scrollBehavior?: string
}

const sizeMap = {
  sm: 'max-w-sm',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
  '2xl': 'max-w-5xl',
  full: 'max-w-full mx-4',
}

export function Modal({ isOpen, onOpenChange, onClose, children, size = 'md', isDismissable = true }: ModalProps) {
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
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          className="fixed inset-0 z-[9990] flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
          onClick={isDismissable ? handleClose : undefined}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.92, y: 16 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.92, y: 16 }}
            transition={{ duration: 0.25, ease: [0.34, 1.56, 0.64, 1] }}
            className={`relative w-full ${sizeClass} bg-[var(--bg-surface)] border border-[var(--border-color)] backdrop-blur-xl rounded-2xl shadow-[0_24px_64px_rgba(0,0,0,0.4)] overflow-hidden`}
            onClick={e => e.stopPropagation()}
          >
            {typeof children === 'function'
              ? (children as (onClose: () => void) => React.ReactNode)(handleClose)
              : React.isValidElement(children)
                ? React.cloneElement(children as React.ReactElement<{ onClose?: () => void }>, { onClose: handleClose })
                : children}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>,
    document.body
  )
}

export interface ModalContentProps {
  children: React.ReactNode | ((onClose: () => void) => React.ReactNode)
  onClose?: () => void
}

export function ModalContent({ children, onClose }: ModalContentProps) {
  if (typeof children === 'function') {
    return <>{children(onClose ?? (() => {}))}</>
  }
  return <>{children}</>
}

export interface ModalHeaderProps {
  children?: React.ReactNode
  className?: string
  onClose?: () => void
}

export function ModalHeader({ children, className = '', onClose }: ModalHeaderProps) {
  return (
    <div className={`flex items-center justify-between px-6 pt-5 pb-3 border-b border-[var(--border-color)] ${className}`}>
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

export function ModalBody({ children, className = '' }: { children?: React.ReactNode; className?: string }) {
  return (
    <div className={`px-6 py-4 text-[var(--text-primary)] ${className}`}>
      {children}
    </div>
  )
}

export function ModalFooter({ children, className = '' }: { children?: React.ReactNode; className?: string }) {
  return (
    <div className={`px-6 pt-3 pb-5 border-t border-[var(--border-color)] flex items-center justify-end gap-2 ${className}`}>
      {children}
    </div>
  )
}
