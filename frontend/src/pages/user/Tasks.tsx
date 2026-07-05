import { useState, useEffect } from 'react';
import { Button, Chip, Pagination, Tabs, Tab } from '../../components/ui';
import { RefreshCw, Image as ImageIcon, Music } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import PageHeader from '../../components/PageHeader';
import { formatQuota } from '../../lib/quota';

interface MJTask {
  id: number;
  action: string;
  mj_id: string;
  prompt: string;
  status: string;
  progress: string;
  image_url: string;
  submit_time: number;
  quota: number;
}

interface SunoTask {
  id: number;
  task_id: string;
  action: string;
  status: string;
  created_at: number;
  quota: number;
}

const MJ_STATUS_COLORS: Record<string, React.CSSProperties> = {
  SUBMITTED:  { background: 'rgba(59,130,246,0.12)',  color: '#60a5fa' },
  PROCESSING: { background: 'rgba(234,179,8,0.12)',   color: '#fbbf24' },
  SUCCESS:    { background: 'rgba(16,185,129,0.12)',  color: '#34d399' },
  FAILURE:    { background: 'rgba(248,113,113,0.12)', color: '#f87171' },
};

const SUNO_STATUS_COLORS: Record<string, React.CSSProperties> = {
  pending:    { background: 'rgba(59,130,246,0.12)',  color: '#60a5fa' },
  processing: { background: 'rgba(234,179,8,0.12)',   color: '#fbbf24' },
  completed:  { background: 'rgba(16,185,129,0.12)',  color: '#34d399' },
  failed:     { background: 'rgba(248,113,113,0.12)', color: '#f87171' },
};

function formatDate(ts: number) {
  if (!ts) return '-';
  return new Date(ts * 1000).toLocaleString('zh-CN');
}

function MJTasksTab({ token }: { token: string }) {
  const [tasks, setTasks] = useState<MJTask[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const size = 15;

  const fetchTasks = async (p: number) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/user/tasks/mj?p=${p}&size=${size}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setTasks(data.data.tasks || []);
        setTotal(data.data.total || 0);
      } else {
        toast.error(data.msg || '获取任务失败');
      }
    } catch {
      toast.error('获取任务失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchTasks(page); }, [page]);

  const totalPages = Math.ceil(total / size);

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>共 {total} 条记录</span>
        <Button isIconOnly variant="flat" size="sm" onPress={() => fetchTasks(page)} isLoading={loading}
          style={{ background: 'var(--bg-elevated)', color: 'var(--accent-primary)', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
          <RefreshCw size={16} />
        </Button>
      </div>

      <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
        <div className="grid text-xs font-semibold px-4 py-3"
          style={{ gridTemplateColumns: '80px 1fr 100px 80px 160px 90px', color: 'var(--text-secondary)', borderBottom: '1px solid var(--border-color)' }}>
          <span>操作</span>
          <span>Prompt</span>
          <span>状态</span>
          <span>进度</span>
          <span>提交时间</span>
          <span className="text-right">费用</span>
        </div>

        {loading ? (
          <div className="p-6 space-y-3">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="skeleton-shimmer" style={{ height: '36px', borderRadius: '8px', animationDelay: `${i * 0.07}s` }} />
            ))}
          </div>
        ) : tasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-14 gap-3" style={{ color: 'var(--text-secondary)' }}>
            <ImageIcon size={40} style={{ opacity: 0.4 }} />
            <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>暂无 Midjourney 任务</p>
            <p className="text-xs">通过 API 提交的绘图任务将在此显示</p>
          </div>
        ) : (
          tasks.map(task => (
            <div key={task.id}
              className="grid items-center px-4 py-3 transition-colors"
              style={{ gridTemplateColumns: '80px 1fr 100px 80px 160px 90px', borderBottom: '1px solid var(--border-color)' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-elevated)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <Chip size="sm" variant="flat" style={{ background: 'rgba(124,58,237,0.12)', color: 'var(--accent-primary)' }}>
                {task.action}
              </Chip>
              <span className="text-sm truncate pr-2" style={{ color: 'var(--text-primary)' }} title={task.prompt}>
                {task.prompt || '-'}
              </span>
              <div className="flex items-center gap-2">
                <Chip size="sm" variant="flat" style={MJ_STATUS_COLORS[task.status] ?? {}}>
                  {task.status}
                </Chip>
                {task.image_url && task.status === 'SUCCESS' && (
                  <a href={task.image_url} target="_blank" rel="noopener noreferrer"
                    className="text-xs underline" style={{ color: 'var(--accent-primary)' }}>
                    查看
                  </a>
                )}
              </div>
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{task.progress || '-'}</span>
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{formatDate(task.submit_time)}</span>
              <span className="text-sm text-right" style={{ color: 'var(--accent-cosmic)' }}>{formatQuota(task.quota, 4)}</span>
            </div>
          ))
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex justify-end">
          <Pagination total={totalPages} page={page} onChange={setPage} />
        </div>
      )}
    </div>
  );
}

