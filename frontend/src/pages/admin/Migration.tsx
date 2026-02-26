import { useEffect, useState } from 'react';
import {
  Card,
  CardHeader,
  CardBody,
  Button,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Input,
  Alert,
} from '@heroui/react';
import { RefreshCw, Play, RotateCcw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface SQLMigration {
  id: number;
  version: number;
  name: string;
  applied_at: number;
}

export default function MigrationPage() {
  const { token } = useAuthStore();
  const [migrations, setMigrations] = useState<SQLMigration[]>([]);
  const [loading, setLoading] = useState(false);
  const [steps, setSteps] = useState('1');
  const [message, setMessage] = useState<{ type: 'success' | 'danger'; text: string } | null>(null);

  const fetchMigrations = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/migration/sql', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setMigrations(data.data || []);
      }
    } catch (error) {
      console.error('获取迁移记录失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchMigrations();
  }, [token]);

  const handleApply = async () => {
    setMessage(null);
    const res = await fetch('/api/migration/sql/apply', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (res.ok) {
      setMessage({ type: 'success', text: data.message || '迁移执行完成' });
      fetchMigrations();
    } else {
      setMessage({ type: 'danger', text: data.error || '迁移执行失败' });
    }
  };

  const handleRollback = async () => {
    setMessage(null);
    const count = parseInt(steps, 10) || 1;
    const res = await fetch(`/api/migration/sql/rollback?steps=${count}`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (res.ok) {
      setMessage({ type: 'success', text: data.message || '回滚执行完成' });
      fetchMigrations();
    } else {
      setMessage({ type: 'danger', text: data.error || '回滚执行失败' });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">SQL 迁移管理</h1>
          <p className="text-default-500">查看与执行 SQL 迁移文件</p>
        </div>
        <div className="flex gap-2">
          <Button isIconOnly variant="flat" onPress={fetchMigrations} isLoading={loading}>
            <RefreshCw size={18} />
          </Button>
          <Button color="primary" startContent={<Play size={18} />} onPress={handleApply}>
            执行迁移
          </Button>
        </div>
      </div>

      {message && (
        <Alert color={message.type}>
          {message.text}
        </Alert>
      )}

      <Card>
        <CardHeader className="flex justify-between items-center">
          <span className="font-semibold">迁移记录</span>
          <div className="flex items-center gap-2">
            <Input
              label="回滚步数"
              type="number"
              value={steps}
              onValueChange={setSteps}
              className="w-32"
            />
            <Button color="warning" variant="flat" startContent={<RotateCcw size={16} />} onPress={handleRollback}>
              回滚
            </Button>
          </div>
        </CardHeader>
        <CardBody>
          <Table aria-label="迁移记录表格">
            <TableHeader>
              <TableColumn>版本</TableColumn>
              <TableColumn>名称</TableColumn>
              <TableColumn>执行时间</TableColumn>
            </TableHeader>
            <TableBody emptyContent="暂无记录">
              {migrations.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.version}</TableCell>
                  <TableCell>{item.name}</TableCell>
                  <TableCell>{new Date(item.applied_at * 1000).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardBody>
      </Card>
    </div>
  );
}
