import { useEffect, useMemo, useState } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Button, Input, Select, SelectItem, Switch, Chip, Textarea } from '../../components/ui';
import { Plus, RefreshCw, Edit, Trash2, RotateCcw, Shield, ScrollText, ListFilter } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';
import QingyuanRules from './QingyuanRules';

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
  detection: { remove_zero_width_for_detection: true, decode_json_escapes_for_detection: true, decode_url_encoding_for_detection: true, decode_html_entities_for_detection: true, detect_base64: true, max_decode_layers: 2, max_scan_chars: 65536 },
  risk: { annotate_threshold: 40, block_threshold: 85, block_structural_tool_abuse: true },
  tools: { enabled: true, validate_tool_schema: true, validate_tool_choice: true, validate_assistant_tool_calls: true, allowed_tool_names: [] as string[], blocked_tool_names: [] as string[], tool_name_regex: '^[a-zA-Z0-9_-]{1,64}$', max_tools: 128, max_tool_schema_bytes: 65536, max_tool_arguments_bytes: 262144 },
  response: { detect_ads: true, ad_policy: 'monitor', ad_confidence_threshold: 75, known_ad_patterns: [] as string[], preserve_code_blocks: true, preserve_json_output: true, validate_output_tool_calls: true, block_invalid_output_tool_calls: false, stream_tail_buffer_size: 2048, detect_prompt_injection: true, detect_multimodal: true, detect_thinking_attacks: true },
  trust: { enable_provenance: true, retrieved_doc_risk_multiplier: 1.6, tool_output_risk_multiplier: 1.4, scan_tool_descriptions: true, scan_memory_poisoning: true, memory_scan_max_messages: 50 },
  logging: { log_events: true, log_raw_content: false, log_snippet_chars: 160, hash_content: true },
  circuit_breaker: { enabled: true, failure_threshold: 5, timeout_per_req_ms: 500, cooldown_seconds: 30 },
  degradation: { monitor_timeout_action: 'skip', protect_timeout_action: 'fallback_to_guard' },
  ai_review: { mode: 'off', channel_id: 0, model: '', timeout_sec: 10, block_score: 70, max_text_chars: 2000, pre_prompt: '', re_prompt: '' },
};

