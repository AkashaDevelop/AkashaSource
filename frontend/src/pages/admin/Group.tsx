import { useState, useEffect, useRef } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Chip,
} from '../../components/ui';
import { Plus, Edit, Trash2, RefreshCw, X } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Group {
  id: number;
  name: string;
  description: string;
  model_ratios: string;
  allowed_channels: string;
  qpm: number;
}

interface RatioRow { model: string; ratio: string; }

// ── 模型倍率可视化编辑器 ──────────────────────────────────────
function RatioEditor({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const parse = (s: string): RatioRow[] => {
    try {
      const obj = JSON.parse(s || '{}');
      return Object.entries(obj).map(([model, ratio]) => ({ model, ratio: String(ratio) }));
    } catch { return []; }
  };

  const [rows, setRows] = useState<RatioRow[]>(() => parse(value));

  const sync = (next: RatioRow[]) => {
    setRows(next);
    const obj: Record<string, number> = {};
    next.forEach(r => { if (r.model) obj[r.model] = parseFloat(r.ratio) || 1; });
    onChange(JSON.stringify(obj));
  };

  const update = (i: number, field: keyof RatioRow, v: string) => {
    const next = rows.map((r, idx) => idx === i ? { ...r, [field]: v } : r);
    sync(next);
  };

  const remove = (i: number) => sync(rows.filter((_, idx) => idx !== i));
  const add = () => sync([...rows, { model: '', ratio: '1' }]);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>模型倍率覆盖</span>
        <Button size="sm" variant="flat" color="primary" startContent={<Plus size={13} />} onPress={add}>
          添加
        </Button>
      </div>
      {rows.length === 0 ? (
        <div className="text-xs text-center py-3 rounded-xl" style={{ color: 'var(--text-muted)', border: '1px dashed var(--border-color)' }}>
          暂无覆盖规则，点击添加
        </div>
      ) : (
        <div className="space-y-1.5">
          {rows.map((row, i) => (
            <div key={i} className="flex gap-2 items-center">
              <Input placeholder="模型名称，如 gpt-4" value={row.model}
                onValueChange={(v) => update(i, 'model', v)} className="flex-1" />
              <Input placeholder="倍率" type="number" value={row.ratio}
                onValueChange={(v) => update(i, 'ratio', v)} className="w-24" />
              <button onClick={() => remove(i)}
                className="p-1.5 rounded-lg transition-colors"
                style={{ color: 'var(--text-muted)' }}
                onMouseEnter={e => (e.currentTarget.style.color = '#f87171')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-muted)')}>
                <X size={15} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ── 渠道多选 ──────────────────────────────────────────────────
interface ChannelOption { id: number; name: string; }

function ChannelMultiSelect({ value, onChange, token }: {
  value: string; onChange: (v: string) => void; token: string;
}) {
  const parseIds = (s: string) => s.split(',').map(v => v.trim()).filter(Boolean).map(Number).filter(Boolean);
  const [selected, setSelected] = useState<number[]>(() => parseIds(value));
  const [channels, setChannels] = useState<ChannelOption[]>([]);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetch('/api/channel', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(d => {
        const list = d.code === 0 ? d.data : (d.data || []);
        setChannels(list.map((c: { id: number; name: string }) => ({ id: c.id, name: c.name })));
      })
      .catch(() => {});
  }, [token]);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const toggle = (id: number) => {
    const next = selected.includes(id) ? selected.filter(s => s !== id) : [...selected, id];
    setSelected(next);
    onChange(next.join(','));
  };

  const remove = (id: number) => {
    const next = selected.filter(s => s !== id);
    setSelected(next);
    onChange(next.join(','));
  };

  const filtered = channels.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase()) || String(c.id).includes(search)
  );

  const selectedChannels = channels.filter(c => selected.includes(c.id));

  return (
    <div className="space-y-1.5" ref={ref}>
      <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>允许渠道</span>
      {/* 已选 tags */}
      <div
        className="flex flex-wrap gap-1.5 min-h-[42px] px-3 py-2 rounded-xl cursor-pointer"
        style={{ background: 'var(--bg-elevated)', border: `1px solid ${open ? 'var(--accent-primary)' : 'var(--border-color)'}` }}
        onClick={() => setOpen(o => !o)}
      >
        {selectedChannels.length === 0 ? (
          <span className="text-sm self-center" style={{ color: 'var(--text-muted)' }}>点击选择渠道（留空=允许全部）</span>
        ) : selectedChannels.map(c => (
          <span key={c.id} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-lg text-xs font-medium"
            style={{ background: 'var(--accent-primary)15', color: 'var(--accent-primary)', border: '1px solid var(--accent-primary)30' }}>
            #{c.id} {c.name}
            <X size={11} className="cursor-pointer" onClick={(e) => { e.stopPropagation(); remove(c.id); }} />
          </span>
        ))}
      </div>
      {/* 下拉面板 */}
      {open && (
        <div className="rounded-xl shadow-lg overflow-hidden"
          style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', zIndex: 50, position: 'relative' }}>
          <div className="p-2 border-b" style={{ borderColor: 'var(--border-color)' }}>
            <input
              autoFocus
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="搜索渠道名称或 ID…"
              className="w-full bg-transparent outline-none text-sm px-2 py-1"
              style={{ color: 'var(--text-primary)' }}
              onClick={e => e.stopPropagation()}
            />
          </div>
          <div className="max-h-48 overflow-y-auto">
            {filtered.length === 0 ? (
              <div className="text-xs text-center py-4" style={{ color: 'var(--text-muted)' }}>无匹配渠道</div>
            ) : filtered.map(c => {
              const isSelected = selected.includes(c.id);
              return (
                <div key={c.id}
                  className="flex items-center gap-2 px-3 py-2 cursor-pointer text-sm transition-colors"
                  style={{ background: isSelected ? 'var(--accent-primary)10' : 'transparent', color: 'var(--text-primary)' }}
                  onMouseEnter={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'var(--bg-elevated)'; }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = isSelected ? 'var(--accent-primary)10' : 'transparent'; }}
                  onClick={e => { e.stopPropagation(); toggle(c.id); }}
                >
                  <div className="w-4 h-4 rounded flex items-center justify-center flex-shrink-0"
                    style={{ background: isSelected ? 'var(--accent-primary)' : 'transparent', border: `1.5px solid ${isSelected ? 'var(--accent-primary)' : 'var(--border-color)'}` }}>
                    {isSelected && <svg width="10" height="10" viewBox="0 0 10 10"><polyline points="1.5,5 4,7.5 8.5,2.5" fill="none" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>}
                  </div>
                  <span className="text-xs" style={{ color: 'var(--text-muted)' }}>#{c.id}</span>
                  <span className="truncate">{c.name}</span>
                </div>
              );
            })}
          </div>
          {selected.length > 0 && (
            <div className="px-3 py-2 border-t flex justify-between items-center" style={{ borderColor: 'var(--border-color)' }}>
              <span className="text-xs" style={{ color: 'var(--text-muted)' }}>已选 {selected.length} 个</span>
              <button className="text-xs" style={{ color: '#f87171' }}
                onClick={e => { e.stopPropagation(); setSelected([]); onChange(''); }}>清空</button>
            </div>
          )}
        </div>
      )}
      <p className="text-xs" style={{ color: 'var(--text-muted)' }}>留空表示允许所有渠道</p>
    </div>
  );
}

// ── 主页面 ────────────────────────────────────────────────────
export default function GroupManagement() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(false);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editing, setEditing] = useState<Group | null>(null);
  const [formData, setFormData] = useState({
    name: '', description: '', model_ratios: '{}', allowed_channels: '', qpm: '0',
  });

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/group', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setGroups(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchGroups(); }, []);

  const handleAdd = () => {
    setEditing(null);
    setFormData({ name: '', description: '', model_ratios: '{}', allowed_channels: '', qpm: '0' });
    onOpen();
  };

  const handleEdit = (g: Group) => {
    setEditing(g);
    setFormData({
      name: g.name, description: g.description,
      model_ratios: g.model_ratios || '{}',
      allowed_channels: g.allowed_channels,
      qpm: g.qpm.toString(),
    });
    onOpen();
  };

  const handleSubmit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = { ...formData, id: editing?.id, qpm: parseInt(formData.qpm) || 0 };
    try {
      const res = await fetch('/api/group', {
        method, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code === 0) { fetchGroups(); onClose(); }
      else toast.error('操作失败');
    } catch (e) { console.error(e); }
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({ title: '删除分组', message: '确定删除此分组？', danger: true })) return;
    try {
      const res = await fetch(`/api/group/${id}`, {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) fetchGroups();
    } catch (e) { console.error(e); }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="分组管理"
        description="管理用户分组及其权限配置"
        actions={
          <div className="flex gap-2">
            <Button startContent={<RefreshCw size={18} />} onPress={fetchGroups} variant="flat">刷新</Button>
            <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>添加分组</Button>
          </div>
        }
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th><th>名称</th><th>描述</th><th>QPM</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={5} rows={5} />
            ) : groups.length === 0 ? (
              <tr><td colSpan={5}><EmptyState icon="🏷️" title="暂无分组" /></td></tr>
            ) : groups.map((g) => (
              <tr key={g.id}>
                <td>{g.id}</td>
                <td><Chip size="sm" variant="flat" color="primary">{g.name}</Chip></td>
                <td>{g.description || '-'}</td>
                <td>{g.qpm || '无限制'}</td>
                <td>
                  <div className="flex gap-2">
                    <span className="cursor-pointer" style={{ color: 'var(--text-muted)' }} onClick={() => handleEdit(g)}><Edit size={18} /></span>
                    <span className="cursor-pointer" style={{ color: '#f87171' }} onClick={() => handleDelete(g.id)}><Trash2 size={18} /></span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="2xl">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editing ? '编辑分组' : '添加分组'}</ModalHeader>
              <ModalBody className="gap-4">
                <Input label="分组名称" value={formData.name}
                  onValueChange={(v) => setFormData({ ...formData, name: v })} isRequired />
                <Input label="描述" value={formData.description}
                  onValueChange={(v) => setFormData({ ...formData, description: v })} />
                <Input label="QPM (每分钟请求数)" type="number" value={formData.qpm}
                  onValueChange={(v) => setFormData({ ...formData, qpm: v })}
                  description="0 表示不限制" />
                <RatioEditor
                  value={formData.model_ratios}
                  onChange={(v) => setFormData({ ...formData, model_ratios: v })}
                />
                <ChannelMultiSelect
                  value={formData.allowed_channels}
                  onChange={(v) => setFormData({ ...formData, allowed_channels: v })}
                  token={token ?? ''}
                />
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleSubmit(onClose)}>保存</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
