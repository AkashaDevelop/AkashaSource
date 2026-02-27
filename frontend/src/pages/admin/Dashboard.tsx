import { useState, useEffect } from 'react';
import { Card, CardBody, CardHeader, Progress, Button } from '../../components/ui';
import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { Server, Activity, DollarSign, Users, RefreshCw, Cpu, MemoryStick, Clock } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import PageHeader from '../../components/PageHeader';
import StatCard from '../../components/StatCard';

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

export default function AdminDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [perf, setPerf] = useState<PerformanceData | null>(null);
  const [loading, setLoading] = useState(false);
  const { token } = useAuthStore();

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

  useEffect(() => {
    if (token) {
      fetchDashboard();
      fetchPerformance();
    }
  }, [token]);

  const channelHealthPct = stats?.channel_count ? (stats.active_channels / stats.channel_count) * 100 : 0;

  return (
    <div className="space-y-6">
      <PageHeader
        title="系统概览"
        description="监控系统运行状态与核心指标"
        actions={
          <Button isIconOnly variant="flat" onPress={fetchDashboard} isLoading={loading}
            style={{ background: 'var(--bg-elevated)', color: 'var(--accent-primary)', borderRadius: '10px', border: '1px solid var(--border-color)' }}>
            <RefreshCw size={20} />
          </Button>
        }
      />

      {/* 核心指标 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCard
          title="总请求数"
          value={stats?.request_count || 0}
          icon={<Activity size={24} style={{ color: 'var(--accent-primary)' }} />}
          iconBg="rgba(124,58,237,0.15)"
          footer="总调用次数"
        />
        <StatCard
          title="总消耗金额"
          value={<span style={{ color: 'var(--accent-cosmic)' }}>${(stats?.total_quota_used || 0).toFixed(2)}</span>}
          icon={<DollarSign size={24} style={{ color: 'var(--accent-cosmic)' }} />}
          iconBg="rgba(8,145,178,0.12)"
          footer="平台累计消耗"
        />
        <StatCard
          title="渠道状态"
          value={`${stats?.active_channels || 0}/${stats?.channel_count || 0}`}
          icon={<Server size={24} style={{ color: 'var(--accent-star)' }} />}
          iconBg="rgba(217,119,6,0.12)"
          footer={
            <Progress aria-label="渠道健康度" size="sm" value={channelHealthPct} color="warning" />
          }
        />
        <StatCard
          title="总用户数"
          value={<span style={{ color: 'var(--accent-cosmic)' }}>{stats?.user_count || 0}</span>}
          icon={<Users size={24} style={{ color: 'var(--accent-cosmic)' }} />}
          iconBg="rgba(8,145,178,0.12)"
          footer="Active Users"
        />
      </div>

      {/* 流量趋势图 */}
      <Card style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
        <CardHeader>
          <h3 className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>近7日调用趋势</h3>
        </CardHeader>
        <CardBody className="h-[350px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--accent-primary)" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="var(--accent-primary)" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis dataKey="date" tick={{ fill: 'var(--text-secondary)' }} />
              <YAxis tick={{ fill: 'var(--text-secondary)' }} />
              <Tooltip contentStyle={{ backgroundColor: 'var(--bg-surface)', borderRadius: '12px', border: '1px solid var(--border-color)' }} />
              <Area type="monotone" dataKey="count" stroke="var(--accent-primary)" fillOpacity={1} fill="url(#colorRequests)" />
            </AreaChart>
          </ResponsiveContainer>
        </CardBody>
      </Card>

      {/* 系统性能 */}
      {perf && (
        <Card style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
          <CardHeader>
            <h3 className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>系统性能</h3>
          </CardHeader>
          <CardBody>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'rgba(124,58,237,0.12)', color: 'var(--accent-primary)' }}><Cpu size={20} /></div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Goroutines</p>
                  <p className="font-bold" style={{ color: 'var(--text-primary)' }}>{perf.goroutines}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'rgba(217,119,6,0.12)', color: 'var(--accent-star)' }}><MemoryStick size={20} /></div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>内存使用</p>
                  <p className="font-bold" style={{ color: 'var(--text-primary)' }}>{(perf.memory_mb ?? 0).toFixed(1)} MB</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'rgba(16,185,129,0.12)', color: '#34d399' }}><RefreshCw size={20} /></div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>GC 次数</p>
                  <p className="font-bold" style={{ color: 'var(--text-primary)' }}>{perf.gc_cycles}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'rgba(8,145,178,0.12)', color: 'var(--accent-cosmic)' }}><Clock size={20} /></div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>运行时间</p>
                  <p className="font-bold" style={{ color: 'var(--text-primary)' }}>{perf.uptime}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'var(--bg-elevated)' }}><Server size={20} style={{ color: 'var(--text-secondary)' }} /></div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>Go 版本</p>
                  <p className="font-bold text-sm" style={{ color: 'var(--text-primary)' }}>{perf.go_version}</p>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
