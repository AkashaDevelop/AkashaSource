import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Button, Input, Select, SelectItem, Switch, Chip, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure, Divider } from '../../components/ui';
import { Plus, RefreshCw, Edit, Trash2, RotateCcw, Shield, ScrollText } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface Policy {
  id: number;
  name: string;
  enabled: boolean;
  scope: string;
  channel_id?: number | null;
  model_name: string;
  mode: string;
  config: string;
  version: number;
  created_at: number;
  updated_at: number;
}

interface EventRow {
  id: number;
  created_at: number;
  direction: string;
  stage: string;
  action: string;
  mode: string;
  risk_score: number;
  channel_name: string;
  requested_model: string;
  mapped_model: string;
  categories: string;
  snippet: string;
  findings: string;
  error: string;
}

interface Channel {
  id: number;
  name: string;
}

const defaultConfig = {
  request: { inject_guard: true, guard_position: 'first_system', guard_language: 'auto', preserve_user_system_messages: true },
  tools: { enabled: true, validate_tool_schema: true, validate_tool_choice: true, validate_assistant_tool_calls: true, tool_name_regex: '^[a-zA-Z0-9_-]{1,64}$', max_tools: 128, max_tool_schema_bytes: 65536 },
  response: { detect_ads: true, ad_policy: 'monitor', ad_confidence_threshold: 75, known_ad_patterns: [], preserve_code_blocks: true },
  logging: { log_events: true, log_raw_content: false, log_snippet_chars: 160, hash_content: true },
  circuit_breaker: { enabled: true, failure_threshold: 5, timeout_per_req_ms: 500, cooldown_seconds: 30 },
};

