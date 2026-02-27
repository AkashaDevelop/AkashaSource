import { useState, useEffect } from 'react';
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
  Textarea,
  Tabs,
  Tab,
} from '../../components/ui';
import { Edit, Trash2, Plus, RefreshCw, Power, Activity, ArrowRight, Upload, Zap, Download } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Channel {
  id: number;
  name: string;
  type: number;
  key: string;
  base_url: string;
  models: string;
  group: string;
  model_mapping: string;
  tags: string;
  priority: number;
  weight: number;
  status: number;
  response_time: number;
  balance: number;
}

const CHANNEL_TYPES = [
  { key: '1', label: 'OpenAI' },
  { key: '3', label: 'Azure' },
  { key: '8', label: 'Custom' },
  { key: '14', label: 'Claude' },
  { key: '18', label: 'Gemini' },
  { key: '30', label: 'Midjourney' },
  { key: '40', label: '通义千问' },
  { key: '41', label: '混元' },
  { key: '42', label: '文心一言' },
  { key: '43', label: '讯飞星火' },
  { key: '44', label: 'Deepseek' },
  { key: '45', label: '智谱 ChatGLM' },
  { key: '46', label: 'Moonshot' },
  { key: '47', label: 'Ollama' },
  { key: '50', label: 'Suno' },
  { key: '51', label: 'Dify' },
];

