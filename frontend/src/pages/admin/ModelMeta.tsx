import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import {
  Button,
  Chip,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Select,
  SelectItem,
  Switch,
  useDisclosure,
} from '../../components/ui';
import { Pagination } from '../../components/ui/Pagination';
import { useAuthStore } from '../../store/auth';
import { confirm } from '../../store/confirm';
import { toast } from '../../store/toast';

interface ModelMeta {
  id: number;
  vendor_id: number;
  model_name: string;
  display_name: string;
  model_type: string;
  context_length: number;
  input_price: number;
  output_price: number;
  enabled: boolean;
}

interface VendorOption {
  id: number;
  name: string;
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

const MODEL_TYPES = [
  { key: 'text', label: '文本' },
  { key: 'chat', label: '对话' },
  { key: 'embedding', label: '向量' },
  { key: 'image', label: '图像' },
  { key: 'audio', label: '音频' },
  { key: 'rerank', label: '重排' },
  { key: 'other', label: '其他' },
];

export default function ModelMetaManagement() {
  const { token } = useAuthStore();
  const { isOpen, onOpen, onOpenChange } = useDisclosure();

  const [items, setItems] = useState<ModelMeta[]>([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  const [editing, setEditing] = useState<ModelMeta | null>(null);
  const [vendors, setVendors] = useState<VendorOption[]>([]);
  const [form, setForm] = useState<Partial<ModelMeta>>({
    vendor_id: 0,
    model_name: '',
    display_name: '',
    model_type: 'text',
    context_length: 0,
    input_price: 0,
    output_price: 0,
    enabled: true,
  });

  const [channelIdsInput, setChannelIdsInput] = useState('');
  const [missingModels, setMissingModels] = useState<string[]>([]);
  const [missingChannelCount, setMissingChannelCount] = useState(0);
  const [previewResult, setPreviewResult] = useState<UpstreamResult | null>(null);
  const [syncResult, setSyncResult] = useState<UpstreamResult | null>(null);

  const [missingLoading, setMissingLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);

  const parsedChannelIds = useMemo(() => {
    return channelIdsInput
      .split(',')
      .map((v) => Number(v.trim()))
      .filter((v) => Number.isInteger(v) && v > 0);
  }, [channelIdsInput]);

  const channelQuery = useMemo(() => {
    return parsedChannelIds.length > 0 ? `?channel_ids=${parsedChannelIds.join(',')}` : '';
  }, [parsedChannelIds]);

  const fetchData = async (targetPage = 1) => {
    const pageSize = 20;
    const q = keyword.trim();
    const base = q ? `/api/models/search?keyword=${encodeURIComponent(q)}` : '/api/models';
    const url = `${base}${base.includes('?') ? '&' : '?'}page=${targetPage}&page_size=${pageSize}`;
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

  const vendorNameMap = useMemo(() => {
    const map: Record<number, string> = {};
    vendors.forEach((v) => {
      map[v.id] = v.name;
    });
    return map;
  }, [vendors]);

  useEffect(() => {
    fetchData(1);
    fetchVendors();
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({
      vendor_id: 0,
      model_name: '',
      display_name: '',
      model_type: 'text',
      context_length: 0,
      input_price: 0,
      output_price: 0,
      enabled: true,
    });
    onOpen();
  };

  const openEdit = (v: ModelMeta) => {
    setEditing(v);
    setForm(v);
    onOpen();
  };

  const submit = async (onClose: () => void) => {
    const method = editing ? 'PUT' : 'POST';
    const body = editing ? { ...form, id: editing.id } : form;
    const res = await fetch('/api/models', {
      method,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    if (data.code === 0) {
      toast.success(editing ? '更新成功' : '创建成功');
      await fetchData(page);
      onClose();
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
      const res = await fetch('/api/models/sync_upstream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ channel_ids: parsedChannelIds, enabled: true }),
      });
      const data = await res.json();
      if (data.code === 0) {
        const payload = data.data || {};
        setSyncResult(payload);
        toast.success(`新增 ${Number(payload.created ?? 0)} 个模型元数据`);
        fetchData(1);
      } else {
        toast.error(data.msg || '同步失败');
      }
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
            <Input
              size="sm"
              placeholder="搜索模型名"
              value={keyword}
              onValueChange={setKeyword}
              onKeyDown={(e) => e.key === 'Enter' && fetchData(1)}
            />
            <Button variant="flat" onPress={() => fetchData(1)}>查询</Button>
            <Button color="primary" onPress={openCreate}>新增</Button>
          </div>
        )}
      />

      <div className="rounded-xl border p-4" style={{ borderColor: 'var(--border-color)', background: 'var(--bg-surface)' }}>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
          <Input
            label="渠道过滤（可选）"
            size="sm"
            placeholder="1,2,3"
            value={channelIdsInput}
            onValueChange={setChannelIdsInput}
          />
          <Button variant="flat" isLoading={missingLoading} onPress={fetchMissingModels}>扫描缺失模型</Button>
          <Button variant="flat" color="warning" isLoading={previewLoading} onPress={previewSync}>上游预览</Button>
          <Button color="primary" isLoading={syncLoading} onPress={doSync}>一键同步到元数据</Button>
        </div>

        <div className="mt-3 flex flex-wrap gap-2 text-xs" style={{ color: 'var(--text-secondary)' }}>
          <Chip size="sm" variant="flat">缺失模型: {missingModels.length}</Chip>
          <Chip size="sm" variant="flat">涉及渠道: {missingChannelCount}</Chip>
          <Chip size="sm" variant="flat">筛选渠道ID: {parsedChannelIds.length > 0 ? parsedChannelIds.join(',') : '全部启用渠道'}</Chip>
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

      <div className="data-table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Model</th>
              <th>Type</th>
              <th>Vendor</th>
              <th>Input</th>
              <th>Output</th>
              <th>Enabled</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((v) => (
              <tr key={v.id}>
                <td>{v.id}</td>
                <td className="font-mono text-xs">{v.model_name}</td>
                <td>{v.model_type}</td>
                <td>
                  <Chip size="sm" variant="flat" color="secondary">
                    {vendorNameMap[v.vendor_id] || `#${v.vendor_id}`}
                  </Chip>
                </td>
                <td>{v.input_price}</td>
                <td>{v.output_price}</td>
                <td>{v.enabled ? 'Y' : 'N'}</td>
                <td>
                  <div className="flex gap-2">
                    <Button size="sm" variant="flat" onPress={() => openEdit(v)}>编辑</Button>
                    <Button size="sm" color="danger" variant="flat" onPress={() => remove(v.id)}>删除</Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end">
        <Pagination total={totalPages} page={page} onChange={(p) => fetchData(p)} />
      </div>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editing ? '编辑模型元数据' : '新增模型元数据'}</ModalHeader>
              <ModalBody className="gap-3">
                <Select
                  label="供应商"
                  selectedKeys={[String(form.vendor_id ?? 0)]}
                  onSelectionChange={(keys) => setForm({ ...form, vendor_id: Number([...keys][0] as string) || 0 })}
                >
                  <SelectItem key="0">未指定</SelectItem>
                  {vendors.map((v) => (
                    <SelectItem key={String(v.id)}>{v.name}</SelectItem>
                  ))}
                </Select>
                <Input
                  label="模型名"
                  value={form.model_name || ''}
                  onValueChange={(v) => setForm({ ...form, model_name: v })}
                />
                <Input
                  label="显示名"
                  value={form.display_name || ''}
                  onValueChange={(v) => setForm({ ...form, display_name: v })}
                />
                <Select
                  label="类型"
                  selectedKeys={[form.model_type || 'text']}
                  onSelectionChange={(keys) => setForm({ ...form, model_type: [...keys][0] as string || 'text' })}
                >
                  {MODEL_TYPES.map((t) => <SelectItem key={t.key}>{t.label}</SelectItem>)}
                </Select>
                <Input
                  label="上下文长度"
                  value={String(form.context_length ?? 0)}
                  onValueChange={(v) => setForm({ ...form, context_length: Number(v) || 0 })}
                />
                <Input
                  label="输入单价"
                  value={String(form.input_price ?? 0)}
                  onValueChange={(v) => setForm({ ...form, input_price: Number(v) || 0 })}
                />
                <Input
                  label="输出单价"
                  value={String(form.output_price ?? 0)}
                  onValueChange={(v) => setForm({ ...form, output_price: Number(v) || 0 })}
                />
                <Switch isSelected={!!form.enabled} onValueChange={(v) => setForm({ ...form, enabled: v })}>启用</Switch>
              </ModalBody>
              <ModalFooter>
                <Button variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => submit(onClose)}>保存</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
