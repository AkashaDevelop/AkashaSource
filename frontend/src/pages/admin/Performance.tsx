import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Divider,
  Chip,
} from '@heroui/react';
import { Cpu, HardDrive, RefreshCw, Clock, Layers, Recycle } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface PerfData {
  goroutines: number;
  memory_alloc: number;
  memory_sys: number;
  gc_cycles: number;
  uptime: number;
  go_version: string;
}

export default function Performance() {
  const { token } = useAuthStore();
  const [data, setData] = useState<PerfData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchPerf = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/performance', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const json = await res.json();
      if (res.ok) setData(json);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchPerf(); }, []);

  const formatUptime = (seconds: number) => {
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const parts = [];
    if (d > 0) parts.push(`${d}天`);
    if (h > 0) parts.push(`${h}小时`);
    parts.push(`${m}分钟`);
    return parts.join(' ');
  };

  const metrics = data ? [
    { label: '运行时间', value: formatUptime(data.uptime), icon: Clock, color: 'primary' },
    { label: 'Goroutines', value: data.goroutines.toString(), icon: Layers, color: 'secondary' },
    { label: '已分配内存', value: `${data.memory_alloc} MB`, icon: Cpu, color: 'success' },
    { label: '系统内存', value: `${data.memory_sys} MB`, icon: HardDrive, color: 'warning' },
    { label: 'GC 次数', value: data.gc_cycles.toString(), icon: Recycle, color: 'danger' },
  ] : [];

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">系统性能</h1>
          <p className="text-default-500">监控后端服务运行状态</p>
        </div>
        <Button startContent={<RefreshCw size={18} />} onPress={fetchPerf} variant="flat" isLoading={loading}>
          刷新
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {metrics.map((m) => (
          <Card key={m.label} className="p-4">
            <div className="flex justify-between items-start">
              <div className="flex flex-col gap-1">
                <span className="text-small text-default-500">{m.label}</span>
                <span className="text-2xl font-bold">{m.value}</span>
              </div>
              <div className={`p-2 bg-${m.color}/10 rounded-lg text-${m.color}`}>
                <m.icon size={24} />
              </div>
            </div>
          </Card>
        ))}
      </div>

      {data && (
        <Card>
          <CardHeader><h3 className="font-semibold">运行环境</h3></CardHeader>
          <Divider />
          <CardBody>
            <div className="flex gap-2">
              <Chip variant="flat" color="primary">{data.go_version}</Chip>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}
