import { ChevronLeft, ChevronRight } from 'lucide-react'

export interface PaginationProps {
  total: number
  page: number
  onChange: (page: number) => void
  className?: string
}

export function Pagination({ total, page, onChange, className = '' }: PaginationProps) {
  if (total <= 1) return null

  function getPages(): (number | '...')[] {
    if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
    const pages: (number | '...')[] = [1]
    if (page > 3) pages.push('...')
    for (let i = Math.max(2, page - 1); i <= Math.min(total - 1, page + 1); i++) pages.push(i)
    if (page < total - 2) pages.push('...')
    pages.push(total)
    return pages
  }

  const btnBase = 'w-9 h-9 rounded-xl text-sm font-medium border transition-all duration-150 flex items-center justify-center'
  const btnNormal = `${btnBase} border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-secondary)] hover:bg-[var(--nav-hover-bg)] hover:text-[var(--accent-primary)] hover:border-[var(--border-strong)]`
  const btnActive = `${btnBase} bg-gradient-to-br from-[var(--accent-primary)] to-[var(--accent-cosmic)] border-transparent text-white font-bold`
  const btnDisabled = `${btnBase} border-[var(--border-color)] bg-[var(--bg-elevated)] text-[var(--text-faint)] opacity-40 cursor-not-allowed`

  return (
    <div className={`flex items-center gap-1 ${className}`}>
      <button
        className={page === 1 ? btnDisabled : btnNormal}
        disabled={page === 1}
        onClick={() => onChange(page - 1)}
      >
        <ChevronLeft size={14} />
      </button>

      {getPages().map((p, i) =>
        p === '...' ? (
          <span key={`ellipsis-${i}`} className="w-9 h-9 flex items-center justify-center text-[var(--text-muted)] text-sm">…</span>
        ) : (
          <button
            key={p}
            className={p === page ? btnActive : btnNormal}
            onClick={() => onChange(p as number)}
          >
            {p}
          </button>
        )
      )}

      <button
        className={page === total ? btnDisabled : btnNormal}
        disabled={page === total}
        onClick={() => onChange(page + 1)}
      >
        <ChevronRight size={14} />
      </button>
    </div>
  )
}