const sectionStyle = { background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' };

export default function ContextSanitization() {
  const { token } = useAuthStore();
  const [activeTab, setActiveTab] = useState<'policies' | 'rules' | 'events' | 'stats' | 'edit'>('policies');
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [events, setEvents] = useState<EventRow[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<Policy | null>(null);
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

  const openCreate = () => { resetForm(); setActiveTab('edit'); };
  const openEdit = (p: Policy) => {
    setEditing(p);
    setForm({ name: p.name, enabled: p.enabled, scope: p.scope, channel_id: p.channel_id ? String(p.channel_id) : '', model_name: p.model_name || '', mode: p.mode });
    try {
      const parsed = p.config ? JSON.parse(p.config) : {};
      setConfigForm({
        request: { ...defaultConfig.request, ...(parsed.request || {}) },
        detection: { ...defaultConfig.detection, ...(parsed.detection || {}) },
        risk: { ...defaultConfig.risk, ...(parsed.risk || {}) },
        tools: { ...defaultConfig.tools, ...(parsed.tools || {}) },
        response: { ...defaultConfig.response, ...(parsed.response || {}) },
        trust: { ...defaultConfig.trust, ...(parsed.trust || {}) },
        logging: { ...defaultConfig.logging, ...(parsed.logging || {}) },
        circuit_breaker: { ...defaultConfig.circuit_breaker, ...(parsed.circuit_breaker || {}) },
        degradation: { ...defaultConfig.degradation, ...(parsed.degradation || {}) },
        ai_review: { ...defaultConfig.ai_review, ...(parsed.ai_review || {}) },
      });
    } catch { setConfigForm(defaultConfig); }
    setActiveTab('edit');
  };

  const savePolicy = async () => {
    const body: any = { ...form, id: editing?.id, channel_id: form.channel_id ? parseInt(form.channel_id) : null, config: JSON.stringify(configForm) };
    const res = await fetch('/api/admin/sanitization/policies', { method: editing ? 'PUT' : 'POST', headers: { ...authHeaders, 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    const data = await res.json();
    if (data.code === 0) { toast.success('保存成功'); fetchPolicies(); fetchStats(); setActiveTab('policies'); }
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
  const stageLabel = (s: string) => ({ request_detect: '请求检测', tool_validate: '工具校验', guard_inject: 'Guard注入', response_ad_detect: '响应广告检测', degraded: '已降级', circuit_open: '熔断开路', stream_finalize: '流式收尾' }[s] || s);
  const actionLabel = (a: string) => ({ monitor: '监控', allow: '放行', inject_guard: '注入Guard', block: '阻断', strip_known_suffix: '清理广告', skip: '跳过', degrade: '降级' }[a] || a);
  const categoryLabel = (t: string) => ({
    tool_poisoning_priority_hijack: '工具投毒·抢占调用', tool_poisoning_confirmation_bypass: '工具投毒·绕过确认', tool_poisoning_param_injection: '工具投毒·参数注入',
    memory_poisoning_self_instruction: '记忆投毒·自我指令', memory_poisoning_tool_rewrite: '记忆投毒·篡改指令',
    retrieved_doc_injection: '检索投毒', multimodal_blind_spot: '多模态盲区', advanced_obfuscation: '高级混淆',
    stream_undeclared_tool_injection: '流式未声明工具注入',
  } as Record<string, string>)[t] || t;
  const parseCategories = (raw: string): string[] => {
    try { const parsed = JSON.parse(raw || '[]'); return Array.isArray(parsed) ? parsed : []; } catch { return []; }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="宸汐清源"
        description="上下文净化结界 · 按全局、渠道或渠道指定模型配置安全 guard、工具/记忆投毒检测、信任边界分级与上游广告处理"
        actions={<div className="flex gap-2"><Button startContent={<RefreshCw size={16} />} onPress={reloadCache} variant="flat">刷新缓存</Button><Button startContent={<Plus size={16} />} color="primary" onPress={openCreate}>新增策略</Button></div>}
      />

      <div className="flex gap-2 flex-wrap">
        <Button variant={activeTab === 'policies' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('policies')} startContent={<Shield size={16} />}>策略配置</Button>
        <Button variant={activeTab === 'rules' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('rules')} startContent={<ListFilter size={16} />}>规则库管理</Button>
        <Button variant={activeTab === 'events' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('events')} startContent={<ScrollText size={16} />}>事件日志</Button>
        <Button variant={activeTab === 'stats' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('stats')}>缓存状态</Button>
      </div>

      {activeTab === 'rules' && <QingyuanRules />}

      {activeTab === 'policies' && (
        <div className="data-table-wrap"><table className="data-table"><thead><tr><th>名称</th><th>范围</th><th>渠道</th><th>模型</th><th>模式</th><th>状态</th><th>版本</th><th>操作</th></tr></thead><tbody>
          {loading ? <LoadingRows cols={8} rows={5} /> : policies.length === 0 ? <tr><td colSpan={8}><EmptyState icon="🛡️" title="暂无策略" /></td></tr> : policies.map(p => <tr key={p.id}>
            <td>{p.name}</td><td><Chip size="sm" variant="flat">{scopeLabel(p.scope)}</Chip></td><td>{channels.find(c => c.id === p.channel_id)?.name || p.channel_id || '-'}</td><td>{p.model_name || '-'}</td><td><Chip size="sm" color={p.mode === 'protect' ? 'warning' : 'default'} variant="flat">{modeLabel(p.mode)}</Chip></td><td><Chip size="sm" color={p.enabled ? 'success' : 'default'} variant="flat">{p.enabled ? '启用' : '禁用'}</Chip></td><td>v{p.version}</td><td><div className="flex gap-2"><span className="cursor-pointer text-default-400" onClick={() => openEdit(p)}><Edit size={18} /></span><span className="cursor-pointer text-warning" onClick={() => rollbackLatest(p)}><RotateCcw size={18} /></span><span className="cursor-pointer text-danger" onClick={() => deletePolicy(p.id)}><Trash2 size={18} /></span></div></td>
          </tr>)}
        </tbody></table></div>
      )}

      {activeTab === 'events' && <div className="space-y-4">
        <div className="flex gap-2 flex-wrap"><Select placeholder="方向" className="w-36" selectedKeys={eventFilter.direction ? [eventFilter.direction] : []} onSelectionChange={keys => setEventFilter({ ...eventFilter, direction: [...keys][0] as string || '' })}><SelectItem key="request">请求</SelectItem><SelectItem key="response">响应</SelectItem></Select><Input className="w-48" placeholder="模型" value={eventFilter.model} onValueChange={v => setEventFilter({ ...eventFilter, model: v })} /></div>
        <div className="data-table-wrap"><table className="data-table"><thead><tr><th>时间</th><th>方向</th><th>阶段</th><th>动作</th><th>风险</th><th>渠道</th><th>模型</th><th>类别</th><th>片段</th></tr></thead><tbody>{events.length === 0 ? <tr><td colSpan={9}><EmptyState icon="📜" title="暂无事件" /></td></tr> : events.map(e => <tr key={e.id}><td>{new Date(e.created_at * 1000).toLocaleString()}</td><td>{directionLabel(e.direction)}</td><td>{stageLabel(e.stage)}</td><td>{actionLabel(e.action)}</td><td>{e.risk_score}</td><td>{e.channel_name || '-'}</td><td>{e.mapped_model || e.requested_model}</td><td className="text-xs"><div className="flex gap-1 flex-wrap">{parseCategories(e.categories).map(c => <Chip key={c} size="sm" variant="flat">{categoryLabel(c)}</Chip>)}</div></td><td className="max-w-xs truncate">{e.snippet || e.error || '-'}</td></tr>)}</tbody></table></div>
      </div>}

      {activeTab === 'stats' && <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">策略总数</div><div className="text-2xl font-bold">{stats?.policy_count || 0}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">事件总数</div><div className="text-2xl font-bold">{stats?.event_count || 0}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">缓存命中</div><div className="text-2xl font-bold">{stats?.cache?.loaded ? '已加载' : '未加载'}</div></div>
        <div className="p-4 rounded-xl" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}><div className="text-sm text-gray-500">熔断开路</div><div className="text-2xl font-bold">{stats?.circuit?.states?.open || 0}</div></div>
      </div>}

      {activeTab === 'edit' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="text-lg font-semibold">{editing ? '编辑策略' : '新增策略'}</div>
            <div className="flex gap-2">
              <Button variant="light" onPress={() => setActiveTab('policies')}>取消</Button>
              <Button color="primary" onPress={savePolicy}>保存</Button>
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">基本信息</div>
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
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">请求侧配置</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.request.inject_guard} onValueChange={v => setConfigForm({ ...configForm, request: { ...configForm.request, inject_guard: v } })}>注入 Guard</Switch>
              <Switch isSelected={configForm.request.preserve_user_system_messages} onValueChange={v => setConfigForm({ ...configForm, request: { ...configForm.request, preserve_user_system_messages: v } })}>保留用户原有 system 消息</Switch>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Select label="Guard 语言" selectedKeys={[configForm.request.guard_language]} onSelectionChange={keys => setConfigForm({ ...configForm, request: { ...configForm.request, guard_language: [...keys][0] as string || 'auto' } })}><SelectItem key="auto">自动</SelectItem><SelectItem key="en">英文</SelectItem><SelectItem key="zh">中文</SelectItem></Select>
              <Input label="Guard 插入位置" value={configForm.request.guard_position} onValueChange={v => setConfigForm({ ...configForm, request: { ...configForm.request, guard_position: v || 'first_system' } })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">检测预处理（去混淆）</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.detection.remove_zero_width_for_detection} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, remove_zero_width_for_detection: v } })}>移除零宽字符</Switch>
              <Switch isSelected={configForm.detection.decode_json_escapes_for_detection} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, decode_json_escapes_for_detection: v } })}>解码 JSON 转义</Switch>
              <Switch isSelected={configForm.detection.decode_url_encoding_for_detection} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, decode_url_encoding_for_detection: v } })}>解码 URL 编码</Switch>
              <Switch isSelected={configForm.detection.decode_html_entities_for_detection} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, decode_html_entities_for_detection: v } })}>解码 HTML 实体</Switch>
              <Switch isSelected={configForm.detection.detect_base64} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, detect_base64: v } })}>检测 Base64 编码</Switch>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="最大解码层数" value={String(configForm.detection.max_decode_layers)} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, max_decode_layers: parseInt(v) || 2 } })} />
              <Input type="number" label="最大扫描字符数" value={String(configForm.detection.max_scan_chars)} onValueChange={v => setConfigForm({ ...configForm, detection: { ...configForm.detection, max_scan_chars: parseInt(v) || 65536 } })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">风险阈值</div>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="标注阈值" value={String(configForm.risk.annotate_threshold)} onValueChange={v => setConfigForm({ ...configForm, risk: { ...configForm.risk, annotate_threshold: parseInt(v) || 40 } })} />
              <Input type="number" label="阻断阈值" value={String(configForm.risk.block_threshold)} onValueChange={v => setConfigForm({ ...configForm, risk: { ...configForm.risk, block_threshold: parseInt(v) || 85 } })} />
            </div>
            <Switch isSelected={configForm.risk.block_structural_tool_abuse} onValueChange={v => setConfigForm({ ...configForm, risk: { ...configForm.risk, block_structural_tool_abuse: v } })}>阻断结构性工具滥用</Switch>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">工具校验配置</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.tools.enabled} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, enabled: v } })}>启用工具校验</Switch>
              <Input type="number" label="最大工具数" value={String(configForm.tools.max_tools)} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, max_tools: parseInt(v) || 128 } })} />
              <Switch isSelected={configForm.tools.validate_tool_schema} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, validate_tool_schema: v } })}>校验工具 Schema</Switch>
              <Switch isSelected={configForm.tools.validate_tool_choice} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, validate_tool_choice: v } })}>校验 tool_choice</Switch>
              <Switch isSelected={configForm.tools.validate_assistant_tool_calls} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, validate_assistant_tool_calls: v } })}>校验 assistant 工具调用</Switch>
              <Input label="工具名正则" value={configForm.tools.tool_name_regex} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, tool_name_regex: v } })} />
              <Input type="number" label="最大工具 Schema 字节数" value={String(configForm.tools.max_tool_schema_bytes)} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, max_tool_schema_bytes: parseInt(v) || 65536 } })} />
              <Input type="number" label="最大工具参数字节数" value={String(configForm.tools.max_tool_arguments_bytes)} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, max_tool_arguments_bytes: parseInt(v) || 262144 } })} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Input label="允许的工具名（逗号分隔）" value={configForm.tools.allowed_tool_names.join(',')} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, allowed_tool_names: v.split(',').map(s => s.trim()).filter(Boolean) } })} />
              <Input label="禁止的工具名（逗号分隔）" value={configForm.tools.blocked_tool_names.join(',')} onValueChange={v => setConfigForm({ ...configForm, tools: { ...configForm.tools, blocked_tool_names: v.split(',').map(s => s.trim()).filter(Boolean) } })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">响应侧检测</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.response.detect_ads} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, detect_ads: v } })}>检测广告</Switch>
              <Select label="广告策略" selectedKeys={[configForm.response.ad_policy]} onSelectionChange={keys => setConfigForm({ ...configForm, response: { ...configForm.response, ad_policy: [...keys][0] as string || 'monitor' } })}><SelectItem key="off">关闭</SelectItem><SelectItem key="monitor">监控</SelectItem><SelectItem key="mark">标记</SelectItem><SelectItem key="strip_known_suffix">清理已知尾部</SelectItem></Select>
              <Input type="number" label="广告置信度阈值" value={String(configForm.response.ad_confidence_threshold)} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, ad_confidence_threshold: parseInt(v) || 75 } })} />
              <Input type="number" label="流式尾部缓冲大小" value={String(configForm.response.stream_tail_buffer_size)} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, stream_tail_buffer_size: parseInt(v) || 2048 } })} />
              <Switch isSelected={configForm.response.preserve_code_blocks} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, preserve_code_blocks: v } })}>保留代码块</Switch>
              <Switch isSelected={configForm.response.preserve_json_output} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, preserve_json_output: v } })}>保留 JSON 输出</Switch>
              <Switch isSelected={configForm.response.validate_output_tool_calls} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, validate_output_tool_calls: v } })}>校验响应工具调用</Switch>
              <Switch isSelected={configForm.response.block_invalid_output_tool_calls} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, block_invalid_output_tool_calls: v } })}>阻断非法响应工具调用</Switch>
              <Switch isSelected={configForm.response.detect_prompt_injection} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, detect_prompt_injection: v } })}>检测响应注入</Switch>
              <Switch isSelected={configForm.response.detect_multimodal} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, detect_multimodal: v } })}>检测多模态风险</Switch>
              <Switch isSelected={configForm.response.detect_thinking_attacks} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, detect_thinking_attacks: v } })}>检测 Thinking 攻击</Switch>
            </div>
            <Input label="已知广告特征词（逗号分隔）" value={configForm.response.known_ad_patterns.join(',')} onValueChange={v => setConfigForm({ ...configForm, response: { ...configForm.response, known_ad_patterns: v.split(',').map(s => s.trim()).filter(Boolean) } })} />
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">信任边界配置（宸汐清源 2026）</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.trust.enable_provenance} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, enable_provenance: v } })}>启用内容来源可信度分级</Switch>
              <Switch isSelected={configForm.trust.scan_tool_descriptions} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, scan_tool_descriptions: v } })}>扫描工具描述投毒</Switch>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.trust.scan_memory_poisoning} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, scan_memory_poisoning: v } })}>扫描历史记忆投毒</Switch>
              <Input type="number" label="记忆扫描最大消息数" value={String(configForm.trust.memory_scan_max_messages)} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, memory_scan_max_messages: parseInt(v) || 50 } })} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="检索文档风险倍率" value={String(configForm.trust.retrieved_doc_risk_multiplier)} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, retrieved_doc_risk_multiplier: parseFloat(v) || 1.6 } })} />
              <Input type="number" label="工具输出风险倍率" value={String(configForm.trust.tool_output_risk_multiplier)} onValueChange={v => setConfigForm({ ...configForm, trust: { ...configForm.trust, tool_output_risk_multiplier: parseFloat(v) || 1.4 } })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">AI 审核（宸汐清源 2026）</div>
            <Select
              label="审核模式"
              selectedKeys={[configForm.ai_review.mode]}
              onSelectionChange={keys => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, mode: [...keys][0] as string || 'off' } })}
            >
              <SelectItem key="off">关闭</SelectItem>
              <SelectItem key="pre">AI 预审（用户消息先经 AI 审核，通过才转发）</SelectItem>
              <SelectItem key="re">规则初审 + AI 复审（规则命中后异步复核，不阻塞请求）</SelectItem>
              <SelectItem key="both">双重审核（先 AI 预审，再规则初审 + AI 复审）</SelectItem>
            </Select>
            <div className="text-xs leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
              <span className="font-medium">AI 预审</span>：请求转发前先用 AI 模型审核，未通过直接拦截，会增加请求延迟。
              <br />
              <span className="font-medium">规则初审 + AI 复审</span>：规则引擎命中风险后异步交给 AI 复核，不阻塞本次请求，仅用于事件记录和事后复盘。
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Select
                label="审核渠道"
                placeholder="选择渠道"
                selectedKeys={configForm.ai_review.channel_id ? [String(configForm.ai_review.channel_id)] : []}
                onSelectionChange={keys => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, channel_id: parseInt([...keys][0] as string) || 0 } })}
              >
                {channels.map(c => <SelectItem key={String(c.id)}>{c.name}</SelectItem>)}
              </Select>
              <Input
                label="审核模型名"
                placeholder="如 gpt-4o-mini"
                value={configForm.ai_review.model}
                onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, model: v } })}
              />
              <Input
                type="number"
                label="审核超时（秒）"
                value={String(configForm.ai_review.timeout_sec)}
                onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, timeout_sec: parseInt(v) || 10 } })}
              />
              <Input
                type="number"
                label="拦截阈值（风险分 0-100）"
                value={String(configForm.ai_review.block_score)}
                onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, block_score: parseInt(v) || 70 } })}
              />
              <Input
                type="number"
                label="送审文本最大字符数"
                value={String(configForm.ai_review.max_text_chars)}
                onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, max_text_chars: parseInt(v) || 2000 } })}
              />
            </div>
            <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
              填写一个支持 OpenAI 兼容接口的渠道（在「渠道管理」中查看），建议使用低成本快速模型。审核服务不可用时自动放行（fail-open）。
            </div>
            <Textarea
              label="AI 预审提示词（留空用内置默认）"
              minRows={4}
              placeholder="留空使用内置默认提示词，系统会自动追加 JSON 返回格式要求"
              value={configForm.ai_review.pre_prompt}
              onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, pre_prompt: v } })}
            />
            <Textarea
              label="AI 复审提示词（留空用内置默认）"
              minRows={4}
              placeholder="留空使用内置默认提示词，复审时会把规则引擎命中的风险信号一并传给 AI"
              value={configForm.ai_review.re_prompt}
              onValueChange={v => setConfigForm({ ...configForm, ai_review: { ...configForm.ai_review, re_prompt: v } })}
            />
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">日志记录</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={configForm.logging.log_events} onValueChange={v => setConfigForm({ ...configForm, logging: { ...configForm.logging, log_events: v } })}>记录事件</Switch>
              <Switch isSelected={configForm.logging.log_raw_content} onValueChange={v => setConfigForm({ ...configForm, logging: { ...configForm.logging, log_raw_content: v } })}>记录原文内容</Switch>
              <Switch isSelected={configForm.logging.hash_content} onValueChange={v => setConfigForm({ ...configForm, logging: { ...configForm.logging, hash_content: v } })}>内容哈希留存</Switch>
              <Input type="number" label="日志片段长度" value={String(configForm.logging.log_snippet_chars)} onValueChange={v => setConfigForm({ ...configForm, logging: { ...configForm.logging, log_snippet_chars: parseInt(v) || 160 } })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={sectionStyle}>
            <div className="text-sm font-semibold">熔断与降级</div>
            <div className="grid grid-cols-3 gap-4">
              <Switch isSelected={configForm.circuit_breaker.enabled} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, enabled: v } })}>启用熔断</Switch>
              <Input type="number" label="失败阈值" value={String(configForm.circuit_breaker.failure_threshold)} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, failure_threshold: parseInt(v) || 5 } })} />
              <Input type="number" label="超时(ms)" value={String(configForm.circuit_breaker.timeout_per_req_ms)} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, timeout_per_req_ms: parseInt(v) || 500 } })} />
            </div>
            <Input type="number" label="熔断冷却时间(秒)" value={String(configForm.circuit_breaker.cooldown_seconds)} onValueChange={v => setConfigForm({ ...configForm, circuit_breaker: { ...configForm.circuit_breaker, cooldown_seconds: parseInt(v) || 30 } })} />
            <div className="grid grid-cols-2 gap-4">
              <Input label="monitor 模式超时动作" value={configForm.degradation.monitor_timeout_action} onValueChange={v => setConfigForm({ ...configForm, degradation: { ...configForm.degradation, monitor_timeout_action: v || 'skip' } })} />
              <Input label="protect 模式超时动作" value={configForm.degradation.protect_timeout_action} onValueChange={v => setConfigForm({ ...configForm, degradation: { ...configForm.degradation, protect_timeout_action: v || 'fallback_to_guard' } })} />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
