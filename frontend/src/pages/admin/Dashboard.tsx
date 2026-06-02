import { useState, useEffect } from 'react';
import { Card, CardBody, CardHeader, Progress, Button } from '../../components/ui';
import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { Server, Activity, DollarSign, Users, RefreshCw, Cpu, MemoryStick, Clock } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import PageHeader from '../../components/PageHeader';
import StatCard from '../../components/StatCard';
import { SkeletonCard } from '../../components/SkeletonCard';

interface DashboardStats {
  user_count: number;
  channel_count: number;
  request_count: number;
  total_quota_used: number;
  active_channels: number;
}

interface ChartData {
  date: string;
  count: number;
}

interface PerformanceData {
  goroutines: number;
  memory_mb: number;
  gc_cycles: number;
  uptime: string;
  go_version: string;
}

interface SystemStats {
  cpu_usage: number;
  memory_usage: number;
  memory_total: number;
  memory_used: number;
  goroutines: number;
  timestamp: number;
}

function SkeletonChart({ height = 350 }: { height?: number }) {
  return (
    <div className="akasha-card p-5" style={{ height }}>
      <div className="skeleton-shimmer h-5 w-36 rounded mb-4" />
      <div className="skeleton-shimmer rounded" style={{ height: height - 76 }} />
    </div>
  );
}

