import { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Pagination,
  Chip,
  Button,
  Input,
  Select,
  SelectItem,
} from '@heroui/react';
import { RefreshCw, Trash2 } from 'lucide-react';
import { useAuthStore } from '../store/auth';

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

export default function LogPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const { token } = useAuthStore();
  const location = useLocation();
  const isGlobalLog = location.pathname.startsWith('/admin');
  const [filters, setFilters] = useState({
    type: '',
    model_name: '',
    token_name: '',
    content: '',
    user_id: '',
    channel_id: '',
    start_time: '',
    end_time: '',
    min_quota: '',
    max_quota: '',
  });

  const buildParams = (includePage = true) => {
    const params = new URLSearchParams();
    if (includePage) {
      params.set('page', page.toString());
      params.set('size', '20');
    }
    Object.entries(filters).forEach(([key, value]) => {
      if (value) {
        params.set(key, value);
      }
    });
    return params;
  };

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const endpoint = isGlobalLog ? '/api/log' : '/api/log/self';
      const params = buildParams();
      const res = await fetch(`${endpoint}?${params.toString()}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setLogs(data.data || []);
        setTotal(data.total || 0);
      }
    } catch (error) {
      console.error('获取日志失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchLogs();
  }, [page, token]);

  const getTypeColor = (type: number) => {
    switch (type) {
      case 1: return "success"; // 消费
      case 2: return "warning"; // 充值
      case 3: return "primary"; // 系统
      case 4: return "danger"; // 失败
      default: return "default";
    }
  };

  const getTypeName = (type: number) => {
    switch (type) {
      case 1: return "消费";
      case 2: return "充值";
      case 3: return "系统";
      case 4: return "失败";
      default: return "未知";
    }
  };

  const handleExport = async () => {
    const params = buildParams(false);
    const res = await fetch(`/api/export/log?${params.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) return;
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'logs.csv';
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  };

  const [cleanupDays, setCleanupDays] = useState('30');

  const handleCleanup = async () => {
    const days = parseInt(cleanupDays);
    if (!days || days < 1) { alert('请输入有效天数'); return; }
    if (!confirm(`确定清理 ${days} 天前的日志?`)) return;
    const ts = Math.floor(Date.now() / 1000) - days * 86400;
    try {
      const res = await fetch('/api/log', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ before_timestamp: ts }),
      });
      const data = await res.json();
      if (res.ok) { alert(`已清理 ${data.deleted} 条日志`); fetchLogs(); }
      else alert(data.error || '清理失败');
    } catch (e) { console.error(e); }
  };

  const handleReset = () => {
    setFilters({
      type: '',
      model_name: '',
      token_name: '',
      content: '',
      user_id: '',
      channel_id: '',
      start_time: '',
      end_time: '',
      min_quota: '',
      max_quota: '',
    });
    setPage(1);
    setTimeout(fetchLogs, 0);
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">{isGlobalLog ? '系统日志' : '我的日志'}</h1>
          <p className="text-default-500">{isGlobalLog ? '查看所有的请求与操作记录' : '查看您的API调用记录'}</p>
        </div>
        <div className="flex gap-2">
          {isGlobalLog && (
            <>
              <Input
                type="number"
                placeholder="天数"
                size="sm"
                value={cleanupDays}
                onValueChange={setCleanupDays}
                className="w-20"
              />
              <Button variant="flat" color="danger" startContent={<Trash2 size={16} />} onPress={handleCleanup}>
                清理日志
              </Button>
              <Button variant="flat" onPress={handleExport}>
                导出 CSV
              </Button>
            </>
          )}
          <Button startContent={<RefreshCw size={18} />} onPress={fetchLogs} variant="flat">
            刷新
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Select
          label="类型"
          selectedKeys={filters.type ? [filters.type] : []}
          onChange={(e) => setFilters(prev => ({ ...prev, type: e.target.value }))}
        >
          <SelectItem key="1">消费</SelectItem>
          <SelectItem key="2">充值</SelectItem>
          <SelectItem key="3">系统</SelectItem>
          <SelectItem key="4">失败</SelectItem>
        </Select>
        <Input
          label="模型名称"
          value={filters.model_name}
          onValueChange={(val) => setFilters(prev => ({ ...prev, model_name: val }))}
        />
        <Input
          label="令牌名称"
          value={filters.token_name}
          onValueChange={(val) => setFilters(prev => ({ ...prev, token_name: val }))}
        />
        <Input
          label="内容关键词"
          value={filters.content}
          onValueChange={(val) => setFilters(prev => ({ ...prev, content: val }))}
        />
        <Input
          type="date"
          label="开始日期"
          value={filters.start_time}
          onValueChange={(val) => setFilters(prev => ({ ...prev, start_time: val }))}
        />
        <Input
          type="date"
          label="结束日期"
          value={filters.end_time}
          onValueChange={(val) => setFilters(prev => ({ ...prev, end_time: val }))}
        />
        <Input
          label="最小额度"
          value={filters.min_quota}
          onValueChange={(val) => setFilters(prev => ({ ...prev, min_quota: val }))}
        />
        <Input
          label="最大额度"
          value={filters.max_quota}
          onValueChange={(val) => setFilters(prev => ({ ...prev, max_quota: val }))}
        />
        {isGlobalLog && (
          <>
            <Input
              label="用户ID"
              value={filters.user_id}
              onValueChange={(val) => setFilters(prev => ({ ...prev, user_id: val }))}
            />
            <Input
              label="渠道ID"
              value={filters.channel_id}
              onValueChange={(val) => setFilters(prev => ({ ...prev, channel_id: val }))}
            />
          </>
        )}
        <div className="flex gap-2 items-end">
          <Button color="primary" onPress={() => { setPage(1); fetchLogs(); }}>
            查询
          </Button>
          <Button variant="flat" onPress={handleReset}>
            重置
          </Button>
        </div>
      </div>

      <Table 
        aria-label="日志表格"
        bottomContent={
          <div className="flex w-full justify-center">
            <Pagination
              isCompact
              showControls
              showShadow
              color="primary"
              page={page}
              total={Math.ceil(total / 20) || 1}
              onChange={(page) => setPage(page)}
            />
          </div>
        }
      >
        <TableHeader>
          <TableColumn>时间</TableColumn>
          <TableColumn>类型</TableColumn>
          {isGlobalLog ? <TableColumn>用户</TableColumn> : (null as any)}
          <TableColumn>模型</TableColumn>
          <TableColumn>令牌名称</TableColumn>
          <TableColumn>提示/补全（令牌）</TableColumn>
          <TableColumn>额度消耗</TableColumn>
          <TableColumn>详情</TableColumn>
        </TableHeader>
        <TableBody emptyContent="暂无日志" isLoading={loading}>
          {logs.map((log) => (
            <TableRow key={log.id}>
              <TableCell>{new Date(log.created_at * 1000).toLocaleString()}</TableCell>
              <TableCell>
                <Chip size="sm" variant="flat" color={getTypeColor(log.type)}>
                  {getTypeName(log.type)}
                </Chip>
              </TableCell>
              {isGlobalLog ? (
                <TableCell>
                  {log.username ? `${log.username} (${log.user_id})` : log.user_id || '-'}
                </TableCell>
              ) : (null as any)}
              <TableCell>
                {log.model_name ? <Chip size="sm" variant="dot">{log.model_name}</Chip> : '-'}
              </TableCell>
              <TableCell>{log.token_name || '-'}</TableCell>
              <TableCell>
                {log.type === 1 ? `${log.prompt_tokens} / ${log.completion_tokens}` : '-'}
              </TableCell>
              <TableCell>${(log.quota / 500000).toFixed(6)}</TableCell>
              <TableCell className="text-default-400 text-xs truncate max-w-[200px]">
                {log.content}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
