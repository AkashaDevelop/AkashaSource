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
} from '@heroui/react';
import { RefreshCw } from 'lucide-react';
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
}

export default function LogPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const { token } = useAuthStore();
  const location = useLocation();
  const isGlobalLog = location.pathname.startsWith('/admin');

  const fetchLogs = async () => {
    setLoading(true);
    try {
      const endpoint = isGlobalLog ? '/api/log' : '/api/log/self';
      const res = await fetch(`${endpoint}?page=${page}&size=20`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setLogs(data.data || []);
        setTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch logs:', error);
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
      default: return "default";
    }
  };

  const getTypeName = (type: number) => {
    switch (type) {
      case 1: return "消费";
      case 2: return "充值";
      case 3: return "系统";
      default: return "未知";
    }
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">{isGlobalLog ? '系统日志' : '我的日志'}</h1>
          <p className="text-default-500">{isGlobalLog ? '查看所有的请求与操作记录' : '查看您的API调用记录'}</p>
        </div>
        <Button startContent={<RefreshCw size={18} />} onPress={fetchLogs} variant="flat">
          刷新
        </Button>
      </div>

      <Table 
        aria-label="Log table"
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
          <TableColumn>模型</TableColumn>
          <TableColumn>令牌名称</TableColumn>
          <TableColumn>提示/补全 (Tokens)</TableColumn>
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
