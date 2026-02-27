import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Select, SelectItem, Chip, Switch,
} from '../../components/ui';
import { Plus, Edit, Trash2, RefreshCw, ArrowUpDown } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface ModelConfig {
  id: number;
  model_name: string;
  display_name: string;
  category: string;
  input_ratio: number;
  output_ratio: number;
  max_context: number;
  enabled: boolean;
}

const CATEGORIES = [
  { key: 'chat', label: '对话' },
  { key: 'embedding', label: '嵌入' },
  { key: 'image', label: '图像' },
  { key: 'audio', label: '音频' },
  { key: 'rerank', label: '重排序' },
  { key: 'other', label: '其他' },
];

export default function ModelManagement() {
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editing, setEditing] = useState<ModelConfig | null>(null);
  const [categoryFilter, setCategoryFilter] = useState('');
  const [formData, setFormData] = useState({
    model_name: '', display_name: '', category: 'chat',
    input_ratio: '1', output_ratio: '1', max_context: '4096', enabled: true,
  });

  const fetchModels = async () => {
    setLoading(true);
    try {
      const params = categoryFilter ? `?category=${categoryFilter}` : '';
      const res = await fetch(`/api/model${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) setModels(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchModels(); }, [categoryFilter]);

  const handleAdd = () => {
    setEditing(null);
    setFormData({
      model_name: '', display_name: '', category: 'chat',
      input_ratio: '1', output_ratio: '1', max_context: '4096', enabled: true,
    });
    onOpen();
  };

  const handleEdit = (m: ModelConfig) => {
    setEditing(m);
    setFormData({
      model_name: m.model_name, display_name: m.display_name,
      category: m.category, input_ratio: m.input_ratio.toString(),
      output_ratio: m.output_ratio.toString(), max_context: m.max_context.toString(),
      enabled: m.enabled,
    });
    onOpen();
  };

  const handleSubmit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = {
      ...formData, id: editing?.id,
      input_ratio: parseFloat(formData.input_ratio) || 1,
      output_ratio: parseFloat(formData.output_ratio) || 1,
      max_context: parseInt(formData.max_context) || 4096,
    };
    try {
      const res = await fetch('/api/model', {
        method, headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(body),
      });
      if (res.ok) { fetchModels(); onClose(); }
      else toast.error('操作失败');
    } catch (e) { console.error(e); }
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({ title: '删除模型', message: '确定删除此模型配置？', danger: true })) return;
    try {
      const res = await fetch(`/api/model/${id}`, {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) fetchModels();
    } catch (e) { console.error(e); }
  };

  const handleSyncPricing = async () => {
    setSyncing(true);
    try {
      const res = await fetch('/api/model/sync-pricing', {
        method: 'POST', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) toast.success(data.message || '同步成功');
      else toast.error(data.error || '同步失败');
    } catch (e) { console.error(e); }
    finally { setSyncing(false); }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="模型管理"
        description="管理模型配置与定价倍率"
        actions={
          <div className="flex gap-2">
            <Select
              placeholder="分类筛选" size="sm" className="w-32"
              selectedKeys={categoryFilter ? [categoryFilter] : []}
              onChange={(e) => setCategoryFilter(e.target.value)}
            >
              {CATEGORIES.map(c => <SelectItem key={c.key}>{c.label}</SelectItem>)}
            </Select>
            <Button startContent={<RefreshCw size={18} />} onPress={fetchModels} variant="flat">刷新</Button>
            <Button startContent={<ArrowUpDown size={18} />} onPress={handleSyncPricing} variant="flat" color="warning" isLoading={syncing}>
              同步定价
            </Button>
            <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>添加模型</Button>
          </div>
        }
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>模型名称</th><th>显示名称</th><th>分类</th><th>输入倍率</th><th>输出倍率</th><th>上下文</th><th>状态</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={8} rows={5} />
            ) : models.length === 0 ? (
              <tr><td colSpan={8}><EmptyState icon="🤖" title="暂无模型映射" /></td></tr>
            ) : models.map((m) => (
              <tr key={m.id}>
                <td className="font-mono text-sm">{m.model_name}</td>
                <td>{m.display_name || '-'}</td>
                <td><Chip size="sm" variant="flat" color="secondary">{CATEGORIES.find(c => c.key === m.category)?.label || m.category}</Chip></td>
                <td>{m.input_ratio}</td>
                <td>{m.output_ratio}</td>
                <td>{m.max_context.toLocaleString()}</td>
                <td><Chip size="sm" color={m.enabled ? 'success' : 'default'} variant="flat">{m.enabled ? '启用' : '禁用'}</Chip></td>
                <td>
                  <div className="flex gap-2">
                    <span className="cursor-pointer text-default-400" onClick={() => handleEdit(m)}><Edit size={18} /></span>
                    <span className="cursor-pointer text-danger" onClick={() => handleDelete(m.id)}><Trash2 size={18} /></span>
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
              <ModalHeader>{editing ? '编辑模型' : '添加模型'}</ModalHeader>
              <ModalBody className="gap-4">
                <Input label="模型名称" placeholder="gpt-4" value={formData.model_name}
                  onValueChange={(v) => setFormData({...formData, model_name: v})} isRequired />
                <Input label="显示名称" placeholder="GPT-4" value={formData.display_name}
                  onValueChange={(v) => setFormData({...formData, display_name: v})} />
                <Select label="分类" selectedKeys={[formData.category]}
                  onChange={(e) => setFormData({...formData, category: e.target.value})}>
                  {CATEGORIES.map(c => <SelectItem key={c.key}>{c.label}</SelectItem>)}
                </Select>
                <div className="grid grid-cols-2 gap-4">
                  <Input label="输入倍率" type="number" value={formData.input_ratio}
                    onValueChange={(v) => setFormData({...formData, input_ratio: v})} />
                  <Input label="输出倍率" type="number" value={formData.output_ratio}
                    onValueChange={(v) => setFormData({...formData, output_ratio: v})} />
                </div>
                <Input label="最大上下文" type="number" value={formData.max_context}
                  onValueChange={(v) => setFormData({...formData, max_context: v})} />
                <Switch isSelected={formData.enabled}
                  onValueChange={(v) => setFormData({...formData, enabled: v})}>
                  启用
                </Switch>
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
