import React from 'react'

export interface TextareaProps {
  label?: string
  placeholder?: string
  value?: string
  defaultValue?: string
  onValueChange?: (v: string) => void
  onChange?: React.ChangeEventHandler<HTMLTextAreaElement>
  isRequired?: boolean
  isDisabled?: boolean
  isReadOnly?: boolean
  errorMessage?: string
  description?: string
  minRows?: number
  maxRows?: number
  maxLength?: number
  className?: string
  name?: string
}

export function Textarea({
  label,
  placeholder,
  value,
  defaultValue,
  onValueChange,
  onChange,
  isRequired,
  isDisabled,
  isReadOnly,
  errorMessage,
  description,
  minRows = 3,
  maxLength,
  className = '',
  name,
}: TextareaProps) {
  const hasError = !!errorMessage

  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      {label && (
        <label className="text-xs font-medium text-[var(--text-secondary)]">
          {label}{isRequired && <span className="text-red-400 ml-0.5">*</span>}
        </label>
      )}
      <div className={`px-3 py-2 rounded-xl border transition-all duration-200
        bg-[var(--bg-elevated)]
        ${hasError
          ? 'border-red-400 focus-within:border-red-400 focus-within:ring-2 focus-within:ring-red-400/20'
          : 'border-[var(--border-color)] focus-within:border-[var(--accent-primary)] focus-within:ring-2 focus-within:ring-[var(--accent-glow)] hover:border-[var(--border-strong)]'
        }
        ${isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
      `}>
        <textarea
          value={value}
          defaultValue={defaultValue}
          placeholder={placeholder}
          required={isRequired}
          disabled={isDisabled}
          readOnly={isReadOnly}
          maxLength={maxLength}
          rows={minRows}
          name={name}
          className="w-full bg-transparent outline-none text-[var(--text-primary)] placeholder:text-[var(--text-faint)] text-sm resize-none disabled:cursor-not-allowed"
          onChange={(e) => {
            onValueChange?.(e.target.value)
            onChange?.(e)
          }}
        />
      </div>
      {errorMessage && <p className="text-xs text-red-400">{errorMessage}</p>}
      {description && !errorMessage && <p className="text-xs text-[var(--text-muted)]">{description}</p>}
    </div>
  )
}
