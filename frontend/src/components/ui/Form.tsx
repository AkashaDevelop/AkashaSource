import React from 'react'

export interface FormProps extends React.FormHTMLAttributes<HTMLFormElement> {
  children?: React.ReactNode
  validationBehavior?: 'native' | 'aria'
}

export function Form({ children, validationBehavior: _vb, ...rest }: FormProps) {
  return <form {...rest}>{children}</form>
}