export default function AdminDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [perf, setPerf] = useState<PerformanceData | null>(null);
  const [systemStats, setSystemStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(false);
  const { token, user } = useAuthStore();

  const fetchDashboard = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/dashboard', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setStats(data.data.stats);
        setChartData(data.data.chart || []);
      }
    } catch (error) {
      console.error('Failed to fetch dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchPerformance = async () => {
    try {
      const res = await fetch('/api/performance', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) setPerf(data.data);
    } catch (e) { console.error(e); }
  };

  const fetchSystemStats = async () => {
    try {
      const res = await fetch('/api/admin/system/monitor', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data) {
        setSystemStats(data.data);
      }
    } catch (e) { console.error(e); }
  };

  useEffect(() => {
    if (token) {
      fetchDashboard();
      if ((user?.role ?? 0) >= 100) {
        fetchPerformance();
        fetchSystemStats();
        const interval = setInterval(fetchSystemStats, 5000);
        return () => clearInterval(interval);
      } else {
        setPerf(null);
      }
    }
  }, [token, user?.role]);

  const channelHealthPct = stats?.channel_count ? (stats.active_channels / stats.channel_count) * 100 : 0;

  return (
    <div className="space-y-5">
      <PageHeader
        title="系统概览"
        description={perf ? `监控系统运行状态与核心指标 · Go ${perf.go_version} · 已运行 ${perf.uptime}` : '监控系统运行状态与核心指标'}
        actions={
          <Button isIconOnly variant="flat" onPress={fetchDashboard} isLoading={loading}
            style={{ background: 'var(--bg-elevated)', color: 'var(--accent-primary)', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border-color)' }}>
            <RefreshCw size={18} />
          </Button>
        }
      />

      {/* 核心指标 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {loading && !stats ? (
          Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} lines={1} />)
        ) : (
          <>
            <StatCard
              title="总请求数"
              value={stats?.request_count?.toLocaleString() ?? 0}
              icon={<Activity size={22} style={{ color: 'var(--accent-primary)' }} />}
              iconBg="var(--color-info-bg)"
              footer="平台累计调用"
            />
            <StatCard
              title="总消耗金额"
              value={<span style={{ color: 'var(--accent-cosmic)' }}>${(stats?.total_quota_used || 0).toFixed(2)}</span>}
              icon={<DollarSign size={22} style={{ color: 'var(--accent-cosmic)' }} />}
              iconBg="var(--color-info-bg)"
              footer="平台累计消耗"
            />
            <StatCard
              title="渠道状态"
              value={`${stats?.active_channels ?? 0} / ${stats?.channel_count ?? 0}`}
              icon={<Server size={22} style={{ color: 'var(--accent-star)' }} />}
              iconBg="var(--color-warning-bg)"
              footer={
                <Progress aria-label="渠道健康度" size="sm" value={channelHealthPct}
                  color={channelHealthPct >= 80 ? 'success' : channelHealthPct >= 50 ? 'warning' : 'danger'} />
              }
            />
            <StatCard
              title="总用户数"
              value={<span style={{ color: 'var(--accent-cosmic)' }}>{stats?.user_count?.toLocaleString() ?? 0}</span>}
              icon={<Users size={22} style={{ color: 'var(--accent-cosmic)' }} />}
              iconBg="var(--color-info-bg)"
              footer="注册用户总数"
            />
          </>
        )}
      </div>

      {/* 流量趋势图 */}
      {loading && !chartData.length ? (
        <SkeletonChart height={380} />
      ) : (
        <Card style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-xl)' }}>
          <CardHeader>
            <h3 className="text-subtitle" style={{ color: 'var(--text-primary)' }}>近 7 日调用趋势</h3>
          </CardHeader>
          <CardBody className="h-[320px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--accent-primary)" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="var(--accent-primary)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                <XAxis dataKey="date" tick={{ fill: 'var(--text-muted)', fontSize: 12 }} />
                <YAxis tick={{ fill: 'var(--text-muted)', fontSize: 12 }} />
                <Tooltip contentStyle={{
                  backgroundColor: 'var(--bg-elevated)',
                  borderRadius: 'var(--radius-lg)',
                  border: '1px solid var(--border-color)',
                  fontSize: 13,
                }} />
                <Area type="monotone" dataKey="count" stroke="var(--accent-primary)" strokeWidth={2} fillOpacity={1} fill="url(#colorRequests)" />
              </AreaChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>
      )}

      {/* 系统性能 */}
      {perf && (
        <Card style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-xl)' }}>
          <CardHeader>
            <h3 className="text-subtitle" style={{ color: 'var(--text-primary)' }}>运行时性能</h3>
          </CardHeader>
          <CardBody>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg flex-shrink-0" style={{ background: 'var(--color-info-bg)', color: 'var(--accent-primary)' }}>
                  <Cpu size={18} />
                </div>
                <div>
                  <p className="text-label" style={{ color: 'var(--text-muted)' }}>Goroutines</p>
                  <p className="font-bold text-sm" style={{ color: 'var(--text-primary)' }}>{perf.goroutines}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg flex-shrink-0" style={{ background: 'var(--color-warning-bg)', color: 'var(--color-warning-fg)' }}>
                  <MemoryStick size={18} />
                </div>
                <div>
                  <p className="text-label" style={{ color: 'var(--text-muted)' }}>内存使用</p>
                  <p className="font-bold text-sm" style={{ color: 'var(--text-primary)' }}>{(perf.memory_mb ?? 0).toFixed(1)} MB</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg flex-shrink-0" style={{ background: 'var(--color-success-bg)', color: 'var(--color-success-fg)' }}>
                  <RefreshCw size={18} />
                </div>
                <div>
                  <p className="text-label" style={{ color: 'var(--text-muted)' }}>GC 次数</p>
                  <p className="font-bold text-sm" style={{ color: 'var(--text-primary)' }}>{perf.gc_cycles}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg flex-shrink-0" style={{ background: 'var(--color-info-bg)', color: 'var(--accent-cosmic)' }}>
                  <Clock size={18} />
                </div>
                <div>
                  <p className="text-label" style={{ color: 'var(--text-muted)' }}>运行时间</p>
                  <p className="font-bold text-sm" style={{ color: 'var(--text-primary)' }}>{perf.uptime}</p>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      )}

      {/* 实时系统监控 */}
      {systemStats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <StatCard
            title="CPU 使用率"
            value={`${systemStats.cpu_usage.toFixed(1)}%`}
            icon={<Cpu size={22} style={{ color: 'var(--accent-primary)' }} />}
            iconBg="var(--color-info-bg)"
            footer={
              <Progress size="sm" value={systemStats.cpu_usage} aria-label="CPU 使用率"
                color={systemStats.cpu_usage > 80 ? 'danger' : systemStats.cpu_usage > 60 ? 'warning' : 'success'} />
            }
          />
          <StatCard
            title="内存使用率"
            value={`${systemStats.memory_usage.toFixed(1)}%`}
            icon={<MemoryStick size={22} style={{ color: 'var(--color-warning-fg)' }} />}
            iconBg="var(--color-warning-bg)"
            footer={
              <Progress size="sm" value={systemStats.memory_usage} aria-label="内存使用率"
                color={systemStats.memory_usage > 80 ? 'danger' : systemStats.memory_usage > 60 ? 'warning' : 'success'} />
            }
          />
          <StatCard
            title="活跃协程"
            value={systemStats.goroutines.toLocaleString()}
            icon={<Activity size={22} style={{ color: 'var(--color-success-fg)' }} />}
            iconBg="var(--color-success-bg)"
            footer={`实时 Goroutines 数量`}
          />
        </div>
      )}
    </div>
  );
}
