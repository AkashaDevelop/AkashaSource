import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Chip, Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Pagination, Select, SelectItem,
} from '../../components/ui';
import { Plus, RefreshCw, Copy, Power, PowerOff, Trash2, Download } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Redemption {
  id: number;
  name: string;
  code: string;
  quota: number;
  max_uses: number;
  used_count: number;
  status: number;
  created_at: number;
  used_by: number;
}

const STATUS_MAP: Record<number, { label: string; color: 'default' | 'success' | 'danger' | 'warning' }> = {
  1: { label: '未使用', color: 'success' },
  2: { label: '已用完', color: 'default' },
  3: { label: '已禁用', color: 'danger' },
};

export default function RedemptionManagement() {
  const [codes, setCodes] = useState<Redemption[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [statusFilter, setStatusFilter] = useState('');
  const [keyword, setKeyword] = useState('');
  const [selected, setSelected] = useState<Set<number>>(new Set());

  const [formData, setFormData] = useState({
    name: '', quota: '1', count: '1', max_uses: '1',
  });

  const fetchCodes = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ p: page.toString(), size: '10' });
      if (statusFilter) params.set('status', statusFilter);
      if (keyword) params.set('keyword', keyword);
      const res = await fetch(`/api/redemption/search?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setCodes(data.data.data || []);
        setTotal(data.data.total || 0);
        setSelected(new Set());
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { if (token) fetchCodes(); }, [page, token, statusFilter]);

  const handleGenerate = async (onClose: () => void) => {
    const res = await fetch('/api/redemption', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({
        name: formData.name,
        quota: parseInt(formData.quota) * 500000,
        count: parseInt(formData.count),
        max_uses: parseInt(formData.max_uses) || 1,
      }),
    });
    const data = await res.json();
    if (data.code === 0) { toast.success('生成成功'); fetchCodes(); onClose(); }
    else toast.error(data.msg || '生成兑换码失败');
  };

  const handleToggleStatus = async (item: Redemption) => {
    if (item.status === 2) return;
    const newStatus = item.status === 3 ? 1 : 3;
    const res = await fetch(`/api/redemption/${item.id}/status`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ status: newStatus }),
    });
    const data = await res.json();
    if (data.code === 0) fetchCodes();
    else toast.error('操作失败');
  };

  const handleBatch = async (action: 'enable' | 'disable' | 'delete') => {
    if (selected.size === 0) return;
    if (action === 'delete') {
      if (!await confirm({ title: '批量删除', message: `确定删除选中的 ${selected.size} 条兑换码？`, danger: true })) return;
    }
    const res = await fetch('/api/redemption/batch', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ ids: Array.from(selected), action }),
    });
    const data = await res.json();
    if (data.code === 0) { toast.success('操作成功'); fetchCodes(); }
    else toast.error('操作失败');
  };

  const toggleSelect = (id: number) => {
    setSelected(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    setSelected(selected.size === codes.length ? new Set() : new Set(codes.map(c => c.id)));
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('已复制');
  };

  const handleExport = async () => {
    const params = new URLSearchParams();
    if (statusFilter) params.set('status', statusFilter);
    const res = await fetch(`/api/export/redemption?${params}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) return;
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'redemptions.csv';
    document.body.appendChild(a); a.click(); a.remove();
    window.URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-4">
      <PageHeader
        title="兑换码管理"
        description="生成和管理额度兑换码"
        actions={
          <div className="flex gap-2 flex-wrap">
            <Input placeholder="搜索名称/码" value={keyword} onValueChange={setKeyword} className="w-36"
              onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); fetchCodes(); } }} />
            <Select label="状态" selectedKeys={statusFilter ? [statusFilter] : []}
              onSelectionChange={(keys) => { setStatusFilter([...keys][0] as string || ''); setPage(1); }}
              className="w-28" placeholder="全部">
              <SelectItem key="1">未使用</SelectItem>
              <SelectItem key="2">已用完</SelectItem>
              <SelectItem key="3">已禁用</SelectItem>
            </Select>
            <Button startContent={<RefreshCw size={16} />} onPress={fetchCodes} variant="flat">刷新</Button>
            <Button startContent={<Download size={16} />} onPress={handleExport} variant="flat">导出</Button>
            <Button startContent={<Plus size={16} />} color="primary" onPress={onOpen}>生成兑换码</Button>
          </div>
        }
      />

      {selected.size > 0 && (
        <div className="flex items-center gap-3 px-4 py-2.5 rounded-xl"
          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
          <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>
            已选 <strong style={{ color: 'var(--accent-primary)' }}>{selected.size}</strong> 条
          </span>
          <div className="flex gap-2 ml-auto">
            <Button size="sm" variant="flat" color="success" startContent={<Power size={14} />}
              onPress={() => handleBatch('enable')}>批量启用</Button>
            <Button size="sm" variant="flat" color="warning" startContent={<PowerOff size={14} />}
              onPress={() => handleBatch('disable')}>批量禁用</Button>
            <Button size="sm" variant="flat" color="danger" startContent={<Trash2 size={14} />}
              onPress={() => handleBatch('delete')}>批量删除</Button>
          </div>
        </div>
      )}

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th style={{ width: 36 }}>
                <input type="checkbox" checked={codes.length > 0 && selected.size === codes.length}
                  onChange={toggleSelectAll}
                  style={{ cursor: 'pointer', accentColor: 'var(--accent-primary)' }} />
              </th>
              <th>ID</th><th>名称</th><th>兑换码</th><th>额度</th>
              <th>可用次数</th><th>已用次数</th><th>状态</th><th>创建时间</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={10} rows={5} />
            ) : codes.length === 0 ? (
              <tr><td colSpan={10}><EmptyState icon="🎫" title="暂无兑换码" /></td></tr>
            ) : codes.map((item) => (
              <tr key={item.id}>
                <td>
                  <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggleSelect(item.id)}
                    style={{ cursor: 'pointer', accentColor: 'var(--accent-primary)' }} />
                </td>
                <td>{item.id}</td>
                <td>{item.name || '-'}</td>
                <td>
                  <div className="flex items-center gap-1.5">
                    <span className="font-mono text-xs">{item.code.slice(0, 18)}…</span>
                    <Copy size={13} className="cursor-pointer" style={{ color: 'var(--text-muted)' }}
                      onClick={() => copyToClipboard(item.code)} />
                  </div>
                </td>
                <td>${(item.quota / 500000).toFixed(2)}</td>
                <td>{item.max_uses === 0 ? '∞' : item.max_uses}</td>
                <td>{item.used_count}</td>
                <td>
                  <Chip size="sm" color={STATUS_MAP[item.status]?.color ?? 'default'}>
                    {STATUS_MAP[item.status]?.label ?? '未知'}
                  </Chip>
                </td>
                <td className="text-xs">{new Date(item.created_at * 1000).toLocaleString()}</td>
                <td>
                  {item.status !== 2 && (
                    <Button size="sm" variant="flat"
                      color={item.status === 3 ? 'success' : 'warning'}
                      onPress={() => handleToggleStatus(item)}>
                      {item.status === 3 ? '启用' : '禁用'}
                    </Button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex justify-center mt-4">
        <Pagination page={page} total={Math.ceil(total / 10) || 1} onChange={(p) => setPage(p)} />
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>生成兑换码</ModalHeader>
              <ModalBody>
                <Input label="名称" placeholder="例如：活动赠送" value={formData.name}
                  onValueChange={(v) => setFormData({ ...formData, name: v })} />
                <Input label="额度 ($)" type="number" placeholder="1" value={formData.quota}
                  onValueChange={(v) => setFormData({ ...formData, quota: v })}
                  description="1 美元 = 500000 额度" />
                <Input label="可用次数" type="number" placeholder="1" value={formData.max_uses}
                  onValueChange={(v) => setFormData({ ...formData, max_uses: v })}
                  description="0 = 不限次数；每次使用均发放一次额度" />
                <Input label="生成数量" type="number" value={formData.count}
                  onValueChange={(v) => setFormData({ ...formData, count: v })} />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleGenerate(onClose)}>生成</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
