import { useEffect, useState } from 'react';
import { X, Copy, Check, Info, Gauge, Code2 } from 'lucide-react';
import { type ModelPrice, getVendor, formatContext } from './lib';
import DrawerOverviewTab from './DrawerOverviewTab';
import DrawerPerformanceTab from './DrawerPerformanceTab';
import DrawerApiTab from './DrawerApiTab';

// 🌸 模型详情抽屉：概览 / 性能 / API 三标签页，对齐 new-api 设计～

export interface ModelStats {
  request_count: number;
  success_count: number;
  fail_count: number;
  success_rate: number;
  avg_latency_ms: number;
  tps: number;
  requests_24h: number;
  total_tokens: number;
  daily: { day: string; count: number }[];
}

export interface GroupLimit {
  group: string;
  rpm: number;
  tpm: number;
  rpd: number;
}

interface Props {
  model: ModelPrice | null;
  unit: 'M' | 'K';
  copied: string | null;
  groupLimits: GroupLimit[];
  onCopy: (name: string) => void;
  onClose: () => void;
}

const TABS = [
  { key: 'overview', label: '概览', icon: Info },
  { key: 'perf', label: '性能', icon: Gauge },
  { key: 'api', label: 'API', icon: Code2 },
] as const;

type TabKey = typeof TABS[number]['key'];

export default function ModelDetailsDrawer({ model, unit, copied, groupLimits, onCopy, onClose }: Props) {
  const [tab, setTab] = useState<TabKey>('overview');
  const [stats, setStats] = useState<ModelStats | null>(null);

  // ～ESC 关闭～
  useEffect(() => {
    if (!model) return;
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [model, onClose]);

  // ～打开时拉取真实统计数据～
  useEffect(() => {
    if (!model) return;
    setTab('overview');
    setStats(null);
    fetch(`/api/pricing/stats?model=${encodeURIComponent(model.model)}`)
      .then(r => r.json())
      .then(d => { if (d.code === 0 && d.data) setStats(d.data); })
      .catch(() => {});
  }, [model?.model]);

  if (!model) return null;

  const v = getVendor(model);
  const ctx = formatContext(model.context_length);
  const isCopied = copied === model.model;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-end">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      <div className="relative h-full w-full max-w-xl shadow-xl flex flex-col animate-slide-in-right" style={{ background: 'var(--bg-base)' }}>
        {/* 头部 */}
        <div style={{ padding: '20px 24px 0' }}>
          <div className="flex items-start justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <div style={{ width: '48px', height: '48px', borderRadius: '14px', flexShrink: 0, background: `${v.color}14`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '24px' }}>
                {v.icon}
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <h2 style={{ fontFamily: 'SF Mono, Menlo, monospace', fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={model.model}>
                    {model.model}
                  </h2>
                  <button
                    onClick={() => onCopy(model.model)}
                    style={{ padding: '4px', borderRadius: '6px', border: 'none', background: 'transparent', color: isCopied ? '#10b981' : 'var(--text-faint)', cursor: 'pointer', display: 'flex', flexShrink: 0 }}
                    title="复制模型名"
                  >
                    {isCopied ? <Check size={14} /> : <Copy size={14} />}
                  </button>
                </div>
                <div className="flex items-center gap-2 mt-0.5">
                  <span style={{ fontSize: '13px', color: v.color, fontWeight: 600 }}>{v.name}</span>
                  <span style={{ fontSize: '11px', color: 'var(--accent-primary)' }}>按量计费</span>
                  {ctx && (
                    <span style={{ fontSize: '11px', color: 'var(--text-muted)', background: 'var(--bg-elevated)', padding: '1px 8px', borderRadius: '6px' }}>
                      {ctx} 上下文
                    </span>
                  )}
                </div>
              </div>
            </div>
            <button onClick={onClose} style={{ padding: '6px', borderRadius: '8px', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: 'var(--text-muted)', cursor: 'pointer', display: 'flex', flexShrink: 0 }}>
              <X size={16} />
            </button>
          </div>

          {model.description && (
            <p style={{ fontSize: '13px', color: 'var(--text-secondary)', lineHeight: 1.6, margin: '12px 0 0' }}>
              {model.description}
            </p>
          )}

          {/* 标签页切换 */}
          <div className="flex items-center gap-1 mt-4" style={{ background: 'var(--bg-elevated)', borderRadius: '12px', padding: '3px', border: '1px solid var(--border-color)' }}>
            {TABS.map(t => {
              const Icon = t.icon;
              const active = tab === t.key;
              return (
                <button
                  key={t.key}
                  onClick={() => setTab(t.key)}
                  className="flex-1 flex items-center justify-center gap-1.5"
                  style={{
                    padding: '8px 0', borderRadius: '9px', border: 'none', fontSize: '13px', fontWeight: 600, cursor: 'pointer',
                    background: active ? 'var(--bg-surface)' : 'transparent',
                    color: active ? 'var(--text-primary)' : 'var(--text-muted)',
                    boxShadow: active ? 'var(--shadow-card)' : 'none',
                    transition: 'all 0.12s',
                  }}
                >
                  <Icon size={14} /> {t.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '20px 24px' }}>
          {tab === 'overview' && <DrawerOverviewTab model={model} unit={unit} stats={stats} />}
          {tab === 'perf' && <DrawerPerformanceTab stats={stats} />}
          {tab === 'api' && <DrawerApiTab model={model} groupLimits={groupLimits} />}
        </div>
      </div>
    </div>
  );
}