export default function ContextSanitization() {
  const { token } = useAuthStore();
  const [activeTab, setActiveTab] = useState<'policies' | 'events' | 'stats'>('policies');
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [events, setEvents] = useState<EventRow[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<Policy | null>(null);
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [form, setForm] = useState({ name: '', enabled: true, scope: 'global', channel_id: '', model_name: '', mode: 'monitor' });
  const [configForm, setConfigForm] = useState(defaultConfig);
  const [eventFilter, setEventFilter] = useState({ direction: '', action: '', mode: '', model: '' });

  const authHeaders = useMemo(() => ({ Authorization: `Bearer ${token}` }), [token]);

  const fetchPolicies = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/admin/sanitization/policies', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setPolicies(data.data || []);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  const fetchEvents = async () => {
    const params = new URLSearchParams({ size: '50' });
    Object.entries(eventFilter).forEach(([k, v]) => { if (v) params.set(k, v); });
    try {
      const res = await fetch(`/api/admin/sanitization/events?${params}`, { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setEvents(data.data?.items || []);
    } catch (e) { console.error(e); }
  };

  const fetchStats = async () => {
    try {
      const res = await fetch('/api/admin/sanitization/stats', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setStats(data.data);
    } catch (e) { console.error(e); }
  };

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/channel', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0 || data.data) setChannels(data.data || []);
    } catch (e) { console.error(e); }
  };

  useEffect(() => { if (token) { fetchPolicies(); fetchEvents(); fetchStats(); fetchChannels(); } }, [token]);
  useEffect(() => { if (token && activeTab === 'events') fetchEvents(); }, [eventFilter, activeTab]);

  const resetForm = () => {
    setEditing(null);
    setForm({ name: '', enabled: true, scope: 'global', channel_id: '', model_name: '', mode: 'monitor' });
    setConfigForm(defaultConfig);
  };

  const openCreate = () => { resetForm(); onOpen(); };
  const openEdit = (p: Policy) => {
    setEditing(p);
    setForm({ name: p.name, enabled: p.enabled, scope: p.scope, channel_id: p.channel_id ? String(p.channel_id) : '', model_name: p.model_name || '', mode: p.mode });
    try { setConfigForm(p.config ? JSON.parse(p.config) : defaultConfig); } catch { setConfigForm(defaultConfig); }
    onOpen();
  };

  const savePolicy = async (onClose: () => void) => {
    const body: any = { ...form, id: editing?.id, channel_id: form.channel_id ? parseInt(form.channel_id) : null, config: JSON.stringify(configForm) };
    const res = await fetch('/api/admin/sanitization/policies', { method: editing ? 'PUT' : 'POST', headers: { ...authHeaders, 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    const data = await res.json();
    if (data.code === 0) { toast.success('保存成功'); fetchPolicies(); fetchStats(); onClose(); }
    else toast.error(data.msg || '保存失败');
  };

  const deletePolicy = async (id: number) => {
    if (!await confirm({ title: '删除策略', message: '确定删除此上下文净化策略？', danger: true })) return;
    const res = await fetch(`/api/admin/sanitization/policies/${id}`, { method: 'DELETE', headers: authHeaders });
    const data = await res.json();
    if (data.code === 0) { toast.success('删除成功'); fetchPolicies(); } else toast.error(data.msg || '删除失败');
  };

  const reloadCache = async () => {
    const res = await fetch('/api/admin/sanitization/reload', { method: 'POST', headers: authHeaders });
    const data = await res.json();
    if (data.code === 0) { toast.success('缓存已刷新'); fetchStats(); } else toast.error(data.msg || '刷新失败');
  };

  const rollbackLatest = async (p: Policy) => {
    if (!await confirm({ title: '回滚策略', message: '将尝试回滚到最近一个历史版本，确定继续？' })) return;
    const revRes = await fetch(`/api/admin/sanitization/policies/${p.id}/revisions`, { headers: authHeaders });
    const revData = await revRes.json();
    const revision = revData.data?.[0];
    if (!revision) { toast.error('没有可回滚的版本'); return; }
    const res = await fetch(`/api/admin/sanitization/policies/${p.id}/rollback`, { method: 'POST', headers: { ...authHeaders, 'Content-Type': 'application/json' }, body: JSON.stringify({ revision_id: revision.id }) });
    const data = await res.json();
    if (data.code === 0) { toast.success('已回滚'); fetchPolicies(); } else toast.error(data.msg || '回滚失败');
  };

  const scopeLabel = (s: string) => ({ global: '全局', model: '模型', channel: '渠道', channel_model: '渠道模型' }[s] || s);
  const modeLabel = (m: string) => ({ off: '关闭', monitor: '监控', protect: '防护', balanced: '均衡', strict: '严格' }[m] || m);
  const directionLabel = (d: string) => ({ request: '请求', response: '响应' }[d] || d);
  const stageLabel = (s: string) => ({ request_detect: '请求检测', tool_validate: '工具校验', guard_inject: 'Guard注入', response_ad_detect: '响应广告检测', degraded: '已降级', circuit_open: '熔断开路' }[s] || s);
  const actionLabel = (a: string) => ({ monitor: '监控', allow: '放行', inject_guard: '注入Guard', block: '阻断', strip_known_suffix: '清理广告', skip: '跳过', degrade: '降级' }[a] || a);

  return (
    <div className="space-y-6">
      <PageHeader
        title="上下文净化"
        description="按全局、渠道或渠道指定模型配置安全 guard、工具结构校验与上游广告处理"
        actions={<div className="flex gap-2"><Button startContent={<RefreshCw size={16} />} onPress={reloadCache} variant="flat">刷新缓存</Button><Button startContent={<Plus size={16} />} color="primary" onPress={openCreate}>新增策略</Button></div>}
      />

      <div className="flex gap-2 flex-wrap">
        <Button variant={activeTab === 'policies' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('policies')} startContent={<Shield size={16} />}>策略配置</Button>
        <Button variant={activeTab === 'events' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('events')} startContent={<ScrollText size={16} />}>事件日志</Button>
        <Button variant={activeTab === 'stats' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('stats')}>缓存状态</Button>
      </div>

      {activeTab === 'policies' && (
        <div className="data-table-wrap"><table className="data-table"><thead><tr><th>名称</th><th>范围</th><th>渠道</th><th>模型</th><th>模式</th><th>状态</th><th>版本</th><th>操作</th></tr></thead><tbody>
          {loading ? <LoadingRows cols={8} rows={5} /> : policies.length === 0 ? <tr><td colSpan={8}><EmptyState icon="🛡️" title="暂无策略" /></td></tr> : policies.map(p => <tr key={p.id}>
            <td>{p.name}</td><td><Chip size="sm" variant="flat">{scopeLabel(p.scope)}</Chip></td><td>{channels.find(c => c.id === p.channel_id)?.name || p.channel_id || '-'}</td><td>{p.model_name || '-'}</td><td><Chip size="sm" color={p.mode === 'protect' ? 'warning' : 'default'} variant="flat">{modeLabel(p.mode)}</Chip></td><td><Chip size="sm" color={p.enabled ? 'success' : 'default'} variant="flat">{p.enabled ? '启用' : '禁用'}</Chip></td><td>v{p.version}</td><td><div className="flex gap-2"><span className="cursor-pointer text-default-400" onClick={() => openEdit(p)}><Edit size={18} /></span><span className="cursor-pointer text-warning" onClick={() => rollbackLatest(p)}><RotateCcw size={18} /></span><span className="cursor-pointer text-danger" onClick={() => deletePolicy(p.id)}><Trash2 size={18} /></span></div></td>
          </tr>)}
        </tbody></table></div>
      )}

      {activeTab === 'events' && <div className="space-y-4">
        <div className="flex gap-2 flex-wrap"><Select placeholder="方向" className="w-36" selectedKeys={eventFilter.direction ? [eventFilter.direction] : []} onSelectionChange={keys => setEventFilter({ ...eventFilter, direction: [...keys][0] as string || '' })}><SelectItem key="request">请求</SelectItem><SelectItem key="response">响应</SelectItem></Select><Input className="w-48" placeholder="模型" value={eventFilter.model} onValueChange={v => setEventFilter({ ...eventFilter, model: v })} /></div>
        <div className="data-table-wrap"><table className="data-table"><thead><tr><th>时间</th><th>方向</th><th>阶段</th><th>动作</th><th>风险</th><th>渠道</th><th>模型</th><th>类别</th><th>片段</th></tr></thead><tbody>{events.length === 0 ? <tr><td colSpan={9}><EmptyState icon="📜" title="暂无事件" /></td></tr> : events.map(e => <tr key={e.id}><td>{new Date(e.created_at * 1000).toLocaleString()}</td><td>{directionLabel(e.direction)}</td><td>{stageLabel(e.stage)}</td><td>{actionLabel(e.action)}</td><td>{e.risk_score}</td><td>{e.channel_name || '-'}</td><td>{e.mapped_model || e.requested_model}</td><td className="text-xs">{e.categories}</td><td className="max-w-xs truncate">{e.snippet || e.error || '-'}</td></tr>)}</tbody></table></div>
      </div>}

      {activeTab === 'stats' && <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">策略总数</div><div className="text-2xl font-bold">{stats?.policy_count || 0}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">事件总数</div><div className="text-2xl font-bold">{stats?.event_count || 0}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">缓存命中</div><div className="text-2xl font-bold">{stats?.cache?.loaded ? '已加载' : '未加载'}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">熔断开路</div><div className="text-2xl font-bold">{stats?.circuit?.states?.open || 0}</div></div>
      </div>}

      <Modal isOpen={isOpen} onOpenChange={onOpenChange} size="4xl" scrollBehavior="inside">
        <ModalContent>{onClose => <><ModalHeader>{editing ? '编辑策略' : '新增策略'}</ModalHeader><ModalBody className="gap-4">
          <Input label="名称" value={form.name} onValueChange={v => setForm({ ...form, name: v })} isRequired />
          <div className="grid grid-cols-2 gap-4">
            <Select label="范围" selectedKeys={[form.scope]} onSelectionChange={keys => setForm({ ...form, scope: [...keys][0] as string || 'global' })}><SelectItem key="global">全局</SelectItem><SelectItem key="model">模型</SelectItem><SelectItem key="channel">渠道</SelectItem><SelectItem key="channel_model">渠道模型</SelectItem></Select>
            <Select label="模式" selectedKeys={[form.mode]} onSelectionChange={keys => setForm({ ...form, mode: [...keys][0] as string || 'monitor' })}><SelectItem key="off">off</SelectItem><SelectItem key="monitor">monitor</SelectItem><SelectItem key="protect">protect</SelectItem><SelectItem key="balanced">balanced</SelectItem><SelectItem key="strict">strict</SelectItem></Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Select label="渠道" placeholder="选择渠道" selectedKeys={form.channel_id ? [form.channel_id] : []} onSelectionChange={keys => setForm({ ...form, channel_id: [...keys][0] as string || '' })}>{channels.map(c => <SelectItem key={String(c.id)}>{c.name}</SelectItem>)}</Select>
            <Input label="模型名" value={form.model_name} onValueChange={v => setForm({ ...form, model_name: v })} />
          </div>
          <Switch isSelected={form.enabled} onValueChange={v => setForm({ ...form, enabled: v })}>启用策略</Switch>
          <Divider className="my-2" />
          <div className="text-sm font-semibold">请求侧配置</div>
          <div className="grid grid-cols-2 gap-4">
            <Switch isSelected={configForm.request.inject_guard} onValueChange={v => setConfigForm({ ...configForm, request: { ...configForm.request, inject_guard: v } })}>注入 Guard</Switch>
            <Select label="Guard 语言" selectedKeys={[configForm.request.guard_language]} onSelectionChange={keys => setConfigForm({ ...configForm, request: { ...configForm.request, guard_language: [...keys][0] as string || 'auto' } })}><SelectItem key="auto">自动</SelectItem><SelectItem key="en">英文</SelectItem><SelectItem key="zh">中文</SelectItem></Select>
          </div>
          <Divider className="my-2" />
          <div className="text-sm font-semibold">工具校验配置</div>
          <div className="grid grid-cols-2 gap-4">
            <Switch isSelected={configForm.tools.enabled} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, enabled: v } })}>启用工具校验</Switch>
            <Input type="number" label="最大工具数" value={String(configForm.tools.max_tools)} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, max_tools: parseInt(v) || 128 } })} />
          </div>
          <Divider className="my-2" />
          <div className="text-sm font-semibold">响应广告检测</div>
          <div className="grid grid-cols-2 gap-4">
            <Switch isSelected={configForm.response.detect_ads} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, detect_ads: v } })}>检测广告</Switch>
            <Select label="广告策略" selectedKeys={[configForm.response.ad_policy]} onSelectionChange={keys => setConfigForm({ ...configForm, response: { ...configForm.response, ad_policy: [...keys][0] as string || 'monitor' } })}><SelectItem key="off">关闭</SelectItem><SelectItem key="monitor">监控</SelectItem><SelectItem key="mark">标记</SelectItem><SelectItem key="strip_known_suffix">清理已知尾部</SelectItem></Select>
          </div>
          <Divider className="my-2" />
          <div className="text-sm font-semibold">熔断与降级</div>
          <div className="grid grid-cols-3 gap-4">
            <Switch isSelected={configForm.circuit_breaker.enabled} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, enabled: v } })}>启用熔断</Switch>
            <Input type="number" label="失败阈值" value={String(configForm.circuit_breaker.failure_threshold)} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, failure_threshold: parseInt(v) || 5 } })} />
            <Input type="number" label="超时(ms)" value={String(configForm.circuit_breaker.timeout_per_req_ms)} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, timeout_per_req_ms: parseInt(v) || 500 } })} />
          </div>
        </ModalBody><ModalFooter><Button variant="light" onPress={onClose}>取消</Button><Button color="primary" onPress={() => savePolicy(onClose)}>保存</Button></ModalFooter></>}</ModalContent>
      </Modal>
    </div>
  );
}
