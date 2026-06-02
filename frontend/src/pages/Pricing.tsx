import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import {
  Search, X, Copy, Check, SlidersHorizontal,
  Grid3X3, List, EyeOff, ChevronDown, ChevronUp, RotateCcw,
} from 'lucide-react';

/* ═══ 类型 ═══ */
interface ModelPrice {
  model: string;
  input_ratio: number;
  output_ratio: number;
  upstream_input_price: number;
  upstream_output_price: number;
  actual_input_price: number;
  actual_output_price: number;
}

/* ═══ 供应商 ═══ */
const VENDORS: { pattern: RegExp; name: string; icon: string; color: string }[] = [
  { pattern: /^(gpt|o1|o3|dall-e|whisper|tts|text-)/i, name: 'OpenAI',     icon: '⚡', color: '#10a37f' },
  { pattern: /^claude/i,                                  name: 'Anthropic',  icon: '🟣', color: '#d97706' },
  { pattern: /^gemini/i,                                  name: 'Google',     icon: '🔵', color: '#4285f4' },
  { pattern: /^deepseek/i,                                name: 'DeepSeek',   icon: '🌊', color: '#4f46e5' },
  { pattern: /^(glm|chatglm)/i,                           name: '智谱 AI',    icon: '🔮', color: '#7c3aed' },
  { pattern: /^qwen/i,                                    name: '阿里云',     icon: '☁️', color: '#ff6a00' },
  { pattern: /^hunyuan/i,                                 name: '腾讯混元',   icon: '💙', color: '#0052d9' },
  { pattern: /^ernie/i,                                   name: '百度文心',   icon: '🔍', color: '#2932e1' },
  { pattern: /^spark/i,                                   name: '讯飞星火',   icon: '🔥', color: '#ca0d19' },
  { pattern: /^moonshot/i,                                name: 'Moonshot',   icon: '🌙', color: '#6d28d9' },
  { pattern: /^(llama|ollama)/i,                          name: 'Ollama',     icon: '🦙', color: '#8b5cf6' },
  { pattern: /^mistral/i,                                 name: 'Mistral',    icon: '🌬️', color: '#ffb300' },
  { pattern: /^silicon/i,                                 name: 'SiliconFlow',icon: '💎', color: '#6366f1' },
  { pattern: /^azure/i,                                   name: 'Azure',      icon: '☁️', color: '#0078d4' },
];

function vName(m: string) {
  return VENDORS.find(v => v.pattern.test(m)) ?? { name: '其他', icon: '🧠', color: 'var(--accent-primary)' };
}

function fp(p: number) {
  if (p <= 0) return '—';
  if (p < 0.01) return p.toFixed(4);
  return p.toFixed(2);
}

/* 倍率分级颜色 */
function tierColor(r: number) {
  if (r <= 1)  return { fg: '#10b981', bg: 'rgba(16,185,129,0.12)' };
  if (r <= 5)  return { fg: '#0891b2', bg: 'rgba(8,145,178,0.12)' };
  if (r <= 15) return { fg: '#d97706', bg: 'rgba(217,119,6,0.12)' };
  return            { fg: '#dc2626', bg: 'rgba(220,38,38,0.12)' };
}

