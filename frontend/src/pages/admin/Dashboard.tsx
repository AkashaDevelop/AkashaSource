import { useState, useEffect, useRef, useCallback } from 'react';
import {
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  AreaChart, Area, Cell, Line, LineChart, PieChart, Pie,
} from 'recharts';
import {
  Server, Activity, DollarSign, Users, RefreshCw, Cpu, MemoryStick,
  Clock, Hash, ArrowDownToLine, ArrowUpFromLine, Gauge, AlertTriangle,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import StatCard from '../../components/StatCard';
import { SkeletonCard } from '../../components/SkeletonCard';

/* ───── 类型 ───── */
interface DashboardStats {
  user_count: number;
  channel_count: number;
  request_count: number;
  total_quota_used: number;
  active_channels: number;
}
interface DailyCount { date: string; count: number; }
interface PerformanceData { goroutines: number; memory_mb: number; gc_cycles: number; uptime: string; go_version: string; }
interface SystemStats { cpu_usage: number; memory_usage: number; memory_total: number; memory_used: number; goroutines: number; timestamp: number; }
interface ModelStat { model_name: string; total_quota: number; prompt_tokens: number; completion_tokens: number; request_count: number; }
interface RecentLog { id: number; created_at: number; model_name: string; quota: number; type: number; username: string; token_name: string; prompt_tokens: number; completion_tokens: number; }
interface PeriodStats { requests: number; quota: number; promptTokens: number; completionTokens: number; tokens: number; }

/* ───── 常量 ───── */
const LOG_TYPE_COLOR: Record<number, string> = {
  1: 'var(--accent-primary)',
  2: 'var(--color-success-fg)',
  3: 'var(--accent-cosmic)',
  4: 'var(--color-danger-fg)',
};
const BAR_COLORS = ['#a78bfa', '#22d3ee', '#fbbf24', '#34d399', '#f87171', '#94a3b8'];

/* ───── 实时迷你折线 ───── */
function MiniSpark({ data, color }: { data: number[]; color: string }) {
  const chartData = data.map((v, i) => ({ i, v }));
  return (
    <div style={{ height: '36px', width: '100%' }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData}>
          <Line type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

/* ───── 骨架图表 ───── */
function SkeletonChart({ height = 300 }: { height?: number }) {
  return (
    <div className="akasha-card p-5" style={{ height }}>
      <div className="skeleton-shimmer h-5 w-36 rounded mb-4" />
      <div className="skeleton-shimmer rounded" style={{ height: height - 76 }} />
    </div>
  );
}

/* ═══════════════════════════════════════ */
export default function AdminDashboard() {
  const { token, user } = useAuthStore();
  const isSuperAdmin = (user?.role ?? 0) >= 100;

  const [stats,      setStats]      = useState<DashboardStats | null>(null);
  const [chartReq,   setChartReq]   = useState<DailyCount[]>([]);
  const [chartDau,   setChartDau]   = useState<DailyCount[]>([]);
  const [chartErr,   setChartErr]   = useState<DailyCount[]>([]);
  const [perf,       setPerf]       = useState<PerformanceData | null>(null);
  const [systemStats,setSystemStats]= useState<SystemStats | null>(null);
  const [modelStats, setModelStats] = useState<ModelStat[]>([]);
  const [recentLogs, setRecentLogs] = useState<RecentLog[]>([]);
  const [periodStats,setPeriodStats]= useState<PeriodStats | null>(null);
  const [loading,    setLoading]    = useState(false);
  const [statLoading,setStatLoading]= useState(false);

  // 实时历史缓冲（迷你折线）
  const cpuHist = useRef<number[]>([]);
  const memHist = useRef<number[]>([]);
  const [, forceTick] = useState(0);

  const fetchDashboard = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/dashboard', { headers: { Authorization: `Bearer ${token}` } });
      const d = await res.json();
      if (d.code === 0) {
        setStats(d.data.stats);
        setChartReq(d.data.chart || []);
        setChartDau(d.data.chart_dau || []);
        setChartErr(d.data.chart_error || []);
      }
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  }, [token]);

  const fetchPeriodData = useCallback(async () => {
    if (!token) return;
    setStatLoading(true);
    try {
      const [sRes, lRes] = await Promise.all([
        fetch('/api/log/stat',                 { headers: { Authorization: `Bearer ${token}` } }),
        fetch('/api/log/search?page=1&size=8', { headers: { Authorization: `Bearer ${token}` } }),
      ]);
      const [sData, lData] = await Promise.all([sRes.json(), lRes.json()]);
      if (sData.code === 0 && sData.data) {
        const items: ModelStat[] = sData.data.items || [];
        setModelStats([...items].sort((a, b) => b.total_quota - a.total_quota).slice(0, 6));
        const promptTok = items.reduce((s, i) => s + i.prompt_tokens, 0);
        const compTok   = items.reduce((s, i) => s + i.completion_tokens, 0);
        setPeriodStats({
          requests:        items.reduce((s, i) => s + i.request_count, 0),
          quota:           sData.data.total_quota || 0,
          promptTokens:    promptTok,
          completionTokens:compTok,
          tokens:          promptTok + compTok,
        });
      }
      if (lData.code === 0) setRecentLogs((lData.data?.data || []).slice(0, 7));
    } catch (e) { console.error(e); }
    finally { setStatLoading(false); }
  }, [token]);

  const fetchPerformance = useCallback(async () => {
    try {
      const res = await fetch('/api/performance', { headers: { Authorization: `Bearer ${token}` } });
      const d = await res.json();
      if (d.code === 0) setPerf(d.data);
    } catch (e) { console.error(e); }
  }, [token]);

  const fetchSystemStats = useCallback(async () => {
    try {
      const res = await fetch('/api/admin/system/monitor', { headers: { Authorization: `Bearer ${token}` } });
      const d = await res.json();
      if (d.code === 0 && d.data) {
        setSystemStats(d.data);
        cpuHist.current = [...cpuHist.current, d.data.cpu_usage].slice(-20);
        memHist.current = [...memHist.current, d.data.memory_usage].slice(-20);
        forceTick(t => t + 1);
      }
    } catch (e) { console.error(e); }
  }, [token]);

  useEffect(() => {
    if (!token) return;
    fetchDashboard();
    fetchPeriodData();
    if (isSuperAdmin) {
      fetchPerformance();
      fetchSystemStats();
      const interval = setInterval(fetchSystemStats, 5000);
      return () => clearInterval(interval);
    }
  }, [token, isSuperAdmin, fetchDashboard, fetchPeriodData, fetchPerformance, fetchSystemStats]);

  /* 衍生值 */
  const channelHealthPct = stats?.channel_count ? (stats.active_channels / stats.channel_count) * 100 : 0;
  const totalModelQ = modelStats.reduce((s, m) => s + m.total_quota, 0);
  const todayDau = chartDau.length ? chartDau[chartDau.length - 1].count : 0;
  const totalErr = chartErr.reduce((s, e) => s + e.count, 0);
  const totalReq7d = chartReq.reduce((s, r) => s + r.count, 0);
  const errRate = totalReq7d > 0 ? (totalErr / totalReq7d) * 100 : 0;
  const avgCostPerReq = periodStats && periodStats.requests > 0 ? periodStats.quota / periodStats.requests : 0;
  const pieData = modelStats.slice(0, 5).map(m => ({
    name: m.model_name.length > 16 ? m.model_name.slice(0, 14) + '…' : m.model_name,
    value: m.total_quota,
  }));

  /* 合并请求/错误图数据 */
  const mergedChart = chartReq.map((r, i) => ({
    date: r.date,
    requests: r.count,
    errors: chartErr[i]?.count ?? 0,
  }));

  const refreshAll = () => { fetchDashboard(); fetchPeriodData(); if (isSuperAdmin) { fetchPerformance(); fetchSystemStats(); } };

  return (
    <div className="space-y-5 animate-fade-in-up">
      {/* ══ 头部 ══ */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold gradient-text">系统概览</h1>
          <p style={{ fontSize: '13px', color: 'var(--text-muted)', marginTop: '2px' }}>
            {perf ? `平台运行状态 · Go ${perf.go_version} · 已运行 ${perf.uptime}` : '监控平台运行状态与核心指标'}
          </p>
        </div>
        <button onClick={refreshAll}
          className="flex items-center justify-center"
          style={{ width: '38px', height: '38px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', color: 'var(--accent-primary)', border: '1px solid var(--border-color)', cursor: 'pointer' }}>
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* ══ 核心指标 ══ */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {loading && !stats ? Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} lines={1} />) : (<>
          <StatCard
            title="总请求数"
            value={stats?.request_count?.toLocaleString() ?? 0}
            icon={<Activity size={20} style={{ color: 'var(--accent-primary)' }} />}
            iconBg="var(--color-info-bg)"
            footer={`近7日 ${totalReq7d.toLocaleString()} 次`}
          />
          <StatCard
            title="总消耗金额"
            value={<span style={{ color: 'var(--accent-cosmic)' }}>${(stats?.total_quota_used || 0).toFixed(2)}</span>}
            icon={<DollarSign size={20} style={{ color: 'var(--accent-cosmic)' }} />}
            iconBg="var(--color-info-bg)"
            footer="平台累计消耗"
          />
          <StatCard
            title="渠道健康"
            value={`${stats?.active_channels ?? 0} / ${stats?.channel_count ?? 0}`}
            icon={<Server size={20} style={{ color: 'var(--accent-star)' }} />}
            iconBg="var(--color-warning-bg)"
            footer={<Progress pct={channelHealthPct} />}
          />
          <StatCard
            title="总用户数"
            value={<span style={{ color: 'var(--accent-cosmic)' }}>{stats?.user_count?.toLocaleString() ?? 0}</span>}
            icon={<Users size={20} style={{ color: 'var(--accent-cosmic)' }} />}
            iconBg="var(--color-info-bg)"
            footer={`今日活跃 ${todayDau} 人`}
          />
        </>)}
      </div>

      {/* ══ 性能指标横条 ══ */}
      <div className="akasha-card px-5 py-3.5">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: '总 Token',     value: statLoading ? '—' : (periodStats?.tokens ?? 0).toLocaleString(),            icon: <Hash size={14} />,            color: 'var(--accent-primary)' },
            { label: '输入 Token',   value: statLoading ? '—' : (periodStats?.promptTokens ?? 0).toLocaleString(),      icon: <ArrowDownToLine size={14} />, color: 'var(--accent-cosmic)' },
            { label: '输出 Token',   value: statLoading ? '—' : (periodStats?.completionTokens ?? 0).toLocaleString(),  icon: <ArrowUpFromLine size={14} />, color: 'var(--accent-star)' },
            { label: '平均单次消耗', value: statLoading ? '—' : `$${(avgCostPerReq / 500000).toFixed(5)}`,               icon: <Gauge size={14} />,           color: 'var(--color-success-fg)' },
          ].map(m => (
            <div key={m.label} className="flex items-center gap-2.5">
              <span style={{ color: m.color, opacity: 0.85 }}>{m.icon}</span>
              <div>
                <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>{m.label}</p>
                <p style={{ fontSize: '14px', fontWeight: 700, color: 'var(--text-primary)', margin: 0, lineHeight: 1.3, fontFamily: 'monospace' }}>{m.value}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Token 构成条 */}
        {!statLoading && periodStats && periodStats.tokens > 0 && (
          <div className="mt-3 pt-3" style={{ borderTop: '1px solid var(--border-color)' }}>
            <div className="flex items-center justify-between mb-1.5">
              <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Token 构成</span>
              <div className="flex items-center gap-3">
                <span style={{ fontSize: '11px', color: 'var(--accent-cosmic)' }}>■ 输入 {((periodStats.promptTokens / periodStats.tokens) * 100).toFixed(1)}%</span>
                <span style={{ fontSize: '11px', color: 'var(--accent-star)' }}>■ 输出 {((periodStats.completionTokens / periodStats.tokens) * 100).toFixed(1)}%</span>
              </div>
            </div>
            <div style={{ height: '5px', borderRadius: 'var(--radius-full)', background: 'var(--bg-elevated)', overflow: 'hidden', display: 'flex' }}>
              <div style={{ height: '100%', width: `${(periodStats.promptTokens / periodStats.tokens) * 100}%`, background: 'var(--accent-cosmic)', transition: 'width 0.6s ease' }} />
              <div style={{ height: '100%', flex: 1, background: 'var(--accent-star)' }} />
            </div>
          </div>
        )}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* 调用 + 错误趋势 */}
        {loading && !mergedChart.length ? <div className="lg:col-span-2"><SkeletonChart height={320} /></div> : (
          <div className="lg:col-span-2 akasha-card p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-subtitle" style={{ color: 'var(--text-primary)' }}>近 7 日调用趋势</h3>
              <div className="flex items-center gap-3">
                <span className="flex items-center gap-1.5" style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  <span style={{ width: '8px', height: '8px', borderRadius: '2px', background: 'var(--accent-primary)' }} /> 调用
                </span>
                <span className="flex items-center gap-1.5" style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  <span style={{ width: '8px', height: '8px', borderRadius: '2px', background: 'var(--color-danger-fg)' }} /> 失败
                </span>
              </div>
            </div>
            <div style={{ height: '240px' }}>
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={mergedChart}>
                  <defs>
                    <linearGradient id="gReq" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="var(--accent-primary)" stopOpacity={0.4} />
                      <stop offset="95%" stopColor="var(--accent-primary)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                  <XAxis dataKey="date" tick={{ fill: 'var(--text-muted)', fontSize: 11 }} />
                  <YAxis tick={{ fill: 'var(--text-muted)', fontSize: 11 }} />
                  <Tooltip contentStyle={{ backgroundColor: 'var(--bg-elevated)', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border-color)', fontSize: 12 }} />
                  <Area type="monotone" dataKey="requests" name="调用" stroke="var(--accent-primary)" strokeWidth={2} fillOpacity={1} fill="url(#gReq)" />
                  <Area type="monotone" dataKey="errors" name="失败" stroke="var(--color-danger-fg)" strokeWidth={1.5} fillOpacity={0} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}

        {/* 右侧：实时系统状态 */}
        <div className="flex flex-col gap-4">
          {isSuperAdmin && systemStats ? (
            <>
              {/* CPU */}
              <div className="akasha-card p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Cpu size={15} style={{ color: 'var(--accent-primary)' }} />
                    <span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)' }}>CPU 使用率</span>
                  </div>
                  <span style={{ fontSize: '16px', fontWeight: 700, color: systemStats.cpu_usage > 80 ? 'var(--color-danger-fg)' : 'var(--text-primary)', fontFamily: 'monospace' }}>
                    {systemStats.cpu_usage.toFixed(1)}%
                  </span>
                </div>
                <MiniSpark data={cpuHist.current} color={systemStats.cpu_usage > 80 ? '#f87171' : 'var(--accent-primary)'} />
              </div>
              {/* 内存 */}
              <div className="akasha-card p-4">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <MemoryStick size={15} style={{ color: 'var(--color-warning-fg)' }} />
                    <span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)' }}>内存使用率</span>
                  </div>
                  <span style={{ fontSize: '16px', fontWeight: 700, color: systemStats.memory_usage > 80 ? 'var(--color-danger-fg)' : 'var(--text-primary)', fontFamily: 'monospace' }}>
                    {systemStats.memory_usage.toFixed(1)}%
                  </span>
                </div>
                <MiniSpark data={memHist.current} color={systemStats.memory_usage > 80 ? '#f87171' : 'var(--color-warning-fg)'} />
                <p style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px', textAlign: 'right' }}>
                  {(systemStats.memory_used / 1073741824).toFixed(2)} / {(systemStats.memory_total / 1073741824).toFixed(2)} GB
                </p>
              </div>
              {/* 运行时信息 */}
              {perf && (
                <div className="akasha-card p-4 flex-1">
                  <p style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '10px' }}>运行时</p>
                  <div className="space-y-2.5">
                    {[
                      { icon: Activity, label: 'Goroutines', value: systemStats.goroutines.toLocaleString(), color: 'var(--accent-primary)' },
                      { icon: RefreshCw, label: 'GC 次数',   value: perf.gc_cycles.toLocaleString(),         color: 'var(--color-success-fg)' },
                      { icon: Clock,     label: '运行时间',   value: perf.uptime,                             color: 'var(--accent-cosmic)' },
                    ].map(r => (
                      <div key={r.label} className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <r.icon size={13} style={{ color: r.color }} />
                          <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{r.label}</span>
                        </div>
                        <span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)', fontFamily: 'monospace' }}>{r.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </>
          ) : (
            /* 非超管：用错误率卡片占位 */
            <>
              <div className="akasha-card p-5">
                <div className="flex items-center gap-2 mb-3">
                  <AlertTriangle size={15} style={{ color: 'var(--color-danger-fg)' }} />
                  <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>近 7 日错误率</span>
                </div>
                <p style={{ fontSize: '28px', fontWeight: 800, color: errRate > 5 ? 'var(--color-danger-fg)' : 'var(--text-primary)', margin: 0, fontFamily: 'monospace' }}>
                  {errRate.toFixed(2)}%
                </p>
                <p style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '4px' }}>
                  {totalErr} 次失败 / {totalReq7d.toLocaleString()} 次调用
                </p>
              </div>
              <div className="akasha-card p-5 flex-1">
                <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>实时系统监控需超级管理员权限</p>
              </div>
            </>
          )}
        </div>
      </div>

      {/* ══ 底部三栏：模型分布 + 模型排行 + 最近调用 ══ */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {/* 模型消耗分布饼图 */}
        <div className="akasha-card p-5">
          <h3 className="text-subtitle mb-2" style={{ color: 'var(--text-primary)' }}>模型消耗分布</h3>
          {!statLoading && pieData.length === 0 ? (
            <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>暂无数据</div>
          ) : statLoading && pieData.length === 0 ? (
            <div className="skeleton-shimmer rounded-xl" style={{ height: '160px' }} />
          ) : (
            <>
              <div style={{ height: '150px' }}>
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={pieData} cx="50%" cy="50%" innerRadius={40} outerRadius={66} paddingAngle={3} dataKey="value">
                      {pieData.map((_, i) => <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />)}
                    </Pie>
                    <Tooltip
                      contentStyle={{ background: 'var(--bg-elevated)', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border-color)', fontSize: 11 }}
                      formatter={(v: number) => [`$${(v / 500000).toFixed(4)}`, '消耗']}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="space-y-1.5 mt-1">
                {pieData.map((m, i) => (
                  <div key={m.name} className="flex items-center gap-2">
                    <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: BAR_COLORS[i], flexShrink: 0 }} />
                    <span style={{ fontSize: '11px', color: 'var(--text-muted)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>{m.name}</span>
                    <span style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-primary)', flexShrink: 0 }}>
                      {totalModelQ > 0 ? ((m.value / totalModelQ) * 100).toFixed(1) : 0}%
                    </span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* 模型排行 */}
        <div className="akasha-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-subtitle" style={{ color: 'var(--text-primary)' }}>模型消耗排行</h3>
            {statLoading && <span className="text-xs animate-nebula-pulse" style={{ color: 'var(--text-muted)' }}>加载中</span>}
          </div>
          {!statLoading && modelStats.length === 0 ? (
            <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>暂无数据</div>
          ) : (
            <div className="space-y-3">
              {(statLoading && !modelStats.length ? Array.from({ length: 4 }) : modelStats).map((m, i) => {
                if (!m) return <div key={i} className="skeleton-shimmer h-7 rounded" />;
                const pct = totalModelQ > 0 ? (m.total_quota / totalModelQ) * 100 : 0;
                const color = BAR_COLORS[i] || 'var(--text-muted)';
                return (
                  <div key={m.model_name}>
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-1.5 min-w-0">
                        <span style={{ width: '16px', height: '16px', borderRadius: '50%', background: color, color: 'white', fontSize: '9px', fontWeight: 700, display: 'inline-flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>{i + 1}</span>
                        <span style={{ fontSize: '11px', fontFamily: 'monospace', color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{m.model_name}</span>
                      </div>
                      <span style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-secondary)', flexShrink: 0, marginLeft: '6px' }}>${(m.total_quota / 500000).toFixed(2)}</span>
                    </div>
                    <div style={{ height: '3px', borderRadius: 'var(--radius-full)', background: 'var(--bg-elevated)' }}>
                      <div style={{ height: '100%', width: `${pct}%`, borderRadius: 'var(--radius-full)', background: color, transition: 'width 0.6s ease' }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 最近调用 */}
        <div className="akasha-card p-5">
          <h3 className="text-subtitle mb-4" style={{ color: 'var(--text-primary)' }}>最近调用</h3>
          {!statLoading && recentLogs.length === 0 ? (
            <div style={{ padding: '40px 0', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>暂无记录</div>
          ) : (
            <div className="space-y-2">
              {(statLoading && !recentLogs.length ? Array.from({ length: 5 }) : recentLogs).map((log, i) => {
                if (!log) return <div key={i} className="skeleton-shimmer rounded-lg" style={{ height: '46px' }} />;
                return (
                  <div key={log.id} style={{ display: 'flex', alignItems: 'center', gap: '9px', padding: '7px 10px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                    <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: LOG_TYPE_COLOR[log.type] || 'var(--text-muted)', flexShrink: 0 }} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <p style={{ fontSize: '12px', fontWeight: 500, color: 'var(--text-primary)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {log.model_name || '—'}
                      </p>
                      <p style={{ fontSize: '10px', color: 'var(--text-muted)', margin: 0 }}>
                        {log.username || `#${log.id}`} · {new Date(log.created_at * 1000).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
                      </p>
                    </div>
                    <span style={{ fontSize: '11px', fontWeight: 600, color: log.type === 4 ? 'var(--color-danger-fg)' : 'var(--text-primary)', flexShrink: 0 }}>
                      ${(log.quota / 500000).toFixed(4)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* 内联渠道健康进度条 */
function Progress({ pct }: { pct: number }) {
  const color = pct >= 80 ? 'var(--color-success-fg)' : pct >= 50 ? 'var(--color-warning-fg)' : 'var(--color-danger-fg)';
  return (
    <div style={{ height: '5px', borderRadius: 'var(--radius-full)', background: 'var(--bg-elevated)', marginTop: '2px' }}>
      <div style={{ height: '100%', width: `${pct}%`, borderRadius: 'var(--radius-full)', background: color, transition: 'width 0.5s ease' }} />
    </div>
  );
}
