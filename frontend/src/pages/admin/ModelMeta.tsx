import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import {
  Button,
  Checkbox,
  Chip,
  Dropdown,
  DropdownTrigger,
  DropdownMenu,
  DropdownItem,
  Input,
  Select,
  SelectItem,
  Switch,
  Textarea,
} from '../../components/ui';
import { Pagination } from '../../components/ui/Pagination';
import { useAuthStore } from '../../store/auth';
import { confirm } from '../../store/confirm';
import { toast } from '../../store/toast';
import { MoreHorizontal, Eye, EyeOff, Copy, Trash2, X, Plus } from 'lucide-react';

interface ModelMeta {
  id: number;
  vendor_id: number;
  model_name: string;
  display_name: string;
  model_type: string;
  name_rule: number; // 0=精确, 1=前缀, 2=后缀, 3=中缀
  matched_count?: number;
  matched_models?: string[];
  context_length: number;
  input_price: number;
  output_price: number;
  enabled: boolean;
  tags?: string;
  endpoints?: string;
  description?: string;
  icon?: string;
  sync_official?: boolean;
}

interface VendorOption {
  id: number;
  name: string;
}

// 🎀 渠道下拉选项～
interface ChannelOption {
  id: number;
  name: string;
  status?: number;
}

interface UpstreamResult {
  missing?: string[];
  missing_count?: number;
  upstream_count?: number;
  channel_count?: number;
  failed_channels?: Array<{ channel_id: number; channel_name: string; error: string }>;
  created?: number;
  skipped?: number;
  created_models?: string[];
  skipped_models?: string[];
}

// 🌸 名称匹配规则配置～
const NAME_RULE_CONFIG = {
  0: { label: '精确', color: 'success' },
  1: { label: '前缀', color: 'warning' },
  2: { label: '后缀', color: 'warning' },
  3: { label: '中缀', color: 'danger' },
} as const;

// 🌸 匹配规则选项～
const MATCH_RULE_OPTIONS = [
  { value: 0, label: '精确匹配', desc: '完全相等' },
  { value: 1, label: '前缀匹配', desc: '以此开头' },
  { value: 2, label: '后缀匹配', desc: '以此结尾' },
  { value: 3, label: '中缀匹配', desc: '包含此串' },
];

// 🌸 端点模板 - 对齐 new-api ENDPOINT_TEMPLATES～
const ENDPOINT_TEMPLATES: Record<string, { path: string; method: string }> = {
  'openai': { path: '/v1/chat/completions', method: 'POST' },
  'openai-response': { path: '/v1/responses', method: 'POST' },
  'anthropic': { path: '/v1/messages', method: 'POST' },
  'gemini': { path: '/v1beta/models/{model}:generateContent', method: 'POST' },
  'jina-rerank': { path: '/rerank', method: 'POST' },
  'image-generation': { path: '/v1/images/generations', method: 'POST' },
  'embeddings': { path: '/v1/embeddings', method: 'POST' },
};

// 🎀 端点行数据结构～
interface EndpointRow {
  key: string;
  path: string;
  method: string;
}

// 🎀 解析端点 JSON 为可视化行～
const parseEndpointRows = (endpoints?: string): EndpointRow[] => {
  if (!endpoints) return [];
  try {
    const parsed = JSON.parse(endpoints);
    if (Array.isArray(parsed)) {
      // 兼容旧格式：["openai", "embeddings"]
      return parsed.map((k) => ({
        key: String(k),
        path: ENDPOINT_TEMPLATES[String(k)]?.path || '',
        method: ENDPOINT_TEMPLATES[String(k)]?.method || 'POST',
      }));
    }
    if (parsed && typeof parsed === 'object') {
      return Object.entries(parsed).map(([key, val]) => {
        if (val && typeof val === 'object') {
          const obj = val as { path?: string; method?: string };
          return { key, path: obj.path || '', method: obj.method || 'POST' };
        }
        return { key, path: String(val ?? ''), method: 'POST' };
      });
    }
    return [];
  } catch {
    // 逗号分隔的旧数据
    return endpoints.split(',').map((k) => k.trim()).filter(Boolean).map((k) => ({
      key: k,
      path: ENDPOINT_TEMPLATES[k]?.path || '',
      method: ENDPOINT_TEMPLATES[k]?.method || 'POST',
    }));
  }
};

// 🎀 序列化端点行为 JSON～
const serializeEndpointRows = (rows: EndpointRow[]): string => {
  const valid = rows.filter((r) => r.key.trim() !== '');
  if (valid.length === 0) return '';
  const obj: Record<string, { path: string; method: string }> = {};
  valid.forEach((r) => {
    obj[r.key.trim()] = { path: r.path.trim(), method: r.method || 'POST' };
  });
  return JSON.stringify(obj);
};