/* ═══ 骨架 ═══ */
function SkeletonCard() {
  return (
    <div style={{ borderRadius: '16px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)', overflow: 'hidden' }}>
      <div style={{ padding: '20px' }}>
        <div className="flex items-start gap-3 mb-4">
          <div className="skeleton-shimmer" style={{ width: '42px', height: '42px', borderRadius: '12px', flexShrink: 0 }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="skeleton-shimmer" style={{ height: '16px', width: '60%', borderRadius: '4px', marginBottom: '8px' }} />
            <div className="skeleton-shimmer" style={{ height: '12px', width: '35%', borderRadius: '4px' }} />
          </div>
        </div>
        <div className="skeleton-shimmer" style={{ height: '12px', width: '90%', borderRadius: '4px', marginBottom: '6px' }} />
        <div className="skeleton-shimmer" style={{ height: '12px', width: '70%', borderRadius: '4px' }} />
      </div>
      <div style={{ padding: '12px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-elevated)' }}>
        <div className="skeleton-shimmer" style={{ height: '14px', width: '50%', borderRadius: '4px' }} />
      </div>
    </div>
  );
}

/* ═══ 侧边栏分组 ═══ */
function FilterGroup({ title, children, defaultOpen = true }: { title: string; children: React.ReactNode; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div>
      <button
        onClick={() => setOpen(v => !v)}
        className="w-full flex items-center justify-between"
        style={{ padding: '14px 16px 8px', background: 'none', border: 'none', cursor: 'pointer' }}
      >
        <span style={{ fontSize: '11px', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.07em' }}>{title}</span>
        {open ? <ChevronUp size={14} style={{ color: 'var(--text-faint)' }} /> : <ChevronDown size={14} style={{ color: 'var(--text-faint)' }} />}
      </button>
      {open && <div style={{ padding: '0 10px 14px' }}>{children}</div>}
    </div>
  );
}

function FilterItem({ label, count, active, onClick, icon, dot }: { label: string; count?: number; active: boolean; onClick: () => void; icon?: string; dot?: string }) {
  return (
    <button
      onClick={onClick}
      className="w-full flex items-center justify-between rounded-lg"
      style={{
        padding: '7px 9px', fontSize: '13px', marginBottom: '1px',
        background: active ? 'var(--nav-active-bg)' : 'transparent',
        color: active ? 'var(--accent-primary)' : 'var(--text-secondary)',
        fontWeight: active ? 600 : 400,
        border: 'none', cursor: 'pointer', transition: 'background 0.1s',
      }}
      onMouseEnter={e => { if (!active) (e.currentTarget as HTMLElement).style.background = 'var(--nav-hover-bg)'; }}
      onMouseLeave={e => { if (!active) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
    >
      <div className="flex items-center gap-2 min-w-0">
        {icon && <span style={{ fontSize: '14px', flexShrink: 0 }}>{icon}</span>}
        {dot && <span style={{ width: '7px', height: '7px', borderRadius: '50%', background: dot, flexShrink: 0 }} />}
        <span className="truncate">{label}</span>
      </div>
      {count !== undefined && (
        <span style={{ fontSize: '11px', color: active ? 'var(--accent-primary)' : 'var(--text-faint)', fontWeight: 500, fontVariantNumeric: 'tabular-nums' }}>{count}</span>
      )}
    </button>
  );
}

/* ═══ 主页面 ═══ */
export default function Pricing() {
  const [models, setModels] = useState<ModelPrice[]>([]);
  const [loading, setLoading] = useState(false);

  const [search,       setSearch]       = useState('');
  const [vendorF,      setVendorF]      = useState<string | null>(null);
  const [tierF,        setTierF]        = useState<string | null>(null);
  const [sort,         setSort]         = useState<'name' | 'input' | 'output'>('name');
  const [view,         setView]         = useState<'card' | 'table'>('card');
  const [mobileFilter, setMobileFilter] = useState(false);
  const [copied,       setCopied]       = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const handleCopy = useCallback(async (n: string) => {
    await navigator.clipboard.writeText(n);
    setCopied(n);
    setTimeout(() => setCopied(null), 1200);
  }, []);

  useEffect(() => {
    setLoading(true);
    fetch('/api/pricing')
      .then(r => r.json())
      .then(d => {
        const list = d.code === 0 ? d.data?.models : d.models;
        if (Array.isArray(list)) setModels(list);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  /* ⌘K 聚焦搜索 */
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  /* 供应商列表 */
  const vlist = useMemo(() => {
    const m: Record<string, { icon: string; color: string; count: number }> = {};
    models.forEach(x => {
      const v = vName(x.model);
      if (!m[v.name]) m[v.name] = { icon: v.icon, color: v.color, count: 0 };
      m[v.name].count++;
    });
    return Object.entries(m).sort(([, a], [, b]) => b.count - a.count);
  }, [models]);

  /* 过滤 */
  const filtered = useMemo(() => {
    let list = models;
    const q = search.toLowerCase().trim();
    if (q) list = list.filter(x => x.model.toLowerCase().includes(q));
    if (vendorF) list = list.filter(x => vName(x.model).name === vendorF);
    if (tierF) {
      if (tierF === 'low')  list = list.filter(x => x.input_ratio <= 5 && x.output_ratio <= 5);
      if (tierF === 'mid')  list = list.filter(x => Math.max(x.input_ratio, x.output_ratio) > 5 && Math.max(x.input_ratio, x.output_ratio) <= 15);
      if (tierF === 'high') list = list.filter(x => Math.max(x.input_ratio, x.output_ratio) > 15);
    }
    return [...list].sort((a, b) => {
      if (sort === 'input')  return b.actual_input_price - a.actual_input_price;
      if (sort === 'output') return b.actual_output_price - a.actual_output_price;
      return a.model.localeCompare(b.model);
    });
  }, [models, search, vendorF, tierF, sort]);

  const hasF = !!(vendorF || tierF);
  const fCnt = (vendorF ? 1 : 0) + (tierF ? 1 : 0);
  const clearAll = () => { setSearch(''); setVendorF(null); setTierF(null); };

  /* ═══ 侧边栏 ═══ */
  const TIER_OPTS = [
    { key: 'low',  label: '经济 ≤5x',   dot: '#10b981', test: (x: ModelPrice) => x.input_ratio <= 5 && x.output_ratio <= 5 },
    { key: 'mid',  label: '标准 5-15x', dot: '#d97706', test: (x: ModelPrice) => Math.max(x.input_ratio, x.output_ratio) > 5 && Math.max(x.input_ratio, x.output_ratio) <= 15 },
    { key: 'high', label: '高级 >15x',  dot: '#dc2626', test: (x: ModelPrice) => Math.max(x.input_ratio, x.output_ratio) > 15 },
  ];

  const sidebar = (
    <div>
      <div className="flex items-center justify-between" style={{ padding: '16px 16px 4px' }}>
        <span style={{ fontSize: '14px', fontWeight: 700, color: 'var(--text-primary)' }}>筛选</span>
        {hasF && (
          <button onClick={clearAll} className="flex items-center gap-1" style={{ fontSize: '11px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer' }}>
            <RotateCcw size={11} /> 重置
          </button>
        )}
      </div>

      <FilterGroup title="价格档位">
        <FilterItem label="全部档位" count={models.length} active={!tierF} onClick={() => setTierF(null)} />
        {TIER_OPTS.map(t => (
          <FilterItem key={t.key} label={t.label} dot={t.dot} count={models.filter(t.test).length} active={tierF === t.key} onClick={() => setTierF(tierF === t.key ? null : t.key)} />
        ))}
      </FilterGroup>

      <div style={{ borderTop: '1px solid var(--border-color)', margin: '0 16px' }} />

      <FilterGroup title="供应商">
        <FilterItem label="全部供应商" count={models.length} active={!vendorF} onClick={() => setVendorF(null)} />
        {vlist.map(([name, info]) => (
          <FilterItem key={name} label={name} count={info.count} active={vendorF === name} onClick={() => setVendorF(vendorF === name ? null : name)} icon={info.icon} />
        ))}
      </FilterGroup>
    </div>
  );

  /* ═══ 渲染 ═══ */
  return (
    <div className="animate-fade-in-up">

      {/* ══════ Hero 头部 ══════ */}
      <div style={{ textAlign: 'center', padding: '8px 0 28px', position: 'relative' }}>
        <h1 style={{ fontSize: '34px', fontWeight: 800, letterSpacing: '-0.02em', margin: 0 }} className="gradient-text">
          模型广场
        </h1>
        <p style={{ fontSize: '14px', color: 'var(--text-muted)', marginTop: '8px' }}>
          {loading ? '正在加载模型数据...' : `本站当前已启用模型，总计 ${models.length} 个 · 探索并比较各模型的定价`}
        </p>

        {/* 居中大搜索框 */}
        <div style={{ maxWidth: '560px', margin: '20px auto 0', position: 'relative' }}>
          <Search size={17} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-faint)', pointerEvents: 'none' }} />
          <input
            ref={searchRef}
            placeholder="搜索模型名称、供应商..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{
              width: '100%', height: '48px', paddingLeft: '46px', paddingRight: '60px',
              borderRadius: 'var(--radius-full)', fontSize: '14px',
              background: 'var(--bg-surface)', color: 'var(--text-primary)',
              border: '1px solid var(--border-color)',
              boxShadow: 'var(--shadow-card)', outline: 'none',
              transition: 'border-color 0.15s, box-shadow 0.15s',
            }}
            onFocus={e => { e.currentTarget.style.borderColor = 'var(--accent-primary)'; e.currentTarget.style.boxShadow = '0 0 0 3px var(--accent-glow)'; }}
            onBlur={e => { e.currentTarget.style.borderColor = 'var(--border-color)'; e.currentTarget.style.boxShadow = 'var(--shadow-card)'; }}
          />
          {search ? (
            <button onClick={() => setSearch('')} style={{ position: 'absolute', right: '16px', top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-faint)', padding: '4px' }}>
              <X size={16} />
            </button>
          ) : (
            <kbd style={{ position: 'absolute', right: '14px', top: '50%', transform: 'translateY(-50%)', fontSize: '11px', fontFamily: 'monospace', color: 'var(--text-faint)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', borderRadius: '6px', padding: '2px 7px' }}>
              ⌘K
            </kbd>
          )}
        </div>
      </div>

      {/* 移动端筛选触发 */}
      <div className="md:hidden flex items-center justify-between mb-4">
        <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>{filtered.length} 个模型</span>
        <button
          onClick={() => setMobileFilter(true)}
          className="flex items-center gap-1.5"
          style={{
            padding: '7px 12px', borderRadius: '10px', fontSize: '12px', fontWeight: 500,
            border: `1px solid ${hasF ? 'var(--accent-primary)' : 'var(--border-color)'}`,
            background: hasF ? 'var(--nav-active-bg)' : 'var(--bg-elevated)',
            color: hasF ? 'var(--accent-primary)' : 'var(--text-secondary)', cursor: 'pointer',
          }}
        >
          <SlidersHorizontal size={14} /> 筛选
          {fCnt > 0 && <span style={{ background: 'var(--accent-primary)', color: 'white', borderRadius: '50%', width: '16px', height: '16px', fontSize: '10px', fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{fCnt}</span>}
        </button>
      </div>

      {/* 移动端抽屉 */}
      {mobileFilter && (
        <>
          <div className="sidebar-mobile-overlay md:hidden" onClick={() => setMobileFilter(false)} />
          <div className="sidebar-mobile-drawer md:hidden open" style={{ width: '280px' }}>
            <div style={{ height: '100%', overflowY: 'auto' }}>{sidebar}</div>
          </div>
        </>
      )}

      {/* ══════ 双栏主体 ══════ */}
      <div className="flex gap-6">
        {/* 侧边栏 */}
        <aside className="hidden md:block" style={{ width: '232px', flexShrink: 0 }}>
          <div style={{ borderRadius: 'var(--radius-xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', overflow: 'hidden', position: 'sticky', top: '68px', paddingBottom: '8px' }}>
            {sidebar}
          </div>
        </aside>

        {/* 内容区 */}
        <main style={{ flex: 1, minWidth: 0 }}>
          {/* 工具栏 */}
          <div className="flex items-center justify-between gap-3 mb-4 flex-wrap">
            <div className="flex items-center gap-2 flex-wrap">
              <span style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 500 }}>
                {loading ? '加载中' : `${filtered.length} 个模型`}
              </span>
              {/* 已选标签 */}
              {tierF && (
                <span style={{ padding: '3px 9px', borderRadius: 'var(--radius-full)', fontSize: '11px', fontWeight: 600, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)', display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                  {({ low: '经济档', mid: '标准档', high: '高级档' } as any)[tierF]}
                  <button onClick={() => setTierF(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'inherit', padding: 0, lineHeight: 1, fontSize: '13px', fontWeight: 700 }}>×</button>
                </span>
              )}
              {vendorF && (
                <span style={{ padding: '3px 9px', borderRadius: 'var(--radius-full)', fontSize: '11px', fontWeight: 600, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)', display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                  {vName(vendorF).icon} {vendorF}
                  <button onClick={() => setVendorF(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'inherit', padding: 0, lineHeight: 1, fontSize: '13px', fontWeight: 700 }}>×</button>
                </span>
              )}
            </div>

            <div className="flex items-center gap-2">
              {/* 排序分段控件 */}
              <div className="flex items-center" style={{ background: 'var(--bg-elevated)', borderRadius: '9px', border: '1px solid var(--border-color)', padding: '2px' }}>
                {([['name', '名称'], ['input', '输入价'], ['output', '输出价']] as const).map(([k, l]) => (
                  <button key={k} onClick={() => setSort(k)} style={{
                    padding: '4px 10px', borderRadius: '7px', border: 'none', fontSize: '12px', fontWeight: 500, cursor: 'pointer',
                    background: sort === k ? 'var(--bg-surface)' : 'transparent',
                    color: sort === k ? 'var(--accent-primary)' : 'var(--text-muted)',
                    boxShadow: sort === k ? 'var(--shadow-card)' : 'none',
                    transition: 'all 0.12s',
                  }}>{l}</button>
                ))}
              </div>

              {/* 视图切换 */}
              <div className="hidden sm:flex items-center" style={{ background: 'var(--bg-elevated)', borderRadius: '9px', border: '1px solid var(--border-color)', padding: '2px' }}>
                <button onClick={() => setView('card')} style={{ padding: '5px 8px', borderRadius: '7px', border: 'none', background: view === 'card' ? 'var(--bg-surface)' : 'transparent', color: view === 'card' ? 'var(--accent-primary)' : 'var(--text-muted)', cursor: 'pointer', boxShadow: view === 'card' ? 'var(--shadow-card)' : 'none', display: 'flex' }}><Grid3X3 size={14} /></button>
                <button onClick={() => setView('table')} style={{ padding: '5px 8px', borderRadius: '7px', border: 'none', background: view === 'table' ? 'var(--bg-surface)' : 'transparent', color: view === 'table' ? 'var(--accent-primary)' : 'var(--text-muted)', cursor: 'pointer', boxShadow: view === 'table' ? 'var(--shadow-card)' : 'none', display: 'flex' }}><List size={14} /></button>
              </div>
            </div>
          </div>

          {/* 内容 */}
          {loading ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
              {Array.from({ length: 9 }).map((_, i) => <SkeletonCard key={i} />)}
            </div>
          ) : filtered.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '100px 24px', borderRadius: 'var(--radius-xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
              <div style={{ width: '52px', height: '52px', borderRadius: '16px', background: 'var(--bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 16px' }}>
                <EyeOff size={22} style={{ color: 'var(--text-faint)' }} />
              </div>
              <p style={{ fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>{search ? `未找到「${search}」相关模型` : '无匹配模型'}</p>
              <button onClick={clearAll} style={{ marginTop: '14px', fontSize: '12px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', fontWeight: 500 }}>清除全部筛选 →</button>
            </div>
          ) : view === 'table' ? (
            /* ─── 表格视图 ─── */
            <div className="data-table-wrap">
              <table className="data-table">
                <thead>
                  <tr><th>模型</th><th>供应商</th><th>输入价格</th><th>输出价格</th><th>倍率</th></tr>
                </thead>
                <tbody>
                  {filtered.map(m => {
                    const v = vName(m.model);
                    const it = tierColor(m.input_ratio), ot = tierColor(m.output_ratio);
                    return (
                      <tr key={m.model} style={{ fontSize: '12px' }}>
                        <td><span style={{ fontFamily: 'SF Mono, Menlo, monospace', fontWeight: 600 }}>{m.model}</span></td>
                        <td><span style={{ color: v.color, fontWeight: 600 }}>{v.icon} {v.name}</span></td>
                        <td>¥{fp(m.actual_input_price)}<span style={{ color: 'var(--text-faint)' }}>/M</span></td>
                        <td><span style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>¥{fp(m.actual_output_price)}</span><span style={{ color: 'var(--text-faint)' }}>/M</span></td>
                        <td>
                          <span style={{ padding: '1px 7px', borderRadius: '5px', fontSize: '10px', fontWeight: 600, background: it.bg, color: it.fg, marginRight: '4px' }}>{m.input_ratio}x</span>
                          <span style={{ padding: '1px 7px', borderRadius: '5px', fontSize: '10px', fontWeight: 600, background: ot.bg, color: ot.fg }}>{m.output_ratio}x</span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            /* ─── 卡片视图 ─── */
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
              {filtered.map(m => {
                const v = vName(m.model);
                const isCopied = copied === m.model;
                const it = tierColor(m.input_ratio), ot = tierColor(m.output_ratio);

                return (
                  <div
                    key={m.model}
                    className="model-card-item"
                    style={{
                      borderRadius: '16px', border: '1px solid var(--border-color)',
                      background: 'var(--bg-surface)', overflow: 'hidden', position: 'relative',
                      transition: 'box-shadow 0.2s, transform 0.2s, border-color 0.2s',
                    }}
                    onMouseEnter={e => { const el = e.currentTarget; el.style.transform = 'translateY(-3px)'; el.style.boxShadow = 'var(--shadow-hover)'; el.style.borderColor = v.color === 'var(--accent-primary)' ? 'var(--accent-primary)' : `${v.color}66`; }}
                    onMouseLeave={e => { const el = e.currentTarget; el.style.transform = 'translateY(0)'; el.style.boxShadow = 'none'; el.style.borderColor = 'var(--border-color)'; }}
                  >
                    {/* 顶部品牌色强调线 */}
                    <div style={{ height: '3px', background: `linear-gradient(90deg, ${v.color}, transparent)` }} />

                    {/* 卡片主体 */}
                    <div style={{ padding: '18px 20px 16px' }}>
                      {/* 头行 */}
                      <div className="flex items-start justify-between gap-3 mb-4">
                        <div className="flex min-w-0 items-center gap-3">
                          <div style={{ width: '42px', height: '42px', borderRadius: '12px', flexShrink: 0, background: `${v.color}14`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '20px' }}>
                            {v.icon}
                          </div>
                          <div className="min-w-0">
                            <h3 style={{ fontFamily: 'SF Mono, Menlo, monospace', fontSize: '15px', lineHeight: 1.25, fontWeight: 700, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', margin: 0 }}>
                              {m.model}
                            </h3>
                            <span style={{ fontSize: '12px', color: v.color, fontWeight: 600 }}>{v.name}</span>
                          </div>
                        </div>
                        <button
                          onClick={() => handleCopy(m.model)}
                          style={{ padding: '6px', borderRadius: '8px', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: isCopied ? '#10b981' : 'var(--text-muted)', cursor: 'pointer', display: 'flex', flexShrink: 0, transition: 'color 0.15s' }}
                          title="复制模型名"
                        >
                          {isCopied ? <Check size={14} /> : <Copy size={14} />}
                        </button>
                      </div>

                      {/* 价格双列 */}
                      <div className="grid grid-cols-2 gap-2">
                        <div style={{ padding: '10px 12px', borderRadius: 'var(--radius-md)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                          <p style={{ fontSize: '10px', color: 'var(--text-muted)', margin: '0 0 3px' }}>输入 / 百万 Token</p>
                          <p style={{ fontSize: '16px', fontWeight: 700, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)', margin: 0 }}>
                            ¥{fp(m.actual_input_price)}
                          </p>
                        </div>
                        <div style={{ padding: '10px 12px', borderRadius: 'var(--radius-md)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                          <p style={{ fontSize: '10px', color: 'var(--text-muted)', margin: '0 0 3px' }}>输出 / 百万 Token</p>
                          <p style={{ fontSize: '16px', fontWeight: 700, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--accent-primary)', margin: 0 }}>
                            ¥{fp(m.actual_output_price)}
                          </p>
                        </div>
                      </div>
                    </div>

                    {/* 底部元信息 */}
                    <div style={{ padding: '10px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                      <div className="flex items-center gap-1.5">
                        <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: it.bg, color: it.fg }}>输入 {m.input_ratio}x</span>
                        <span style={{ padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: ot.bg, color: ot.fg }}>输出 {m.output_ratio}x</span>
                      </div>
                      {m.upstream_input_price > 0 && (
                        <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>上游 ¥{m.upstream_input_price.toFixed(2)}</span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