export default function ChannelManagement() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);
  const [testingId, setTestingId] = useState<number | null>(null);
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const { isOpen: isBatchOpen, onOpen: onBatchOpen, onOpenChange: onBatchOpenChange } = useDisclosure();
  const [editingChannel, setEditingChannel] = useState<Partial<Channel> | null>(null);
  const [batchText, setBatchText] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [batchTesting, setBatchTesting] = useState(false);
  const [fetchingModels, setFetchingModels] = useState<number | null>(null);

  // Form State
  const [formData, setFormData] = useState({
    name: '',
    type: '1',
    key: '',
    base_url: '',
    models: '',
    group: 'default',
    priority: '10',
    weight: '1',
    tags: '',
  });
  
  // Model Mapping State (Visual)
  const [modelMapping, setModelMapping] = useState<{from: string, to: string}[]>([]);
  const [newMapping, setNewMapping] = useState({from: '', to: ''});

  const fetchChannels = async () => {
    setLoading(true);
    try {
      const params = tagFilter ? `?tag=${encodeURIComponent(tagFilter)}` : '';
      const res = await fetch(`/api/channel${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setChannels(data.data);
      }
    } catch (error) {
      console.error('Failed to fetch channels:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async (id: number) => {
    setTestingId(id);
    try {
      const res = await fetch(`/api/channel/test/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data.success) {
        setChannels(prev => prev.map(c =>
          c.id === id ? { ...c, response_time: data.data.time } : c
        ));
      } else {
        toast.error(data.data?.msg || data.msg || '测试失败');
      }
    } catch (error) {
      console.error('Test error:', error);
    } finally {
      setTestingId(null);
    }
  };

  useEffect(() => {
    fetchChannels();
  }, []);

  const parseMapping = (jsonStr: string) => {
    try {
      const map = JSON.parse(jsonStr || '{}');
      if (!map || typeof map !== 'object') return [];
      return Object.entries(map).map(([from, to]) => ({ from, to: to as string }));
    } catch {
      return [];
    }
  };

  const handleEdit = (channel: Channel) => {
    setEditingChannel(channel);
    setFormData({
      name: channel.name,
      type: channel.type.toString(),
      key: channel.key,
      base_url: channel.base_url,
      models: channel.models,
      group: channel.group || 'default',
      priority: channel.priority.toString(),
      weight: channel.weight.toString(),
      tags: channel.tags || '',
    });
    setModelMapping(parseMapping(channel.model_mapping));
    onOpen();
  };

  const handleAdd = () => {
    setEditingChannel(null);
    setFormData({
      name: '',
      type: '1',
      key: '',
      base_url: '',
      models: '',
      group: 'default',
      priority: '10',
      weight: '1',
      tags: '',
    });
    setModelMapping([]);
    onOpen();
  };

  const handleAddMapping = () => {
    if (newMapping.from && newMapping.to) {
      setModelMapping([...modelMapping, newMapping]);
      setNewMapping({from: '', to: ''});
    }
  };

  const handleDeleteMapping = (index: number) => {
    setModelMapping((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async (onClose: () => void) => {
    const url = '/api/channel';
    const method = editingChannel ? 'PUT' : 'POST';
    
    // Convert mapping array back to object then JSON
    const mappingObj: Record<string, string> = {};
    modelMapping.forEach(m => mappingObj[m.from] = m.to);

    const body = {
      ...formData,
      id: editingChannel?.id,
      type: parseInt(formData.type),
      priority: parseInt(formData.priority),
      weight: parseInt(formData.weight),
      model_mapping: JSON.stringify(mappingObj),
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
        fetchChannels();
        onClose();
      } else {
        toast.error('操作失败');
      }
    } catch (error) {
      console.error('Operation error:', error);
    }
  };

  const handleDelete = async (id: number) => {
    if (!await confirm({ title: '删除渠道', message: '确定要删除这个渠道吗？此操作不可撤销。', danger: true })) return;
    try {
      const res = await fetch(`/api/channel/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) fetchChannels();
    } catch (error) {
      console.error('Delete error:', error);
    }
  };

  const handleBatchSubmit = async (onClose: () => void) => {
    if (!batchText.trim()) return;
    
    let channels: any[] = [];
    try {
        // Try JSON first
        const json = JSON.parse(batchText);
        if (Array.isArray(json)) {
            channels = json;
        }
    } catch (e) {
        // Fallback to CSV-like
        // Format: Type|Name|Key|BaseURL|Models|Group
        const lines = batchText.split('\n');
        channels = lines.filter(l => l.trim()).map(line => {
            const parts = line.split('|');
            if (parts.length < 3) return null;
            return {
                type: parseInt(parts[0]) || 1,
                name: parts[1],
                key: parts[2],
                base_url: parts[3] || '',
                models: parts[4] || '',
                group: parts[5] || 'default',
                status: 1,
                priority: 10,
                weight: 1,
            };
        }).filter(c => c !== null);
    }

    if (channels.length === 0) {
        toast.error('未能解析出有效的渠道数据');
        return;
    }

    try {
        const res = await fetch('/api/channel/batch', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${token}`,
            },
            body: JSON.stringify(channels),
        });
        const data = await res.json();
        if (data.code === 0) {
            toast.success(`成功导入 ${data.data?.count ?? data.count} 个渠道`);
            fetchChannels();
            setBatchText('');
            onClose();
        } else {
            toast.error(data.msg || '导入失败');
        }
    } catch (error) {
        console.error(error);
        toast.error('请求失败');
    }
  };

  const getResponseTimeColor = (ms: number) => {
    if (ms === 0) return "default";
    if (ms < 1000) return "success";
    if (ms < 3000) return "warning";
    return "danger";
  };

  const handleBatchTest = async () => {
    setBatchTesting(true);
    try {
      const res = await fetch('/api/channel/test-all', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`测试完成: ${data.data?.success ?? data.success}/${data.data?.total ?? data.total} 成功`);
        fetchChannels();
      }
    } catch (error) {
      console.error('Batch test error:', error);
    } finally {
      setBatchTesting(false);
    }
  };

  const handleFetchModels = async (channelId: number) => {
    setFetchingModels(channelId);
    try {
      const res = await fetch(`/api/channel/models/${channelId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data?.models) {
        const modelStr = data.data.models.join(',');
        if (editingChannel) {
          setFormData(prev => ({ ...prev, models: modelStr }));
        }
        toast.info(`获取到 ${data.data.models.length} 个模型`);
      } else {
        toast.error(data.msg || '获取模型失败');
      }
    } catch (error) {
      console.error('Fetch models error:', error);
    } finally {
      setFetchingModels(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">渠道管理</h1>
          <p className="text-default-500">管理所有模型上游渠道</p>
        </div>
        <div className="flex gap-2">
          <Input
            placeholder="按标签筛选"
            size="sm"
            value={tagFilter}
            onValueChange={setTagFilter}
            className="w-32"
            onKeyDown={(e) => e.key === 'Enter' && fetchChannels()}
          />
          <Button startContent={<RefreshCw size={18} />} onPress={fetchChannels} variant="flat">
            刷新
          </Button>
          <Button startContent={<Zap size={18} />} onPress={handleBatchTest} variant="flat" color="warning" isLoading={batchTesting}>
            全部测试
          </Button>
          <Button startContent={<Upload size={18} />} onPress={onBatchOpen} variant="flat" color="secondary">
            批量导入
          </Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>
            添加渠道
          </Button>
        </div>
      </div>

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th><th>名称</th><th>类型</th><th>分组</th><th>状态</th><th>响应时间</th><th>余额</th><th>优先级</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={9} rows={5} />
            ) : channels.length === 0 ? (
              <tr><td colSpan={9}><EmptyState icon="📡" title="暂无渠道" description="添加您的第一个 AI 渠道" /></td></tr>
            ) : channels.map((channel) => (
              <tr key={channel.id}>
                <td>{channel.id}</td>
                <td className="font-medium">{channel.name}</td>
                <td><Chip size="sm" variant="flat" color="primary">{CHANNEL_TYPES.find(t => t.key === channel.type.toString())?.label || 'Unknown'}</Chip></td>
                <td><Chip size="sm" variant="dot">{channel.group || 'default'}</Chip></td>
                <td><Chip size="sm" color={channel.status === 1 ? "success" : "danger"} startContent={<Power size={12} />}>{channel.status === 1 ? "已启用" : "已禁用"}</Chip></td>
                <td><Chip size="sm" variant="flat" color={getResponseTimeColor(channel.response_time)}>{channel.response_time > 0 ? `${channel.response_time}ms` : '未测试'}</Chip></td>
                <td>${channel.balance.toFixed(2)}</td>
                <td>{channel.priority}</td>
                <td>
                  <div className="flex items-center gap-2">
                    <Tooltip content="测试"><span className={`text-lg text-default-400 cursor-pointer active:opacity-50 ${testingId === channel.id ? 'animate-spin' : ''}`} onClick={() => handleTest(channel.id)}><Activity size={18} /></span></Tooltip>
                    <Tooltip content="编辑"><span className="text-lg text-default-400 cursor-pointer active:opacity-50" onClick={() => handleEdit(channel)}><Edit size={18} /></span></Tooltip>
                    <Tooltip color="danger" content="删除"><span className="text-lg text-danger cursor-pointer active:opacity-50" onClick={() => handleDelete(channel.id)}><Trash2 size={18} /></span></Tooltip>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="3xl" scrollBehavior="inside">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader className="flex flex-col gap-1">
                {editingChannel ? '编辑渠道' : '添加新渠道'}
              </ModalHeader>
              <ModalBody>
                <Form className="grid grid-cols-2 gap-4">
                  <Input
                    label="渠道名称"
                    placeholder="例如: Azure GPT-4"
                    value={formData.name}
                    onValueChange={(v) => setFormData({...formData, name: v})}
                    isRequired
                    className="col-span-2"
                  />
                  <Select 
                    label="渠道类型" 
                    defaultSelectedKeys={[formData.type]}
                    onChange={(e) => setFormData({...formData, type: e.target.value})}
                  >
                    {CHANNEL_TYPES.map((type) => (
                      <SelectItem key={type.key}>{type.label}</SelectItem>
                    ))}
                  </Select>
                  <Input
                    label="优先级"
                    type="number"
                    value={formData.priority}
                    onValueChange={(v) => setFormData({...formData, priority: v})}
                  />
                  <Input
                    label="分组"
                    placeholder="default,vip"
                    value={formData.group}
                    onValueChange={(v) => setFormData({...formData, group: v})}
                  />
                  <Input
                    label="Base URL"
                    placeholder="https://api.openai.com"
                    value={formData.base_url}
                    onValueChange={(v) => setFormData({...formData, base_url: v})}
                    className="col-span-2"
                  />
                  <Input
                    label="权重"
                    type="number"
                    value={formData.weight}
                    onValueChange={(v) => setFormData({...formData, weight: v})}
                  />
                  <Input
                    label="标签"
                    placeholder="逗号分隔，如: gpt,claude"
                    value={formData.tags}
                    onValueChange={(v) => setFormData({...formData, tags: v})}
                  />
                  <div className="col-span-2 space-y-2">
                    <span className="text-small font-medium">密钥 (Key)</span>
                    <Textarea
                      placeholder="sk-... (多个Key每行一个)"
                      value={formData.key}
                      onValueChange={(v) => setFormData({...formData, key: v})}
                      minRows={2}
                      description="支持多Key轮换，每行一个"
                    />
                  </div>
                  
                  <div className="col-span-2 space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-small font-medium">模型列表 (逗号分隔)</span>
                      {editingChannel?.id && (
                        <Button
                          size="sm"
                          variant="flat"
                          color="secondary"
                          startContent={<Download size={14} />}
                          isLoading={fetchingModels === editingChannel.id}
                          onPress={() => handleFetchModels(editingChannel.id!)}
                        >
                          拉取模型
                        </Button>
                      )}
                    </div>
                    <Textarea
                      placeholder="gpt-3.5-turbo,gpt-4"
                      value={formData.models}
                      onValueChange={(v) => setFormData({...formData, models: v})}
                      minRows={2}
                    />
                  </div>

                  <div className="col-span-2 space-y-2">
                    <span className="text-small font-medium">模型映射 (重定向)</span>
                    <div className="flex gap-2 items-end">
                      <Input 
                        placeholder="原模型 (e.g. gpt-4)" 
                        size="sm" 
                        value={newMapping.from}
                        onValueChange={v => setNewMapping({...newMapping, from: v})}
                      />
                      <ArrowRight className="text-default-400 mb-2" size={16} />
                      <Input 
                        placeholder="目标模型 (e.g. azure-gpt-4)" 
                        size="sm"
                        value={newMapping.to}
                        onValueChange={v => setNewMapping({...newMapping, to: v})}
                      />
                      <Button isIconOnly size="sm" color="primary" variant="flat" onPress={handleAddMapping} className="mb-0.5">
                        <Plus size={16} />
                      </Button>
                    </div>
                    
                    <div className="flex flex-wrap gap-2 mt-2">
                      {modelMapping.map((m, i) => (
                        <Chip 
                          key={i} 
                          onClose={() => handleDeleteMapping(i)} 
                          variant="flat" 
                          color="secondary"
                          classNames={{ content: "flex items-center gap-1" }}
                        >
                          {m.from} <ArrowRight size={12} /> {m.to}
                        </Chip>
                      ))}
                      {modelMapping.length === 0 && <span className="text-xs text-default-400">暂无映射</span>}
                    </div>
                  </div>
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

      {/* Batch Import Modal */}
      <Modal isOpen={isBatchOpen} onOpenChange={onBatchOpenChange} size="2xl">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>批量导入渠道</ModalHeader>
              <ModalBody>
                <Tabs aria-label="Import Options">
                    <Tab key="text" title="文本导入">
                        <div className="space-y-2">
                            <p className="text-small text-default-500">
                                支持 JSON 数组或 CSV 格式 (竖线分隔):
                                <br />
                                <code>Type|Name|Key|BaseURL|Models|Group</code>
                                <br />
                                例如: <code>1|OpenAI|sk-xxx||gpt-3.5-turbo|default</code>
                            </p>
                            <Textarea 
                                minRows={10} 
                                placeholder="在此粘贴渠道数据..." 
                                value={batchText}
                                onValueChange={setBatchText}
                            />
                        </div>
                    </Tab>
                </Tabs>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleBatchSubmit(onClose)}>开始导入</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
