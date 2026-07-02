import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Card, CardBody, Button, Input, Select, SelectItem, Pagination, Chip, Tooltip,
} from '../../components/ui';
import { RefreshCw, Filter } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface AuditLog {
  id: number;
  admin_id: number;
  admin_username: string;
  method: string;
  path: string;
  target_type: string;
  status_code: number;
  ip: string;
  request_id: string;
  request_body: string;
  created_at: number;
}

const METHOD_COLOR: Record<string, string> = {
  POST: 'success', PUT: 'warning', PATCH: 'warning', DELETE: 'danger',
};

// ～这个页面专门给超管大人翻看管理员们的操作足迹哦～
export default function AdminAuditLog() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [showFilters, setShowFilters] = useState(false);
  const { token } = useAuthStore();
  const [filters, setFilters] = useState({
    admin_username: '', method: '', path: '', target_type: '', ip: '', start_time: '', end_time: '',
  });

  const setFilter = (key: string, val: string) => setFilters(prev => ({ ...prev, [key]: val }));

  const buildParams = () => {
    const p = new URLSearchParams();
    p.set('page', page.toString());
    p.set('size', pageSize.toString());
    Object.entries(filters).forEach(([k, v]) => { if (v) p.set(k, v); });
    return p;
  };

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/admin/audit-log?${buildParams()}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) { setLogs(data.data.data || []); setTotal(data.data.total || 0); }
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { if (token) fetchLogs(); }, [page, pageSize, token]);

  const handleReset = () => {
    setFilters({ admin_username: '', method: '', path: '', target_type: '', ip: '', start_time: '', end_time: '' });
    setPage(1);
    setTimeout(fetchLogs, 0);
  };

  const activeFilterCount = Object.values(filters).filter(Boolean).length;

  return (
    <div className="space-y-5">
      <PageHeader
        title="操作审计"
        description="查看管理员对系统资源的写操作记录，仅超级管理员可见"
        actions={
          <div className="flex gap-2 flex-wrap items-center">
            <Button
              size="sm"
              variant={showFilters ? 'solid' : 'flat'}
              color={activeFilterCount > 0 ? 'primary' : 'default'}
              startContent={<Filter size={14} />}
              onPress={() => setShowFilters(v => !v)}
            >
              筛选{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
            </Button>
            <Button size="sm" variant="flat" startContent={<RefreshCw size={14} />} onPress={fetchLogs} isLoading={loading}>
              刷新
            </Button>
          </div>
        }
      />

      {showFilters && (
        <Card>
          <CardBody className="p-4">
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
              <Input label="管理员用户名" size="sm" value={filters.admin_username} onValueChange={v => setFilter('admin_username', v)} />
              <Select label="方法" size="sm" selectedKeys={filters.method ? [filters.method] : []}
                onSelectionChange={keys => setFilter('method', [...keys][0] as string || '')}>
                {['POST', 'PUT', 'PATCH', 'DELETE'].map(m => <SelectItem key={m}>{m}</SelectItem>)}
              </Select>
              <Input label="路径关键字" size="sm" value={filters.path} onValueChange={v => setFilter('path', v)} placeholder="例如 /api/user" />
              <Input label="目标类型" size="sm" value={filters.target_type} onValueChange={v => setFilter('target_type', v)} placeholder="例如 user / channel" />
              <Input label="IP" size="sm" value={filters.ip} onValueChange={v => setFilter('ip', v)} />
              <Input label="开始日期" size="sm" type="date" value={filters.start_time} onValueChange={v => setFilter('start_time', v)} />
              <Input label="结束日期" size="sm" type="date" value={filters.end_time} onValueChange={v => setFilter('end_time', v)} />
            </div>
            <div className="flex justify-end gap-2 mt-3">
              <Button size="sm" variant="light" onPress={handleReset}>重置</Button>
              <Button size="sm" color="primary" onPress={() => { setPage(1); fetchLogs(); }}>应用筛选</Button>
            </div>
          </CardBody>
        </Card>
      )}

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>管理员</th>
              <th>方法</th>
              <th>路径</th>
              <th>目标类型</th>
              <th>状态码</th>
              <th>IP</th>
              <th>详情</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={8} rows={8} />
            ) : logs.length === 0 ? (
              <tr><td colSpan={8}>
                <EmptyState icon="🛡️" title="暂无审计记录" description={activeFilterCount > 0 ? '当前筛选条件下没有记录' : '还没有管理员写操作记录'} />
              </td></tr>
            ) : logs.map(item => (
              <tr key={item.id}>
                <td style={{ whiteSpace: 'nowrap', fontSize: '12px' }}>{new Date(item.created_at * 1000).toLocaleString()}</td>
                <td style={{ fontSize: '12px' }}>{item.admin_username || item.admin_id || '-'}</td>
                <td><Chip size="sm" variant="flat" color={(METHOD_COLOR[item.method] as any) || 'default'}>{item.method}</Chip></td>
                <td style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'monospace' }}>{item.path}</td>
                <td>{item.target_type ? <Chip size="sm" variant="dot">{item.target_type}</Chip> : '-'}</td>
                <td style={{ fontSize: '12px', color: item.status_code >= 400 ? '#f87171' : 'var(--text-secondary)' }}>{item.status_code}</td>
                <td style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'monospace' }}>{item.ip || '-'}</td>
                <td style={{ maxWidth: '220px' }}>
                  <Tooltip content={item.request_body || '无请求体'}>
                    <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '12px', color: 'var(--text-muted)', cursor: 'help' }}>
                      {item.request_body || '-'}
                    </span>
                  </Tooltip>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {total > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-3 mt-1">
          <span className="text-sm" style={{ color: 'var(--text-muted)' }}>
            共 <strong style={{ color: 'var(--text-primary)' }}>{total}</strong> 条，
            第 {page}/{Math.ceil(total / pageSize)} 页
          </span>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5">
              <span className="text-xs" style={{ color: 'var(--text-muted)' }}>每页</span>
              <select
                className="akasha-select"
                value={pageSize}
                onChange={e => { setPageSize(Number(e.target.value)); setPage(1); }}
              >
                {[20, 50, 100].map(n => <option key={n} value={n}>{n} 条</option>)}
              </select>
            </div>
            <Pagination page={page} total={Math.ceil(total / pageSize)} onChange={setPage} />
          </div>
        </div>
      )}
    </div>
  );
}
