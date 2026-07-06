import { useState, useEffect, useMemo, useRef } from 'react';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import BatchChannelActions from '../../components/BatchChannelActions';
import {
  Chip,
  Tooltip,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerBody,
  DrawerFooter,
  useDisclosure,
  Input,
  Select,
  SelectItem,
  Form,
  Textarea,
  Tabs,
  Tab,
  Pagination,
  Checkbox,
  Switch,
} from '../../components/ui';
import { Edit, Trash2, Plus, RefreshCw, Power, Activity, ArrowRight, Upload, Zap, Download, DollarSign, Search, KeyRound, Settings } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import type { Channel, CustomChannelConfig, MultiKeyStatusItem, PromptDialogConfig } from './channelTypes';
import { CHANNEL_TYPES } from './channelConstants';

export default function ChannelManagement() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);
  const [testingId, setTestingId] = useState<number | null>(null);
  const { token, user } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const { isOpen: isBatchOpen, onOpen: onBatchOpen, onOpenChange: onBatchOpenChange } = useDisclosure();
  const [editingChannel, setEditingChannel] = useState<Partial<Channel> | null>(null);
  const [batchText, setBatchText] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [nameSearch, setNameSearch] = useState('');
  const [batchTesting, setBatchTesting] = useState(false);
  const [selectedChannelIds, setSelectedChannelIds] = useState<number[]>([]);
  const [fetchingModels, setFetchingModels] = useState<number | null>(null);
  const [fetchingBalance, setFetchingBalance] = useState<number | null>(null);
  const [operatingId, setOperatingId] = useState<number | null>(null);
  const { isOpen: isMultiKeyOpen, onOpen: onMultiKeyOpen, onOpenChange: onMultiKeyOpenChange } = useDisclosure();
  const [multiKeyChannel, setMultiKeyChannel] = useState<Channel | null>(null);
  const [multiKeyRows, setMultiKeyRows] = useState<MultiKeyStatusItem[]>([]);
  const [multiKeyPage, setMultiKeyPage] = useState(1);
  const [multiKeyPageSize, setMultiKeyPageSize] = useState(10);
  const [multiKeyTotalPages, setMultiKeyTotalPages] = useState(1);
  const [multiKeyTotal, setMultiKeyTotal] = useState(0);
  const [multiKeyStatusFilter, setMultiKeyStatusFilter] = useState<'all' | '1' | '2' | '3'>('all');
  const [multiKeySummary, setMultiKeySummary] = useState({ enabled: 0, manual: 0, auto: 0 });
  const [multiKeyLoading, setMultiKeyLoading] = useState(false);
  const [multiKeyActionLoading, setMultiKeyActionLoading] = useState<string | null>(null);
  const [multiKeyEditorText, setMultiKeyEditorText] = useState('');
  const [batchOpLoading, setBatchOpLoading] = useState<string | null>(null);
  const [modelSourceMode, setModelSourceMode] = useState<'manual' | 'sync'>('manual');
  const [newModelInput, setNewModelInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const { isOpen: isTestOpen, onOpen: onTestOpen, onOpenChange: onTestOpenChange } = useDisclosure();
  const [testChannel, setTestChannel] = useState<Channel | null>(null);
  const [testMode, setTestMode] = useState<string>('quick');
  const [testModel, setTestModel] = useState<string>('');
  const [testPrompt, setTestPrompt] = useState<string>('');
  const { isOpen: isPromptOpen, onOpen: onPromptOpen, onOpenChange: onPromptOpenChange } = useDisclosure();
  const [promptConfig, setPromptConfig] = useState<PromptDialogConfig>({ title: '' });
  const [promptValue, setPromptValue] = useState('');
  const [promptSubmitting, setPromptSubmitting] = useState(false);
  const promptResolveRef = useRef<((value: string | null) => void) | null>(null);

  // 自定义配置相关状态
  const { isOpen: isCustomConfigOpen, onOpen: onCustomConfigOpen, onOpenChange: onCustomConfigOpenChange } = useDisclosure();
  const [editingCustomConfig, setEditingCustomConfig] = useState<CustomChannelConfig | null>(null);
  const [customConfigFormData, setCustomConfigFormData] = useState({
    name: '',
    description: '',
    request_endpoint: '/v1/chat/completions',
    auth_header_template: 'Bearer {key}',
    field_model: 'model',
    field_messages: 'messages',
    field_temperature: 'temperature',
    field_max_tokens: 'max_tokens',
    response_content_path: 'choices.0.message.content',
    response_prompt_tokens_path: 'usage.prompt_tokens',
    response_completion_tokens_path: 'usage.completion_tokens',
    stream_enabled: 1,
    stream_data_prefix: 'data: ',
    stream_end_marker: '[DONE]',
    stream_content_path: 'choices.0.delta.content',
  });

  const channelSummary = useMemo(() => {
    const total = channels.length;
    const enabled = channels.filter((c) => c.status === 1).length;
    const disabled = total - enabled;
    const withMultiKey = channels.filter((c) => c.key?.includes('\n')).length;
    const avgLatency = total > 0
      ? Math.round(channels.reduce((sum, c) => sum + (c.response_time || 0), 0) / total)
      : 0;

    return { total, enabled, disabled, withMultiKey, avgLatency };
  }, [channels]);

  const allVisibleChannelIds = useMemo(() => channels.map((c) => c.id), [channels]);
  const allSelected = allVisibleChannelIds.length > 0 && allVisibleChannelIds.every((id) => selectedChannelIds.includes(id));

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
    is_custom: 0,
    custom_config_id: 0,
  });

  // Model Mapping State (Visual)
  const [modelMapping, setModelMapping] = useState<{from: string, to: string}[]>([]);
  const [newMapping, setNewMapping] = useState({from: '', to: ''});

  // Custom Config State
  const [customConfigs, setCustomConfigs] = useState<CustomChannelConfig[]>([]);
  const [loadingConfigs, setLoadingConfigs] = useState(false);

  const fetchChannels = async () => {
    setLoading(true);
    try {
      let url = '/api/channel';
      if (nameSearch.trim()) {
        url = `/api/channel/search?keyword=${encodeURIComponent(nameSearch.trim())}`;
      } else if (tagFilter) {
        url = `/api/channel?tag=${encodeURIComponent(tagFilter)}`;
      }
      const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) setChannels(data.data);
      else setSelectedChannelIds([]);
    } catch (error) {
      console.error('Failed to fetch channels:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleTest = async (id: number) => {
    const channel = channels.find(c => c.id === id);
    if (!channel) return;

    setTestChannel(channel);
    setTestMode('quick');
    setTestModel('');
    setTestPrompt('');
    onTestOpen();
  };

  const executeTest = async () => {
    if (!testChannel) return;

    if (testMode === 'quick') {
      await executeChannelTest(testChannel.id);
    } else if (testMode === 'custom') {
      if (!testPrompt.trim()) {
        toast.error('请输入测试问题');
        return;
      }
      await executeChannelTest(testChannel.id, testPrompt.trim());
    } else if (testMode === 'model') {
      if (!testModel.trim()) {
        toast.error('请选择或输入模型名称');
        return;
      }
      await executeModelTest(testChannel.id, testModel.trim(), testPrompt.trim() || undefined);
    }

    onTestOpenChange();
  };

  const executeChannelTest = async (id: number, prompt?: string) => {
    setTestingId(id);
    try {
      const res = await fetch(`/api/channel/test/${id}`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ prompt: prompt || '' }),
      });
      const data = await res.json();
      if (data.code === 0 && data.data.success) {
        toast.success(`测试通过 (${data.data.time}ms)`);
        setChannels(prev => prev.map(c =>
          c.id === id ? { ...c, response_time: data.data.time } : c
        ));
      } else {
        toast.error(data.data?.msg || data.msg || '测试失败');
      }
    } catch (error) {
      console.error('Test error:', error);
      toast.error('测试请求失败');
    } finally {
      setTestingId(null);
    }
  };

  const executeModelTest = async (id: number, model: string, prompt?: string) => {
    setTestingId(id);
    try {
      const res = await fetch(`/api/channel/test-model/${id}`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ model, prompt: prompt || '' }),
      });
      const data = await res.json();
      if (data.code === 0 && data.data.success) {
        toast.success(`模型 ${model} 测试通过 (${data.data.time}ms)`);
      } else {
        toast.error(data.data?.msg || data.msg || '模型测试失败');
      }
    } catch (error) {
      console.error('Model test error:', error);
      toast.error('模型测试请求失败');
    } finally {
      setTestingId(null);
    }
  };

  const showPromptDialog = (config: PromptDialogConfig): Promise<string | null> => {
    return new Promise((resolve) => {
      setPromptConfig(config);
      setPromptValue(config.defaultValue || '');
      promptResolveRef.current = resolve;
      onPromptOpen();
    });
  };

  useEffect(() => {
    fetchChannels();
    fetchCustomConfigs();
  }, []);

  useEffect(() => {
    setSelectedChannelIds((prev) => prev.filter((id) => channels.some((channel) => channel.id === id)));
  }, [channels]);

  const fetchCustomConfigs = async () => {
    setLoadingConfigs(true);
    try {
      const res = await fetch('/api/custom-channel-config', {
        headers: { Authorization: `Bearer ${token}` }
      });
      const data = await res.json();
      if (data.code === 0) {
        setCustomConfigs(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch custom configs:', error);
    } finally {
      setLoadingConfigs(false);
    }
  };

  const handleCreateCustomConfig = () => {
    setEditingCustomConfig(null);
    setCustomConfigFormData({
      name: '',
      description: '',
      request_endpoint: '/v1/chat/completions',
      auth_header_template: 'Bearer {key}',
      field_model: 'model',
      field_messages: 'messages',
      field_temperature: 'temperature',
      field_max_tokens: 'max_tokens',
      response_content_path: 'choices.0.message.content',
      response_prompt_tokens_path: 'usage.prompt_tokens',
      response_completion_tokens_path: 'usage.completion_tokens',
      stream_enabled: 1,
      stream_data_prefix: 'data: ',
      stream_end_marker: '[DONE]',
      stream_content_path: 'choices.0.delta.content',
    });
    onCustomConfigOpen();
  };

  const handleSubmitCustomConfig = async (onClose: () => void) => {
    if (!customConfigFormData.name.trim()) {
      toast.error('请输入配置名称');
      return;
    }

    const url = '/api/custom-channel-config';
    const method = editingCustomConfig ? 'PUT' : 'POST';

    const body = editingCustomConfig
      ? { ...customConfigFormData, id: editingCustomConfig.id }
      : customConfigFormData;

    setSubmitting(true);
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
      if (data.code !== 0) {
        toast.error(data.msg || '操作失败');
        return;
      }

      toast.success(editingCustomConfig ? '配置更新成功' : '配置创建成功');
      fetchCustomConfigs();
      onClose();
    } catch (error) {
      console.error('Operation error:', error);
      toast.error('请求失败');
    } finally {
      setSubmitting(false);
    }
  };

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
      is_custom: channel.is_custom || 0,
      custom_config_id: channel.custom_config_id || 0,
    });
    setModelMapping(parseMapping(channel.model_mapping));
    setModelSourceMode('manual');
    setNewModelInput('');
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
      is_custom: 0,
      custom_config_id: 0,
    });
    setModelMapping([]);
    setModelSourceMode('manual');
    setNewModelInput('');
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

    const mappingObj: Record<string, string> = {};
    modelMapping.forEach(m => mappingObj[m.from] = m.to);

    // 检查自定义渠道类型
    const isCustomChannel = formData.type === '100';

    const body = {
      ...formData,
      id: editingChannel?.id,
      type: parseInt(formData.type),
      priority: parseInt(formData.priority),
      weight: parseInt(formData.weight),
      model_mapping: JSON.stringify(mappingObj),
      is_custom: isCustomChannel ? 1 : 0,
      custom_config_id: isCustomChannel ? formData.custom_config_id : 0,
    };

    setSubmitting(true);
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
      if (data.code !== 0) {
        toast.error(data.msg || '操作失败');
        return;
      }

      toast.success(editingChannel ? '渠道更新成功' : '渠道创建成功');

      fetchChannels();
      onClose();
    } catch (error) {
      console.error('Operation error:', error);
      toast.error('请求失败');
    } finally {
      setSubmitting(false);
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

  const handleUpdateAllBalances = async () => {
    setBatchOpLoading('update_balance');
    try {
      const res = await fetch('/api/channel/update_balance', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('已触发全量余额刷新');
        fetchChannels();
      } else {
        toast.error(data.msg || '更新余额失败');
      }
    } finally {
      setBatchOpLoading(null);
    }
  };

  const handleFixChannelAbilities = async () => {
    setBatchOpLoading('fix');
    try {
      const res = await fetch('/api/channel/fix', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('渠道能力修复完成');
      } else {
        toast.error(data.msg || '修复渠道能力失败');
      }
    } finally {
      setBatchOpLoading(null);
    }
  };

  const handleDeleteAllDisabled = async () => {
    if (!await confirm({ title: '清理已禁用渠道', message: '将删除全部已禁用渠道，此操作不可恢复。', danger: true })) return;
    setBatchOpLoading('delete_disabled');
    try {
      const res = await fetch('/api/channel/disabled', {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('已清理禁用渠道');
        fetchChannels();
      } else {
        toast.error(data.msg || '清理禁用渠道失败');
      }
    } finally {
      setBatchOpLoading(null);
    }
  };

  const toggleChannelSelection = (channelId: number, checked: boolean) => {
    setSelectedChannelIds((prev) => checked
      ? (prev.includes(channelId) ? prev : [...prev, channelId])
      : prev.filter((id) => id !== channelId)
    );
  };

  const toggleSelectAllChannels = (checked: boolean) => {
    setSelectedChannelIds(checked ? allVisibleChannelIds : []);
  };

  const handleToggleStatus = async (channelId: number) => {
    try {
      const res = await fetch(`/api/channel/${channelId}/status`, {
        method: 'PATCH',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setChannels(prev => prev.map(c => c.id === channelId ? { ...c, status: data.data.status } : c));
      }
    } catch (error) {
      toast.error('操作失败');
    }
  };

  const handleFetchBalance = async (channelId: number) => {
    setFetchingBalance(channelId);
    try {
      const res = await fetch(`/api/channel/balance/${channelId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setChannels(prev => prev.map(c => c.id === channelId ? { ...c, balance: data.data.balance } : c));
        toast.success(`余额: $${data.data.balance.toFixed(4)}`);
      } else {
        toast.error(data.msg || '查询余额失败');
      }
    } catch (error) {
      toast.error('请求失败');
    } finally {
      setFetchingBalance(null);
    }
  };

  const handleFetchModels = async (channelId: number) => {
    setFetchingModels(channelId);
    try {
      const res = await fetch(`/api/channel/models/${channelId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      // 兼容多种格式:
      // new-api GET:  { data: [...] }  (无 code/success 字段)
      // new-api POST: { success: true, data: [...] }
      // backend:      { code: 0, data: { models: [...] } }
      const models: string[] = Array.isArray(data?.data)
        ? data.data
        : (Array.isArray(data?.data?.models) ? data.data.models : []);
      // 乐观匹配：只有明确返回失败时才报错
      const isFailed = data?.success === false || (data?.code !== undefined && data?.code !== 0);
      if (isFailed) {
        toast.error(data.message || data.msg || '获取模型失败');
      } else if (models.length > 0) {
        const modelStr = models.join(',');
        setFormData(prev => ({ ...prev, models: modelStr }));
        toast.success(`已拉取 ${models.length} 个上游模型`);
      } else {
        toast.error('上游未返回任何模型');
      }
    } catch (error) {
      console.error('Fetch models error:', error);
    } finally {
      setFetchingModels(null);
    }
  };

  const handleFetchModelsByForm = async () => {
    setFetchingModels(-1);
    try {
      const res = await fetch('/api/channel/fetch_models', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          type: parseInt(formData.type || '1', 10),
          key: formData.key,
          base_url: formData.base_url,
        }),
      });
      const data = await res.json();
      // 兼容两种后端响应格式:
      // new-api: { success: true, data: [...] }
      // backend: { code: 0, msg: "...", data: { models: [...] } }
      const isSuccess = data?.success === true || data?.code === 0;
      const models = Array.isArray(data?.data)
        ? data.data
        : (Array.isArray(data?.data?.models) ? data.data.models : []);
      if (isSuccess && Array.isArray(models) && models.length > 0) {
        const modelStr = models.join(',');
        setFormData(prev => ({ ...prev, models: modelStr }));
        toast.success(`已同步 ${models.length} 个上游模型，可继续删改后保存`);
      } else if (!isSuccess) {
        toast.error(data.message || data.msg || '同步上游模型失败');
      } else {
        toast.error('上游未返回任何模型');
      }
    } catch (error) {
      console.error('Fetch upstream models by form error:', error);
      toast.error('同步上游模型失败');
    } finally {
      setFetchingModels(null);
    }
  };

  const addModel = (name: string) => {
    const trimmed = name.trim();
    if (!trimmed) return;
    const existing = formData.models ? formData.models.split(',').map((m: string) => m.trim()).filter(Boolean) : [];
    if (existing.includes(trimmed)) return;
    setFormData((prev: typeof formData) => ({ ...prev, models: [...existing, trimmed].join(',') }));
    setNewModelInput('');
  };

  const removeModel = (name: string) => {
    const existing = formData.models ? formData.models.split(',').map((m: string) => m.trim()).filter(Boolean) : [];
    setFormData((prev: typeof formData) => ({ ...prev, models: existing.filter((m: string) => m !== name).join(',') }));
  };

  const openPromptDialog = (config: PromptDialogConfig): Promise<string | null> => {
    setPromptConfig(config);
    setPromptValue(config.defaultValue ?? '');
    setPromptSubmitting(false);
    onPromptOpen();
    return new Promise((resolve) => {
      promptResolveRef.current = resolve;
    });
  };

  const closePromptDialog = (value: string | null) => {
    promptResolveRef.current?.(value);
    promptResolveRef.current = null;
    onPromptOpenChange();
  };

  const handleCopyChannel = async (channelId: number) => {
    const suffixInput = await openPromptDialog({
      title: '复制渠道',
      placeholder: '请输入复制后缀',
      defaultValue: '_复制',
      description: '会在原渠道名后追加该后缀',
      confirmText: '继续复制',
    });
    if (suffixInput === null) return;
    const suffix = (suffixInput || '_复制').trim() || '_复制';

    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/copy/${channelId}?suffix=${encodeURIComponent(suffix)}&reset_balance=true`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('复制成功');
        fetchChannels();
      } else {
        toast.error(data.msg || '复制失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleGetChannelKey = async (channelId: number) => {
    if ((user?.role ?? 0) < 100) {
      toast.error('仅超级管理员可查看渠道 Key');
      return;
    }
    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/${channelId}/key`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const key = data?.data?.key ?? '';
        await openPromptDialog({
          title: `渠道 #${channelId} 的 Key`,
          defaultValue: key || '(空)',
          description: '仅超级管理员可查看，支持复制',
          confirmText: '关闭',
          multiline: true,
          readOnly: true,
        });
      } else {
        toast.error(data.msg || '获取渠道 Key 失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleOllamaPull = async (channelId: number) => {
    const modelName = await openPromptDialog({
      title: 'Ollama 拉取模型',
      placeholder: '例如 llama3:8b',
      defaultValue: 'llama3',
      description: '输入要拉取的 Ollama 模型名',
      confirmText: '开始拉取',
    });
    if (!modelName?.trim()) return;

    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/ollama/pull', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId, model_name: modelName.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(data.msg || '拉取成功');
      } else {
        toast.error(data.msg || '拉取失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleOllamaDelete = async (channelId: number) => {
    const modelName = await openPromptDialog({
      title: 'Ollama 删除模型',
      placeholder: '例如 llama3:8b',
      defaultValue: 'llama3',
      description: '输入要删除的 Ollama 模型名',
      confirmText: '确认删除',
    });
    if (!modelName?.trim()) return;

    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/ollama/delete', {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId, model_name: modelName.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(data.msg || '删除成功');
      } else {
        toast.error(data.msg || '删除失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleOllamaPullStream = async (channelId: number) => {
    const modelName = await openPromptDialog({
      title: 'Ollama 流式拉取模型',
      placeholder: '例如 llama3:8b',
      defaultValue: 'llama3',
      description: '输入要流式拉取的 Ollama 模型名',
      confirmText: '开始拉取',
    });
    if (!modelName?.trim()) return;

    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/ollama/pull/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId, model_name: modelName.trim() }),
      });
      if (!res.ok || !res.body) {
        toast.error('流式拉取失败');
        return;
      }
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let lastStatus = '';
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';
        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed.startsWith('data:')) continue;
          const payload = trimmed.slice(5).trim();
          if (payload === '[DONE]') continue;
          try {
            const parsed = JSON.parse(payload);
            if (parsed?.status) lastStatus = parsed.status;
            if (parsed?.message) lastStatus = parsed.message;
            if (parsed?.error) {
              toast.error(parsed.error);
              return;
            }
          } catch {
            // ignore non-json chunks
          }
        }
      }
      toast.success(lastStatus || '流式拉取完成');
    } catch {
      toast.error('流式拉取失败');
    } finally {
      setOperatingId(null);
    }
  };

  const handleOllamaVersion = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/ollama/version/${channelId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const payload = data.data?.data || data.data;
        toast.info(`Ollama 版本: ${payload?.version || '-'}`);
      } else {
        toast.error(data.msg || '查询版本失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const fetchMultiKeyStatus = async (channelId: number, opts?: { page?: number; pageSize?: number; status?: 'all' | '1' | '2' | '3' }) => {
    const page = opts?.page ?? multiKeyPage;
    const pageSize = opts?.pageSize ?? multiKeyPageSize;
    const status = opts?.status ?? multiKeyStatusFilter;
    setMultiKeyLoading(true);
    try {
      const body: any = { channel_id: channelId, action: 'get_key_status', page, page_size: pageSize };
      if (status !== 'all') body.status = parseInt(status, 10);
      const res = await fetch('/api/channel/multi_key/manage', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '获取多Key状态失败');
        return;
      }
      const stat = data.data?.data || data.data;
      setMultiKeyRows(stat?.keys || []);
      setMultiKeyPage(stat?.page || page);
      setMultiKeyPageSize(stat?.page_size || pageSize);
      setMultiKeyTotal(stat?.total || 0);
      setMultiKeyTotalPages(stat?.total_pages || 1);
      setMultiKeySummary({
        enabled: stat?.enabled_count || 0,
        manual: stat?.manual_disabled_count || 0,
        auto: stat?.auto_disabled_count || 0,
      });
    } finally {
      setMultiKeyLoading(false);
    }
  };

  const handleGetMultiKeyStatus = async (channelId: number) => {
    const current = channels.find((c) => c.id === channelId) || null;
    setMultiKeyChannel(current);
    setMultiKeyStatusFilter('all');
    setMultiKeyPage(1);
    setMultiKeyPageSize(10);
    setMultiKeyEditorText(current?.key || '');
    onMultiKeyOpen();
    await fetchMultiKeyStatus(channelId, { page: 1, pageSize: 10, status: 'all' });
  };

  const doMultiKeyAction = async (action: string, keyIndex?: number) => {
    if (!multiKeyChannel) return;
    const loadingKey = `${action}:${keyIndex ?? 'all'}`;
    setMultiKeyActionLoading(loadingKey);
    try {
      const body: any = { channel_id: multiKeyChannel.id, action };
      if (typeof keyIndex === 'number') body.key_index = keyIndex;
      const res = await fetch('/api/channel/multi_key/manage', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('操作成功');
        await fetchMultiKeyStatus(multiKeyChannel.id);
      } else {
        toast.error(data.msg || '操作失败');
      }
    } finally {
      setMultiKeyActionLoading(null);
    }
  };

  const getKeyStatusText = (status: number) => {
    if (status === 1) return '启用';
    if (status === 2) return '手动禁用';
    if (status === 3) return '自动禁用';
    return `未知(${status})`;
  };

  const getKeyStatusColor = (status: number): 'success' | 'warning' | 'danger' | 'default' => {
    if (status === 1) return 'success';
    if (status === 2) return 'warning';
    if (status === 3) return 'danger';
    return 'default';
  };

  const formatUnix = (ts: number) => {
    if (!ts) return '-';
    return new Date(ts * 1000).toLocaleString();
  };

  const handleEnableAllKeys = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/multi_key/manage', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId, action: 'enable_all_keys' }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('已启用全部 Key');
      } else {
        toast.error(data.msg || '启用失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleDisableAllKeys = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/multi_key/manage', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: channelId, action: 'disable_all_keys' }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('已禁用全部 Key');
      } else {
        toast.error(data.msg || '禁用失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleCodexOAuthStart = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/${channelId}/codex/oauth/start`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '获取授权链接失败');
        return;
      }
      const payload = data.data?.data || data.data;
      const authorizeUrl = payload?.authorize_url;
      if (!authorizeUrl) {
        toast.error('后端未返回授权链接');
        return;
      }
      window.open(authorizeUrl, '_blank');
      toast.success('已打开 Codex 授权链接');
    } finally {
      setOperatingId(null);
    }
  };

  const handleCodexOAuthComplete = async (channelId: number) => {
    const input = await openPromptDialog({
      title: 'Codex OAuth 完成',
      placeholder: '粘贴 code#state 或完整 URL',
      defaultValue: '',
      description: '请从回调地址中粘贴完整参数',
      confirmText: '提交回调',
      multiline: true,
    });
    if (!input?.trim()) return;

    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/${channelId}/codex/oauth/complete`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ input: input.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('Codex OAuth 完成并已写入渠道');
        fetchChannels();
      } else {
        toast.error(data.msg || 'Codex OAuth 完成失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleCodexRefresh = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/${channelId}/codex/refresh`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('Codex 凭据刷新成功');
      } else {
        toast.error(data.msg || 'Codex 凭据刷新失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleSetSingleTag = async (channelId: number) => {
    const tag = await openPromptDialog({
      title: '设置渠道标签',
      placeholder: '输入标签',
      defaultValue: '',
      description: '会覆盖该渠道 tags 字段',
      confirmText: '保存标签',
    });
    if (!tag?.trim()) return;

    setOperatingId(channelId);
    try {
      const res = await fetch('/api/channel/batch/tag', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ ids: [channelId], tag: tag.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('标签更新成功');
        fetchChannels();
      } else {
        toast.error(data.msg || '标签更新失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleGetTagModels = async (initialTag?: string) => {
    const tag = await openPromptDialog({
      title: '查询标签模型',
      placeholder: '输入标签',
      defaultValue: initialTag || '',
      description: '查询该标签下可用模型列表',
      confirmText: '开始查询',
    });
    if (!tag?.trim()) return;

    setOperatingId(-2);
    try {
      const res = await fetch(`/api/channel/tag/models?tag=${encodeURIComponent(tag.trim())}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const payload = data.data?.data ?? data.data;
        toast.info(`标签[${tag}]模型: ${payload || '(空)'}`);
      } else {
        toast.error(data.msg || '查询标签模型失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleCodexUsage = async (channelId: number) => {
    setOperatingId(channelId);
    try {
      const res = await fetch(`/api/channel/${channelId}/codex/usage`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const payload = data.data?.data || data.data;
        toast.info(`Codex 用量: ${JSON.stringify(payload)}`);
      } else {
        toast.error(data.msg || '查询 Codex 用量失败');
      }
    } finally {
      setOperatingId(null);
    }
  };

  const handleSetMultiKeys = async () => {
    if (!multiKeyChannel) return;
    const keys = multiKeyEditorText.split('\n').map((v) => v.trim()).filter(Boolean);
    if (keys.length === 0) {
      toast.error('至少保留一个 Key');
      return;
    }
    if (!await confirm({ title: '覆盖多 Key', message: `将覆盖该渠道全部 Key，共 ${keys.length} 个，是否继续？`, danger: true })) return;
    setMultiKeyActionLoading('set:all');
    try {
      const res = await fetch('/api/channel/multi_key/manage', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ channel_id: multiKeyChannel.id, action: 'set', keys }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('多 Key 已更新');
        await fetchMultiKeyStatus(multiKeyChannel.id, { page: 1 });
        await fetchChannels();
      } else {
        toast.error(data.msg || '更新多 Key 失败');
      }
    } finally {
      setMultiKeyActionLoading(null);
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
            placeholder="搜索渠道名称"
            size="sm"
            value={nameSearch}
            onValueChange={setNameSearch}
            className="w-36"
            startContent={<Search size={14} />}
            onKeyDown={(e) => e.key === 'Enter' && fetchChannels()}
          />
          <Input
            placeholder="按标签筛选"
            size="sm"
            value={tagFilter}
            onValueChange={setTagFilter}
            className="w-28"
            onKeyDown={(e) => e.key === 'Enter' && fetchChannels()}
          />
          <Button startContent={<RefreshCw size={18} />} onPress={fetchChannels} variant="flat">
            刷新
          </Button>
          <Button startContent={<Zap size={18} />} onPress={handleBatchTest} variant="flat" color="warning" isLoading={batchTesting}>
            全部测试
          </Button>
          <Button startContent={<RefreshCw size={18} />} onPress={handleUpdateAllBalances} variant="flat" isLoading={batchOpLoading === 'update_balance'}>
            全量余额
          </Button>
          <Button startContent={<Activity size={18} />} onPress={handleFixChannelAbilities} variant="flat" color="secondary" isLoading={batchOpLoading === 'fix'}>
            修复能力
          </Button>
          <Button startContent={<Trash2 size={18} />} onPress={handleDeleteAllDisabled} variant="flat" color="danger" isLoading={batchOpLoading === 'delete_disabled'}>
            清理禁用
          </Button>
          <Button startContent={<Upload size={18} />} onPress={onBatchOpen} variant="flat" color="secondary">
            批量导入
          </Button>
          <Button startContent={<Search size={18} />} onPress={() => handleGetTagModels(tagFilter)} variant="flat" color="secondary">
            标签模型
          </Button>
          <Button startContent={<Download size={18} />} variant="flat" onPress={async () => {
            const res = await fetch('/api/export/channel', { headers: { Authorization: `Bearer ${token}` } });
            if (!res.ok) return;
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url; a.download = 'channels.json';
            document.body.appendChild(a); a.click(); a.remove();
            URL.revokeObjectURL(url);
          }}>
            导出
          </Button>
          <Button startContent={<Plus size={18} />} color="primary" onPress={handleAdd}>
            添加渠道
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        <Chip size="sm" variant="flat" color="primary">总数 {channelSummary.total}</Chip>
        <Chip size="sm" variant="flat" color="success">启用 {channelSummary.enabled}</Chip>
        <Chip size="sm" variant="flat" color="danger">禁用 {channelSummary.disabled}</Chip>
        <Chip size="sm" variant="flat" color="secondary">多 Key {channelSummary.withMultiKey}</Chip>
        <Chip size="sm" variant="flat" color={channelSummary.avgLatency > 0 && channelSummary.avgLatency < 1000 ? 'success' : channelSummary.avgLatency >= 3000 ? 'danger' : 'warning'}>
          平均响应 {channelSummary.avgLatency > 0 ? `${channelSummary.avgLatency}ms` : '未测试'}
        </Chip>
      </div>

      <BatchChannelActions
        selectedIds={selectedChannelIds}
        token={token || ''}
        onSuccess={fetchChannels}
        onClearSelection={() => setSelectedChannelIds([])}
      />

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>
                <Checkbox isSelected={allSelected} onValueChange={toggleSelectAllChannels} />
              </th>
              <th>ID</th><th>名称</th><th>类型</th><th>分组</th><th>状态</th><th>响应时间</th><th>余额</th><th>优先级</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <LoadingRows cols={10} rows={5} />
            ) : channels.length === 0 ? (
              <tr><td colSpan={10}><EmptyState icon="📡" title="暂无渠道" description="添加您的第一个 AI 渠道" /></td></tr>
            ) : channels.map((channel) => (
              <tr key={channel.id}>
                <td>
                  <Checkbox
                    isSelected={selectedChannelIds.includes(channel.id)}
                    onValueChange={(checked) => toggleChannelSelection(channel.id, checked)}
                  />
                </td>
                <td>{channel.id}</td>
                <td className="font-medium">{channel.name}</td>
                <td><Chip size="sm" variant="flat" color="primary">{CHANNEL_TYPES.find(t => t.key === channel.type.toString())?.label || 'Unknown'}</Chip></td>
                <td><Chip size="sm" variant="dot">{channel.group || 'default'}</Chip></td>
                <td><Chip size="sm" color={channel.status === 1 ? "success" : "danger"} startContent={<Power size={12} />} className="cursor-pointer" onClick={() => handleToggleStatus(channel.id)}>{channel.status === 1 ? "已启用" : "已禁用"}</Chip></td>
                <td><Chip size="sm" variant="flat" color={getResponseTimeColor(channel.response_time)}>{channel.response_time > 0 ? `${channel.response_time}ms` : '未测试'}</Chip></td>
                <td>${channel.balance.toFixed(2)}</td>
                <td>{channel.priority}</td>
                <td>
                  <div className="flex items-center gap-2 flex-wrap">
                    <Tooltip content="测试"><span className={`text-lg text-default-400 cursor-pointer active:opacity-50 ${testingId === channel.id ? 'animate-spin' : ''}`} onClick={() => handleTest(channel.id)}><Activity size={18} /></span></Tooltip>
                    <Tooltip content="查询余额"><span className={`text-lg text-default-400 cursor-pointer active:opacity-50 ${fetchingBalance === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleFetchBalance(channel.id)}><DollarSign size={18} /></span></Tooltip>
                    <Tooltip content="复制渠道"><span className={`text-lg text-default-400 cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleCopyChannel(channel.id)}><Plus size={18} /></span></Tooltip>
                    <Tooltip content="查看 Key (Root)"><span className={`text-lg ${(user?.role ?? 0) >= 100 ? 'text-warning cursor-pointer' : 'text-default-300 cursor-not-allowed'} active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleGetChannelKey(channel.id)}><KeyRound size={18} /></span></Tooltip>
                    {channel.type === 47 && (
                      <>
                        <Tooltip content="Ollama 拉取模型"><span className={`text-lg text-secondary cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleOllamaPull(channel.id)}><Download size={18} /></span></Tooltip>
                        <Tooltip content="Ollama 流式拉取"><span className={`text-lg text-primary cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleOllamaPullStream(channel.id)}><RefreshCw size={18} /></span></Tooltip>
                        <Tooltip content="Ollama 版本"><span className={`text-lg text-success cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleOllamaVersion(channel.id)}><Activity size={18} /></span></Tooltip>
                        <Tooltip content="Ollama 删除模型"><span className={`text-lg text-warning cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleOllamaDelete(channel.id)}><Trash2 size={18} /></span></Tooltip>
                      </>
                    )}
                    {channel.key?.includes('\n') && (
                      <>
                        <Tooltip content="查看多 Key 状态"><span className={`text-lg text-primary cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleGetMultiKeyStatus(channel.id)}><Zap size={18} /></span></Tooltip>
                        <Tooltip content="启用全部 Key"><span className={`text-lg text-success cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleEnableAllKeys(channel.id)}><Power size={18} /></span></Tooltip>
                        <Tooltip content="禁用全部 Key"><span className={`text-lg text-warning cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleDisableAllKeys(channel.id)}><RefreshCw size={18} /></span></Tooltip>
                      </>
                    )}
                    <Tooltip content="设置标签"><span className={`text-lg text-success cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleSetSingleTag(channel.id)}><Search size={18} /></span></Tooltip>
                    {channel.type === 58 && (
                      <>
                        <Tooltip content="Codex OAuth 开始"><span className={`text-lg text-primary cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleCodexOAuthStart(channel.id)}><Power size={18} /></span></Tooltip>
                        <Tooltip content="Codex OAuth 完成"><span className={`text-lg text-secondary cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleCodexOAuthComplete(channel.id)}><Activity size={18} /></span></Tooltip>
                        <Tooltip content="Codex 凭据刷新"><span className={`text-lg text-warning cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleCodexRefresh(channel.id)}><RefreshCw size={18} /></span></Tooltip>
                        <Tooltip content="Codex 用量"><span className={`text-lg text-success cursor-pointer active:opacity-50 ${operatingId === channel.id ? 'animate-pulse' : ''}`} onClick={() => handleCodexUsage(channel.id)}><DollarSign size={18} /></span></Tooltip>
                      </>
                    )}
                    <Tooltip content="编辑"><span className="text-lg text-default-400 cursor-pointer active:opacity-50" onClick={() => handleEdit(channel)}><Edit size={18} /></span></Tooltip>
                    <Tooltip content="删除"><span className="text-lg text-danger cursor-pointer active:opacity-50" onClick={() => handleDelete(channel.id)}><Trash2 size={18} /></span></Tooltip>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Drawer isOpen={isOpen} onOpenChange={onOpenChange} size="lg">
        <DrawerContent>
          {(onClose) => (
            <>
              <DrawerHeader onClose={onClose}>
                {editingChannel ? '编辑渠道' : '添加新渠道'}
              </DrawerHeader>
              <DrawerBody>
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
                    onSelectionChange={(keys) => {
                      const newType = [...keys][0] as string || '1';
                      setFormData({...formData, type: newType, custom_config_id: newType === '100' ? 0 : formData.custom_config_id});
                    }}
                  >
                    {CHANNEL_TYPES.map((type) => (
                      <SelectItem key={type.key}>{type.label}</SelectItem>
                    ))}
                  </Select>

                  {/* 自定义渠道配置选择器 */}
                  {formData.type === '100' && (
                    <div className="col-span-2">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-small font-medium">自定义配置</span>
                        <Button
                          type="button"
                          size="sm"
                          variant="flat"
                          color="primary"
                          startContent={<Plus size={14} />}
                          onPress={handleCreateCustomConfig}
                        >
                          创建新配置
                        </Button>
                      </div>
                      <Select
                        placeholder="选择配置模板"
                        selectedKeys={formData.custom_config_id > 0 ? [formData.custom_config_id.toString()] : []}
                        onSelectionChange={(keys) => {
                          const configId = parseInt([...keys][0] as string || '0');
                          setFormData({...formData, custom_config_id: configId});
                        }}
                        isRequired
                        isLoading={loadingConfigs}
                        description="选择或创建一个自定义 API 配置模板"
                      >
                        {customConfigs.map((config) => (
                          <SelectItem key={config.id.toString()} textValue={config.name}>
                            <div className="flex flex-col">
                              <span className="text-small">{config.name}</span>
                              {config.description && (
                                <span className="text-tiny text-default-400">{config.description}</span>
                              )}
                            </div>
                          </SelectItem>
                        ))}
                      </Select>
                    </div>
                  )}

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
                    {!editingChannel && (
                      <div className="flex items-center gap-2">
                        <span className="text-small font-medium">模型来源</span>
                        <Button
                          type="button"
                          size="sm"
                          variant={modelSourceMode === 'manual' ? 'solid' : 'flat'}
                          color={modelSourceMode === 'manual' ? 'primary' : 'default'}
                          onPress={() => setModelSourceMode('manual')}
                        >
                          手动输入
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant={modelSourceMode === 'sync' ? 'solid' : 'flat'}
                          color={modelSourceMode === 'sync' ? 'secondary' : 'default'}
                          onPress={() => setModelSourceMode('sync')}
                        >
                          同步上游模型
                        </Button>
                      </div>
                    )}
                    <div className="flex justify-between items-center">
                      <span className="text-small font-medium">模型列表 (逗号分隔)</span>
                      {editingChannel?.id ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="flat"
                          color="secondary"
                          startContent={<Download size={14} />}
                          isLoading={fetchingModels === editingChannel.id}
                          onPress={() => handleFetchModels(editingChannel.id!)}
                        >
                          拉取模型
                        </Button>
                      ) : modelSourceMode === 'sync' ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="flat"
                          color="secondary"
                          startContent={<Download size={14} />}
                          isLoading={fetchingModels === -1}
                          onPress={handleFetchModelsByForm}
                        >
                          同步上游模型
                        </Button>
                      ) : null}
                    </div>
                    <div className="border border-default-200 rounded-xl p-2 min-h-[80px] flex flex-wrap gap-1.5">
                      {formData.models
                        ? formData.models.split(',').map((m: string) => m.trim()).filter(Boolean).map((m: string) => (
                            <Chip
                              key={m}
                              size="sm"
                              variant="flat"
                              color="primary"
                              onClose={() => removeModel(m)}
                            >
                              {m}
                            </Chip>
                          ))
                        : <span className="text-default-400 text-small px-1 py-0.5">
                            {modelSourceMode === 'sync' && !editingChannel ? '点击"同步上游模型"后自动填入，可删改后保存' : '请在下方输入框逐个添加或粘贴逗号分隔列表'}
                          </span>
                      }
                    </div>
                    <div className="flex gap-2 items-center">
                      <Input
                        size="sm"
                        placeholder="输入模型名称后按 Enter 添加，或粘贴逗号分隔列表"
                        value={newModelInput}
                        onValueChange={setNewModelInput}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            newModelInput.split(',').forEach((m) => addModel(m));
                          }
                        }}
                      />
                      <Button
                        type="button"
                        size="sm"
                        isIconOnly
                        color="primary"
                        variant="flat"
                        onPress={() => newModelInput.split(',').forEach((m) => addModel(m))}
                      >
                        <Plus size={16} />
                      </Button>
                    </div>
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
                      <Button type="button" isIconOnly size="sm" color="primary" variant="flat" onPress={handleAddMapping} className="mb-0.5">
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
                          className="flex items-center gap-1"
                        >
                          {m.from} <ArrowRight size={12} /> {m.to}
                        </Chip>
                      ))}
                      {modelMapping.length === 0 && <span className="text-xs text-default-400">暂无映射</span>}
                    </div>
                  </div>
                </Form>
              </DrawerBody>
              <DrawerFooter>
                <Button color="danger" variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" isLoading={submitting} onPress={() => handleSubmit(onClose)}>
                  保存
                </Button>
              </DrawerFooter>
            </>
          )}
        </DrawerContent>
      </Drawer>

      <Modal isOpen={isMultiKeyOpen} onOpenChange={onMultiKeyOpenChange} size="xl" scrollBehavior="inside">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>
                多 Key 管理 {multiKeyChannel ? `#${multiKeyChannel.id} ${multiKeyChannel.name}` : ''}
              </ModalHeader>
              <ModalBody className="space-y-4">
                <div className="flex items-center gap-2 flex-wrap">
                  <Chip color="primary" variant="flat">总数 {multiKeyTotal}</Chip>
                  <Chip color="success" variant="flat">启用 {multiKeySummary.enabled}</Chip>
                  <Chip color="warning" variant="flat">手禁 {multiKeySummary.manual}</Chip>
                  <Chip color="danger" variant="flat">自禁 {multiKeySummary.auto}</Chip>
                </div>

                <div className="flex items-end gap-2 flex-wrap">
                  <Select
                    label="状态筛选"
                    selectedKeys={[multiKeyStatusFilter]}
                    className="w-40"
                    onSelectionChange={(keys) => {
                      if (!multiKeyChannel) return;
                      const v = ([...keys][0] as 'all' | '1' | '2' | '3') || 'all';
                      setMultiKeyStatusFilter(v);
                      setMultiKeyPage(1);
                      fetchMultiKeyStatus(multiKeyChannel.id, { page: 1, pageSize: multiKeyPageSize, status: v });
                    }}
                  >
                    <SelectItem key="all">全部</SelectItem>
                    <SelectItem key="1">启用</SelectItem>
                    <SelectItem key="2">手动禁用</SelectItem>
                    <SelectItem key="3">自动禁用</SelectItem>
                  </Select>

                  <Select
                    label="每页"
                    selectedKeys={[String(multiKeyPageSize)]}
                    className="w-28"
                    onSelectionChange={(keys) => {
                      if (!multiKeyChannel) return;
                      const v = parseInt(([...keys][0] as string) || '10', 10);
                      setMultiKeyPageSize(v);
                      setMultiKeyPage(1);
                      fetchMultiKeyStatus(multiKeyChannel.id, { page: 1, pageSize: v });
                    }}
                  >
                    <SelectItem key="10">10</SelectItem>
                    <SelectItem key="20">20</SelectItem>
                    <SelectItem key="50">50</SelectItem>
                  </Select>

                  <Button
                    variant="flat"
                    startContent={<RefreshCw size={14} />}
                    isLoading={multiKeyLoading}
                    onPress={() => multiKeyChannel && fetchMultiKeyStatus(multiKeyChannel.id)}
                  >
                    刷新
                  </Button>
                  <Button
                    color="success"
                    variant="flat"
                    isLoading={multiKeyActionLoading === 'enable_all_keys:all'}
                    onPress={() => doMultiKeyAction('enable_all_keys')}
                  >
                    全部启用
                  </Button>
                  <Button
                    color="warning"
                    variant="flat"
                    isLoading={multiKeyActionLoading === 'disable_all_keys:all'}
                    onPress={() => doMultiKeyAction('disable_all_keys')}
                  >
                    全部禁用
                  </Button>
                  <Button
                    color="danger"
                    variant="flat"
                    isLoading={multiKeyActionLoading === 'delete_disabled_keys:all'}
                    onPress={async () => {
                      if (!await confirm({ title: '删除自动禁用 Key', message: '将删除全部自动禁用（状态=3）的 Key，是否继续？', danger: true })) return;
                      doMultiKeyAction('delete_disabled_keys');
                    }}
                  >
                    删除自动禁用
                  </Button>
                </div>

                <div className="space-y-2">
                  <span className="text-small font-medium">覆盖设置 Key 列表（每行一个）</span>
                  <Textarea
                    minRows={4}
                    placeholder="每行一个 Key"
                    value={multiKeyEditorText}
                    onValueChange={setMultiKeyEditorText}
                    description="提交后将重建多 Key 列表并清空历史禁用状态"
                  />
                  <div className="flex justify-end">
                    <Button
                      color="secondary"
                      variant="flat"
                      isLoading={multiKeyActionLoading === 'set:all'}
                      onPress={handleSetMultiKeys}
                    >
                      覆盖保存 Key 列表
                    </Button>
                  </div>
                </div>

                <div className="data-table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>#</th>
                        <th>预览</th>
                        <th>状态</th>
                        <th>禁用时间</th>
                        <th>原因</th>
                        <th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {multiKeyLoading ? (
                        <LoadingRows cols={6} rows={5} />
                      ) : multiKeyRows.length === 0 ? (
                        <tr><td colSpan={6}><EmptyState icon="🔑" title="暂无 Key 数据" description="当前筛选条件下没有结果" /></td></tr>
                      ) : multiKeyRows.map((row) => (
                        <tr key={row.index}>
                          <td>{row.index}</td>
                          <td className="font-mono text-xs">{row.key_preview || '-'}</td>
                          <td><Chip size="sm" color={getKeyStatusColor(row.status)} variant="flat">{getKeyStatusText(row.status)}</Chip></td>
                          <td>{formatUnix(row.disabled_time)}</td>
                          <td>{row.reason || '-'}</td>
                          <td>
                            <div className="flex items-center gap-2">
                              {row.status !== 1 ? (
                                <Button
                                  size="sm"
                                  color="success"
                                  variant="flat"
                                  isLoading={multiKeyActionLoading === `enable_key:${row.index}`}
                                  onPress={() => doMultiKeyAction('enable_key', row.index)}
                                >
                                  启用
                                </Button>
                              ) : (
                                <Button
                                  size="sm"
                                  color="warning"
                                  variant="flat"
                                  isLoading={multiKeyActionLoading === `disable_key:${row.index}`}
                                  onPress={() => doMultiKeyAction('disable_key', row.index)}
                                >
                                  禁用
                                </Button>
                              )}
                              <Button
                                size="sm"
                                color="danger"
                                variant="light"
                                isLoading={multiKeyActionLoading === `delete_key:${row.index}`}
                                onPress={async () => {
                                  if (!await confirm({ title: '删除 Key', message: `确定删除索引 ${row.index} 的 Key？`, danger: true })) return;
                                  doMultiKeyAction('delete_key', row.index);
                                }}
                              >
                                删除
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                <div className="flex justify-end">
                  <Pagination
                    page={multiKeyPage}
                    total={multiKeyTotalPages}
                    onChange={(p) => {
                      if (!multiKeyChannel) return;
                      setMultiKeyPage(p);
                      fetchMultiKeyStatus(multiKeyChannel.id, { page: p });
                    }}
                  />
                </div>
              </ModalBody>
              <ModalFooter>
                <Button variant="light" color="danger" onPress={onClose}>关闭</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      <Modal isOpen={isPromptOpen} onOpenChange={onPromptOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{promptConfig.title || '输入信息'}</ModalHeader>
              <ModalBody>
                {promptConfig.multiline ? (
                  <Textarea
                    minRows={promptConfig.readOnly ? 6 : 3}
                    placeholder={promptConfig.placeholder || ''}
                    value={promptValue}
                    onValueChange={setPromptValue}
                    isReadOnly={Boolean(promptConfig.readOnly)}
                    description={promptConfig.description}
                  />
                ) : (
                  <Input
                    placeholder={promptConfig.placeholder || ''}
                    value={promptValue}
                    onValueChange={setPromptValue}
                    isReadOnly={Boolean(promptConfig.readOnly)}
                    description={promptConfig.description}
                  />
                )}
              </ModalBody>
              <ModalFooter>
                <Button
                  variant="light"
                  onPress={() => {
                    closePromptDialog(null);
                    onClose();
                  }}
                >
                  取消
                </Button>
                <Button
                  color="primary"
                  isLoading={promptSubmitting}
                  onPress={() => {
                    setPromptSubmitting(true);
                    closePromptDialog(promptValue);
                    onClose();
                    setPromptSubmitting(false);
                  }}
                >
                  {promptConfig.confirmText || '确认'}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* Test Channel Modal */}
      <Modal isOpen={isTestOpen} onOpenChange={onTestOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>测试渠道可用性</ModalHeader>
              <ModalBody>
                <div className="space-y-4">
                  <div className="p-3 bg-warning-50 border border-warning-200 rounded-lg">
                    <p className="text-sm text-warning-800">⚠️ 测试会向上游发送真实请求，可能产生费用</p>
                  </div>

                  <Select
                    label="测试方式"
                    selectedKeys={new Set([testMode])}
                    onSelectionChange={(keys) => {
                      const selected = Array.from(keys)[0] as string;
                      setTestMode(selected);
                    }}
                  >
                    <SelectItem key="quick">快速测试（默认问题：现在几点了）</SelectItem>
                    <SelectItem key="custom">自定义问题测试</SelectItem>
                    <SelectItem key="model">测试指定模型</SelectItem>
                  </Select>

                  {testMode === 'custom' && (
                    <Input
                      label="测试问题"
                      placeholder="输入测试问题"
                      value={testPrompt}
                      onValueChange={setTestPrompt}
                    />
                  )}

                  {testMode === 'model' && (
                    <>
                      <Select
                        label="选择模型"
                        placeholder="选择模型"
                        selectedKeys={testModel ? new Set([testModel]) : new Set()}
                        onSelectionChange={(keys) => {
                          const selected = Array.from(keys)[0] as string;
                          setTestModel(selected || '');
                        }}
                      >
                        {(testChannel?.models?.split(',').filter(m => m.trim()) || []).map(m => (
                          <SelectItem key={m.trim()}>{m.trim()}</SelectItem>
                        ))}
                      </Select>
                      <Input
                        label="测试问题（可选）"
                        placeholder="留空使用默认问题"
                        value={testPrompt}
                        onValueChange={setTestPrompt}
                      />
                    </>
                  )}
                </div>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={executeTest} isLoading={testingId === testChannel?.id}>开始测试</Button>
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

      {/* 自定义配置创建模态框 */}
      <Modal isOpen={isCustomConfigOpen} onOpenChange={onCustomConfigOpenChange} size="full" scrollBehavior="inside" className="max-w-6xl mx-4">
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>
                <div className="flex items-center gap-2">
                  <Settings size={20} />
                  {editingCustomConfig ? '编辑自定义配置' : '创建自定义配置'}
                </div>
              </ModalHeader>
              <ModalBody className="py-6">
                <div className="grid grid-cols-3 gap-6">
                  {/* 左列 */}
                  <div className="space-y-4">
                    {/* 基础信息 */}
                    <div className="p-4 bg-default-50 dark:bg-default-100/50 rounded-xl border border-default-200">
                      <h4 className="font-semibold mb-3 flex items-center gap-2 text-sm">
                        <span>📝</span>
                        <span>基础信息</span>
                      </h4>
                      <div className="space-y-3">
                        <Input
                          label="配置名称"
                          placeholder="例如: DeepSeek Compatible"
                          value={customConfigFormData.name}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, name: v })}
                          isRequired
                          size="sm"
                        />
                        <Textarea
                          label="配置描述"
                          placeholder="简要说明此配置的用途"
                          value={customConfigFormData.description}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, description: v })}
                          size="sm"
                          minRows={2}
                        />
                      </div>
                    </div>

                    {/* 请求配置 */}
                    <div className="p-4 bg-default-50 dark:bg-default-100/50 rounded-xl border border-default-200">
                      <h4 className="font-semibold mb-3 flex items-center gap-2 text-sm">
                        <span>🔧</span>
                        <span>请求配置</span>
                      </h4>
                      <div className="space-y-3">
                        <Input
                          label="请求端点"
                          placeholder="/v1/chat/completions"
                          value={customConfigFormData.request_endpoint}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, request_endpoint: v })}
                          size="sm"
                        />
                        <Input
                          label="认证头模板"
                          placeholder="Bearer {key}"
                          value={customConfigFormData.auth_header_template}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, auth_header_template: v })}
                          description="使用 {key} 作为占位符"
                          size="sm"
                        />
                      </div>
                    </div>
                  </div>

                  {/* 中列：字段映射 + 响应路径 */}
                  <div className="space-y-4">
                    {/* 字段映射 */}
                    <div className="p-4 bg-default-50 dark:bg-default-100/50 rounded-xl border border-default-200">
                      <h4 className="font-semibold mb-3 flex items-center gap-2 text-sm">
                        <span>🔄</span>
                        <span>字段映射（OpenAI → 目标）</span>
                      </h4>
                      <div className="space-y-3">
                        <Input
                          label="model"
                          value={customConfigFormData.field_model}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, field_model: v })}
                          size="sm"
                        />
                        <Input
                          label="messages"
                          value={customConfigFormData.field_messages}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, field_messages: v })}
                          size="sm"
                        />
                        <Input
                          label="temperature"
                          value={customConfigFormData.field_temperature}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, field_temperature: v })}
                          size="sm"
                        />
                        <Input
                          label="max_tokens"
                          value={customConfigFormData.field_max_tokens}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, field_max_tokens: v })}
                          size="sm"
                        />
                      </div>
                      <p className="text-xs text-default-500 mt-3">
                        💡 不支持的字段填 "-" 或留空
                      </p>
                    </div>

                    {/* 响应路径 */}
                    <div className="p-4 bg-default-50 dark:bg-default-100/50 rounded-xl border border-default-200">
                      <h4 className="font-semibold mb-3 flex items-center gap-2 text-sm">
                        <span>📊</span>
                        <span>响应路径（点路径语法）</span>
                      </h4>
                      <div className="space-y-3">
                        <Input
                          label="内容路径"
                          placeholder="choices.0.message.content"
                          value={customConfigFormData.response_content_path}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, response_content_path: v })}
                          size="sm"
                        />
                        <Input
                          label="Prompt Tokens"
                          placeholder="usage.prompt_tokens"
                          value={customConfigFormData.response_prompt_tokens_path}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, response_prompt_tokens_path: v })}
                          size="sm"
                        />
                        <Input
                          label="Completion Tokens"
                          placeholder="usage.completion_tokens"
                          value={customConfigFormData.response_completion_tokens_path}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, response_completion_tokens_path: v })}
                          size="sm"
                        />
                      </div>
                    </div>
                  </div>

                  {/* 右列：流式配置 + 提示 */}
                  <div className="space-y-4">

                    {/* 流式配置 */}
                    <div className="p-4 bg-default-50 dark:bg-default-100/50 rounded-xl border border-default-200">
                      <h4 className="font-semibold mb-3 flex items-center gap-2 text-sm">
                        <span>📡</span>
                        <span>流式响应配置</span>
                      </h4>
                      <div className="space-y-3">
                        <Switch
                          isSelected={customConfigFormData.stream_enabled === 1}
                          onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, stream_enabled: v ? 1 : 0 })}
                          size="sm"
                        >
                          支持流式响应（SSE）
                        </Switch>
                        {customConfigFormData.stream_enabled === 1 && (
                          <div className="space-y-3 pt-2">
                            <div className="grid grid-cols-2 gap-3">
                              <Input
                                label="数据前缀"
                                placeholder="data: "
                                value={customConfigFormData.stream_data_prefix}
                                onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, stream_data_prefix: v })}
                                size="sm"
                              />
                              <Input
                                label="结束标记"
                                placeholder="[DONE]"
                                value={customConfigFormData.stream_end_marker}
                                onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, stream_end_marker: v })}
                                size="sm"
                              />
                            </div>
                            <Input
                              label="流内容路径"
                              placeholder="choices.0.delta.content"
                              value={customConfigFormData.stream_content_path}
                              onValueChange={(v) => setCustomConfigFormData({ ...customConfigFormData, stream_content_path: v })}
                              size="sm"
                            />
                          </div>
                        )}
                      </div>
                    </div>

                    {/* 提示信息 */}
                    <div className="p-4 bg-primary-50 dark:bg-primary-900/20 rounded-xl border border-primary-200 dark:border-primary-800">
                      <div className="flex items-start gap-3">
                        <span className="text-xl flex-shrink-0">💡</span>
                        <div className="flex-1 space-y-1">
                          <p className="text-xs font-semibold text-primary">配置说明</p>
                          <p className="text-xs text-default-700 dark:text-default-300 leading-relaxed space-y-1">
                            <span className="block">• 字段映射：将 OpenAI 字段映射到目标 API</span>
                            <span className="block">• 响应路径：使用点路径提取嵌套字段</span>
                            <span className="block">• 示例：<code className="bg-default-200 dark:bg-default-700 px-1.5 py-0.5 rounded text-xs">data.result.text</code></span>
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>
                  取消
                </Button>
                <Button color="primary" onPress={() => handleSubmitCustomConfig(onClose)} isLoading={submitting}>
                  {editingCustomConfig ? '更新配置' : '创建配置'}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
