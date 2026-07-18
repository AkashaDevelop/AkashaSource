import { Activity, Timer, CheckCircle2, BarChart3, Flame } from 'lucide-react';
import type { ModelStats } from './ModelDetailsDrawer';

// 🌸 详情抽屉 - 性能标签页：真实日志聚合的近 7 天表现～

export default function DrawerPerformanceTab({ stats }: { stats: ModelStats | null }) {
  const hasData = !!stats && stats.request_count > 0;

  if (!hasData) {
    return (
      <div style={{ textAlign: 'center', padding: '80px 24px' }}>
        <div style={{ width: '52px', height: '52px', borderRadius: '16px', background: 'var(--bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 16px' }}>
          <BarChart3 size={22} style={{ color: 'var(--text-faint)' }} />
        </div>
        <p style={{ fontSize: '14px', color: 'var(--text-muted)', margin: 0 }}>近 7 天暂无调用数据</p>
        <p style={{ fontSize: '12px', color: 'var(--text-faint)', marginTop: '6px' }}>该模型被调用后，这里会展示真实的性能表现哦～</p>
      </div>
    );
  }

  const s = stats!;
  const maxDay = Math.max(...s.daily.map(d => d.count), 1);

  const metrics = [
    { icon: <Activity size={15} />, label: '吞吐（TPS）', value: s.tps > 0 ? `${s.tps.toFixed(1)} tok/s` : '—', hint: '输出 Token / 耗时' },
    { icon: <Timer size={15} />, label: '平均延迟', value: s.avg_latency_ms > 0 ? `${(s.avg_latency_ms / 1000).toFixed(2)}s` : '—', hint: '每次请求平均耗时' },
    { icon: <CheckCircle2 size={15} />, label: '成功率', value: `${s.success_rate.toFixed(2)}%`, hint: `${s.success_count} 成功 / ${s.fail_count} 失败` },
    { icon: <Flame size={15} />, label: '24h 请求', value: String(s.requests_24h), hint: `7 天共 ${s.request_count} 次` },
  ];

  return (
    <>
      {/* 指标卡 */}
      <div className="grid grid-cols-2 gap-3 mb-6">
        {metrics.map(m => (
          <div key={m.label} style={{ padding: '14px 16px', borderRadius: '12px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
            <div className="flex items-center gap-1.5" style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '6px' }}>
              <span style={{ color: 'var(--accent-primary)', display: 'flex' }}>{m.icon}</span> {m.label}
            </div>
            <div style={{ fontSize: '20px', fontWeight: 800, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)' }}>{m.value}</div>
            <div style={{ fontSize: '10px', color: 'var(--text-faint)', marginTop: '4px' }}>{m.hint}</div>
          </div>
        ))}
      </div>

      {/* 近 7 天请求量柱状图 */}
      <div className="flex items-center gap-2 mb-3">
        <BarChart3 size={15} style={{ color: 'var(--accent-primary)' }} />
        <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--text-primary)' }}>近 7 天请求量</span>
      </div>
      <div style={{ padding: '18px 16px 8px', borderRadius: '12px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
        <div className="flex items-end justify-around gap-2" style={{ height: '120px' }}>
          {s.daily.map(d => (
            <div key={d.day} className="flex flex-col items-center gap-1" style={{ flex: 1, minWidth: 0 }}>
              <span style={{ fontSize: '10px', color: 'var(--text-muted)', fontVariantNumeric: 'tabular-nums' }}>{d.count}</span>
              <div
                style={{
                  width: '70%', maxWidth: '36px',
                  height: `${Math.max((d.count / maxDay) * 90, 3)}px`,
                  borderRadius: '5px 5px 2px 2px',
                  background: 'linear-gradient(180deg, var(--accent-primary), var(--accent-primary-soft, var(--accent-primary)))',
                  opacity: 0.85,
                }}
              />
              <span style={{ fontSize: '9px', color: 'var(--text-faint)' }}>{d.day}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 总量 */}
      <div style={{ marginTop: '14px', padding: '12px 16px', borderRadius: '12px', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', fontSize: '12px', color: 'var(--text-secondary)' }}>
        近 7 天累计处理 <b style={{ fontFamily: 'SF Mono, Menlo, monospace' }}>{s.total_tokens.toLocaleString()}</b> tokens，
        统计来自本站真实调用日志～
      </div>
    </>
  );
}
