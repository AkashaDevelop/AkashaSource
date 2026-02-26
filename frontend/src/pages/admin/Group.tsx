import { useState, useEffect } from 'react';
import {
  Table, TableHeader, TableColumn, TableBody, TableRow, TableCell,
  Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Textarea, Chip,
} from '@heroui/react';
import { Plus, Edit, Trash2, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface Group {
  id: number;
  name: string;
  description: string;
  model_ratios: string;
  allowed_channels: string;
  qpm: number;
}

export default function GroupManagement() {
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(false);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editing, setEditing] = useState<Group | null>(null);
  const [formData, setFormData] = useState({
    name: '', description: '', model_ratios: '', allowed_channels: '', qpm: '0',
  });

  const fetchGroups = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/group', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (res.ok) setGroups(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchGroups(); }, []);

  const handleAdd = () => {
    setEditing(null);
    setFormData({ name: '', description: '', model_ratios: '', allowed_channels: '', qpm: '0' });
    onOpen();
  };

  const handleEdit = (g: Group) => {
    setEditing(g);
    setFormData({
      name: g.name, description: g.description,
      model_ratios: g.model_ratios, allowed_channels: g.allowed_channels,
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
      if (res.ok) { fetchGroups(); onClose(); }
      else alert('操作失败');
    } catch (e) { console.error(e); }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除此分组?')) return;
    try {
      const res = await fetch(`/api/group/${id}`, {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) fetchGroups();
    } catch (e) { console.error(e); }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">分组管理</h1>
          <p className="text-default-500">管理用户分组及其权限配置</p>
        </div>
        <div className="flex gap-2">
          <Button startContent={<RefreshCw size={18} />} onPress={fetchGroups} variant="flat">刷新</Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>添加分组</Button>
        </div>
      </div>

      <Table aria-label="Group table">
        <TableHeader>
          <TableColumn>ID</TableColumn>
          <TableColumn>名称</TableColumn>
          <TableColumn>描述</TableColumn>
          <TableColumn>QPM</TableColumn>
          <TableColumn>操作</TableColumn>
        </TableHeader>
        <TableBody emptyContent="暂无分组" isLoading={loading}>
          {groups.map((g) => (
            <TableRow key={g.id}>
              <TableCell>{g.id}</TableCell>
              <TableCell><Chip size="sm" variant="flat" color="primary">{g.name}</Chip></TableCell>
              <TableCell>{g.description || '-'}</TableCell>
              <TableCell>{g.qpm || '无限制'}</TableCell>
              <TableCell>
                <div className="flex gap-2">
                  <span className="cursor-pointer text-default-400" onClick={() => handleEdit(g)}><Edit size={18} /></span>
                  <span className="cursor-pointer text-danger" onClick={() => handleDelete(g.id)}><Trash2 size={18} /></span>
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
              <ModalHeader>{editing ? '编辑分组' : '添加分组'}</ModalHeader>
              <ModalBody className="gap-4">
                <Input label="分组名称" value={formData.name} onValueChange={(v) => setFormData({...formData, name: v})} isRequired />
                <Input label="描述" value={formData.description} onValueChange={(v) => setFormData({...formData, description: v})} />
                <Input label="QPM (每分钟请求数)" type="number" value={formData.qpm} onValueChange={(v) => setFormData({...formData, qpm: v})} description="0 表示不限制" />
                <Textarea label="模型倍率覆盖 (JSON)" placeholder='{"gpt-4": 2.0}' value={formData.model_ratios} onValueChange={(v) => setFormData({...formData, model_ratios: v})} minRows={2} />
                <Input label="允许渠道ID (逗号分隔)" placeholder="1,2,3" value={formData.allowed_channels} onValueChange={(v) => setFormData({...formData, allowed_channels: v})} />
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
