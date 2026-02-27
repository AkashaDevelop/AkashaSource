import React from 'react'

export interface LinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  color?: 'primary' | 'foreground' | 'secondary'
  isExternal?: boolean
  showAnchorIcon?: boolean
  children?: React.ReactNode
}

export function Link({ color = 'primary', isExternal, children, className = '', ...rest }: LinkProps) {
  const colorClass = {
    primary: 'text-[var(--accent-primary)] hover:text-[var(--accent-cosmic)]',
    secondary: 'text-[var(--accent-cosmic)] hover:text-[var(--accent-primary)]',
    foreground: 'text-[var(--text-primary)] hover:text-[var(--accent-primary)]',
  }[color]

  return (
    <a
      className={`underline-offset-2 hover:underline transition-colors duration-150 ${colorClass} ${className}`}
      target={isExternal ? '_blank' : undefined}
      rel={isExternal ? 'noopener noreferrer' : undefined}
      {...rest}
    >
      {children}
    </a>
  )
}
