import { Zap, Network, Cpu, DollarSign, Timer, CheckCircle2, Activity } from 'lucide-react';
import { type ModelPrice, getVendor, parseTags, parseEndpoints, formatPrice, formatContext, tierColor } from './lib';
import type { ModelStats } from './ModelDetailsDrawer';

// 🌸 详情抽屉 - 概览标签页：性能速览条 + 定价 + 标签 + 端点 + 规格～

function Section({ icon, title, children }: { icon: React.ReactNode; title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: '20px' }}>
      <div className="flex items-center gap-2 mb-3">
        <span style={{ color: 'var(--accent-primary)', display: 'flex' }}>{icon}</span>
        <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--text-primary)' }}>{title}</span>
      </div>
      {children}
    </div>
  );
}

export default function DrawerOverviewTab({ model, unit, stats }: { model: ModelPrice; unit: 'M' | 'K'; stats: ModelStats | null }) {
  const v = getVendor(model);
  const tags = parseTags(model.tags);
  const endpoints = parseEndpoints(model.endpoints);
  const ctx = formatContext(model.context_length);
  const it = tierColor(model.input_ratio);
  const ot = tierColor(model.output_ratio);
  const unitLabel = unit === 'M' ? '百万' : '千';

  const hasStats = !!stats && stats.request_count > 0;

  return (
    <>
      {/* 性能速览条（对齐 new-api 概览页顶部的 TPS/延迟/成功率）～ */}
      <div className="grid grid-cols-3 gap-2 mb-5">
        {[
          { icon: <Activity size={13} />, label: 'TPS', value: hasStats && stats!.tps > 0 ? stats!.tps.toFixed(1) : '—' },
          { icon: <Timer size={13} />, label: '平均延迟', value: hasStats && stats!.avg_latency_ms > 0 ? `${(stats!.avg_latency_ms / 1000).toFixed(1)}s` : '—' },
          { icon: <CheckCircle2 size={13} />, label: '成功率', value: hasStats ? `${stats!.success_rate.toFixed(1)}%` : '—' },
        ].map(x => (
          <div key={x.label} style={{ padding: '10px 14px', borderRadius: '10px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
            <div className="flex items-center gap-1" style={{ fontSize: '10px', color: 'var(--text-muted)', marginBottom: '3px' }}>
              {x.icon} {x.label}
            </div>
            <div style={{ fontSize: '15px', fontWeight: 700, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)' }}>{x.value}</div>
          </div>
        ))}
      </div>

      {/* 定价 */}
      <Section icon={<DollarSign size={15} />} title="定价信息">
        <div className="grid grid-cols-2 gap-3">
          <div style={{ padding: '14px 16px', borderRadius: '12px', background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
            <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: '0 0 4px' }}>输入 / {unitLabel} Token</p>
            <p style={{ fontSize: '20px', fontWeight: 800, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)', margin: 0 }}>
              {model.actual_input_price <= 0 ? '免费' : `¥${formatPrice(model.actual_input_price, unit)}`}
            </p>
            <span style={{ display: 'inline-block', marginTop: '6px', padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: it.bg, color: it.fg }}>倍率 {model.input_ratio}x</span>
          </div>
          <div style={{ padding: '14px 16px', borderRadius: '12px', background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
            <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: '0 0 4px' }}>输出 / {unitLabel} Token</p>
            <p style={{ fontSize: '20px', fontWeight: 800, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--accent-primary)', margin: 0 }}>
              {model.actual_output_price <= 0 ? '免费' : `¥${formatPrice(model.actual_output_price, unit)}`}
            </p>
            <span style={{ display: 'inline-block', marginTop: '6px', padding: '2px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 600, background: ot.bg, color: ot.fg }}>倍率 {model.output_ratio}x</span>
          </div>
        </div>
        {model.upstream_input_price > 0 && (
          <p style={{ fontSize: '11px', color: 'var(--text-faint)', margin: '10px 0 0' }}>
            上游参考价：输入 ¥{model.upstream_input_price.toFixed(2)} / 输出 ¥{model.upstream_output_price.toFixed(2)}（每百万 Token）
          </p>
        )}

        {/* 计费示例 */}
        <div style={{ marginTop: '10px', padding: '12px 16px', borderRadius: '12px', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', fontSize: '12px', color: 'var(--text-secondary)', lineHeight: 1.8 }}>
          一次请求消耗 <b>1000</b> 输入 + <b>500</b> 输出 Token 约花费：
          <span style={{ fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)', marginLeft: '6px', fontWeight: 700 }}>
            ¥{((model.actual_input_price * 1000 + model.actual_output_price * 500) / 1000000).toFixed(6)}
          </span>
        </div>
      </Section>

      {/* 能力标签 */}
      {tags.length > 0 && (
        <Section icon={<Zap size={15} />} title="能力标签">
          <div className="flex flex-wrap gap-2">
            {tags.map(t => (
              <span key={t} style={{ padding: '4px 12px', borderRadius: '8px', fontSize: '12px', fontWeight: 500, background: 'var(--nav-active-bg)', color: 'var(--accent-primary)' }}>{t}</span>
            ))}
          </div>
        </Section>
      )}

      {/* 可用端点 */}
      {endpoints.length > 0 && (
        <Section icon={<Network size={15} />} title="可用端点">
          <div className="flex flex-wrap gap-2">
            {endpoints.map(ep => (
              <span key={ep} style={{ padding: '4px 12px', borderRadius: '8px', fontSize: '12px', fontWeight: 500, fontFamily: 'SF Mono, Menlo, monospace', background: 'rgba(8,145,178,0.10)', color: '#0891b2' }}>{ep}</span>
            ))}
          </div>
        </Section>
      )}

      {/* 模型规格 */}
      <Section icon={<Cpu size={15} />} title="模型规格">
        <div style={{ borderRadius: '12px', border: '1px solid var(--border-color)', overflow: 'hidden' }}>
          {[
            ['模型名称', model.model, true],
            ['供应商', v.name, false],
            ['计费类型', '按量计费', false],
            ['上下文窗口', ctx ? `${ctx} tokens` : '未知', false],
            ['输入倍率', `${model.input_ratio}x`, false],
            ['输出倍率', `${model.output_ratio}x`, false],
          ].map(([label, value, mono], idx) => (
            <div key={String(label)} className="flex items-center justify-between" style={{ padding: '10px 16px', fontSize: '13px', background: idx % 2 === 0 ? 'var(--bg-surface)' : 'var(--bg-elevated)' }}>
              <span style={{ color: 'var(--text-muted)' }}>{label}</span>
              <span style={{ color: 'var(--text-primary)', fontWeight: 500, fontFamily: mono ? 'SF Mono, Menlo, monospace' : undefined, maxWidth: '60%', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={String(value)}>
                {value}
              </span>
            </div>
          ))}
        </div>
      </Section>
    </>
  );
}
