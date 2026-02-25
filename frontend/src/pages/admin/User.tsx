import { useState, useEffect } from 'react';
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
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
} from '@heroui/react';
import { Edit, Trash2, Plus, RefreshCw, Power, Key } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

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
  const [total, setTotal] = useState(0);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingUser, setEditingUser] = useState<Partial<User> | null>(null);

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
      const res = await fetch(`/api/user?p=${page}&size=10`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setUsers(data.data);
        setTotal(data.total);
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, [page]);

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

      if (res.ok) {
        fetchUsers();
        onClose();
      } else {
        alert('Operation failed');
      }
    } catch (error) {
      console.error('Operation error:', error);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除这个用户吗?')) return;
    try {
      const res = await fetch(`/api/user/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) fetchUsers();
    } catch (error) {
      console.error('Delete error:', error);
    }
  };

  const renderQuota = (quota: number) => {
    return `$${(quota / 500000).toFixed(2)}`;
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">用户管理</h1>
          <p className="text-default-500">管理系统所有用户账户</p>
        </div>
        <div className="flex gap-2">
          <Button startContent={<RefreshCw size={18} />} onPress={fetchUsers} variant="flat">
            刷新
          </Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>
            添加用户
          </Button>
        </div>
      </div>

      <Table 
        aria-label="User table"
        bottomContent={
          <div className="flex w-full justify-center">
            <Pagination
              isCompact
              showControls
              showShadow
              color="primary"
              page={page}
              total={Math.ceil(total / 10) || 1}
              onChange={(page) => setPage(page)}
            />
          </div>
        }
      >
        <TableHeader>
          <TableColumn>ID</TableColumn>
          <TableColumn>用户名</TableColumn>
          <TableColumn>显示名称</TableColumn>
          <TableColumn>角色</TableColumn>
          <TableColumn>状态</TableColumn>
          <TableColumn>额度</TableColumn>
          <TableColumn>已用额度</TableColumn>
          <TableColumn>分组</TableColumn>
          <TableColumn>操作</TableColumn>
        </TableHeader>
        <TableBody emptyContent="暂无用户" isLoading={loading}>
          {users.map((user) => (
            <TableRow key={user.id}>
              <TableCell>{user.id}</TableCell>
              <TableCell className="font-medium">{user.username}</TableCell>
              <TableCell>{user.display_name}</TableCell>
              <TableCell>
                <Chip size="sm" variant="flat" color={user.role >= 10 ? "secondary" : "default"}>
                  {ROLES.find(r => r.key === user.role.toString())?.label || 'Unknown'}
                </Chip>
              </TableCell>
              <TableCell>
                <Chip 
                  size="sm" 
                  color={user.status === 1 ? "success" : "danger"}
                  startContent={<Power size={12} />}
                >
                  {user.status === 1 ? "正常" : "封禁"}
                </Chip>
              </TableCell>
              <TableCell>{renderQuota(user.quota)}</TableCell>
              <TableCell>{renderQuota(user.used_quota)}</TableCell>
              <TableCell>
                <Chip size="sm" variant="dot">{user.group}</Chip>
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Tooltip content="编辑">
                    <span 
                      className="text-lg text-default-400 cursor-pointer active:opacity-50"
                      onClick={() => handleEdit(user)}
                    >
                      <Edit size={18} />
                    </span>
                  </Tooltip>
                  <Tooltip color="danger" content="删除">
                    <span 
                      className="text-lg text-danger cursor-pointer active:opacity-50"
                      onClick={() => handleDelete(user.id)}
                    >
                      <Trash2 size={18} />
                    </span>
                  </Tooltip>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

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
                    onChange={(e) => setFormData({...formData, role: e.target.value})}
                  >
                    {ROLES.map((role) => (
                      <SelectItem key={role.key}>{role.label}</SelectItem>
                    ))}
                  </Select>
                  <Select 
                    label="状态" 
                    selectedKeys={[formData.status]}
                    onChange={(e) => setFormData({...formData, status: e.target.value})}
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
    </div>
  );
}
