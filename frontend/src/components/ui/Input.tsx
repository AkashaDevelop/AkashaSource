import React from 'react'

export interface InputProps {
  label?: string
  placeholder?: string
  value?: string
  defaultValue?: string
  onValueChange?: (v: string) => void
  onChange?: React.ChangeEventHandler<HTMLInputElement>
  type?: string
  isRequired?: boolean
  isDisabled?: boolean
  isReadOnly?: boolean
  errorMessage?: string
  description?: string
  startContent?: React.ReactNode
  endContent?: React.ReactNode
  maxLength?: number
  className?: string
  size?: 'sm' | 'md' | 'lg'
  name?: string
  autoComplete?: string
  autoFocus?: boolean
  min?: string | number
  max?: string | number
  step?: string | number
  onKeyDown?: React.KeyboardEventHandler<HTMLInputElement>
  onBlur?: React.FocusEventHandler<HTMLInputElement>
  isInvalid?: boolean
  style?: React.CSSProperties
}

export function Input({
  label,
  placeholder,
  value,
  defaultValue,
  onValueChange,
  onChange,
  type = 'text',
  isRequired,
  isDisabled,
  isReadOnly,
  errorMessage,
  description,
  startContent,
  endContent,
  maxLength,
  className = '',
  size = 'md',
  name,
  autoComplete,
  autoFocus,
  min,
  max,
  step,
  onKeyDown,
  onBlur,
  isInvalid,
  style,
}: InputProps) {
  const hasError = !!errorMessage || isInvalid
  const py = size === 'sm' ? 'py-1.5' : size === 'lg' ? 'py-3' : 'py-2.5'

  return (
    <div className={`flex flex-col gap-1 ${className}`} style={style}>
      {label && (
        <label className="text-xs font-medium text-[var(--text-secondary)]">
          {label}{isRequired && <span className="text-red-400 ml-0.5">*</span>}
        </label>
      )}
      <div className={`flex items-center gap-2 px-3 rounded-xl border transition-all duration-200
        bg-[var(--bg-elevated)]
        ${hasError
          ? 'border-red-400 focus-within:border-red-400 focus-within:ring-2 focus-within:ring-red-400/20'
          : 'border-[var(--border-color)] focus-within:border-[var(--accent-primary)] focus-within:ring-2 focus-within:ring-[var(--accent-glow)] hover:border-[var(--border-strong)]'
        }
        ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
      `}>
        {startContent && <span className="text-[var(--text-muted)] flex-shrink-0">{startContent}</span>}
        <input
          type={type}
          value={value}
          defaultValue={defaultValue}
          placeholder={placeholder}
          required={isRequired}
          disabled={isDisabled}
          readOnly={isReadOnly}
          maxLength={maxLength}
          name={name}
          autoComplete={autoComplete}
          autoFocus={autoFocus}
          min={min}
          max={max}
          step={step}
          onKeyDown={onKeyDown}
          onBlur={onBlur}
          className={`flex-1 bg-transparent outline-none text-[var(--text-primary)] placeholder:text-[var(--text-faint)] text-sm ${py} disabled:cursor-not-allowed`}
          onChange={(e) => {
            onValueChange?.(e.target.value)
            onChange?.(e)
          }}
        />
        {endContent && <span className="text-[var(--text-muted)] flex-shrink-0">{endContent}</span>}
      </div>
      {errorMessage && <p className="text-xs text-red-400">{errorMessage}</p>}
      {description && !errorMessage && <p className="text-xs text-[var(--text-muted)]">{description}</p>}
    </div>
  )
}
