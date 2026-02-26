import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Progress,
  Button,
} from '@heroui/react';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts';
import { Server, Activity, DollarSign, Users, RefreshCw, Cpu, MemoryStick, Clock } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

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
      if (res.ok) {
        setStats(data.stats);
        setChartData(data.chart || []);
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
      if (res.ok) setPerf(data);
    } catch (e) { console.error(e); }
  };

  useEffect(() => {
    if (token) {
      fetchDashboard();
      fetchPerformance();
    }
  }, [token]);

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">系统概览</h1>
          <p className="text-default-500">监控系统运行状态与核心指标</p>
        </div>
        <Button 
            isIconOnly 
            variant="flat" 
            onPress={fetchDashboard} 
            isLoading={loading}
        >
            <RefreshCw size={20} />
        </Button>
      </div>

      {/* 核心指标 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="p-4 bg-primary text-primary-foreground">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small opacity-80">总请求数</span>
              <span className="text-2xl font-bold">{stats?.request_count || 0}</span>
            </div>
            <Activity className="opacity-80" />
          </div>
          <div className="mt-4 text-small opacity-80">
            总调用次数
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">总消耗金额</span>
              <span className="text-2xl font-bold text-success">
                ${(stats?.total_quota_used || 0).toFixed(2)}
              </span>
            </div>
            <div className="p-2 bg-success/10 rounded-lg text-success">
              <DollarSign size={24} />
            </div>
          </div>
          <div className="mt-4 text-small text-default-400">
            平台累计消耗
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">渠道状态</span>
              <span className="text-2xl font-bold">
                {stats?.active_channels || 0}/{stats?.channel_count || 0}
              </span>
            </div>
            <div className="p-2 bg-warning/10 rounded-lg text-warning">
              <Server size={24} />
            </div>
          </div>
          <Progress 
            size="sm" 
            value={stats?.channel_count ? (stats.active_channels / stats.channel_count) * 100 : 0} 
            color="warning" 
            className="mt-4"
          />
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">总用户数</span>
              <span className="text-2xl font-bold text-secondary">
                {stats?.user_count || 0}
              </span>
            </div>
            <div className="p-2 bg-secondary/10 rounded-lg text-secondary">
              <Users size={24} />
            </div>
          </div>
          <div className="mt-4 text-small text-secondary">
             Active Users
          </div>
        </Card>
      </div>

      {/* 流量趋势图 */}
      <Card className="p-4">
        <CardHeader>
          <h3 className="text-lg font-semibold">近7日调用趋势</h3>
        </CardHeader>
        <CardBody className="h-[350px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#006FEE" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="#006FEE" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" opacity={0.1} />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip 
                contentStyle={{ 
                  backgroundColor: 'var(--heroui-content1)', 
                  borderRadius: '12px',
                  border: 'none',
                }} 
              />
              <Area 
                type="monotone" 
                dataKey="count" 
                stroke="#006FEE" 
                fillOpacity={1} 
                fill="url(#colorRequests)" 
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardBody>
      </Card>

      {/* 系统性能 */}
      {perf && (
        <Card className="p-4">
          <CardHeader>
            <h3 className="text-lg font-semibold">系统性能</h3>
          </CardHeader>
          <CardBody>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-primary/10 rounded-lg text-primary"><Cpu size={20} /></div>
                <div>
                  <p className="text-xs text-default-500">Goroutines</p>
                  <p className="font-bold">{perf.goroutines}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-warning/10 rounded-lg text-warning"><MemoryStick size={20} /></div>
                <div>
                  <p className="text-xs text-default-500">内存使用</p>
                  <p className="font-bold">{perf.memory_mb.toFixed(1)} MB</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-success/10 rounded-lg text-success"><RefreshCw size={20} /></div>
                <div>
                  <p className="text-xs text-default-500">GC 次数</p>
                  <p className="font-bold">{perf.gc_cycles}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-secondary/10 rounded-lg text-secondary"><Clock size={20} /></div>
                <div>
                  <p className="text-xs text-default-500">运行时间</p>
                  <p className="font-bold">{perf.uptime}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-default/10 rounded-lg"><Server size={20} /></div>
                <div>
                  <p className="text-xs text-default-500">Go 版本</p>
                  <p className="font-bold text-sm">{perf.go_version}</p>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
