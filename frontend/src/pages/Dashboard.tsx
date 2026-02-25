import { useState, useEffect } from 'react';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Progress,
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
import { Activity, CreditCard, Key, Server, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useNavigate } from 'react-router-dom';

interface UserStats {
  token_count: number;
  request_count: number;
}

interface UserInfo {
  username: string;
  quota: number;
  used_quota: number;
  role: number;
}

interface ChartData {
  date: string;
  usage: number;
}

export default function Dashboard() {
  const { user, token } = useAuthStore();
  const navigate = useNavigate();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchDashboard = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/dashboard', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setStats(data.stats);
        setUserInfo(data.user);
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
    <div className="p-6 space-y-6 max-w-[1400px] mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">仪表盘</h1>
          <p className="text-default-500">欢迎回来, {userInfo?.username || user?.username}</p>
        </div>
        <div className="flex gap-2">
            <Button isIconOnly variant="flat" onPress={fetchDashboard} isLoading={loading}>
                <RefreshCw size={20} />
            </Button>
            <Button color="primary" variant="shadow" onPress={() => navigate('/topup')}>
            充值余额
            </Button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">总调用次数</span>
              <span className="text-2xl font-bold">{stats?.request_count || 0}</span>
            </div>
            <div className="p-2 bg-primary/10 rounded-lg text-primary">
              <Activity size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
            <span className="text-default-400">历史累计</span>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">剩余额度</span>
              <span className="text-2xl font-bold text-success">
                ${userInfo?.quota ? (userInfo.quota - (userInfo.used_quota || 0)).toFixed(2) : '0.00'}
              </span>
            </div>
            <div className="p-2 bg-success/10 rounded-lg text-success">
              <CreditCard size={24} />
            </div>
          </div>
          <Progress 
            size="sm" 
            value={userInfo?.quota ? ((userInfo.quota - (userInfo.used_quota || 0)) / userInfo.quota) * 100 : 0} 
            color="success" 
            className="mt-4"
          />
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">活跃令牌</span>
              <span className="text-2xl font-bold">{stats?.token_count || 0}</span>
            </div>
            <div className="p-2 bg-warning/10 rounded-lg text-warning">
              <Key size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
            <Button size="sm" variant="light" onPress={() => navigate('/token')} className="px-0 h-6">管理令牌</Button>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">已用额度</span>
              <span className="text-2xl font-bold text-danger">
                 ${(userInfo?.used_quota || 0).toFixed(2)}
              </span>
            </div>
            <div className="p-2 bg-danger/10 rounded-lg text-danger">
              <Server size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
             <span className="text-default-400">历史累计消耗</span>
          </div>
        </Card>
      </div>

      {/* 图表区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="p-4 lg:col-span-2">
          <CardHeader>
            <h3 className="text-lg font-semibold">近7日消耗统计 ($)</h3>
          </CardHeader>
          <CardBody className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorUsage" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#17C964" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#17C964" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" opacity={0.1} />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip 
                  cursor={{fill: 'transparent'}}
                  contentStyle={{ 
                    backgroundColor: 'var(--heroui-content1)', 
                    borderRadius: '12px',
                    border: 'none',
                  }} 
                />
                <Area 
                  type="monotone"
                  dataKey="usage" 
                  stroke="#17C964" 
                  fillOpacity={1}
                  fill="url(#colorUsage)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
