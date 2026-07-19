import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import {
  Search, X, SlidersHorizontal, Grid3X3, List, EyeOff, ChevronDown, ChevronUp, RotateCcw,
} from 'lucide-react';
import {
  type ModelPrice, getVendor, parseTags, parseEndpoints, formatPrice, formatContext, tierColor, TIER_OPTIONS,
} from './pricing/lib';
import ModelCard from './pricing/ModelCard';
import ModelDetailsDrawer from './pricing/ModelDetailsDrawer';
import { useAuthStore } from '../store/auth';

// 🌸 模型广场：探索、比较、选择模型的公开页面～
// 对齐 new-api 新版前端：侧栏筛选（档位/供应商/标签/端点）+ 卡片/表格双视图 + 详情抽屉 + 单位切换

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

/* ～已选筛选小胶囊～ */
function ActiveChip({ label, onClear }: { label: string; onClear: () => void }) {
  return (
    <span style={{ padding: '3px 9px', borderRadius: 'var(--radius-full)', fontSize: '11px', fontWeight: 600, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)', display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
      {label}
      <button onClick={onClear} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'inherit', padding: 0, lineHeight: 1, fontSize: '13px', fontWeight: 700 }}>×</button>
    </span>
  );
}

/* ═══ 主页面 ═══ */
export default function Pricing() {
  const { token } = useAuthStore();
  const [models, setModels] = useState<ModelPrice[]>([]);
  const [groupLimits, setGroupLimits] = useState<{ group: string; rpm: number; tpm: number; rpd: number }[]>([]);
  const [loading, setLoading] = useState(false);

  const [search, setSearch] = useState('');
  const [vendorF, setVendorF] = useState<string | null>(null);
  const [tierF, setTierF] = useState<string | null>(null);
  const [tagF, setTagF] = useState<string | null>(null);
  const [endpointF, setEndpointF] = useState<string | null>(null);
  const [groupF, setGroupF] = useState<string | null>(null);
  const [sort, setSort] = useState<'name' | 'input' | 'output'>('name');
  const [view, setView] = useState<'card' | 'table'>('card');
  const [unit, setUnit] = useState<'M' | 'K'>('M');
  const [mobileFilter, setMobileFilter] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [detailModel, setDetailModel] = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const handleCopy = useCallback(async (n: string) => {
    await navigator.clipboard.writeText(n);
    setCopied(n);
    setTimeout(() => setCopied(null), 1200);
  }, []);

  useEffect(() => {
    setLoading(true);
    // 🌸 带上 token 让后端认出主人～游客不带就只返 default 分组价，登录则按权限返所有分组
    fetch('/api/pricing', token ? { headers: { Authorization: `Bearer ${token}` } } : undefined)
      .then(r => r.json())
      .then(d => {
        const payload = d.code === 0 ? d.data : d;
        if (Array.isArray(payload?.models)) setModels(payload.models);
        if (Array.isArray(payload?.group_limits)) setGroupLimits(payload.group_limits);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [token]);

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

  /* 供应商列表（元数据优先，名称推断兜底） */
  const vlist = useMemo(() => {
    const m: Record<string, { icon: string; color: string; count: number }> = {};
    models.forEach(x => {
      const v = getVendor(x);
      if (!m[v.name]) m[v.name] = { icon: v.icon, color: v.color, count: 0 };
      m[v.name].count++;
    });
    return Object.entries(m).sort(([, a], [, b]) => b.count - a.count);
  }, [models]);

  /* 标签列表 */
  const tagList = useMemo(() => {
    const m: Record<string, number> = {};
    models.forEach(x => parseTags(x.tags).forEach(t => { m[t] = (m[t] || 0) + 1; }));
    return Object.entries(m).sort(([, a], [, b]) => b - a).slice(0, 20);
  }, [models]);

  /* 端点列表 */
  const endpointList = useMemo(() => {
    const m: Record<string, number> = {};
    models.forEach(x => parseEndpoints(x.endpoints).forEach(e => { m[e] = (m[e] || 0) + 1; }));
    return Object.entries(m).sort(([, a], [, b]) => b - a);
  }, [models]);

  /* 分组列表：模型自带的分组 ∪ 后端分组限制表 */
  const groupList = useMemo(() => {
    const m: Record<string, number> = {};
    models.forEach(x => (x.groups || []).forEach(g => { m[g] = (m[g] || 0) + 1; }));
    // 分组限制表里有但还没模型的分组也展示（计数 0）
    groupLimits.forEach(gl => { if (!(gl.group in m)) m[gl.group] = 0; });
    return Object.entries(m).sort(([, a], [, b]) => b - a);
  }, [models, groupLimits]);

  /* 过滤 + 排序 */
  const filtered = useMemo(() => {
    let list = models;
    const q = search.toLowerCase().trim();
    if (q) {
      list = list.filter(x =>
        x.model.toLowerCase().includes(q)
        || getVendor(x).name.toLowerCase().includes(q)
        || (x.description || '').toLowerCase().includes(q)
        || parseTags(x.tags).some(t => t.toLowerCase().includes(q))
        || parseEndpoints(x.endpoints).some(e => e.toLowerCase().includes(q))
      );
    }
    if (vendorF) list = list.filter(x => getVendor(x).name === vendorF);
    if (tagF) list = list.filter(x => parseTags(x.tags).includes(tagF));
    if (endpointF) list = list.filter(x => parseEndpoints(x.endpoints).includes(endpointF));
    if (groupF) list = list.filter(x => (x.groups || []).includes(groupF));
    if (tierF) {
      const opt = TIER_OPTIONS.find(t => t.key === tierF);
      if (opt) list = list.filter(opt.test);
    }
    return [...list].sort((a, b) => {
      if (sort === 'input') return b.actual_input_price - a.actual_input_price;
      if (sort === 'output') return b.actual_output_price - a.actual_output_price;
      return a.model.localeCompare(b.model);
    });
  }, [models, search, vendorF, tierF, tagF, endpointF, groupF, sort]);

  const hasF = !!(vendorF || tierF || tagF || endpointF || groupF);
  const fCnt = (vendorF ? 1 : 0) + (tierF ? 1 : 0) + (tagF ? 1 : 0) + (endpointF ? 1 : 0) + (groupF ? 1 : 0);
  const clearAll = () => { setSearch(''); setVendorF(null); setTierF(null); setTagF(null); setEndpointF(null); setGroupF(null); };

  const selectedModel = useMemo(
    () => (detailModel ? models.find(m => m.model === detailModel) ?? null : null),
    [models, detailModel],
  );

  /* ═══ 侧边栏 ═══ */
  const sidebar = (
    <div>
      <div className="flex items-center justify-between" style={{ padding: '16px 16px 4px' }}>
        <span style={{ fontSize: '14px', fontWeight: 700, color: 'var(--text-primary)' }}>筛选</span>
        {hasF && (
          <button onClick={() => { setVendorF(null); setTierF(null); setTagF(null); setEndpointF(null); setGroupF(null); }} className="flex items-center gap-1" style={{ fontSize: '11px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer' }}>
            <RotateCcw size={11} /> 重置
          </button>
        )}
      </div>

      <FilterGroup title="价格档位">
        <FilterItem label="全部档位" count={models.length} active={!tierF} onClick={() => setTierF(null)} />
        {TIER_OPTIONS.map(t => (
          <FilterItem key={t.key} label={t.label} dot={t.dot} count={models.filter(t.test).length} active={tierF === t.key} onClick={() => setTierF(tierF === t.key ? null : t.key)} />
        ))}
      </FilterGroup>

      {groupList.length > 0 && (
        <>
          <div style={{ borderTop: '1px solid var(--border-color)', margin: '0 16px' }} />
          <FilterGroup title="分组">
            <FilterItem label="全部分组" count={models.length} active={!groupF} onClick={() => setGroupF(null)} />
            {groupList.map(([name, count]) => (
              <FilterItem key={name} label={name} count={count} active={groupF === name} onClick={() => setGroupF(groupF === name ? null : name)} />
            ))}
          </FilterGroup>
        </>
      )}

      <div style={{ borderTop: '1px solid var(--border-color)', margin: '0 16px' }} />

      <FilterGroup title="供应商">
        <FilterItem label="全部供应商" count={models.length} active={!vendorF} onClick={() => setVendorF(null)} />
        {vlist.map(([name, info]) => (
          <FilterItem key={name} label={name} count={info.count} active={vendorF === name} onClick={() => setVendorF(vendorF === name ? null : name)} icon={info.icon} />
        ))}
      </FilterGroup>

      {endpointList.length > 0 && (
        <>
          <div style={{ borderTop: '1px solid var(--border-color)', margin: '0 16px' }} />
          <FilterGroup title="端点类型">
            <FilterItem label="全部端点" count={models.length} active={!endpointF} onClick={() => setEndpointF(null)} />
            {endpointList.map(([name, count]) => (
              <FilterItem key={name} label={name} count={count} active={endpointF === name} onClick={() => setEndpointF(endpointF === name ? null : name)} />
            ))}
          </FilterGroup>
        </>
      )}

      {tagList.length > 0 && (
        <>
          <div style={{ borderTop: '1px solid var(--border-color)', margin: '0 16px' }} />
          <FilterGroup title="能力标签" defaultOpen={false}>
            <FilterItem label="全部标签" count={models.length} active={!tagF} onClick={() => setTagF(null)} />
            {tagList.map(([name, count]) => (
              <FilterItem key={name} label={name} count={count} active={tagF === name} onClick={() => setTagF(tagF === name ? null : name)} />
            ))}
          </FilterGroup>
        </>
      )}
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
          {loading ? '正在加载模型数据...' : `本站当前已启用 ${models.length} 个模型`}
        </p>
        <p style={{ fontSize: '12px', color: 'var(--text-faint)', marginTop: '4px', maxWidth: '520px', marginLeft: 'auto', marginRight: 'auto' }}>
          探索精选 AI 模型，比较定价与能力，为每个场景选择合适的模型。
        </p>

        {/* 居中大搜索框 */}
        <div style={{ maxWidth: '560px', margin: '20px auto 0', position: 'relative' }}>
          <Search size={17} style={{ position: 'absolute', left: '16px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-faint)', pointerEvents: 'none' }} />
          <input
            ref={searchRef}
            placeholder="搜索模型名称、供应商、端点或标签..."
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
        <aside className="hidden md:block" style={{ width: '240px', flexShrink: 0 }}>
          <div style={{ borderRadius: 'var(--radius-xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', overflow: 'hidden', position: 'sticky', top: '68px', paddingBottom: '8px', maxHeight: 'calc(100dvh - 84px)', overflowY: 'auto' }}>
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
              {tierF && <ActiveChip label={TIER_OPTIONS.find(t => t.key === tierF)?.label || tierF} onClear={() => setTierF(null)} />}
              {groupF && <ActiveChip label={`分组: ${groupF}`} onClear={() => setGroupF(null)} />}
              {vendorF && <ActiveChip label={vendorF} onClear={() => setVendorF(null)} />}
              {endpointF && <ActiveChip label={endpointF} onClear={() => setEndpointF(null)} />}
              {tagF && <ActiveChip label={tagF} onClear={() => setTagF(null)} />}
            </div>

            <div className="flex items-center gap-2">
              {/* 单位切换 */}
              <div className="flex items-center" style={{ background: 'var(--bg-elevated)', borderRadius: '9px', border: '1px solid var(--border-color)', padding: '2px' }}>
                {([['M', '/1M'], ['K', '/1K']] as const).map(([k, l]) => (
                  <button key={k} onClick={() => setUnit(k)} style={{
                    padding: '4px 10px', borderRadius: '7px', border: 'none', fontSize: '12px', fontWeight: 500, cursor: 'pointer',
                    background: unit === k ? 'var(--bg-surface)' : 'transparent',
                    color: unit === k ? 'var(--accent-primary)' : 'var(--text-muted)',
                    boxShadow: unit === k ? 'var(--shadow-card)' : 'none',
                    transition: 'all 0.12s',
                  }}>{l}</button>
                ))}
              </div>

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
                  <tr><th>模型</th><th>供应商</th><th>标签</th><th>上下文</th><th>输入价格</th><th>输出价格</th><th>倍率</th></tr>
                </thead>
                <tbody>
                  {filtered.map(m => {
                    const v = getVendor(m);
                    const tags = parseTags(m.tags);
                    const ctx = formatContext(m.context_length);
                    const it = tierColor(m.input_ratio), ot = tierColor(m.output_ratio);
                    return (
                      <tr key={m.model} style={{ fontSize: '12px', cursor: 'pointer' }} onClick={() => setDetailModel(m.model)}>
                        <td><span style={{ fontFamily: 'SF Mono, Menlo, monospace', fontWeight: 600 }}>{m.model}</span></td>
                        <td><span style={{ color: v.color, fontWeight: 600 }}>{v.icon} {v.name}</span></td>
                        <td>
                          {tags.length > 0 ? (
                            <span className="flex flex-wrap gap-1">
                              {tags.slice(0, 3).map(t => (
                                <span key={t} style={{ padding: '1px 7px', borderRadius: '5px', fontSize: '10px', fontWeight: 500, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)' }}>{t}</span>
                              ))}
                              {tags.length > 3 && <span style={{ fontSize: '10px', color: 'var(--text-faint)' }}>+{tags.length - 3}</span>}
                            </span>
                          ) : <span style={{ color: 'var(--text-faint)' }}>—</span>}
                        </td>
                        <td>{ctx || <span style={{ color: 'var(--text-faint)' }}>—</span>}</td>
                        <td>{m.actual_input_price <= 0 ? '免费' : <>¥{formatPrice(m.actual_input_price, unit)}<span style={{ color: 'var(--text-faint)' }}>/{unit === 'M' ? 'M' : 'K'}</span></>}</td>
                        <td>{m.actual_output_price <= 0 ? '免费' : <><span style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>¥{formatPrice(m.actual_output_price, unit)}</span><span style={{ color: 'var(--text-faint)' }}>/{unit === 'M' ? 'M' : 'K'}</span></>}</td>
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
              {filtered.map(m => (
                <ModelCard key={m.model} m={m} unit={unit} copied={copied} onCopy={handleCopy} onClick={setDetailModel} />
              ))}
            </div>
          )}
        </main>
      </div>

      {/* 详情抽屉 */}
      <ModelDetailsDrawer
        model={selectedModel}
        unit={unit}
        copied={copied}
        groupLimits={groupLimits}
        onCopy={handleCopy}
        onClose={() => setDetailModel(null)}
      />
    </div>
  );
}
