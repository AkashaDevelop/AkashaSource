import { Copy, Check } from 'lucide-react';
import { type ModelPrice, getVendor, parseTags, parseEndpoints, formatPrice, formatContext, tierColor } from './lib';

// 🌸 模型卡片：广场卡片视图的单张卡片～

interface Props {
  m: ModelPrice;
  unit: 'M' | 'K';
  copied: string | null;
  onCopy: (name: string) => void;
  onClick: (name: string) => void;
}

export default function ModelCard({ m, unit, copied, onCopy, onClick }: Props) {
  const v = getVendor(m);
  const tags = parseTags(m.tags);
  const endpoints = parseEndpoints(m.endpoints);
  const ctx = formatContext(m.context_length);
  const isCopied = copied === m.model;
  const it = tierColor(m.input_ratio);
  const ot = tierColor(m.output_ratio);
  const unitLabel = unit === 'M' ? '百万' : '千';

  return (
    <div
      className="model-card-item cursor-pointer"
      style={{
        borderRadius: '16px', border: '1px solid var(--border-color)',
        background: 'var(--bg-surface)', overflow: 'hidden', position: 'relative',
        transition: 'box-shadow 0.2s, transform 0.2s, border-color 0.2s',
        display: 'flex', flexDirection: 'column',
      }}
      onClick={() => onClick(m.model)}
      onMouseEnter={e => { const el = e.currentTarget; el.style.transform = 'translateY(-3px)'; el.style.boxShadow = 'var(--shadow-hover)'; el.style.borderColor = `${v.color}66`; }}
      onMouseLeave={e => { const el = e.currentTarget; el.style.transform = 'translateY(0)'; el.style.boxShadow = 'none'; el.style.borderColor = 'var(--border-color)'; }}
    >
      {/* 顶部品牌色强调线 */}
      <div style={{ height: '3px', background: `linear-gradient(90deg, ${v.color}, transparent)` }} />

      <div style={{ padding: '18px 20px 14px', flex: 1 }}>
        {/* 头行 */}
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="flex min-w-0 items-center gap-3">
            <div style={{ width: '42px', height: '42px', borderRadius: '12px', flexShrink: 0, background: `${v.color}14`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '20px' }}>
              {v.icon}
            </div>
            <div className="min-w-0">
              <h3 style={{ fontFamily: 'SF Mono, Menlo, monospace', fontSize: '14px', lineHeight: 1.3, fontWeight: 700, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', margin: 0 }} title={m.model}>
                {m.model}
              </h3>
              <div className="flex items-center gap-2">
                <span style={{ fontSize: '12px', color: v.color, fontWeight: 600 }}>{v.name}</span>
                {ctx && <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>{ctx} 上下文</span>}
              </div>
            </div>
          </div>
          <button
            onClick={e => { e.stopPropagation(); onCopy(m.model); }}
            style={{ padding: '6px', borderRadius: '8px', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: isCopied ? '#10b981' : 'var(--text-muted)', cursor: 'pointer', display: 'flex', flexShrink: 0, transition: 'color 0.15s' }}
            title="复制模型名"
          >
            {isCopied ? <Check size={14} /> : <Copy size={14} />}
          </button>
        </div>

        {/* 描述 */}
        {m.description && (
          <p style={{
            fontSize: '12px', color: 'var(--text-muted)', margin: '0 0 10px', lineHeight: 1.5,
            display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden',
          }} title={m.description}>
            {m.description}
          </p>
        )}

        {/* 标签 + 端点 */}
        {(tags.length > 0 || endpoints.length > 0) && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {tags.slice(0, 4).map(t => (
              <span key={t} style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '10px', fontWeight: 500, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)' }}>{t}</span>
            ))}
            {tags.length > 4 && (
              <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '10px', fontWeight: 500, background: 'var(--bg-elevated)', color: 'var(--text-faint)' }} title={tags.slice(4).join(', ')}>+{tags.length - 4}</span>
            )}
            {endpoints.slice(0, 2).map(ep => (
              <span key={ep} style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '10px', fontWeight: 500, background: 'rgba(8,145,178,0.10)', color: '#0891b2' }}>{ep}</span>
            ))}
          </div>
        )}

        {/* 价格双列 */}
        <div className="grid grid-cols-2 gap-2">
          <div style={{ padding: '9px 12px', borderRadius: 'var(--radius-md)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <p style={{ fontSize: '10px', color: 'var(--text-muted)', margin: '0 0 3px' }}>输入 / {unitLabel} Token</p>
            <p style={{ fontSize: '15px', fontWeight: 700, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)', margin: 0 }}>
              {m.actual_input_price <= 0 ? '免费' : `¥${formatPrice(m.actual_input_price, unit)}`}
            </p>
          </div>
          <div style={{ padding: '9px 12px', borderRadius: 'var(--radius-md)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <p style={{ fontSize: '10px', color: 'var(--text-muted)', margin: '0 0 3px' }}>输出 / {unitLabel} Token</p>
            <p style={{ fontSize: '15px', fontWeight: 700, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--accent-primary)', margin: 0 }}>
              {m.actual_output_price <= 0 ? '免费' : `¥${formatPrice(m.actual_output_price, unit)}`}
            </p>
          </div>
        </div>
      </div>

      {/* 底部元信息 */}
      <div style={{ padding: '9px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div className="flex items-center gap-1.5">
          <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: it.bg, color: it.fg }}>输入 {m.input_ratio}x</span>
          <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: ot.bg, color: ot.fg }}>输出 {m.output_ratio}x</span>
        </div>
        <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>查看详情 →</span>
      </div>
    </div>
  );
}