export default function ModelMetaManagement() {
  const { token } = useAuthStore();
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const [items, setItems] = useState<ModelMeta[]>([]);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [vendorFilter, setVendorFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  const [editing, setEditing] = useState<ModelMeta | null>(null);
  const [vendors, setVendors] = useState<VendorOption[]>([]);
  const [form, setForm] = useState<Partial<ModelMeta>>({
    vendor_id: 0,
    model_name: '',
    display_name: '',
    model_type: 'text',
    name_rule: 0,
    context_length: 0,
    input_price: 0,
    output_price: 0,
    enabled: true,
    tags: '',
    endpoints: '',
    description: '',
    icon: '',
    sync_official: true,
  });

  // 🎀 渠道下拉多选（告别手输ID啦～）
  const [channelOptions, setChannelOptions] = useState<ChannelOption[]>([]);
  const [selectedChannelKeys, setSelectedChannelKeys] = useState<Set<string>>(new Set());
  const [missingModels, setMissingModels] = useState<string[]>([]);
  const [missingChannelCount, setMissingChannelCount] = useState(0);
  const [previewResult, setPreviewResult] = useState<UpstreamResult | null>(null);
  const [syncResult, setSyncResult] = useState<UpstreamResult | null>(null);

  const [missingLoading, setMissingLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);

  // 🎀 批量选择状态～
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());

  // 🎀 端点可视化编辑行～
  const [endpointRows, setEndpointRows] = useState<EndpointRow[]>([]);

  // 🎀 同步向导对话框状态～
  const [isSyncWizardOpen, setIsSyncWizardOpen] = useState(false);
  const [syncLocale, setSyncLocale] = useState<'zh' | 'en' | 'ja'>('zh');

  const parsedChannelIds = useMemo(() => {
    return Array.from(selectedChannelKeys)
      .map((v) => Number(v))
      .filter((v) => Number.isInteger(v) && v > 0);
  }, [selectedChannelKeys]);

  const channelQuery = useMemo(() => {
    return parsedChannelIds.length > 0 ? `?channel_ids=${parsedChannelIds.join(',')}` : '';
  }, [parsedChannelIds]);

  const fetchData = async (targetPage = 1) => {
    const pageSize = 20;
    const q = keyword.trim();
    const base = q ? `/api/models/search?keyword=${encodeURIComponent(q)}` : '/api/models';
    let url = `${base}${base.includes('?') ? '&' : '?'}page=${targetPage}&page_size=${pageSize}`;

    // 添加状态筛选
    if (statusFilter) {
      url += `&enabled=${statusFilter}`;
    }

    // 添加供应商筛选
    if (vendorFilter) {
      url += `&vendor_id=${vendorFilter}`;
    }

    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
    const data = await res.json();
    if (data.code !== 0) {
      toast.error(data.msg || '拉取模型元数据失败');
      return;
    }

    const payload = data.data;
    if (Array.isArray(payload)) {
      setItems(payload);
      setPage(1);
      setTotalPages(1);
      return;
    }

    const nextItems = payload?.items || [];
    const total = Number(payload?.total ?? nextItems.length);
    const ps = Number(payload?.page_size ?? pageSize) || pageSize;
    const nextPage = Number(payload?.page ?? targetPage) || targetPage;
    setItems(nextItems);
    setPage(nextPage);
    setTotalPages(Math.max(1, Math.ceil(total / ps)));
  };

  const fetchVendors = async () => {
    try {
      const res = await fetch('/api/vendors?page=1&page_size=1000', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code !== 0) return;
      const payload = data.data;
      if (Array.isArray(payload)) {
        setVendors(payload);
        return;
      }
      setVendors(payload?.items || []);
    } catch {
      // ignore vendor preload errors
    }
  };

  // 🎀 拉取渠道列表，用于下拉多选～
  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/channel', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code !== 0) return;
      const payload = data.data;
      const list: ChannelOption[] = Array.isArray(payload) ? payload : (payload?.items || []);
      setChannelOptions(list.map((c: any) => ({ id: c.id, name: c.name, status: c.status })));
    } catch {
      // ignore channel preload errors
    }
  };

  const vendorNameMap = useMemo(() => {
    const map: Record<number, string> = {};
    vendors.forEach((v) => {
      map[v.id] = v.name;
    });
    return map;
  }, [vendors]);

  // 🎀 解析标签字符串为数组～
  const parseTags = (tags?: string): string[] => {
    if (!tags) return [];
    try {
      const parsed = JSON.parse(tags);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return tags.split(',').map((t) => t.trim()).filter(Boolean);
    }
  };

  // 🎀 解析端点字符串为数组～
  const parseEndpoints = (endpoints?: string): string[] => {
    if (!endpoints) return [];
    try {
      const parsed = JSON.parse(endpoints);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return endpoints.split(',').map((e) => e.trim()).filter(Boolean);
    }
  };

  useEffect(() => {
    fetchData(1);
    fetchVendors();
    fetchChannels();
  }, []);

  // 筛选条件变化时重新加载
  useEffect(() => {
    if (statusFilter !== '' || vendorFilter !== '' || keyword !== '') {
      fetchData(1);
    }
  }, [statusFilter, vendorFilter, keyword]);

  const openCreate = () => {
    setEditing(null);
    setForm({
      vendor_id: 0,
      model_name: '',
      display_name: '',
      model_type: 'text',
      name_rule: 0,
      context_length: 0,
      input_price: 0,
      output_price: 0,
      enabled: true,
      tags: '',
      endpoints: '',
      description: '',
      icon: '',
      sync_official: true,
    });
    setEndpointRows([]);
    setIsDrawerOpen(true);
  };

  const openEdit = (v: ModelMeta) => {
    setEditing(v);
    setForm(v);
    setEndpointRows(parseEndpointRows(v.endpoints));
    setIsDrawerOpen(true);
  };

  const submit = async () => {
    const method = editing ? 'PUT' : 'POST';
    const payload = { ...form, endpoints: serializeEndpointRows(endpointRows) };
    const body = editing ? { ...payload, id: editing.id } : payload;
    const res = await fetch('/api/models', {
      method,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(editing ? '更新成功' : '创建成功');
      await fetchData(page);
      setIsDrawerOpen(false);
      return;
    }
    toast.error(data.msg || '保存失败');
  };

  const remove = async (id: number) => {
    if (!await confirm({ title: '删除模型元数据', message: '确认删除该记录？', danger: true })) return;
    const res = await fetch(`/api/models/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success('删除成功');
      fetchData(page);
      return;
    }
    toast.error(data.msg || '删除失败');
  };

  // 🎀 批量操作函数～
  const toggleSelection = (id: number) => {
    const newSelected = new Set(selectedIds);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedIds(newSelected);
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === items.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(items.map((item) => item.id)));
    }
  };

  const batchEnable = async (enabled: boolean) => {
    if (selectedIds.size === 0) {
      toast.warning('请先选择要操作的模型');
      return;
    }
    const ids = Array.from(selectedIds);
    const res = await fetch('/api/models/batch', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ ids, enabled }),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(`已${enabled ? '启用' : '禁用'} ${ids.length} 个模型`);
      setSelectedIds(new Set());
      fetchData(page);
      return;
    }
    toast.error(data.msg || '批量操作失败');
  };

  const batchDelete = async () => {
    if (selectedIds.size === 0) {
      toast.warning('请先选择要删除的模型');
      return;
    }
    if (!await confirm({ title: '批量删除', message: `确认删除选中的 ${selectedIds.size} 个模型？`, danger: true })) return;
    const ids = Array.from(selectedIds);
    const res = await fetch('/api/models/batch', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ ids }),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(`已删除 ${ids.length} 个模型`);
      setSelectedIds(new Set());
      fetchData(page);
      return;
    }
    toast.error(data.msg || '批量删除失败');
  };

  const toggleEnabled = async (id: number, currentEnabled: boolean) => {
    const res = await fetch('/api/models', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ id, enabled: !currentEnabled }),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(currentEnabled ? '已禁用' : '已启用');
      fetchData(page);
      return;
    }
    toast.error(data.msg || '操作失败');
  };

  const copyModelName = (name: string) => {
    navigator.clipboard.writeText(name);
    toast.success('已复制到剪贴板');
  };

  const fetchMissingModels = async () => {
    setMissingLoading(true);
    try {
      const res = await fetch(`/api/models/missing${channelQuery}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        const payload = data.data || {};
        setMissingModels(payload.missing || []);
        setMissingChannelCount(Number(payload.channel_count ?? 0));
        toast.success(`缺失模型 ${Number(payload.missing_count ?? 0)} 个`);
      } else {
        toast.error(data.msg || '缺失模型扫描失败');
      }
    } finally {
      setMissingLoading(false);
    }
  };

  const previewSync = async () => {
    setPreviewLoading(true);
    try {
      const res = await fetch(`/api/models/sync_upstream/preview${channelQuery}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setPreviewResult(data.data || {});
        toast.success('已完成上游预览');
      } else {
        toast.error(data.msg || '上游预览失败');
      }
    } finally {
      setPreviewLoading(false);
    }
  };

  const doSync = async () => {
    setSyncLoading(true);
    try {
      // 第一步：从渠道同步模型名称～
      const res = await fetch('/api/models/sync_upstream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ channel_ids: parsedChannelIds, enabled: true }),
      });
      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '同步失败');
        return;
      }
      const payload = data.data || {};
      setSyncResult(payload);

      // 🎀 第二步：从官方元数据仓库补齐描述/图标/标签/端点/供应商～
      toast.info?.('正在从官方仓库补齐模型元信息...');
      const metaRes = await fetch('/api/models/sync_official', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ locale: syncLocale }),
      });
      const metaData = await metaRes.json();
      if (metaData.code === 0) {
        const mp = metaData.data || {};
        toast.success(
          `同步完成：新增 ${Number(payload.created ?? 0)} 个，元信息补齐 ${Number(mp.enriched_models ?? 0)} 个，新建供应商 ${Number(mp.created_vendors ?? 0)} 个`
        );
      } else {
        toast.success(`新增 ${Number(payload.created ?? 0)} 个模型元数据`);
        toast.error(metaData.msg || '官方元信息补齐失败（模型名称已同步）');
      }
      setIsSyncWizardOpen(false);
      fetchData(1);
    } finally {
      setSyncLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="模型元数据"
        description="统一管理模型元数据，并支持从渠道上游预览/同步"
        actions={(
          <div className="flex gap-2">
            <Button color="primary" onPress={openCreate}>新增</Button>
            <Button variant="flat" onPress={() => fetchData(1)}>刷新</Button>
          </div>
        )}
      />

      {/* 顶部筛选栏 */}
      <div className="flex items-center gap-3 flex-wrap">
        <Input
          size="sm"
          placeholder="按模型名称筛选..."
          value={keyword}
          onValueChange={setKeyword}
          className="w-64"
        />
        <div className="flex gap-2">
          <Button
            size="sm"
            variant={statusFilter === '' ? 'solid' : 'bordered'}
            onPress={() => {
              setStatusFilter('');
              setPage(1);
            }}
          >
            全部
          </Button>
          <Button
            size="sm"
            variant={statusFilter === '1' ? 'solid' : 'bordered'}
            color={statusFilter === '1' ? 'success' : 'default'}
            onPress={() => {
              setStatusFilter('1');
              setPage(1);
            }}
          >
            已启用
          </Button>
          <Button
            size="sm"
            variant={statusFilter === '0' ? 'solid' : 'bordered'}
            color={statusFilter === '0' ? 'danger' : 'default'}
            onPress={() => {
              setStatusFilter('0');
              setPage(1);
            }}
          >
            已禁用
          </Button>
        </div>
        <Select
          size="sm"
          placeholder="供应商筛选"
          selectedKeys={vendorFilter ? [vendorFilter] : []}
          onSelectionChange={(keys) => {
            const selected = Array.from(keys)[0]?.toString() || '';
            setVendorFilter(selected);
            setPage(1);
          }}
          className="w-48"
        >
          <SelectItem key="">全部供应商</SelectItem>
          {vendors.map((v) => (
            <SelectItem key={v.id.toString()}>{v.name}</SelectItem>
          ))}
        </Select>
        <Button size="sm" variant="flat" onPress={() => { setKeyword(''); setStatusFilter(''); setVendorFilter(''); setPage(1); }}>
          清空筛选
        </Button>
      </div>

      {/* 🎀 批量操作栏～ */}
      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 p-3 rounded-lg border" style={{ borderColor: 'var(--border-color)', background: 'var(--bg-surface)' }}>
          <span className="text-sm text-default-600">已选择 {selectedIds.size} 项</span>
          <div className="flex gap-2">
            <Button size="sm" color="success" variant="flat" onPress={() => batchEnable(true)}>
              批量启用
            </Button>
            <Button size="sm" color="warning" variant="flat" onPress={() => batchEnable(false)}>
              批量禁用
            </Button>
            <Button size="sm" color="danger" variant="flat" onPress={batchDelete}>
              批量删除
            </Button>
            <Button size="sm" variant="flat" onPress={() => setSelectedIds(new Set())}>
              取消选择
            </Button>
          </div>
        </div>
      )}

      {/* 上游同步工具区 */}
      <div className="rounded-xl border p-4" style={{ borderColor: 'var(--border-color)', background: 'var(--bg-surface)' }}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
          <Select
            label="渠道过滤（可选）"
            size="sm"
            placeholder="全部启用渠道"
            selectionMode="multiple"
            selectedKeys={selectedChannelKeys}
            onSelectionChange={(keys) => setSelectedChannelKeys(new Set(Array.from(keys as Set<string>).map(String)))}
          >
            {channelOptions.map((c) => (
              <SelectItem key={String(c.id)}>{`#${c.id} ${c.name}`}</SelectItem>
            ))}
          </Select>
          <Button variant="flat" isLoading={missingLoading} onPress={fetchMissingModels}>扫描缺失模型</Button>
          <Button variant="flat" color="warning" isLoading={previewLoading} onPress={previewSync}>上游预览</Button>
          <Button color="primary" onPress={() => setIsSyncWizardOpen(true)}>一键同步到元数据</Button>
        </div>

        <div className="mt-3 flex flex-wrap gap-2 text-xs" style={{ color: 'var(--text-secondary)' }}>
          <Chip size="sm" variant="flat">缺失模型: {missingModels.length}</Chip>
          <Chip size="sm" variant="flat">涉及渠道: {missingChannelCount}</Chip>
          <Chip size="sm" variant="flat">
            筛选渠道: {parsedChannelIds.length > 0
              ? parsedChannelIds.map((id) => channelOptions.find((c) => c.id === id)?.name || `#${id}`).join(', ')
              : '全部启用渠道'}
          </Chip>
        </div>

        {missingModels.length > 0 && (
          <div className="mt-3 text-xs" style={{ color: 'var(--text-secondary)' }}>
            缺失模型：{missingModels.slice(0, 20).join(', ')}{missingModels.length > 20 ? ' ...' : ''}
          </div>
        )}

        {previewResult && (
          <div className="mt-3 text-xs" style={{ color: 'var(--text-secondary)' }}>
            预览结果：缺失 {previewResult.missing_count || 0} / 上游总计 {previewResult.upstream_count || 0} / 渠道数 {previewResult.channel_count || 0}
          </div>
        )}

        {syncResult && (
          <div className="mt-2 text-xs" style={{ color: 'var(--text-secondary)' }}>
            同步结果：新增 {syncResult.created || 0}，跳过 {syncResult.skipped || 0}
          </div>
        )}
      </div>

      {/* 紧凑型表格 - 对齐参考项目设计 */}
      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th style={{ width: '40px' }}>
                <Checkbox
                  isSelected={items.length > 0 && selectedIds.size === items.length}
                  onValueChange={toggleSelectAll}
                  aria-label="全选"
                />
              </th>
              <th style={{ width: '60px' }}>ID</th>
              <th style={{ width: '260px', minWidth: '200px' }}>模型名称</th>
              <th style={{ width: '100px' }}>匹配类型</th>
              <th style={{ width: '80px' }}>状态</th>
              <th style={{ width: '120px' }}>供应商</th>
              <th style={{ width: '150px' }}>描述</th>
              <th style={{ width: '120px' }}>标签</th>
              <th style={{ width: '150px' }}>端点</th>
              <th style={{ width: '100px' }}>输入价格</th>
              <th style={{ width: '100px' }}>输出价格</th>
              <th style={{ width: '100px' }}>上下文</th>
              <th style={{ width: '140px' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((v) => {
              const tags = parseTags(v.tags);
              const endpoints = parseEndpoints(v.endpoints);
              const matchRule = NAME_RULE_CONFIG[v.name_rule as 0 | 1 | 2 | 3] || NAME_RULE_CONFIG[0];
              const isSelected = selectedIds.has(v.id);

              return (
                <tr key={v.id}>
                  {/* 🎀 复选框列～ */}
                  <td>
                    <Checkbox
                      isSelected={isSelected}
                      onValueChange={() => toggleSelection(v.id)}
                      aria-label={`选择模型 ${v.model_name}`}
                    />
                  </td>

                  <td className="text-center">{v.id}</td>

                  {/* 模型名称 - 双行显示 */}
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="font-mono text-sm font-medium">{v.model_name}</div>
                    </div>
                    {v.display_name && v.display_name !== v.model_name && (
                      <div className="text-xs text-default-500 mt-0.5">{v.display_name}</div>
                    )}
                  </td>

                  {/* 匹配类型 - 带悬停提示 */}
                  <td>
                    <div title={v.matched_models?.length ? `匹配模型: ${v.matched_models.join(', ')}` : undefined}>
                      <Chip
                        size="sm"
                        variant="flat"
                        color={matchRule.color as any}
                      >
                        {matchRule.label}{v.matched_count ? ` (${v.matched_count})` : ''}
                      </Chip>
                    </div>
                  </td>

                  {/* 状态 */}
                  <td>
                    <Chip size="sm" variant="flat" color={v.enabled ? 'success' : 'default'}>
                      {v.enabled ? '已启用' : '已禁用'}
                    </Chip>
                  </td>

                  {/* 供应商 */}
                  <td>
                    <Chip size="sm" variant="flat" color="secondary">
                      {vendorNameMap[v.vendor_id] || `#${v.vendor_id}`}
                    </Chip>
                  </td>

                  {/* 描述 */}
                  <td>
                    <div className="text-xs text-default-500 truncate max-w-[150px]" title={v.description}>
                      {v.description || '-'}
                    </div>
                  </td>

                  {/* 标签 */}
                  <td>
                    <div className="flex flex-wrap gap-1 max-w-[120px]">
                      {tags.slice(0, 2).map((tag, idx) => (
                        <Chip key={idx} size="sm" variant="dot">
                          {tag}
                        </Chip>
                      ))}
                      {tags.length > 2 && (
                        <div title={tags.join(', ')}>
                          <Chip size="sm" variant="flat">
                            +{tags.length - 2}
                          </Chip>
                        </div>
                      )}
                      {tags.length === 0 && <span className="text-xs text-default-400">-</span>}
                    </div>
                  </td>

                  {/* 端点 */}
                  <td>
                    <div className="flex flex-wrap gap-1 max-w-[150px]">
                      {endpoints.slice(0, 2).map((ep, idx) => (
                        <Chip key={idx} size="sm" variant="dot">
                          {ep}
                        </Chip>
                      ))}
                      {endpoints.length > 2 && (
                        <div title={endpoints.join(', ')}>
                          <Chip size="sm" variant="flat">
                            +{endpoints.length - 2}
                          </Chip>
                        </div>
                      )}
                      {endpoints.length === 0 && <span className="text-xs text-default-400">-</span>}
                    </div>
                  </td>

                  {/* 输入价格 */}
                  <td className="text-xs font-mono">
                    ${v.input_price.toFixed(6)}
                  </td>

                  {/* 输出价格 */}
                  <td className="text-xs font-mono">
                    ${v.output_price.toFixed(6)}
                  </td>

                  {/* 上下文 */}
                  <td className="text-xs">
                    {v.context_length > 0 ? v.context_length.toLocaleString() : '-'}
                  </td>

                  {/* 🎀 操作 - 增强的行操作菜单～ */}
                  <td>
                    <div className="flex gap-2 items-center">
                      <Button size="sm" variant="flat" onPress={() => openEdit(v)}>编辑</Button>
                      <Dropdown>
                        <DropdownTrigger>
                          <Button size="sm" isIconOnly variant="flat">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownTrigger>
                        <DropdownMenu aria-label="行操作">
                          <DropdownItem
                            key="toggle"
                            startContent={v.enabled ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            onPress={() => toggleEnabled(v.id, v.enabled)}
                          >
                            {v.enabled ? '禁用' : '启用'}
                          </DropdownItem>
                          <DropdownItem
                            key="copy"
                            startContent={<Copy className="w-4 h-4" />}
                            onPress={() => copyModelName(v.model_name)}
                          >
                            复制模型名
                          </DropdownItem>
                          <DropdownItem
                            key="delete"
                            color="danger"
                            startContent={<Trash2 className="w-4 h-4" />}
                            onPress={() => remove(v.id)}
                          >
                            删除
                          </DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end">
        <Pagination total={totalPages} page={page} onChange={(p) => fetchData(p)} />
      </div>

      {/* 🎀 侧边抽屉编辑器～ */}
      {isDrawerOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-end">
          {/* 背景遮罩 */}
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setIsDrawerOpen(false)}
          />

          {/* 抽屉内容 */}
          <div className="relative h-full w-full max-w-2xl shadow-xl flex flex-col animate-slide-in-right" style={{ background: 'var(--bg-base)' }}>
            {/* 头部 */}
            <div className="flex items-start justify-between p-6 border-b">
              <div>
                <h2 className="text-lg font-semibold">{editing ? '编辑模型元数据' : '新增模型元数据'}</h2>
                <p className="text-sm text-default-500 mt-1">
                  {editing ? '更新模型配置信息并保存' : '添加新的模型到系统中'}
                </p>
              </div>
              <Button
                isIconOnly
                variant="light"
                size="sm"
                onPress={() => setIsDrawerOpen(false)}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>

            {/* 内容区 - 可滚动 */}
            <div className="flex-1 overflow-y-auto p-6 space-y-6">
              {/* 🌸 基础信息区 */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-default-700">基础信息</h3>

                <Input
                  label="模型名称 *"
                  placeholder="gpt-4, claude-3-opus, etc."
                  description="模型的唯一标识符"
                  value={form.model_name || ''}
                  onValueChange={(v) => setForm({ ...form, model_name: v })}
                  isRequired
                />

                <Input
                  label="显示名称"
                  placeholder="GPT-4 Turbo"
                  description="用于展示的友好名称（可选）"
                  value={form.display_name || ''}
                  onValueChange={(v) => setForm({ ...form, display_name: v })}
                />

                <Textarea
                  label="描述"
                  placeholder="描述这个模型的特点..."
                  description="模型的详细说明"
                  value={form.description || ''}
                  onValueChange={(v) => setForm({ ...form, description: v })}
                  minRows={3}
                />

                <Input
                  label="图标"
                  placeholder="OpenAI, Anthropic, etc."
                  description="@lobehub/icons 图标键名"
                  value={form.icon || ''}
                  onValueChange={(v) => setForm({ ...form, icon: v })}
                />

                <Select
                  label="供应商"
                  placeholder="选择供应商"
                  selectedKeys={form.vendor_id ? [String(form.vendor_id)] : []}
                  onSelectionChange={(keys) => setForm({ ...form, vendor_id: Number([...keys][0] as string) || 0 })}
                >
                  <SelectItem key="0">未指定</SelectItem>
                  {vendors.map((v) => (
                    <SelectItem key={String(v.id)}>{v.name}</SelectItem>
                  ))}
                </Select>

                <Input
                  label="标签"
                  placeholder="vision, function-calling (用逗号分隔)"
                  description="按回车或逗号添加标签"
                  value={form.tags || ''}
                  onValueChange={(v) => setForm({ ...form, tags: v })}
                />
              </div>

              {/* 🌸 匹配规则区 */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-default-700">匹配规则</h3>

                <div className="grid grid-cols-2 gap-3">
                  {MATCH_RULE_OPTIONS.map((rule) => (
                    <Button
                      key={rule.value}
                      variant={(form.name_rule ?? 0) === rule.value ? 'flat' : 'bordered'}
                      color={(form.name_rule ?? 0) === rule.value ? 'primary' : 'default'}
                      className="h-auto py-3 flex-col items-start"
                      onPress={() => setForm((prev) => ({ ...prev, name_rule: rule.value }))}
                    >
                      <div className="font-medium text-sm">{rule.label}</div>
                      <div className="text-xs text-default-500 mt-1">{rule.desc}</div>
                    </Button>
                  ))}
                </div>
                <p className="text-xs text-default-500">
                  此模型名称如何匹配请求中的模型
                </p>
              </div>

              {/* 🌸 端点配置区 - 可视化编辑～ */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-default-700">端点配置</h3>
                  <Select
                    size="sm"
                    placeholder="加载模板..."
                    className="w-48"
                    selectedKeys={[]}
                    onSelectionChange={(keys) => {
                      const templateName = Array.from(keys)[0] as string;
                      const tpl = templateName ? ENDPOINT_TEMPLATES[templateName] : undefined;
                      if (!tpl) return;
                      setEndpointRows((prev) => {
                        // 已存在同名端点就不重复添加哦～
                        if (prev.some((r) => r.key === templateName)) return prev;
                        return [...prev, { key: templateName, path: tpl.path, method: tpl.method }];
                      });
                    }}
                  >
                    {Object.keys(ENDPOINT_TEMPLATES).map((name) => (
                      <SelectItem key={name}>{name}</SelectItem>
                    ))}
                  </Select>
                </div>

                {endpointRows.length === 0 && (
                  <div className="text-center py-6 rounded-lg border border-dashed text-xs text-default-400">
                    暂无端点配置，可以从右上角选择模板快速添加，或点击下方按钮手动添加～
                  </div>
                )}

                {endpointRows.map((row, idx) => (
                  <div key={idx} className="flex items-end gap-2">
                    <Input
                      size="sm"
                      label="端点类型"
                      placeholder="openai"
                      className="w-40"
                      value={row.key}
                      onValueChange={(v) => {
                        setEndpointRows((prev) => prev.map((r, i) => (i === idx ? { ...r, key: v } : r)));
                      }}
                    />
                    <Input
                      size="sm"
                      label="路径"
                      placeholder="/v1/chat/completions"
                      className="flex-1 font-mono"
                      value={row.path}
                      onValueChange={(v) => {
                        setEndpointRows((prev) => prev.map((r, i) => (i === idx ? { ...r, path: v } : r)));
                      }}
                    />
                    <Select
                      size="sm"
                      label="方法"
                      className="w-28"
                      selectedKeys={[row.method || 'POST']}
                      onSelectionChange={(keys) => {
                        const m = (Array.from(keys)[0] as string) || 'POST';
                        setEndpointRows((prev) => prev.map((r, i) => (i === idx ? { ...r, method: m } : r)));
                      }}
                    >
                      <SelectItem key="POST">POST</SelectItem>
                      <SelectItem key="GET">GET</SelectItem>
                    </Select>
                    <Button
                      isIconOnly
                      size="sm"
                      variant="flat"
                      color="danger"
                      onPress={() => setEndpointRows((prev) => prev.filter((_, i) => i !== idx))}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                ))}

                <Button
                  size="sm"
                  variant="flat"
                  startContent={<Plus className="w-4 h-4" />}
                  onPress={() => setEndpointRows((prev) => [...prev, { key: '', path: '', method: 'POST' }])}
                >
                  添加端点
                </Button>
              </div>

              {/* 🌸 定价配置区 */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-default-700">定价配置</h3>

                <div className="grid grid-cols-2 gap-4">
                  <Input
                    label="输入价格（$/1M tokens）"
                    type="number"
                    placeholder="2.50"
                    step="0.000001"
                    value={String(form.input_price ?? '')}
                    onValueChange={(v) => setForm({ ...form, input_price: Number(v) || 0 })}
                  />
                  <Input
                    label="输出价格（$/1M tokens）"
                    type="number"
                    placeholder="10.00"
                    step="0.000001"
                    value={String(form.output_price ?? '')}
                    onValueChange={(v) => setForm({ ...form, output_price: Number(v) || 0 })}
                  />
                </div>

                <Input
                  label="上下文长度"
                  type="number"
                  placeholder="128000"
                  description="最大上下文窗口大小（tokens）"
                  value={String(form.context_length ?? '')}
                  onValueChange={(v) => setForm({ ...form, context_length: Number(v) || 0 })}
                />
              </div>

              {/* 🌸 状态与同步区 */}
              <div className="space-y-4">
                <h3 className="text-sm font-semibold text-default-700">状态与同步</h3>

                <div className="flex items-center justify-between p-3 rounded-lg border">
                  <div>
                    <div className="font-medium text-sm">启用状态</div>
                    <div className="text-xs text-default-500 mt-0.5">启用或禁用此模型</div>
                  </div>
                  <Switch
                    isSelected={!!form.enabled}
                    onValueChange={(v) => setForm({ ...form, enabled: v })}
                  />
                </div>

                <div className="flex items-center justify-between p-3 rounded-lg border">
                  <div>
                    <div className="font-medium text-sm">官方同步</div>
                    <div className="text-xs text-default-500 mt-0.5">与官方上游自动同步</div>
                  </div>
                  <Switch
                    isSelected={!!form.sync_official}
                    onValueChange={(v) => setForm({ ...form, sync_official: v })}
                  />
                </div>
              </div>
            </div>

            {/* 底部操作栏 */}
            <div className="flex items-center justify-end gap-2 p-6 border-t" style={{ background: 'var(--bg-surface)' }}>
              <Button variant="light" onPress={() => setIsDrawerOpen(false)}>
                取消
              </Button>
              <Button color="primary" onPress={submit}>
                {editing ? '更新模型' : '保存'}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 🎀 同步向导对话框 - 对齐 new-api 设计～ */}
      {isSyncWizardOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          {/* 背景遮罩 */}
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => !syncLoading && setIsSyncWizardOpen(false)}
          />

          {/* 对话框内容 */}
          <div className="relative w-full max-w-xl mx-4 rounded-2xl shadow-xl flex flex-col" style={{ background: 'var(--bg-base)' }}>
            {/* 头部 */}
            <div className="flex items-start justify-between p-6 pb-4">
              <div>
                <h2 className="text-lg font-semibold">同步上游模型</h2>
                <p className="text-sm text-default-500 mt-1">从上游源同步模型和供应商</p>
              </div>
              <Button
                isIconOnly
                variant="light"
                size="sm"
                isDisabled={syncLoading}
                onPress={() => setIsSyncWizardOpen(false)}
              >
                <X className="w-4 h-4" />
              </Button>
            </div>

            <div className="px-6 space-y-5">
              {/* 🌸 选择同步源 */}
              <div className="space-y-2">
                <h3 className="text-sm font-semibold">选择同步源</h3>
                <p className="text-xs text-default-500">选择从何处获取上游元数据。</p>
                <div className="grid grid-cols-2 gap-3">
                  <div className="p-4 rounded-xl border-2 border-primary bg-primary-50 dark:bg-primary-900/20 cursor-pointer">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-primary" />
                      <span className="font-medium text-sm">官方仓库</span>
                      <span className="text-xs text-default-400">Default</span>
                    </div>
                    <p className="text-xs text-default-500 mt-1 ml-4">从公共上游元数据仓库同步。</p>
                  </div>
                  <div className="p-4 rounded-xl border-2 border-default-200 opacity-50 cursor-not-allowed">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-default-300" />
                      <span className="font-medium text-sm text-default-400">配置文件</span>
                    </div>
                    <p className="text-xs text-default-400 mt-1 ml-4">上传或引用本地配置文件。</p>
                  </div>
                </div>
              </div>

              {/* 🌸 选择语言 */}
              <div className="space-y-2">
                <h3 className="text-sm font-semibold">选择语言</h3>
                <div className="grid grid-cols-3 gap-3">
                  {([
                    { key: 'zh', label: '中文' },
                    { key: 'en', label: '英文' },
                    { key: 'ja', label: '日语' },
                  ] as const).map((lang) => (
                    <Button
                      key={lang.key}
                      variant={syncLocale === lang.key ? 'flat' : 'bordered'}
                      color={syncLocale === lang.key ? 'primary' : 'default'}
                      onPress={() => setSyncLocale(lang.key)}
                      startContent={
                        <div className={`w-2 h-2 rounded-full ${syncLocale === lang.key ? 'bg-primary' : 'bg-default-300'}`} />
                      }
                      className="justify-start"
                    >
                      {lang.label}
                    </Button>
                  ))}
                </div>
              </div>

              {/* 🌸 说明 */}
              <div className="p-3 rounded-lg text-xs text-default-500" style={{ background: 'var(--bg-surface)' }}>
                同步将从渠道获取缺失的模型，并从选定的源补齐模型元信息（描述、标签、端点、供应商、价格）。已有内容不会被覆盖。
              </div>
            </div>

            {/* 底部操作栏 */}
            <div className="flex items-center justify-end gap-2 p-6 pt-4">
              <Button variant="light" isDisabled={syncLoading} onPress={() => setIsSyncWizardOpen(false)}>
                取消
              </Button>
              <Button color="primary" isLoading={syncLoading} onPress={doSync}>
                立即同步
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
