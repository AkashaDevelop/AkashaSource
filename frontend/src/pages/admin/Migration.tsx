import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import { Card, CardBody, Button, Chip } from '../../components/ui';
import { Play, RotateCcw, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Migration {
  id: number;
  version: string;
  description: string;
  applied_at: number;
}

export default function MigrationPage() {
  const [migrations, setMigrations] = useState<Migration[]>([]);
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [rolling, setRolling] = useState(false);
  const { token } = useAuthStore();

  const fetchMigrations = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/migration/sql', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.data) setMigrations(data.data);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchMigrations(); }, []);

  const handleApply = async () => {
    setApplying(true);
    try {
      const res = await fetch('/api/migration/sql/apply', { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.message) { toast.success(data.message); fetchMigrations(); }
      else toast.error(data.error || '执行失败');
    } finally { setApplying(false); }
  };

  const handleRollback = async () => {
    setRolling(true);
    try {
      const res = await fetch('/api/migration/sql/rollback?steps=1', { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.message) { toast.success(data.message); fetchMigrations(); }
      else toast.error(data.error || '回滚失败');
    } finally { setRolling(false); }
  };

  return (
    <div className="space-y-5 max-w-4xl mx-auto pb-10">
      <PageHeader
        title="数据库迁移"
        description="管理数据库结构迁移脚本"
        actions={
          <div className="flex gap-2">
            <Button variant="flat" startContent={<RefreshCw size={16} />} onPress={fetchMigrations} isLoading={loading}>刷新</Button>
            <Button variant="flat" color="danger" startContent={<RotateCcw size={16} />} onPress={handleRollback} isLoading={rolling}>回滚 1 步</Button>
            <Button color="primary" startContent={<Play size={16} />} onPress={handleApply} isLoading={applying}>执行迁移</Button>
          </div>
        }
      />
      <Card>
        <CardBody style={{ padding: 0 }}>
          <div className="data-table-wrap" style={{ borderRadius: 0, border: 'none', boxShadow: 'none' }}>
            <table className="data-table">
              <thead><tr><th>版本</th><th>描述</th><th>状态</th><th>执行时间</th></tr></thead>
              <tbody>
                {migrations.length === 0 ? (
                  <tr><td colSpan={4} className="text-center py-8 text-default-400">暂无迁移记录</td></tr>
                ) : migrations.map(m => (
                  <tr key={m.id}>
                    <td className="font-mono text-sm">{m.version}</td>
                    <td>{m.description || '-'}</td>
                    <td><Chip size="sm" color="success" variant="flat">已执行</Chip></td>
                    <td className="text-sm text-default-400">{m.applied_at ? new Date(m.applied_at * 1000).toLocaleString() : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
