import { useState, useEffect } from 'react';
import { Card, CardBody, CardHeader } from '../../components/ui';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Cpu, MemoryStick, Activity, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import PageHeader from '../../components/PageHeader';
import StatCard from '../../components/StatCard';

/**
 * 系统监控数据接口
 */
interface SystemStats {
  cpu_usage: number;      // CPU 使用率 (0-100)
  memory_usage: number;   // 内存使用率 (0-100)
  memory_total: number;   // 总内存 (字节)
  memory_used: number;    // 已用内存 (字节)
  goroutines: number;     // Goroutine 数量
  timestamp: number;      // 时间戳
}

/**
 * 历史数据点接口（用于图表展示）
 */
interface HistoryPoint {
  time: string;           // 时间标签
  cpu: number;            // CPU 使用率
  memory: number;         // 内存使用率
  goroutines: number;     // Goroutine 数量
}

export default function SystemMonitor() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [history, setHistory] = useState<HistoryPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const { token } = useAuthStore();

  /**
   * 获取系统监控数据
   */
  const fetchStats = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/admin/system/monitor', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data) {
        setStats(data.data);

        // 添加到历史记录（保留最近20个数据点）
        const now = new Date();
        const timeLabel = `${now.getHours()}:${now.getMinutes().toString().padStart(2, '0')}`;
        setHistory(prev => {
          const newHistory = [...prev, {
            time: timeLabel,
            cpu: data.data.cpu_usage,
            memory: data.data.memory_usage,
            goroutines: data.data.goroutines,
          }];
          return newHistory.slice(-20); // 只保留最近20个点
        });
      }
    } catch (error) {
      console.error('获取系统监控数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 自动刷新：每5秒获取一次数据
  useEffect(() => {
    if (token) {
      fetchStats();
      const interval = setInterval(fetchStats, 5000);
      return () => clearInterval(interval);
    }
  }, [token]);

  /**
   * 格式化字节数为可读格式
   */
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="系统监控"
        description="实时监控系统资源使用情况"
        actions={
          <button
            onClick={fetchStats}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </button>
        }
      />

      {/* 实时统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="CPU 使用率"
          value={`${stats?.cpu_usage.toFixed(1) || 0}%`}
          icon={Cpu}
          trend={stats?.cpu_usage && stats.cpu_usage > 80 ? 'up' : undefined}
          trendValue={stats?.cpu_usage && stats.cpu_usage > 80 ? '高负载' : '正常'}
        />
        <StatCard
          title="内存使用"
          value={stats ? formatBytes(stats.memory_used) : '0 B'}
          subtitle={stats ? `总计 ${formatBytes(stats.memory_total)}` : ''}
          icon={MemoryStick}
          trend={stats?.memory_usage && stats.memory_usage > 80 ? 'up' : undefined}
          trendValue={`${stats?.memory_usage.toFixed(1) || 0}%`}
        />
        <StatCard
          title="Goroutines"
          value={stats?.goroutines.toString() || '0'}
          icon={Activity}
          subtitle="活跃协程数"
        />
      </div>

      {/* CPU 使用率趋势图 */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">CPU 使用率趋势</h3>
        </CardHeader>
        <CardBody>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={history}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={[0, 100]} />
              <Tooltip />
              <Line type="monotone" dataKey="cpu" stroke="#3b82f6" name="CPU %" />
            </LineChart>
          </ResponsiveContainer>
        </CardBody>
      </Card>

      {/* 内存使用率趋势图 */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">内存使用率趋势</h3>
        </CardHeader>
        <CardBody>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={history}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis domain={[0, 100]} />
              <Tooltip />
              <Line type="monotone" dataKey="memory" stroke="#10b981" name="内存 %" />
            </LineChart>
          </ResponsiveContainer>
        </CardBody>
      </Card>

      {/* Goroutines 趋势图 */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">Goroutines 数量趋势</h3>
        </CardHeader>
        <CardBody>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={history}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="goroutines" stroke="#f59e0b" name="Goroutines" />
            </LineChart>
          </ResponsiveContainer>
        </CardBody>
      </Card>
    </div>
  );
}