function SunoTasksTab({ token }: { token: string }) {
  const [tasks, setTasks] = useState<SunoTask[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const size = 15;

  const fetchTasks = async (p: number) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/user/tasks/suno?p=${p}&size=${size}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setTasks(data.data.tasks || []);
        setTotal(data.data.total || 0);
      } else {
        toast.error(data.msg || '获取任务失败');
      }
    } catch {
      toast.error('获取任务失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchTasks(page); }, [page]);

  const totalPages = Math.ceil(total / size);

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>共 {total} 条记录</span>
        <Button isIconOnly variant="flat" size="sm" onPress={() => fetchTasks(page)} isLoading={loading}
          style={{ background: 'var(--bg-elevated)', color: 'var(--accent-primary)', borderRadius: '8px', border: '1px solid var(--border-color)' }}>
          <RefreshCw size={16} />
        </Button>
      </div>

      <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
        <div className="grid text-xs font-semibold px-4 py-3"
          style={{ gridTemplateColumns: '200px 80px 100px 1fr 90px', color: 'var(--text-secondary)', borderBottom: '1px solid var(--border-color)' }}>
          <span>Task ID</span>
          <span>操作</span>
          <span>状态</span>
          <span>创建时间</span>
          <span className="text-right">费用</span>
        </div>

        {loading ? (
          <div className="p-6 space-y-3">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="skeleton-shimmer" style={{ height: '36px', borderRadius: '8px', animationDelay: `${i * 0.07}s` }} />
            ))}
          </div>
        ) : tasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-14 gap-3" style={{ color: 'var(--text-secondary)' }}>
            <Music size={40} style={{ opacity: 0.4 }} />
            <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>暂无 Suno 任务</p>
            <p className="text-xs">通过 API 提交的音乐生成任务将在此显示</p>
          </div>
        ) : (
          tasks.map(task => (
            <div key={task.id}
              className="grid items-center px-4 py-3 transition-colors"
              style={{ gridTemplateColumns: '200px 80px 100px 1fr 90px', borderBottom: '1px solid var(--border-color)' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-elevated)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <span className="text-sm font-mono truncate pr-2" style={{ color: 'var(--text-secondary)' }} title={task.task_id}>
                {task.task_id}
              </span>
              <Chip size="sm" variant="flat" style={{ background: 'rgba(124,58,237,0.12)', color: 'var(--accent-primary)' }}>
                {task.action}
              </Chip>
              <Chip size="sm" variant="flat" style={SUNO_STATUS_COLORS[task.status] ?? {}}>
                {task.status}
              </Chip>
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{formatDate(task.created_at)}</span>
              <span className="text-sm text-right" style={{ color: 'var(--accent-cosmic)' }}>{formatQuota(task.quota, 4)}</span>
            </div>
          ))
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex justify-end">
          <Pagination total={totalPages} page={page} onChange={setPage} />
        </div>
      )}
    </div>
  );
}

export default function UserTasksPage() {
  const { token } = useAuthStore();

  return (
    <div className="space-y-6">
      <PageHeader title="任务记录" description="查看你的 Midjourney 绘图与 Suno 音乐生成任务" />

      <Tabs defaultSelectedKey="mj">
        <Tab key="mj" title={<span className="flex items-center gap-1.5"><ImageIcon size={15} />Midjourney</span>}>
          <MJTasksTab token={token!} />
        </Tab>
        <Tab key="suno" title={<span className="flex items-center gap-1.5"><Music size={15} />Suno</span>}>
          <SunoTasksTab token={token!} />
        </Tab>
      </Tabs>
    </div>
  );
}
