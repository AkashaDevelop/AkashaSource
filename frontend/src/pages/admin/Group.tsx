import { useState, useEffect, useRef, useMemo } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Chip, Pagination, Switch, Select, SelectItem,
} from '../../components/ui';
import { Plus, Edit, Trash2, RefreshCw, X, Layers } from 'lucide-react';
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
  tpm: number;
  rpd: number;
  ratio?: number;       // 🌸 计费倍率(收归 Group 表)
  visibility?: string;  // 🌸 public 公开可选 / hidden 隐藏
  sort?: number;
}

// 🌸 特殊分组授权(基础分组→解锁的特殊分组)
interface SpecialGrant {
  id: number;
  base_group: string;
  special_group: string;
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
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [formData, setFormData] = useState({
    name: '', description: '', model_ratios: '{}', allowed_channels: '', qpm: '0', tpm: '0', rpd: '0', ratio: '1', visibility: 'public',
  });

  // 🌸 特殊分组授权规则～
  const [grants, setGrants] = useState<SpecialGrant[]>([]);
  // 🌸 编辑弹窗里：当前分组解锁的特殊分组名集合，保存时与后端 diff
  const [editUnlocks, setEditUnlocks] = useState<string[]>([]);
  // 🌸 特殊分组倍率(group_group_ratio)：{用户分组: {使用分组: 倍率}} 全量存 options 表～
  const [groupGroupRatio, setGroupGroupRatio] = useState<Record<string, Record<string, number>>>({});
  // 编辑弹窗里当前分组的"使用各分组的专属倍率"编辑行
  const [editGgRows, setEditGgRows] = useState<{ group: string; ratio: string }[]>([]);

  const fetchGroupGroupRatio = async () => {
    try {
      const res = await fetch('/api/option', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        const opt = data.data?.find((o: { key: string; value: string }) => o.key === 'group_group_ratio');
        if (opt?.value) setGroupGroupRatio(JSON.parse(opt.value) || {});
      }
    } catch { /* ignore */ }
  };

  const fetchGrants = async () => {
    try {
      const res = await fetch('/api/group/special_grant', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setGrants(data.data || []);
    } catch { /* ignore */ }
  };

  const handleDeleteGrant = async (id: number) => {
    try {
      const res = await fetch(`/api/group/special_grant/${id}`, {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) fetchGrants();
    } catch { /* ignore */ }
  };

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/group', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setGroups(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchGroups(); fetchGrants(); fetchGroupGroupRatio(); }, []);

  // 分页计算
  const paginatedGroups = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    return groups.slice(startIndex, endIndex);
  }, [groups, currentPage, pageSize]);

  const totalPages = Math.ceil(groups.length / pageSize);

  const handleAdd = () => {
    setEditing(null);
    setFormData({ name: '', description: '', model_ratios: '{}', allowed_channels: '', qpm: '0', tpm: '0', rpd: '0', ratio: '1', visibility: 'public' });
    setEditUnlocks([]);
    setEditGgRows([]);
    onOpen();
  };

  const handleEdit = (g: Group) => {
    setEditing(g);
    setFormData({
      name: g.name, description: g.description,
      model_ratios: g.model_ratios || '{}',
      allowed_channels: g.allowed_channels,
      qpm: g.qpm.toString(),
      tpm: (g.tpm ?? 0).toString(),
      rpd: (g.rpd ?? 0).toString(),
      ratio: String(g.ratio ?? 1),
      visibility: g.visibility || 'public',
    });
    // 当前分组已解锁的特殊分组
    setEditUnlocks(grants.filter((gr) => gr.base_group === g.name).map((gr) => gr.special_group));
    // 当前分组的"使用各分组专属倍率"(group_group_ratio[本分组])
    const gg = groupGroupRatio[g.name] || {};
    setEditGgRows(Object.entries(gg).map(([group, ratio]) => ({ group, ratio: String(ratio) })));
    onOpen();
  };

