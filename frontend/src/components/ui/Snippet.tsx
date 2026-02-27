import { useState } from 'react'
import { Copy, Check } from 'lucide-react'

export interface SnippetProps {
  children: string
  symbol?: string
  hideSymbol?: boolean
  className?: string
}

export function Snippet({ children, symbol = '$', hideSymbol = false, className = '' }: SnippetProps) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(children).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className={`akasha-glass rounded-xl px-4 py-2.5 flex items-center gap-3 font-mono text-sm ${className}`}>
      {!hideSymbol && <span className="text-[var(--accent-primary)] select-none flex-shrink-0">{symbol}</span>}
      <span className="flex-1 text-[var(--text-primary)] overflow-x-auto whitespace-nowrap">{children}</span>
      <button
        onClick={copy}
        className="flex-shrink-0 p-1 rounded-lg text-[var(--text-muted)] hover:text-[var(--accent-primary)] hover:bg-[var(--nav-hover-bg)] transition-all duration-150"
        title="复制"
      >
        {copied ? <Check size={14} className="text-emerald-400" /> : <Copy size={14} />}
      </button>
    </div>
  )
}
