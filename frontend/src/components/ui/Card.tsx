import React from 'react'

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode
  className?: string
}

export function Card({ children, className = '', ...rest }: CardProps) {
  return (
    <div className={`akasha-card ${className}`} {...rest}>
      {children}
    </div>
  )
}

export function CardHeader({ children, className = '', ...rest }: CardProps) {
  return (
    <div className={`px-6 pt-5 pb-3 border-b border-[var(--border-color)] ${className}`} {...rest}>
      {children}
    </div>
  )
}

export function CardBody({ children, className = '', ...rest }: CardProps) {
  return (
    <div className={`px-6 py-4 ${className}`} {...rest}>
      {children}
    </div>
  )
}

export function CardFooter({ children, className = '', ...rest }: CardProps) {
  return (
    <div className={`px-6 pt-3 pb-5 border-t border-[var(--border-color)] flex items-center gap-2 ${className}`} {...rest}>
      {children}
    </div>
  )
}
