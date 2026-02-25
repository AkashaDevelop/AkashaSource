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
import { Server, Activity, DollarSign, Users, RefreshCw } from 'lucide-react';
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

export default function AdminDashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
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

  useEffect(() => {
    if (token) fetchDashboard();
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
    </div>
  );
}
