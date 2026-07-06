import { useState, useEffect, useMemo } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import {
  Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter,
  useDisclosure, Input, Select, SelectItem, Chip, Switch,
} from '../../components/ui';
import { Plus, Edit, Trash2, RefreshCw, ArrowUpDown, Search } from 'lucide-react';
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
  upstream_input_price: number;
  upstream_output_price: number;
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
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'enabled' | 'disabled'>('all');
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const { isOpen: isBatchOpen, onOpen: onBatchOpen, onOpenChange: onBatchOpenChange } = useDisclosure();
  const [batchRatio, setBatchRatio] = useState({ input: '1', output: '1' });
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

  const filteredModels = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    return models.filter((m) => {
      const hitKeyword = !normalizedKeyword
        || m.model_name.toLowerCase().includes(normalizedKeyword)
        || (m.display_name || '').toLowerCase().includes(normalizedKeyword);

      const hitStatus = statusFilter === 'all'
        || (statusFilter === 'enabled' && m.enabled)
        || (statusFilter === 'disabled' && !m.enabled);

      return hitKeyword && hitStatus;
    });
  }, [models, keyword, statusFilter]);

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

  const handleToggleEnabled = async (m: ModelConfig) => {
    setTogglingId(m.id);
    try {
      const res = await fetch('/api/model', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          id: m.id,
          model_name: m.model_name,
          display_name: m.display_name,
          category: m.category,
          input_ratio: m.input_ratio,
          output_ratio: m.output_ratio,
          max_context: m.max_context,
          enabled: !m.enabled,
        }),
      });
      const data = await res.json();
      if (res.ok) {
        toast.success(!m.enabled ? '模型已启用' : '模型已禁用');
        fetchModels();
      } else {
        toast.error(data?.error || '状态切换失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('状态切换失败');
    } finally {
      setTogglingId(null);
    }
  };

  const handleSyncPricing = async () => {
    setSyncing(true);
    try {
      const res = await fetch('/api/model/sync-upstream', {
        method: 'POST', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`同步成功：创建 ${data.data.created} 个，更新 ${data.data.updated} 个`);
        fetchModels();
      } else {
        toast.error(data.msg || '同步失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('同步请求失败');
    }
    finally { setSyncing(false); }
  };

  const handleBatchRatio = async () => {
    if (selectedIds.length === 0) {
      toast.error('请先选择模型');
      return;
    }
    try {
      const res = await fetch('/api/model/batch-ratio', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          ids: selectedIds,
          input_ratio: parseFloat(batchRatio.input),
          output_ratio: parseFloat(batchRatio.output),
        }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`已更新 ${selectedIds.length} 个模型的倍率`);
        setSelectedIds([]);
        fetchModels();
        onBatchOpenChange();
      } else {
        toast.error(data.msg || '批量更新失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('批量更新请求失败');
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="模型管理"
        description="管理模型配置与定价倍率"
        actions={
          <div className="flex gap-2 flex-wrap">
            <Input
              size="sm"
              className="w-44"
              placeholder="搜索模型名/显示名"
              value={keyword}
              onValueChange={setKeyword}
              startContent={<Search size={14} />}
            />
            <Select
              placeholder="状态" size="sm" className="w-28"
              selectedKeys={[statusFilter]}
              onSelectionChange={(keys) => setStatusFilter(([...keys][0] as 'all' | 'enabled' | 'disabled') || 'all')}
            >
              <SelectItem key="all">全部状态</SelectItem>
              <SelectItem key="enabled">仅启用</SelectItem>
              <SelectItem key="disabled">仅禁用</SelectItem>
            </Select>
            <Select
              placeholder="分类筛选" size="sm" className="w-32"
              selectedKeys={categoryFilter ? [categoryFilter] : []}
              onSelectionChange={(keys) => setCategoryFilter([...keys][0] as string || '')}
            >
              {CATEGORIES.map(c => <SelectItem key={c.key}>{c.label}</SelectItem>)}
            </Select>
            <Button startContent={<RefreshCw size={18} />} onPress={fetchModels} variant="flat">刷新</Button>
            {selectedIds.length > 0 && (
              <Button color="secondary" onPress={onBatchOpen}>
                批量设置倍率 ({selectedIds.length})
              </Button>
            )}
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
              <th style={{width: '40px'}}>
                <input
                  type="checkbox"
                  checked={selectedIds.length === filteredModels.length && filteredModels.length > 0}
                  onChange={(e) => setSelectedIds(e.target.checked ? filteredModels.map(m => m.id) : [])}
                />
              </th>
              <th>模型名称</th><th>显示名称</th><th>分类</th><th>输入倍率</th><th>输出倍率</th><th>输入价格</th><th>输出价格</th><th>上下文</th><th>状态</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={11} rows={5} />
            ) : filteredModels.length === 0 ? (
              <tr><td colSpan={11}><EmptyState icon="🤖" title="暂无匹配模型" /></td></tr>
            ) : filteredModels.map((m) => (
              <tr key={m.id}>
                <td>
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(m.id)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedIds([...selectedIds, m.id]);
                      } else {
                        setSelectedIds(selectedIds.filter(id => id !== m.id));
                      }
                    }}
                  />
                </td>
                <td className="font-mono text-sm">{m.model_name}</td>
                <td>{m.display_name || '-'}</td>
                <td><Chip size="sm" variant="flat" color="secondary">{CATEGORIES.find(c => c.key === m.category)?.label || m.category}</Chip></td>
                <td>{m.input_ratio}</td>
                <td>{m.output_ratio}</td>
                <td>
                  {m.upstream_input_price > 0 ? (
                    <span className="text-sm">¥{(m.upstream_input_price * m.input_ratio).toFixed(2)}/M</span>
                  ) : '-'}
                </td>
                <td>
                  {m.upstream_output_price > 0 ? (
                    <span className="text-sm">¥{(m.upstream_output_price * m.output_ratio).toFixed(2)}/M</span>
                  ) : '-'}
                </td>
                <td>{m.max_context.toLocaleString()}</td>
                <td>
                  <div className="flex items-center gap-2">
                    <Chip size="sm" color={m.enabled ? 'success' : 'default'} variant="flat">{m.enabled ? '启用' : '禁用'}</Chip>
                    <Button
                      size="sm"
                      variant="flat"
                      isLoading={togglingId === m.id}
                      onPress={() => handleToggleEnabled(m)}
                    >
                      {m.enabled ? '禁用' : '启用'}
                    </Button>
                  </div>
                </td>
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
                  onSelectionChange={(keys) => setFormData({...formData, category: [...keys][0] as string || 'chat'})}>
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

      <Modal isOpen={isBatchOpen} onOpenChange={onBatchOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>批量设置倍率</ModalHeader>
              <ModalBody className="gap-4">
                <p className="text-sm text-default-500">已选择 {selectedIds.length} 个模型</p>
                <Input label="输入倍率" type="number" value={batchRatio.input}
                  onValueChange={(v) => setBatchRatio({...batchRatio, input: v})} />
                <Input label="输出倍率" type="number" value={batchRatio.output}
                  onValueChange={(v) => setBatchRatio({...batchRatio, output: v})} />
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => { handleBatchRatio(); onClose(); }}>确定</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
