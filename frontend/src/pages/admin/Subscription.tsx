import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import { Button, Input } from '../../components/ui';
import { Plus, Pencil, Trash2, ToggleLeft, ToggleRight } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Plan {
  id: number;
  name: string;
  description: string;
  price: number;
  duration_days: number;
  type: string;
  group_name: string;
  quota: number;
  rpm: number;
  enabled: boolean;
  sort_order: number;
}

const emptyPlan: Omit<Plan, 'id'> = {
  name: '', description: '', price: 0, duration_days: 30,
  type: 'quota', group_name: '', quota: 0, rpm: 0, enabled: true, sort_order: 0,
};

const typeLabels: Record<string, string> = {
  group: '分组订阅', quota: '额度包', rpm: '高速RPM', combo: '组合套餐',
};

export default function SubscriptionManagement() {
  const { token } = useAuthStore();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [editing, setEditing] = useState<Partial<Plan> | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchPlans = async () => {
    const res = await fetch('/api/subscription/admin/plans', { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) setPlans(data.data || []);
  };

  useEffect(() => { fetchPlans(); }, []);

  const openNew = () => { setEditing({ ...emptyPlan }); setIsNew(true); };
  const openEdit = (p: Plan) => { setEditing({ ...p }); setIsNew(false); };

  const save = async () => {
    if (!editing?.name || editing.price == null) { toast.error('请填写套餐名称'); return; }
    setLoading(true);
    try {
      const method = isNew ? 'POST' : 'PUT';
      const url = isNew ? '/api/subscription/admin/plans' : `/api/subscription/admin/plans/${editing.id}`;
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(editing),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(isNew ? '创建成功' : '更新成功');
        setEditing(null);
        fetchPlans();
      } else {
        toast.error(data.msg || '操作失败');
      }
    } finally { setLoading(false); }
  };

  const deletePlan = async (id: number) => {
    if (!confirm('确认删除此套餐？')) return;
    const res = await fetch(`/api/subscription/plan/${id}`, {
      method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (data.code === 0) { toast.success('删除成功'); fetchPlans(); }
    else toast.error(data.msg || '删除失败');
  };

  const toggleEnabled = async (p: Plan) => {
    const res = await fetch(`/api/subscription/admin/plans/${p.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ enabled: !p.enabled }),
    });
    const data = await res.json();
    if (data.code === 0) fetchPlans();
    else toast.error(data.msg || '操作失败');
  };

  const field = (key: keyof Plan, label: string, type = 'text', placeholder = '') => (
    <div>
      <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>{label}</label>
      <Input
        size="sm"
        type={type}
        placeholder={placeholder}
        value={String(editing?.[key] ?? '')}
        onValueChange={v => setEditing(e => ({ ...e, [key]: type === 'number' ? Number(v) : v }))}
      />
    </div>
  );

  return (
    <div className="space-y-6">
      <PageHeader title="订阅套餐管理" description="创建和管理用户订阅套餐" />

      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Button color="primary" startContent={<Plus size={16} />} onPress={openNew}
          style={{ borderRadius: '12px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}>
          新建套餐
        </Button>
      </div>

      {/* Plan list */}
      <div className="space-y-3">
        {plans.map(p => (
          <div key={p.id} style={{
            borderRadius: '16px', background: 'var(--bg-surface)',
            border: '1px solid var(--border-color)', padding: '16px 20px',
            display: 'flex', alignItems: 'center', gap: '16px',
          }}>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                <span style={{ fontWeight: 600, color: 'var(--text-primary)', fontSize: '14px' }}>{p.name}</span>
                <span style={{
                  padding: '2px 8px', borderRadius: '99px', fontSize: '11px', fontWeight: 600,
                  background: 'rgba(124,58,237,0.12)', color: 'var(--accent-primary)',
                }}>{typeLabels[p.type] || p.type}</span>
                {!p.enabled && <span style={{ padding: '2px 8px', borderRadius: '99px', fontSize: '11px', background: 'rgba(248,113,113,0.12)', color: '#f87171' }}>已禁用</span>}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
                <span>¥{p.price.toFixed(2)}</span>
                <span>{p.duration_days > 0 ? `${p.duration_days}天` : '永久'}</span>
                {p.type === 'group' || p.type === 'combo' ? <span>分组: {p.group_name}</span> : null}
                {(p.type === 'quota' || p.type === 'combo') && p.quota > 0 ? <span>额度: {(p.quota / 500000).toFixed(2)} USD</span> : null}
                {(p.type === 'rpm' || p.type === 'combo') && p.rpm > 0 ? <span>RPM: {p.rpm}</span> : null}
                {p.description && <span style={{ color: 'var(--text-faint)' }}>{p.description}</span>}
              </div>
            </div>
            <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
              <Button size="sm" variant="flat" onPress={() => toggleEnabled(p)}
                style={{ borderRadius: '8px', minWidth: 0, padding: '0 10px' }}>
                {p.enabled ? <ToggleRight size={16} color="var(--accent-primary)" /> : <ToggleLeft size={16} />}
              </Button>
              <Button size="sm" variant="flat" onPress={() => openEdit(p)}
                style={{ borderRadius: '8px', minWidth: 0, padding: '0 10px' }}>
                <Pencil size={14} />
              </Button>
              <Button size="sm" variant="flat" onPress={() => deletePlan(p.id)}
                style={{ borderRadius: '8px', minWidth: 0, padding: '0 10px', color: '#f87171' }}>
                <Trash2 size={14} />
              </Button>
            </div>
          </div>
        ))}
        {plans.length === 0 && (
          <div style={{ textAlign: 'center', padding: '48px', color: 'var(--text-faint)', fontSize: '14px' }}>
            暂无套餐，点击「新建套餐」创建
          </div>
        )}
      </div>

      {/* Edit modal */}
      {editing && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 50,
          display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '16px',
        }} onClick={e => { if (e.target === e.currentTarget) setEditing(null); }}>
          <div style={{
            background: 'var(--bg-surface)', borderRadius: '20px',
            border: '1px solid var(--border-color)', padding: '28px',
            width: '100%', maxWidth: '520px', maxHeight: '90vh', overflowY: 'auto',
          }}>
            <h3 style={{ fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', marginBottom: '20px' }}>
              {isNew ? '新建套餐' : '编辑套餐'}
            </h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              {field('name', '套餐名称', 'text', '如：月度会员')}
              {field('price', '价格（CNY）', 'number', '9.9')}
              <div>
                <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>套餐类型</label>
                <select
                  value={editing.type || 'quota'}
                  onChange={e => setEditing(ed => ({ ...ed, type: e.target.value }))}
                  style={{
                    width: '100%', padding: '8px 12px', borderRadius: '10px', fontSize: '13px',
                    background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                    color: 'var(--text-primary)', outline: 'none',
                  }}
                >
                  <option value="quota">额度包</option>
                  <option value="group">分组订阅</option>
                  <option value="rpm">高速RPM</option>
                  <option value="combo">组合套餐</option>
                </select>
              </div>
              {field('duration_days', '有效天数（0=永久）', 'number', '30')}
              {(editing.type === 'group' || editing.type === 'combo') &&
                field('group_name', '目标分组名', 'text', 'vip')}
              {(editing.type === 'quota' || editing.type === 'combo') &&
                field('quota', '赠送额度（单位）', 'number', '500000')}
              {(editing.type === 'rpm' || editing.type === 'combo') &&
                field('rpm', 'RPM 上限', 'number', '120')}
              {field('sort_order', '排序（小的在前）', 'number', '0')}
            </div>
            <div style={{ marginTop: '12px' }}>
              <label style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginBottom: '4px' }}>描述</label>
              <textarea
                value={editing.description || ''}
                onChange={e => setEditing(ed => ({ ...ed, description: e.target.value }))}
                rows={2}
                placeholder="套餐描述（可选）"
                style={{
                  width: '100%', padding: '8px 12px', borderRadius: '10px', fontSize: '13px',
                  background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                  color: 'var(--text-primary)', outline: 'none', resize: 'vertical',
                }}
              />
            </div>
            <div style={{ display: 'flex', gap: '10px', marginTop: '20px', justifyContent: 'flex-end' }}>
              <Button variant="flat" onPress={() => setEditing(null)} style={{ borderRadius: '10px' }}>取消</Button>
              <Button color="primary" isLoading={loading} onPress={save}
                style={{ borderRadius: '10px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}>
                保存
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
