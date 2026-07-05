import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Chip, Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Select, SelectItem, Form, Pagination,
  Dropdown, DropdownTrigger, DropdownMenu, DropdownItem,
} from '../../components/ui';
import {
  Edit, Trash2, Plus, RefreshCw, Power, Key, PlusCircle, Search,
  MoreHorizontal, Link2, Crown, ShieldOff, Fingerprint, Ban, Unlink,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import { formatQuota, moneyToQuota } from '../../lib/quota';

interface User {
  id: number;
  username: string;
  display_name: string;
  role: number;
  status: number;
  quota: number;
  used_quota: number;
  group: string;
  email: string;
  totp_enabled?: boolean;
}

interface OAuthBinding {
  provider_id: number;
  provider_name: string;
  provider_slug: string;
  provider_icon: string;
  provider_user_id: string;
}

interface SubPlan {
  id: number; name: string; type: string; price: number;
  duration_days: number; group_name: string; quota: number; rpm: number; enabled: boolean;
}

interface UserSub {
  id: number; plan_id: number; status: number; started_at: number; expired_at: number;
  plan?: { name: string; type: string };
}

const ROLES = [
  { key: '1', label: '普通用户' },
  { key: '10', label: '管理员' },
  { key: '100', label: '超级管理员' },
];
const STATUS_OPTIONS = [
  { key: '1', label: '正常', color: 'success' },
  { key: '2', label: '封禁', color: 'danger' },
];
const SUB_STATUS: Record<number, { label: string; color: string }> = {
  0: { label: '待支付', color: 'var(--color-warning-fg)' },
  1: { label: '生效中', color: 'var(--color-success-fg)' },
  2: { label: '已过期', color: 'var(--text-muted)' },
  3: { label: '已取消', color: 'var(--color-danger-fg)' },
};
const PLAN_TYPE_LABEL: Record<string, string> = { group: '分组', quota: '额度', rpm: '限速', combo: '组合' };

export default function UserManagement() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const { token, user: currentUser } = useAuthStore();

  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const { isOpen: isQuotaOpen, onOpen: onQuotaOpen, onOpenChange: onQuotaOpenChange } = useDisclosure();
  const { isOpen: isBindOpen, onOpen: onBindOpen, onOpenChange: onBindOpenChange } = useDisclosure();
  const { isOpen: isSubOpen, onOpen: onSubOpen, onOpenChange: onSubOpenChange } = useDisclosure();

  const [editingUser, setEditingUser] = useState<Partial<User> | null>(null);
  const [quotaUser, setQuotaUser] = useState<User | null>(null);
  const [quotaDelta, setQuotaDelta] = useState('');

  // 绑定/订阅管理目标用户
  const [targetUser, setTargetUser] = useState<User | null>(null);
  const [bindings, setBindings] = useState<OAuthBinding[]>([]);
  const [bindLoading, setBindLoading] = useState(false);
  const [userSubs, setUserSubs] = useState<UserSub[]>([]);
  const [plans, setPlans] = useState<SubPlan[]>([]);
  const [subLoading, setSubLoading] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState('');

  const [formData, setFormData] = useState({
    username: '', display_name: '', password: '', role: '1', status: '1', quota: '0', group: 'default', email: '',
  });

  const fetchUsers = async () => {
    setLoading(true);
    try {
      let url = `/api/user?p=${page}&size=${pageSize}`;
      if (search.trim()) url = `/api/user/search?keyword=${encodeURIComponent(search.trim())}`;
      const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        const list = Array.isArray(data.data) ? data.data : data.data.data;
        setUsers(list);
        setTotal(Array.isArray(data.data) ? list.length : data.data.total);
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchUsers(); }, [page, pageSize]);
  useEffect(() => { if (!search.trim()) fetchUsers(); }, [search]);

  /* ── 编辑 / 新增 ── */
  const handleEdit = (u: User) => {
    setEditingUser(u);
    setFormData({
      username: u.username, display_name: u.display_name, password: '',
      role: u.role.toString(), status: u.status.toString(), quota: u.quota.toString(),
      group: u.group, email: u.email,
    });
    onOpen();
  };
  const handleAdd = () => {
    setEditingUser(null);
    setFormData({ username: '', display_name: '', password: '', role: '1', status: '1', quota: '0', group: 'default', email: '' });
    onOpen();
  };
  const handleSubmit = async (onClose: () => void) => {
    const body = { ...formData, id: editingUser?.id, role: parseInt(formData.role), status: parseInt(formData.status), quota: parseInt(formData.quota) };
    try {
      const res = await fetch('/api/user', {
        method: editingUser ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success(editingUser ? '已更新用户' : '已创建用户'); fetchUsers(); onClose(); }
      else toast.error(data.msg || '操作失败');
    } catch (error) { console.error(error); }
  };

  /* ── 删除 ── */
  const handleDelete = async (u: User) => {
    if (!await confirm({ title: '删除用户', message: `确定要删除用户「${u.username}」吗？此操作不可撤销。`, danger: true })) return;
    try {
      const res = await fetch(`/api/user/${u.id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) { toast.success('已删除用户'); fetchUsers(); }
      else toast.error(data.msg || '删除失败');
    } catch (error) { console.error(error); }
  };

  /* ── 额度 ── */
  const handleOpenQuota = (u: User) => { setQuotaUser(u); setQuotaDelta(''); onQuotaOpen(); };
  const handleAdjustQuota = async (onClose: () => void) => {
    if (!quotaUser || !quotaDelta) return;
    const delta = moneyToQuota(parseFloat(quotaDelta));
    try {
      const res = await fetch(`/api/user/${quotaUser.id}/quota`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ delta }),
      });
      const data = await res.json();
      if (data.code === 0) { setUsers(prev => prev.map(u => u.id === quotaUser.id ? { ...u, quota: data.data.quota } : u)); toast.success('额度已调整'); onClose(); }
      else toast.error(data.msg || '操作失败');
    } catch { toast.error('请求失败'); }
  };

  /* ── 启用/禁用 ── */
  const handleToggleStatus = async (u: User) => {
    try {
      const res = await fetch(`/api/user/${u.id}/status`, { method: 'PATCH', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        setUsers(prev => prev.map(x => x.id === u.id ? { ...x, status: data.data.status } : x));
        toast.success(data.data.status === 1 ? '已启用用户' : '已封禁用户');
      } else toast.error(data.msg || '操作失败');
    } catch { toast.error('操作失败'); }
  };

  /* ── 重置 2FA ── */
  const handleReset2FA = async (u: User) => {
    if (!await confirm({ title: '重置两步验证', message: `确定重置「${u.username}」的 2FA 吗？重置后该用户需重新绑定。`, danger: true })) return;
    try {
      const res = await fetch(`/api/user/${u.id}/2fa`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) { toast.success('已重置该用户 2FA'); fetchUsers(); }
      else toast.error(data.msg || '操作失败');
    } catch { toast.error('请求失败'); }
  };

  /* ── 重置 Passkey ── */
  const handleResetPasskey = async (u: User) => {
    if (!await confirm({ title: '重置 Passkey', message: `确定重置「${u.username}」的 Passkey 吗？重置后该用户需重新注册。`, danger: true })) return;
    try {
      const res = await fetch(`/api/user/${u.id}/passkey`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) { toast.success('已重置该用户 Passkey'); }
      else toast.error(data.msg || '操作失败');
    } catch { toast.error('请求失败'); }
  };

  /* ── 管理绑定 ── */
  const handleOpenBindings = async (u: User) => {
    setTargetUser(u); setBindings([]); onBindOpen();
    setBindLoading(true);
    try {
      const res = await fetch(`/api/user/${u.id}/oauth/bindings`, { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setBindings(data.data || []);
    } catch { toast.error('获取绑定失败'); }
    finally { setBindLoading(false); }
  };
  const handleUnbind = async (providerId: number) => {
    if (!targetUser) return;
    if (!await confirm({ title: '解除绑定', message: '确定解除该 OAuth 绑定吗？', danger: true })) return;
    try {
      const res = await fetch(`/api/user/${targetUser.id}/oauth/bindings/${providerId}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) { toast.success('已解绑'); setBindings(prev => prev.filter(b => b.provider_id !== providerId)); }
      else toast.error(data.msg || '解绑失败');
    } catch { toast.error('请求失败'); }
  };

  /* ── 管理订阅 ── */
  const handleOpenSubs = async (u: User) => {
    setTargetUser(u); setUserSubs([]); setSelectedPlan(''); onSubOpen();
    setSubLoading(true);
    try {
      const [sRes, pRes] = await Promise.all([
        fetch(`/api/subscription/admin/users/${u.id}/subscriptions`, { headers: { Authorization: `Bearer ${token}` } }),
        fetch('/api/subscription/admin/plans', { headers: { Authorization: `Bearer ${token}` } }),
      ]);
      const [sData, pData] = await Promise.all([sRes.json(), pRes.json()]);
      if (sData.code === 0) setUserSubs(sData.data || []);
      if (pData.code === 0) setPlans((pData.data || []).filter((p: SubPlan) => p.enabled));
    } catch { toast.error('获取订阅失败'); }
    finally { setSubLoading(false); }
  };
  const refreshSubs = async () => {
    if (!targetUser) return;
    const res = await fetch(`/api/subscription/admin/users/${targetUser.id}/subscriptions`, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code === 0) setUserSubs(data.data || []);
  };
  const handleCreateSub = async () => {
    if (!targetUser || !selectedPlan) { toast.warning('请选择套餐'); return; }
    try {
      const res = await fetch(`/api/subscription/admin/users/${targetUser.id}/subscriptions`, {
        method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ plan_id: parseInt(selectedPlan) }),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success('已为用户开通订阅'); setSelectedPlan(''); refreshSubs(); }
      else toast.error(data.msg || '开通失败');
    } catch { toast.error('请求失败'); }
  };
  const handleInvalidateSub = async (subId: number) => {
    if (!await confirm({ title: '作废订阅', message: '确定作废该订阅吗？用户权益将立即失效。', danger: true })) return;
    try {
      const res = await fetch(`/api/subscription/admin/user_subscriptions/${subId}/invalidate`, { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) { toast.success('已作废订阅'); refreshSubs(); }
      else toast.error(data.msg || '操作失败');
    } catch { toast.error('请求失败'); }
  };

  /* 是否可对该用户操作（不能操作比自己角色高/相同的，root 除外） */
  const canManage = (u: User) => {
    const myRole = currentUser?.role ?? 0;
    return myRole === 100 || myRole > u.role;
  };

  return (
    <div className="space-y-5">
      <PageHeader
        title="用户管理"
        description="管理系统所有用户账户"
        actions={
          <div className="flex gap-2 items-center">
            <Input
              placeholder="搜索用户名/邮箱" size="sm" value={search} onValueChange={setSearch}
              className="w-40" startContent={<Search size={14} />}
              onKeyDown={(e) => e.key === 'Enter' && fetchUsers()}
            />
            <Button startContent={<RefreshCw size={16} />} onPress={fetchUsers} variant="flat" size="sm">刷新</Button>
            <Button startContent={<Plus size={16} />} color="primary" onPress={handleAdd} size="sm">添加用户</Button>
          </div>
        }
      />

      <div className="data-table-wrap" style={{ overflow: 'visible' }}>
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th><th>用户名</th><th>角色</th><th>状态</th><th>额度</th><th>已用额度</th><th>分组</th><th>安全</th><th style={{ textAlign: 'right' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={9} rows={5} />
            ) : users.length === 0 ? (
              <tr><td colSpan={9}><EmptyState icon="👤" title="暂无用户" /></td></tr>
            ) : users.map((u) => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>
                  <div className="font-medium">{u.username}</div>
                  {u.display_name && <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{u.display_name}</div>}
                </td>
                <td><Chip size="sm" variant="flat" color={u.role >= 100 ? 'secondary' : u.role >= 10 ? 'primary' : 'default'}>{ROLES.find(r => r.key === u.role.toString())?.label || '未知'}</Chip></td>
                <td><Chip size="sm" color={u.status === 1 ? 'success' : 'danger'} startContent={<Power size={12} />} className="cursor-pointer" onClick={() => canManage(u) && handleToggleStatus(u)}>{u.status === 1 ? '正常' : '封禁'}</Chip></td>
                <td>{formatQuota(u.quota, 2)}</td>
                <td>{formatQuota(u.used_quota, 2)}</td>
                <td><Chip size="sm" variant="dot">{u.group}</Chip></td>
                <td>
                  {u.totp_enabled
                    ? <Chip size="sm" variant="flat" color="success">2FA</Chip>
                    : <span style={{ fontSize: '12px', color: 'var(--text-faint)' }}>—</span>}
                </td>
                <td>
                  <div className="flex items-center justify-end gap-1">
                    {/* 快捷：调整额度 */}
                    <button onClick={() => handleOpenQuota(u)} title="调整额度"
                      style={{ padding: '5px', borderRadius: '7px', border: 'none', background: 'transparent', color: 'var(--text-muted)', cursor: 'pointer', display: 'flex' }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'var(--nav-hover-bg)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
                      <PlusCircle size={16} />
                    </button>

                    {/* 更多操作下拉 */}
                    <Dropdown placement="bottom-end">
                      <DropdownTrigger>
                        <button title="更多操作"
                          style={{ padding: '5px', borderRadius: '7px', border: 'none', background: 'transparent', color: 'var(--text-muted)', cursor: 'pointer', display: 'flex' }}
                          onMouseEnter={e => (e.currentTarget.style.background = 'var(--nav-hover-bg)')}
                          onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}>
                          <MoreHorizontal size={16} />
                        </button>
                      </DropdownTrigger>
                      <DropdownMenu ariaLabel="用户操作">
                        <DropdownItem key="edit" startContent={<Edit size={15} />} onPress={() => handleEdit(u)}>编辑</DropdownItem>
                        <DropdownItem key="status" startContent={u.status === 1 ? <Ban size={15} /> : <Power size={15} />}
                          isDisabled={!canManage(u)} showDivider
                          onPress={() => handleToggleStatus(u)}>
                          {u.status === 1 ? '封禁' : '启用'}
                        </DropdownItem>
                        <DropdownItem key="bindings" startContent={<Link2 size={15} />} onPress={() => handleOpenBindings(u)}>管理绑定</DropdownItem>
                        <DropdownItem key="subs" startContent={<Crown size={15} />} showDivider onPress={() => handleOpenSubs(u)}>管理订阅</DropdownItem>
                        <DropdownItem key="passkey" startContent={<Fingerprint size={15} />}
                          isDisabled={!canManage(u)} onPress={() => handleResetPasskey(u)}>重置 Passkey</DropdownItem>
                        <DropdownItem key="2fa" startContent={<ShieldOff size={15} />} showDivider
                          isDisabled={!canManage(u) || !u.totp_enabled} onPress={() => handleReset2FA(u)}>重置 2FA</DropdownItem>
                        <DropdownItem key="delete" color="danger" startContent={<Trash2 size={15} />}
                          isDisabled={!canManage(u)} onPress={() => handleDelete(u)}>删除</DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 分页 */}
      <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
        <span className="text-sm" style={{ color: 'var(--text-muted)' }}>
          共 <strong style={{ color: 'var(--text-primary)' }}>{total}</strong> 位用户，第 {page}/{Math.ceil(total / pageSize) || 1} 页
        </span>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <span className="text-xs" style={{ color: 'var(--text-muted)' }}>每页</span>
            <select className="akasha-select" value={pageSize} onChange={e => { setPageSize(Number(e.target.value)); setPage(1); }}>
              {[10, 20, 50].map(n => <option key={n} value={n}>{n} 条</option>)}
            </select>
          </div>
          <Pagination page={page} total={Math.ceil(total / pageSize) || 1} onChange={setPage} />
        </div>
      </div>

      {/* ── 编辑/新增 Modal ── */}
      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="2xl">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editingUser ? '编辑用户' : '添加新用户'}</ModalHeader>
              <ModalBody>
                <Form className="grid grid-cols-2 gap-4">
                  <Input label="用户名" placeholder="john_doe" value={formData.username} onValueChange={(v) => setFormData({ ...formData, username: v })} isRequired />
                  <Input label="显示名称" placeholder="John Doe" value={formData.display_name} onValueChange={(v) => setFormData({ ...formData, display_name: v })} />
                  <Input label="密码" placeholder={editingUser ? '留空则不修改' : '必须填写'} type="password" value={formData.password} onValueChange={(v) => setFormData({ ...formData, password: v })} startContent={<Key className="text-default-400" size={16} />} />
                  <Input label="邮箱" type="email" value={formData.email} onValueChange={(v) => setFormData({ ...formData, email: v })} />
                  <Select label="角色" selectedKeys={[formData.role]} onSelectionChange={(keys) => setFormData({ ...formData, role: [...keys][0] as string || '1' })}>
                    {ROLES.map((r) => <SelectItem key={r.key}>{r.label}</SelectItem>)}
                  </Select>
                  <Select label="状态" selectedKeys={[formData.status]} onSelectionChange={(keys) => setFormData({ ...formData, status: [...keys][0] as string || '1' })}>
                    {STATUS_OPTIONS.map((s) => <SelectItem key={s.key}>{s.label}</SelectItem>)}
                  </Select>
                  <Input label="额度 (Raw)" type="number" value={formData.quota} onValueChange={(v) => setFormData({ ...formData, quota: v })} description={formatQuota(parseInt(formData.quota || '0'), 2)} />
                  <Input label="分组" value={formData.group} onValueChange={(v) => setFormData({ ...formData, group: v })} />
                </Form>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleSubmit(onClose)}>保存</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* ── 额度 Modal ── */}
      <Modal isOpen={isQuotaOpen} onOpenChange={onQuotaOpenChange} size="sm">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>调整额度 — {quotaUser?.username}</ModalHeader>
              <ModalBody>
                <p className="text-sm text-default-500">当前余额: {formatQuota(quotaUser?.quota ?? 0, 2)}</p>
                <Input label="调整金额 ($)" type="number" placeholder="正数增加，负数扣减" value={quotaDelta} onValueChange={setQuotaDelta}
                  description={quotaDelta ? `= ${moneyToQuota(parseFloat(quotaDelta))} quota 单位` : ''} />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleAdjustQuota(onClose)}>确认</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* ── 管理绑定 Modal ── */}
      <Modal isOpen={isBindOpen} onOpenChange={onBindOpenChange} size="md">
        <ModalContent>
          {() => (
            <>
              <ModalHeader>OAuth 绑定 — {targetUser?.username}</ModalHeader>
              <ModalBody className="pb-4">
                {bindLoading ? (
                  <div style={{ padding: '32px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>加载中...</div>
                ) : bindings.length === 0 ? (
                  <div style={{ padding: '32px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>该用户暂无 OAuth 绑定</div>
                ) : (
                  <div className="space-y-2">
                    {bindings.map(b => (
                      <div key={b.provider_id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                        <div className="flex items-center gap-2.5 min-w-0">
                          {b.provider_icon
                            ? <img src={b.provider_icon} alt="" style={{ width: '24px', height: '24px', borderRadius: '6px' }} />
                            : <div style={{ width: '24px', height: '24px', borderRadius: '6px', background: 'var(--nav-active-bg)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px' }}>🔗</div>}
                          <div className="min-w-0">
                            <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>{b.provider_name}</p>
                            <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>ID: {b.provider_user_id}</p>
                          </div>
                        </div>
                        <Button size="sm" variant="flat" color="danger" startContent={<Unlink size={13} />} onPress={() => handleUnbind(b.provider_id)}>解绑</Button>
                      </div>
                    ))}
                  </div>
                )}
              </ModalBody>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* ── 管理订阅 Modal ── */}
      <Modal isOpen={isSubOpen} onOpenChange={onSubOpenChange} size="lg">
        <ModalContent>
          {() => (
            <>
              <ModalHeader>订阅管理 — {targetUser?.username}</ModalHeader>
              <ModalBody className="pb-4">
                {/* 开通新订阅 */}
                <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-end', paddingBottom: '12px', borderBottom: '1px solid var(--border-color)' }}>
                  <Select label="选择套餐" className="flex-1" selectedKeys={selectedPlan ? [selectedPlan] : []} onSelectionChange={(k) => setSelectedPlan([...k][0] as string || '')}>
                    {plans.map(p => <SelectItem key={String(p.id)}>{`${p.name} · ${PLAN_TYPE_LABEL[p.type] || p.type} · ¥${p.price}`}</SelectItem>)}
                  </Select>
                  <Button color="primary" startContent={<Plus size={14} />} onPress={handleCreateSub}>开通</Button>
                </div>

                {/* 当前订阅列表 */}
                {subLoading ? (
                  <div style={{ padding: '32px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>加载中...</div>
                ) : userSubs.length === 0 ? (
                  <div style={{ padding: '32px', textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px' }}>该用户暂无订阅记录</div>
                ) : (
                  <div className="space-y-2">
                    {userSubs.map(s => {
                      const st = SUB_STATUS[s.status] || SUB_STATUS[2];
                      return (
                        <div key={s.id} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
                          <div className="min-w-0">
                            <div className="flex items-center gap-2">
                              <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{s.plan?.name || `套餐 #${s.plan_id}`}</span>
                              <span style={{ fontSize: '11px', padding: '1px 7px', borderRadius: 'var(--radius-full)', background: 'var(--bg-base)', color: st.color, fontWeight: 600 }}>{st.label}</span>
                            </div>
                            <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: '2px 0 0' }}>
                              {s.plan?.type ? `${PLAN_TYPE_LABEL[s.plan.type] || s.plan.type} · ` : ''}
                              {s.expired_at ? `到期 ${new Date(s.expired_at * 1000).toLocaleDateString()}` : '永久'}
                            </p>
                          </div>
                          {(s.status === 0 || s.status === 1) && (
                            <Button size="sm" variant="flat" color="danger" onPress={() => handleInvalidateSub(s.id)}>作废</Button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </ModalBody>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
