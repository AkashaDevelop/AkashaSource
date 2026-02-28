import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import { Button, Input, Chip, Pagination } from '../../components/ui';
import { Plus, Trash2, Copy, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Invitation {
  id: number;
  code: string;
  inviter_id: number;
  invitee_id: number;
  inviter_name: string;
  invitee_name: string;
  status: number;
  max_uses: number;
  used_count: number;
  created_at: number;
  used_at: number;
}

export default function AdminInvitation() {
  const { token } = useAuthStore();
  const [list, setList] = useState<Invitation[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState(0);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [showModal, setShowModal] = useState(false);
  const [count, setCount] = useState('10');
  const [maxUses, setMaxUses] = useState('1');
  const [customCodes, setCustomCodes] = useState('');
  const [generating, setGenerating] = useState(false);
  const pageSize = 20;

  const fetch_ = async (p = page, s = statusFilter) => {
    setLoading(true);
    const params = new URLSearchParams({ p: String(p), size: String(pageSize) });
    if (s) params.set('status', String(s));
    const res = await fetch(`/api/invitation?${params}`, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) { setList(data.data.data || []); setTotal(data.data.total || 0); }
    setLoading(false);
  };

  useEffect(() => { fetch_(); }, [page, statusFilter]);

  const generate = async () => {
    const codes = customCodes.trim().split('\n').map(s => s.trim()).filter(Boolean);
    const n = parseInt(count);
    if (codes.length === 0 && (!n || n <= 0)) { toast.error('请输入有效数量或自定义邀请码'); return; }
    setGenerating(true);
    const body: Record<string, unknown> = { max_uses: parseInt(maxUses) || 1 };
    if (codes.length > 0) body.codes = codes; else body.count = n;
    const res = await fetch('/api/invitation', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(`已生成 ${data.data?.length ?? n} 个邀请码`);
      setShowModal(false); setCustomCodes('');
      fetch_(1, statusFilter); setPage(1);
    } else toast.error(data.msg || '生成失败');
    setGenerating(false);
  };

  const del = async (id: number) => {
    if (!await confirm({ message: '确认删除此邀请码？', danger: true })) return;
    const res = await fetch(`/api/invitation/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) { toast.success('已删除'); fetch_(); }
    else toast.error(data.msg || '删除失败');
  };

  const toggleSelect = (id: number) => setSelected(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const allSelected = list.length > 0 && list.every(i => selected.has(i.id));
  const toggleAll = () => setSelected(allSelected ? new Set() : new Set(list.map(i => i.id)));

  const bulkDelete = async () => {
    if (!await confirm({ message: `确认删除选中的 ${selected.size} 个邀请码？`, danger: true })) return;
    await Promise.all([...selected].map(id =>
      fetch(`/api/invitation/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } })
    ));
    toast.success(`已删除 ${selected.size} 个`);
    setSelected(new Set());
    fetch_();
  };

  const copy = (code: string) => { navigator.clipboard.writeText(code); toast.success('已复制'); };
  const fmtDate = (ts: number) => ts ? new Date(ts * 1000).toLocaleString('zh-CN') : '-';

  return (
    <div className="space-y-5">
      <PageHeader title="邀请码管理" description="批量生成和管理全局邀请码" />

      {/* Toolbar */}
      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
        <Button color="primary" size="sm" startContent={<Plus size={15} />} onPress={() => setShowModal(true)}
          style={{ borderRadius: '10px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}>
          批量生成
        </Button>
        {selected.size > 0 && (
          <>
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>已选 {selected.size} 项</span>
            <Button size="sm" variant="flat" startContent={<Trash2 size={13} />} onPress={bulkDelete}
              style={{ borderRadius: '10px', color: '#f87171' }}>批量删除</Button>
          </>
        )}
        <div style={{ display: 'flex', gap: '6px', marginLeft: 'auto' }}>
          {[{ v: 0, l: '全部' }, { v: 1, l: '未使用' }, { v: 2, l: '已使用' }].map(({ v, l }) => (
            <button key={v} onClick={() => { setStatusFilter(v); setPage(1); }}
              style={{
                padding: '4px 12px', borderRadius: '99px', fontSize: '12px', fontWeight: 600, cursor: 'pointer',
                background: statusFilter === v ? 'var(--accent-primary)' : 'var(--bg-elevated)',
                color: statusFilter === v ? '#fff' : 'var(--text-secondary)',
                border: '1px solid var(--border-color)',
              }}>{l}</button>
          ))}
          <Button size="sm" variant="flat" isIconOnly onPress={() => fetch_()} style={{ borderRadius: '10px' }}>
            <RefreshCw size={14} />
          </Button>
        </div>
      </div>

      {/* Table */}
      <div style={{ borderRadius: '16px', border: '1px solid var(--border-color)', overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ background: 'var(--bg-elevated)', borderBottom: '1px solid var(--border-color)' }}>
              <th style={{ padding: '10px 14px', width: '36px' }}>
                <input type="checkbox" checked={allSelected} onChange={toggleAll} />
              </th>
              {['邀请码', '状态', '使用次数', '邀请人', '被邀请人', '创建时间', '使用时间', '操作'].map(h => (
                <th key={h} style={{ padding: '10px 14px', textAlign: 'left', fontWeight: 600, color: 'var(--text-secondary)', whiteSpace: 'nowrap' }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={8} style={{ padding: '32px', textAlign: 'center', color: 'var(--text-faint)' }}>加载中...</td></tr>
            ) : list.length === 0 ? (
              <tr><td colSpan={8} style={{ padding: '32px', textAlign: 'center', color: 'var(--text-faint)' }}>暂无数据</td></tr>
            ) : list.map(inv => (
              <tr key={inv.id} style={{ borderBottom: '1px solid var(--border-color)', background: selected.has(inv.id) ? 'rgba(124,58,237,0.04)' : undefined }}>
                <td style={{ padding: '10px 14px', width: '36px' }}>
                  <input type="checkbox" checked={selected.has(inv.id)} onChange={() => toggleSelect(inv.id)} />
                </td>
                <td style={{ padding: '10px 14px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <code style={{ fontSize: '11px', color: 'var(--text-secondary)', fontFamily: 'monospace' }}>
                      {inv.code.length > 18 ? inv.code.slice(0, 18) + '…' : inv.code}
                    </code>
                    <button onClick={() => copy(inv.code)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-faint)', padding: 0 }}>
                      <Copy size={12} />
                    </button>
                  </div>
                </td>
                <td style={{ padding: '10px 14px' }}>
                  <Chip size="sm" color={inv.status === 1 ? 'success' : 'default'} variant="flat">
                    {inv.status === 1 ? '未使用' : '已使用'}
                  </Chip>
                </td>
                <td style={{ padding: '10px 14px', color: 'var(--text-secondary)', fontSize: '12px' }}>
                  {inv.used_count}/{inv.max_uses === 0 ? '∞' : inv.max_uses}
                </td>
                <td style={{ padding: '10px 14px', color: 'var(--text-secondary)' }}>{inv.inviter_name || (inv.inviter_id ? `#${inv.inviter_id}` : '-')}</td>
                <td style={{ padding: '10px 14px', color: 'var(--text-secondary)' }}>{inv.invitee_name || (inv.invitee_id ? `#${inv.invitee_id}` : '-')}</td>
                <td style={{ padding: '10px 14px', color: 'var(--text-secondary)', whiteSpace: 'nowrap' }}>{fmtDate(inv.created_at)}</td>
                <td style={{ padding: '10px 14px', color: 'var(--text-secondary)', whiteSpace: 'nowrap' }}>{fmtDate(inv.used_at)}</td>
                <td style={{ padding: '10px 14px' }}>
                  <Button size="sm" variant="flat" isIconOnly onPress={() => del(inv.id)}
                    style={{ borderRadius: '8px', color: '#f87171' }}><Trash2 size={13} /></Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {total > pageSize && (
        <div style={{ display: 'flex', justifyContent: 'center' }}>
          <Pagination total={Math.ceil(total / pageSize)} page={page} onChange={setPage} />
        </div>
      )}

      {/* Generate modal */}
      {showModal && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 50,
          display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '16px',
        }} onClick={e => { if (e.target === e.currentTarget) setShowModal(false); }}>
          <div style={{
            background: 'var(--bg-surface)', borderRadius: '20px',
            border: '1px solid var(--border-color)', padding: '28px',
            width: '100%', maxWidth: '420px',
          }}>
            <h3 style={{ fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', marginBottom: '20px' }}>批量生成邀请码</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                <div style={{ minWidth: 0 }}>
                  <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>生成数量</label>
                  <Input size="sm" type="number" placeholder="10" value={count} onValueChange={setCount} />
                </div>
                <div style={{ minWidth: 0 }}>
                  <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>使用次数（0=无限）</label>
                  <Input size="sm" type="number" placeholder="1" value={maxUses} onValueChange={setMaxUses} />
                </div>
              </div>
              <div>
                <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>
                  自定义邀请码（每行一个，填写后忽略数量）
                </label>
                <textarea
                  value={customCodes}
                  onChange={e => setCustomCodes(e.target.value)}
                  placeholder={'MYCODE2026\nVIP-SPECIAL'}
                  rows={4}
                  style={{
                    width: '100%', padding: '8px 12px', borderRadius: '10px', fontSize: '13px',
                    background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                    color: 'var(--text-primary)', outline: 'none', resize: 'vertical', boxSizing: 'border-box',
                  }}
                />
              </div>
            </div>
            <div style={{ display: 'flex', gap: '10px', marginTop: '20px', justifyContent: 'flex-end' }}>
              <Button variant="flat" onPress={() => setShowModal(false)} style={{ borderRadius: '10px' }}>取消</Button>
              <Button color="primary" isLoading={generating} onPress={generate}
                style={{ borderRadius: '10px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}>
                生成
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