  // 🌸 把编辑弹窗里的"解锁特殊分组"与后端现状做 diff，增删对应授权
  const syncGrants = async (baseGroup: string) => {
    const current = grants.filter((gr) => gr.base_group === baseGroup);
    const currentNames = current.map((gr) => gr.special_group);
    const toAdd = editUnlocks.filter((n) => !currentNames.includes(n));
    const toDel = current.filter((gr) => !editUnlocks.includes(gr.special_group));
    await Promise.all([
      ...toAdd.map((special) => fetch('/api/group/special_grant', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ base_group: baseGroup, special_group: special }),
      })),
      ...toDel.map((gr) => fetch(`/api/group/special_grant/${gr.id}`, {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      })),
    ]);
  };

  const handleSubmit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = {
      ...formData, id: editing?.id,
      qpm: parseInt(formData.qpm) || 0,
      tpm: parseInt(formData.tpm) || 0,
      rpd: parseInt(formData.rpd) || 0,
      ratio: parseFloat(formData.ratio) || 1, // 🌸 倍率收归 Group 表
    };
    try {
      const res = await fetch('/api/group', {
        method, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code === 0) {
        // 🌸 分组存好后，同步它解锁的特殊分组授权
        await syncGrants(formData.name);
        // 🌸 同步"特殊分组倍率"：更新 group_group_ratio[本分组] 这一段后整体写回 options
        const ggNext = { ...groupGroupRatio };
        const seg: Record<string, number> = {};
        editGgRows.forEach((r) => {
          const g = r.group.trim();
          if (g) seg[g] = parseFloat(r.ratio) || 1;
        });
        if (Object.keys(seg).length > 0) ggNext[formData.name] = seg;
        else delete ggNext[formData.name];
        await fetch('/api/option', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify([{ key: 'group_group_ratio', value: JSON.stringify(ggNext) }]),
        });
        setGroupGroupRatio(ggNext);
        fetchGroups(); fetchGrants(); onClose();
      }
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

      {/* 🎴 分组卡片网格～ */}
      {loading ? (
        <div className="rounded-2xl p-6" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
          <LoadingRows cols={1} rows={5} />
        </div>
      ) : groups.length === 0 ? (
        <div className="rounded-2xl p-6" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
          <EmptyState icon="🏷️" title="暂无分组" />
        </div>
      ) : (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))' }}>
          {paginatedGroups.map((g) => {
            const children = grants.filter((gr) => gr.base_group === g.name);
            const hidden = g.visibility === 'hidden';
            return (
              <div key={g.id} className="rounded-2xl p-4 flex flex-col gap-3 transition-all duration-150 hover:shadow-[var(--shadow-hover)]"
                style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
                {/* 头部：名称 + 可见性 + 操作 */}
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    <div className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0"
                      style={{ background: hidden ? 'rgba(217,119,6,0.12)' : 'var(--nav-active-bg)' }}>
                      <Layers size={18} style={{ color: hidden ? '#d97706' : 'var(--accent-primary)' }} />
                    </div>
                    <div className="min-w-0">
                      <p className="font-bold truncate" style={{ color: 'var(--text-primary)' }} title={g.name}>{g.name}</p>
                      <p className="text-xs truncate" style={{ color: 'var(--text-muted)' }} title={g.description}>{g.description || '无描述'}</p>
                    </div>
                  </div>
                  <div className="flex gap-1 flex-shrink-0">
                    <span className="cursor-pointer p-1.5 rounded-lg transition-colors hover:bg-[var(--nav-hover-bg)]" style={{ color: 'var(--text-muted)' }} onClick={() => handleEdit(g)}><Edit size={16} /></span>
                    <span className="cursor-pointer p-1.5 rounded-lg transition-colors hover:bg-[var(--nav-hover-bg)]" style={{ color: '#f87171' }} onClick={() => handleDelete(g.id)}><Trash2 size={16} /></span>
                  </div>
                </div>

                {/* 标签行：可见性 + 倍率 */}
                <div className="flex items-center gap-2">
                  {hidden
                    ? <Chip size="sm" variant="flat" color="warning">隐藏</Chip>
                    : <Chip size="sm" variant="flat" color="success">公开</Chip>}
                  <Chip size="sm" variant="flat" color="primary">倍率 {g.ratio ?? 1}x</Chip>
                </div>

                {/* 限速三件套 */}
                <div className="grid grid-cols-3 gap-2">
                  {[['RPM', g.qpm], ['TPM', g.tpm], ['RPD', g.rpd]].map(([label, val]) => (
                    <div key={label as string} className="rounded-lg px-2 py-1.5 text-center" style={{ background: 'var(--bg-elevated)' }}>
                      <p className="text-[10px]" style={{ color: 'var(--text-faint)' }}>{label}</p>
                      <p className="text-sm font-semibold" style={{ color: 'var(--text-primary)', fontFamily: 'SF Mono, Menlo, monospace' }}>
                        {val ? Number(val).toLocaleString() : '∞'}
                      </p>
                    </div>
                  ))}
                </div>

                {/* 解锁的特殊分组 */}
                {children.length > 0 && (
                  <div className="pt-1 border-t" style={{ borderColor: 'var(--border-color)' }}>
                    <p className="text-[11px] mb-1.5" style={{ color: 'var(--text-muted)' }}>解锁特殊分组</p>
                    <div className="flex flex-wrap gap-1.5">
                      {children.map((gr) => (
                        <span key={gr.id} className="inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-lg text-xs"
                          style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', color: 'var(--text-secondary)' }}>
                          {gr.special_group}
                          <span title="移除授权" className="inline-flex cursor-pointer" onClick={() => handleDeleteGrant(gr.id)}>
                            <X size={12} style={{ color: '#f87171' }} />
                          </span>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* 分页控件 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-sm text-default-500">
            显示 {(currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, groups.length)} 条，共 {groups.length} 条
          </span>
          <Pagination
            total={totalPages}
            page={currentPage}
            onChange={setCurrentPage}
          />
        </div>
      )}

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
                {/* 🌸 可见性～public 普通用户可见可选 / hidden 仅特殊授权或直配可用 */}
                <div className="flex items-center justify-between px-3 py-2.5 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                  <div>
                    <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>公开可选</p>
                    <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
                      开启=普通用户可见可选；关闭=隐藏分组，仅特殊授权或直接分配的用户可用
                    </p>
                  </div>
                  <Switch
                    isSelected={formData.visibility === 'public'}
                    onValueChange={(v) => setFormData({ ...formData, visibility: v ? 'public' : 'hidden' })}
                  />
                </div>
                {/* 🌸 解锁的特殊分组：用户到本分组后自动获得这些额外分组(可选隐藏分组) */}
                <Select
                  label="解锁的特殊分组（可多选）"
                  placeholder="用户到本分组后自动解锁的额外分组，可留空"
                  selectionMode="multiple"
                  selectedKeys={editUnlocks}
                  onSelectionChange={(keys) => setEditUnlocks(Array.from(keys as Set<string>))}
                >
                  {groups.filter((g) => g.name !== formData.name).map((g) => (
                    <SelectItem key={g.name}>{`${g.name}${g.visibility === 'hidden' ? '（隐藏）' : ''}`}</SelectItem>
                  ))}
                </Select>
                {/* 🎀 三维速率限制～ */}
                <div className="grid grid-cols-3 gap-3">
                  <Input label="RPM (每分钟请求)" type="number" value={formData.qpm}
                    onValueChange={(v) => setFormData({ ...formData, qpm: v })}
                    description="0 = 不限" />
                  <Input label="TPM (每分钟 Token)" type="number" value={formData.tpm}
                    onValueChange={(v) => setFormData({ ...formData, tpm: v })}
                    description="0 = 不限" />
                  <Input label="RPD (每日请求)" type="number" value={formData.rpd}
                    onValueChange={(v) => setFormData({ ...formData, rpd: v })}
                    description="0 = 不限" />
                </div>
                <Input label="计费倍率" type="number" value={formData.ratio}
                  onValueChange={(v) => setFormData({ ...formData, ratio: v })}
                  description="1 = 原价，0.8 = 八折，影响该分组下所有用户的消费金额" />
                <RatioEditor
                  value={formData.model_ratios}
                  onChange={(v) => setFormData({ ...formData, model_ratios: v })}
                />
                {/* 🌸 特殊分组倍率：本分组用户在"使用某个分组"时的专属倍率(覆盖该分组默认倍率) */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>特殊分组倍率</span>
                      <p className="text-xs" style={{ color: 'var(--text-muted)' }}>本分组用户使用「某分组」时按此倍率计费，覆盖该分组的默认倍率</p>
                    </div>
                    <Button size="sm" variant="flat" color="primary" startContent={<Plus size={13} />}
                      onPress={() => setEditGgRows([...editGgRows, { group: '', ratio: '1' }])}>
                      添加
                    </Button>
                  </div>
                  {editGgRows.length === 0 ? (
                    <div className="text-xs text-center py-3 rounded-xl" style={{ color: 'var(--text-muted)', border: '1px dashed var(--border-color)' }}>
                      暂无特殊倍率，点击添加
                    </div>
                  ) : (
                    <div className="space-y-1.5">
                      {editGgRows.map((row, i) => (
                        <div key={i} className="flex gap-2 items-center">
                          <Select
                            className="flex-1"
                            placeholder="选择使用分组"
                            selectedKeys={row.group ? [row.group] : []}
                            onSelectionChange={(keys) => {
                              const g = [...keys][0] as string || '';
                              setEditGgRows(editGgRows.map((r, idx) => idx === i ? { ...r, group: g } : r));
                            }}
                          >
                            {groups
                              .filter((g) => g.name === row.group || !editGgRows.some((r, idx) => idx !== i && r.group === g.name))
                              .map((g) => <SelectItem key={g.name}>{g.name}</SelectItem>)}
                          </Select>
                          <Input className="w-28" type="number" placeholder="倍率" value={row.ratio}
                            onValueChange={(v) => setEditGgRows(editGgRows.map((r, idx) => idx === i ? { ...r, ratio: v } : r))} />
                          <span className="cursor-pointer" style={{ color: '#f87171' }}
                            onClick={() => setEditGgRows(editGgRows.filter((_, idx) => idx !== i))}>
                            <Trash2 size={16} />
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
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
