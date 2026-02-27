import { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import LoadingRows from '../components/LoadingRows';
import {
  Pagination, Chip, Button, Input, Select, SelectItem, Card, CardBody,
} from '../components/ui';
import { RefreshCw, Trash2, Download, Filter } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { toast } from '../store/toast';
import { confirm } from '../store/confirm';

interface Log {
  id: number;
  created_at: number;
  type: number;
  token_name: string;
  model_name: string;
  quota: number;
  prompt_tokens: number;
  completion_tokens: number;
  content: string;
  user_id: number;
  username: string;
  channel_id: number;
}

const TYPE_OPTIONS = [
  { key: '1', label: '消费', color: 'success' },
  { key: '2', label: '充值', color: 'warning' },
  { key: '3', label: '系统', color: 'primary' },
  { key: '4', label: '失败', color: 'danger' },
] as const;

export default function LogPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [showFilters, setShowFilters] = useState(false);
  const { token } = useAuthStore();
  const location = useLocation();
  const isGlobalLog = location.pathname.startsWith('/admin');
  const [filters, setFilters] = useState({
    type: '', model_name: '', token_name: '', content: '',
    user_id: '', channel_id: '', start_time: '', end_time: '',
    min_quota: '', max_quota: '',
  });
  const [cleanupDays, setCleanupDays] = useState('30');

  const setFilter = (key: string, val: string) =>
    setFilters(prev => ({ ...prev, [key]: val }));

  const buildParams = (includePage = true) => {
    const p = new URLSearchParams();
    if (includePage) { p.set('page', page.toString()); p.set('size', '20'); }
    Object.entries(filters).forEach(([k, v]) => { if (v) p.set(k, v); });
    return p;
  };

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const endpoint = isGlobalLog ? '/api/log' : '/api/log/self';
      const res = await fetch(`${endpoint}?${buildParams()}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) { setLogs(data.data.data || []); setTotal(data.data.total || 0); }
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { if (token) fetchLogs(); }, [page, token]);

  const getTypeInfo = (type: number) => TYPE_OPTIONS.find(t => t.key === String(type)) ?? { label: '未知', color: 'default' as const };

  const handleExport = async () => {
    const res = await fetch(`/api/export/log?${buildParams(false)}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error('导出失败'); return; }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'logs.csv';
    document.body.appendChild(a); a.click(); a.remove();
    URL.revokeObjectURL(url);
  };

  const handleCleanup = async () => {
    const days = parseInt(cleanupDays);
    if (!days || days < 1) { toast.warning('请输入有效天数'); return; }
    if (!await confirm({ title: '清理日志', message: `确定清理 ${days} 天前的日志？此操作不可撤销。`, danger: true })) return;
    const ts = Math.floor(Date.now() / 1000) - days * 86400;
    try {
      const res = await fetch('/api/log', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ before_timestamp: ts }),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success(`已清理 ${data.data?.deleted ?? data.deleted} 条日志`); fetchLogs(); }
      else toast.error(data.msg || '清理失败');
    } catch (e) { console.error(e); }
  };

  const handleReset = () => {
    setFilters({ type: '', model_name: '', token_name: '', content: '', user_id: '', channel_id: '', start_time: '', end_time: '', min_quota: '', max_quota: '' });
    setPage(1);
    setTimeout(fetchLogs, 0);
  };

  const activeFilterCount = Object.values(filters).filter(Boolean).length;

  return (
    <div className="space-y-5">
      <PageHeader
        title={isGlobalLog ? '系统日志' : '我的日志'}
        description={isGlobalLog ? '查看所有请求与操作记录' : '查看您的 API 调用记录'}
        actions={
          <div className="flex gap-2 flex-wrap">
            {isGlobalLog && (
              <>
                <div className="flex gap-1 items-center">
                  <Input type="number" placeholder="天数" size="sm" value={cleanupDays}
                    onValueChange={setCleanupDays} style={{ width: '72px' }} />
                  <Button size="sm" variant="flat" color="danger" startContent={<Trash2 size={14} />} onPress={handleCleanup}>
                    清理
                  </Button>
                </div>
                <Button size="sm" variant="flat" startContent={<Download size={14} />} onPress={handleExport}>
                  导出 CSV
                </Button>
              </>
            )}
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

      {/* 筛选面板 */}
      {showFilters && (
        <Card>
          <CardBody className="p-4">
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
              <Select label="类型" size="sm"
                selectedKeys={filters.type ? [filters.type] : []}
                onChange={e => setFilter('type', e.target.value)}>
                {TYPE_OPTIONS.map(t => <SelectItem key={t.key}>{t.label}</SelectItem>)}
              </Select>
              <Input size="sm" label="模型名称" value={filters.model_name} onValueChange={v => setFilter('model_name', v)} />
              <Input size="sm" label="令牌名称" value={filters.token_name} onValueChange={v => setFilter('token_name', v)} />
              <Input size="sm" label="内容关键词" value={filters.content} onValueChange={v => setFilter('content', v)} />
              <Input size="sm" type="date" label="开始日期" value={filters.start_time} onValueChange={v => setFilter('start_time', v)} />
              <Input size="sm" type="date" label="结束日期" value={filters.end_time} onValueChange={v => setFilter('end_time', v)} />
              <Input size="sm" label="最小额度" value={filters.min_quota} onValueChange={v => setFilter('min_quota', v)} />
              <Input size="sm" label="最大额度" value={filters.max_quota} onValueChange={v => setFilter('max_quota', v)} />
              {isGlobalLog && (
                <>
                  <Input size="sm" label="用户 ID" value={filters.user_id} onValueChange={v => setFilter('user_id', v)} />
                  <Input size="sm" label="渠道 ID" value={filters.channel_id} onValueChange={v => setFilter('channel_id', v)} />
                </>
              )}
            </div>
            <div className="flex gap-2 mt-3">
              <Button size="sm" color="primary" onPress={() => { setPage(1); fetchLogs(); }}>查询</Button>
              <Button size="sm" variant="flat" onPress={handleReset}>重置</Button>
            </div>
          </CardBody>
        </Card>
      )}

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>类型</th>
              {isGlobalLog && <th>用户</th>}
              <th>模型</th>
              <th>令牌</th>
              <th>提示 / 补全</th>
              <th>额度消耗</th>
              <th>详情</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={isGlobalLog ? 8 : 7} rows={8} />
            ) : logs.length === 0 ? (
              <tr><td colSpan={isGlobalLog ? 8 : 7}>
                <EmptyState icon="📋" title="暂无日志" description={activeFilterCount > 0 ? '当前筛选条件下没有记录' : '暂时没有请求记录'} />
              </td></tr>
            ) : logs.map((log) => {
              const typeInfo = getTypeInfo(log.type);
              return (
                <tr key={log.id}>
                  <td style={{ whiteSpace: 'nowrap', fontSize: '12px' }}>{new Date(log.created_at * 1000).toLocaleString()}</td>
                  <td><Chip size="sm" variant="flat" color={typeInfo.color as any}>{typeInfo.label}</Chip></td>
                  {isGlobalLog && <td style={{ fontSize: '12px' }}>{log.username ? `${log.username}` : log.user_id || '-'}</td>}
                  <td>{log.model_name ? <Chip size="sm" variant="dot">{log.model_name}</Chip> : '-'}</td>
                  <td style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{log.token_name || '-'}</td>
                  <td style={{ fontSize: '12px' }}>{log.type === 1 ? `${log.prompt_tokens} / ${log.completion_tokens}` : '-'}</td>
                  <td style={{ fontWeight: 600, color: log.type === 4 ? '#f87171' : 'var(--text-primary)' }}>
                    ${(log.quota / 500000).toFixed(6)}
                  </td>
                  <td style={{ maxWidth: '180px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '12px', color: 'var(--text-muted)' }}>
                    {log.content || '-'}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {total > 20 && (
        <div className="flex justify-center">
          <Pagination isCompact showControls showShadow color="primary"
            page={page} total={Math.ceil(total / 20)} onChange={setPage} />
        </div>
      )}
    </div>
  );
}
