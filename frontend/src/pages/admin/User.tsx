import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Chip,
  Tooltip,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Select,
  SelectItem,
  Form,
  Pagination,
} from '../../components/ui';
import { Edit, Trash2, Plus, RefreshCw, Power, Key, PlusCircle, Search } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

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

export default function UserManagement() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const { isOpen: isQuotaOpen, onOpen: onQuotaOpen, onOpenChange: onQuotaOpenChange } = useDisclosure();
  const [editingUser, setEditingUser] = useState<Partial<User> | null>(null);
  const [quotaUser, setQuotaUser] = useState<User | null>(null);
  const [quotaDelta, setQuotaDelta] = useState('');

  const [formData, setFormData] = useState({
    username: '',
    display_name: '',
    password: '',
    role: '1',
    status: '1',
    quota: '0',
    group: 'default',
    email: '',
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

  const handleEdit = (user: User) => {
    setEditingUser(user);
    setFormData({
      username: user.username,
      display_name: user.display_name,
      password: '', // Don't show password
      role: user.role.toString(),
      status: user.status.toString(),
      quota: user.quota.toString(),
      group: user.group,
      email: user.email,
    });
    onOpen();
  };

  const handleAdd = () => {
    setEditingUser(null);
    setFormData({
      username: '',
      display_name: '',
      password: '',
      role: '1',
      status: '1',
      quota: '0',
      group: 'default',
      email: '',
    });
    onOpen();
  };

  const handleSubmit = async (onClose: () => void) => {
    const url = '/api/user';
    const method = editingUser ? 'PUT' : 'POST';
    
    const body = {
      ...formData,
      id: editingUser?.id,
      role: parseInt(formData.role),
      status: parseInt(formData.status),
      quota: parseInt(formData.quota),
    };

    try {
      const res = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      });

      const data = await res.json();
      if (data.code === 0) {
        fetchUsers();
        onClose();
      } else {
        toast.error('操作失败');
      }
    } catch (error) {
      console.error('Operation error:', error);
    }
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({ title: '删除用户', message: '确定要删除这个用户吗？此操作不可撤销。', danger: true })) return;
    try {
      const res = await fetch(`/api/user/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) fetchUsers();
    } catch (error) {
      console.error('Delete error:', error);
    }
  };

  const handleOpenQuota = (user: User) => {
    setQuotaUser(user);
    setQuotaDelta('');
    onQuotaOpen();
  };

  const handleToggleStatus = async (userId: number) => {
    try {
      const res = await fetch(`/api/user/${userId}/status`, {
        method: 'PATCH',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setUsers(prev => prev.map(u => u.id === userId ? { ...u, status: data.data.status } : u));
      }
    } catch {
      toast.error('操作失败');
    }
  };

  const handleAdjustQuota = async (onClose: () => void) => {
    if (!quotaUser || !quotaDelta) return;
    const delta = Math.round(parseFloat(quotaDelta) * 500000);
    try {
      const res = await fetch(`/api/user/${quotaUser.id}/quota`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ delta }),
      });
      const data = await res.json();
      if (data.code === 0) {
        setUsers(prev => prev.map(u => u.id === quotaUser.id ? { ...u, quota: data.data.quota } : u));
        toast.success(`额度已调整`);
        onClose();
      } else {
        toast.error(data.msg || '操作失败');
      }
    } catch {
      toast.error('请求失败');
    }
  };

  const renderQuota = (quota: number) => {
    return `$${(quota / 500000).toFixed(2)}`;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="用户管理"
        description="管理系统所有用户账户"
        actions={
          <div className="flex gap-2">
            <Input
              placeholder="搜索用户名/邮箱"
              size="sm"
              value={search}
              onValueChange={setSearch}
              className="w-40"
              startContent={<Search size={14} />}
              onKeyDown={(e) => e.key === 'Enter' && fetchUsers()}
            />
            <Button startContent={<RefreshCw size={18} />} onPress={fetchUsers} variant="flat">刷新</Button>
            <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>添加用户</Button>
          </div>
        }
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th><th>用户名</th><th>显示名称</th><th>角色</th><th>状态</th><th>额度</th><th>已用额度</th><th>分组</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={9} rows={5} />
            ) : users.length === 0 ? (
              <tr><td colSpan={9}><EmptyState icon="👤" title="暂无用户" /></td></tr>
            ) : users.map((user) => (
              <tr key={user.id}>
                <td>{user.id}</td>
                <td className="font-medium">{user.username}</td>
                <td>{user.display_name}</td>
                <td><Chip size="sm" variant="flat" color={user.role >= 10 ? "secondary" : "default"}>{ROLES.find(r => r.key === user.role.toString())?.label || 'Unknown'}</Chip></td>
                <td><Chip size="sm" color={user.status === 1 ? "success" : "danger"} startContent={<Power size={12} />} className="cursor-pointer" onClick={() => handleToggleStatus(user.id)}>{user.status === 1 ? "正常" : "封禁"}</Chip></td>
                <td>{renderQuota(user.quota)}</td>
                <td>{renderQuota(user.used_quota)}</td>
                <td><Chip size="sm" variant="dot">{user.group}</Chip></td>
                <td>
                  <div className="flex items-center gap-2">
                    <Tooltip content="调整额度"><span className="text-lg text-default-400 cursor-pointer active:opacity-50" onClick={() => handleOpenQuota(user)}><PlusCircle size={18} /></span></Tooltip>
                    <Tooltip content="编辑"><span className="text-lg text-default-400 cursor-pointer active:opacity-50" onClick={() => handleEdit(user)}><Edit size={18} /></span></Tooltip>
                    <Tooltip content="删除"><span className="text-lg text-danger cursor-pointer active:opacity-50" onClick={() => handleDelete(user.id)}><Trash2 size={18} /></span></Tooltip>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
        <span className="text-sm" style={{ color: 'var(--text-muted)' }}>
          共 <strong style={{ color: 'var(--text-primary)' }}>{total}</strong> 位用户，
          第 {page}/{Math.ceil(total / pageSize) || 1} 页
        </span>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <span className="text-xs" style={{ color: 'var(--text-muted)' }}>每页</span>
            <select
              className="akasha-select"
              value={pageSize}
              onChange={e => { setPageSize(Number(e.target.value)); setPage(1); }}
            >
              {[10, 20, 50].map(n => <option key={n} value={n}>{n} 条</option>)}
            </select>
          </div>
          <Pagination page={page} total={Math.ceil(total / pageSize) || 1} onChange={(p) => setPage(p)} />
        </div>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="2xl">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>
                {editingUser ? '编辑用户' : '添加新用户'}
              </ModalHeader>
              <ModalBody>
                <Form className="grid grid-cols-2 gap-4">
                  <Input
                    label="用户名"
                    placeholder="john_doe"
                    value={formData.username}
                    onValueChange={(v) => setFormData({...formData, username: v})}
                    isRequired
                  />
                  <Input
                    label="显示名称"
                    placeholder="John Doe"
                    value={formData.display_name}
                    onValueChange={(v) => setFormData({...formData, display_name: v})}
                  />
                  <Input
                    label="密码"
                    placeholder={editingUser ? "留空则不修改" : "必须填写"}
                    type="password"
                    value={formData.password}
                    onValueChange={(v) => setFormData({...formData, password: v})}
                    startContent={<Key className="text-default-400" size={16} />}
                  />
                  <Input
                    label="邮箱"
                    type="email"
                    value={formData.email}
                    onValueChange={(v) => setFormData({...formData, email: v})}
                  />
                  <Select
                    label="角色"
                    selectedKeys={[formData.role]}
                    onSelectionChange={(keys) => setFormData({...formData, role: [...keys][0] as string || '1'})}
                  >
                    {ROLES.map((role) => (
                      <SelectItem key={role.key}>{role.label}</SelectItem>
                    ))}
                  </Select>
                  <Select
                    label="状态"
                    selectedKeys={[formData.status]}
                    onSelectionChange={(keys) => setFormData({...formData, status: [...keys][0] as string || '1'})}
                  >
                    {STATUS_OPTIONS.map((status) => (
                      <SelectItem key={status.key}>{status.label}</SelectItem>
                    ))}
                  </Select>
                  <Input
                    label="额度 (Raw)"
                    type="number"
                    value={formData.quota}
                    onValueChange={(v) => setFormData({...formData, quota: v})}
                    description={`$${(parseInt(formData.quota || '0') / 500000).toFixed(2)}`}
                  />
                  <Input
                    label="分组"
                    value={formData.group}
                    onValueChange={(v) => setFormData({...formData, group: v})}
                  />
                </Form>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={() => handleSubmit(onClose)}>
                  保存
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
      <Modal isOpen={isQuotaOpen} onOpenChange={onQuotaOpenChange} size="sm">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>调整额度 — {quotaUser?.username}</ModalHeader>
              <ModalBody>
                <p className="text-sm text-default-500">当前余额: {renderQuota(quotaUser?.quota ?? 0)}</p>
                <Input
                  label="调整金额 ($)"
                  type="number"
                  placeholder="正数增加，负数扣减"
                  value={quotaDelta}
                  onValueChange={setQuotaDelta}
                  description={quotaDelta ? `= ${Math.round(parseFloat(quotaDelta) * 500000)} quota 单位` : ''}
                />
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleAdjustQuota(onClose)}>确认</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
