import { useState, useEffect } from 'react';
import { Button, Chip, Tooltip } from '../../components/ui';
import { Trash2, RefreshCw, FileText, Download } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import PageHeader from '../../components/PageHeader';

interface StoredFile {
  id: number;
  user_id: number;
  purpose: string;
  filename: string;
  bytes: number;
  created_at: number;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleString('zh-CN');
}

export default function FilesPage() {
  const [files, setFiles] = useState<StoredFile[]>([]);
  const [loading, setLoading] = useState(false);
  const { token } = useAuthStore();

  const fetchFiles = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/files', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setFiles(data.data || []);
      } else {
        toast.error(data.msg || '获取文件列表失败');
      }
    } catch {
      toast.error('获取文件列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (file: StoredFile) => {
    const ok = await confirm(`确认删除文件 "${file.filename}"？此操作不可恢复。`);
    if (!ok) return;
    try {
      const res = await fetch(`/api/user/files/${file.id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('文件已删除');
        setFiles(prev => prev.filter(f => f.id !== file.id));
      } else {
        toast.error(data.msg || '删除失败');
      }
    } catch {
      toast.error('删除失败');
    }
  };

  useEffect(() => {
    if (token) fetchFiles();
  }, [token]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="文件管理"
        description="管理通过 API 上传的文件"
        actions={
          <Button
            isIconOnly
            variant="flat"
            onPress={fetchFiles}
            isLoading={loading}
            style={{
              background: 'var(--bg-elevated)',
              color: 'var(--accent-primary)',
              borderRadius: '10px',
              border: '1px solid var(--border-color)',
            }}
          >
            <RefreshCw size={20} />
          </Button>
        }
      />

      <div
        className="rounded-2xl overflow-hidden"
        style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}
      >
        <div
          className="grid text-xs font-semibold px-4 py-3"
          style={{
            gridTemplateColumns: '1fr 120px 100px 160px 80px',
            color: 'var(--text-secondary)',
            borderBottom: '1px solid var(--border-color)',
          }}
        >
          <span>文件名</span>
          <span>用途</span>
          <span>大小</span>
          <span>上传时间</span>
          <span className="text-right">操作</span>
        </div>

        {loading ? (
          <div className="p-6 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="skeleton-shimmer" style={{ height: '36px', borderRadius: '8px', animationDelay: `${i * 0.07}s` }} />
            ))}
          </div>
        ) : files.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-14 gap-3" style={{ color: 'var(--text-secondary)' }}>
            <FileText size={40} style={{ opacity: 0.4 }} />
            <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>暂无文件</p>
            <p className="text-xs">通过 API /v1/files 上传的文件将在此显示</p>
          </div>
        ) : (
          files.map(file => (
            <div
              key={file.id}
              className="grid items-center px-4 py-3 transition-colors"
              style={{
                gridTemplateColumns: '1fr 120px 100px 160px 80px',
                borderBottom: '1px solid var(--border-color)',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-elevated)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <div className="flex items-center gap-2 min-w-0">
                <FileText size={15} style={{ color: 'var(--accent-primary)', flexShrink: 0 }} />
                <span
                  className="text-sm font-medium truncate"
                  style={{ color: 'var(--text-primary)' }}
                  title={file.filename}
                >
                  {file.filename}
                </span>
              </div>

              <div>
                <Chip size="sm" variant="flat" style={{ background: 'rgba(124,58,237,0.12)', color: 'var(--accent-primary)' }}>
                  {file.purpose}
                </Chip>
              </div>

              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                {formatBytes(file.bytes)}
              </span>

              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                {formatDate(file.created_at)}
              </span>

              <div className="flex justify-end gap-1">
                <Tooltip content="下载文件">
                  <Button
                    isIconOnly
                    size="sm"
                    variant="flat"
                    style={{ background: 'var(--bg-elevated)', color: 'var(--text-secondary)', borderRadius: '8px' }}
                    onPress={() => window.open(`/v1/files/${file.id}/content`, '_blank')}
                  >
                    <Download size={14} />
                  </Button>
                </Tooltip>
                <Tooltip content="删除文件">
                  <Button
                    isIconOnly
                    size="sm"
                    variant="flat"
                    style={{ background: 'rgba(248,113,113,0.1)', color: '#f87171', borderRadius: '8px' }}
                    onPress={() => handleDelete(file)}
                  >
                    <Trash2 size={14} />
                  </Button>
                </Tooltip>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
