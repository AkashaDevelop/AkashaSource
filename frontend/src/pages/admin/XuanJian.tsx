import { useEffect, useState, useMemo } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';
import { Button, Input, Select, SelectItem, Switch, Chip } from '../../components/ui';
import { RefreshCw, Shield, ScrollText, Activity } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface XJConfig {
  mode: string;
  window_minutes: number;
  max_requests_per_win: number;
  max_quota_per_win: number;
  max_models_per_win: number;
  max_ip_cidrs_per_win: number;
  max_tokens_per_user: number;
  short_prompt_max_tokens: number;
  enable_abuse_detection: boolean;
  enable_jailbreak_detection: boolean;
  enable_llm_abuse: boolean;
  enable_agent_detection: boolean;
  enable_duplicate_detection: boolean;
  notify_admin: boolean;
  auto_disable_score: number;
  auto_ban_score: number;
  exempt_token_ids: number[];
  exempt_user_ids: number[];
}

interface EventRow {
  id: number;
  created_at: number;
  token_id: number;
  token_name: string;
  user_id: number;
  risk_score: number;
  finding_type: string;
  finding_group: string;
  action: string;
  evidence: string;
  ip: string;
  model: string;
}

const defaultConfig: XJConfig = {
  mode: 'protect',
  window_minutes: 5,
  max_requests_per_win: 300,
  max_quota_per_win: 10000000,
  max_models_per_win: 8,
  max_ip_cidrs_per_win: 15,
  max_tokens_per_user: 8,
  short_prompt_max_tokens: 20,
  enable_abuse_detection: true,
  enable_jailbreak_detection: true,
  enable_llm_abuse: true,
  enable_agent_detection: false,
  enable_duplicate_detection: true,
  notify_admin: true,
  auto_disable_score: 90,
  auto_ban_score: 95,
  exempt_token_ids: [],
  exempt_user_ids: [],
};

const groupLabel = (g: string) => ({
  llmjacking: 'LLM劫持',
  jailbreak: '模型破限',
  malware_gen: '恶意内容生成',
  reverse_eng: '逆向探测',
  agent_abuse: 'AI代理滥用',
}[g] || g);

const actionLabel = (a: string) => ({
  warn: '记录',
  notify: '告警',
  throttle: '限速',
  disable_token: '封Token',
  ban_user: '封用户',
}[a] || a);

const actionColor = (a: string): 'default' | 'warning' | 'danger' => ({
  warn: 'default',
  notify: 'warning',
  throttle: 'warning',
  disable_token: 'danger',
  ban_user: 'danger',
}[a] as any || 'default');

export default function XuanJian() {
  const { token } = useAuthStore();
  const [activeTab, setActiveTab] = useState<'config' | 'events' | 'profiles'>('config');
  const [enabled, setEnabled] = useState(false);
  const [config, setConfig] = useState<XJConfig>(defaultConfig);
  const [events, setEvents] = useState<EventRow[]>([]);
  const [profiles, setProfiles] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [eventFilter, setEventFilter] = useState({ finding_group: '', min_score: '' });

  const authHeaders = useMemo(() => ({ Authorization: `Bearer ${token}` }), [token]);

  const fetchConfig = async () => {
    try {
      const res = await fetch('/api/admin/xuanjian/config', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) {
        setEnabled(data.data.enabled);
        setConfig({ ...defaultConfig, ...data.data.config });
      }
    } catch (e) { console.error(e); }
  };

  const fetchEvents = async () => {
    const params = new URLSearchParams({ size: '50' });
    if (eventFilter.finding_group) params.set('finding_group', eventFilter.finding_group);
    if (eventFilter.min_score) params.set('min_score', eventFilter.min_score);
    try {
      const res = await fetch(`/api/admin/xuanjian/events?${params}`, { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setEvents(data.data?.items || []);
    } catch (e) { console.error(e); }
  };

  const fetchProfiles = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/admin/xuanjian/profiles', { headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) setProfiles(data.data);
    } catch (e) { console.error(e); } finally { setLoading(false); }
  };

  useEffect(() => {
    if (token) { fetchConfig(); fetchEvents(); }
  }, [token]);

  useEffect(() => {
    if (token && activeTab === 'events') fetchEvents();
    if (token && activeTab === 'profiles') fetchProfiles();
  }, [activeTab, eventFilter]);

  const saveConfig = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/admin/xuanjian/config', {
        method: 'PUT',
        headers: { ...authHeaders, 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled, config }),
      });
      const data = await res.json();
      if (data.code === 0) toast.success('配置已保存');
      else toast.error(data.msg || '保存失败');
    } catch { toast.error('保存失败'); } finally { setSaving(false); }
  };

  const resetProfile = async (tokenId: number) => {
    try {
      const res = await fetch(`/api/admin/xuanjian/reset/${tokenId}`, { method: 'POST', headers: authHeaders });
      const data = await res.json();
      if (data.code === 0) { toast.success('画像已清除'); fetchProfiles(); }
      else toast.error(data.msg || '清除失败');
    } catch { toast.error('清除失败'); }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="宸汐玄鉴"
        description="用户行为风控 · 破限拦截 · 逆向行为检测 · 仅超级管理员可访问"
        actions={
          <div className="flex gap-2 items-center">
            <Switch isSelected={enabled} onValueChange={setEnabled} size="sm">
              {enabled ? '已启用' : '已停用'}
            </Switch>
            <Button color="primary" isLoading={saving} onPress={saveConfig} size="sm">
              保存配置
            </Button>
          </div>
        }
      />

      <div className="flex gap-2 flex-wrap">
        <Button variant={activeTab === 'config' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('config')} startContent={<Shield size={16} />}>策略配置</Button>
        <Button variant={activeTab === 'events' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('events')} startContent={<ScrollText size={16} />}>事件日志</Button>
        <Button variant={activeTab === 'profiles' ? 'solid' : 'flat'} color="primary" onPress={() => setActiveTab('profiles')} startContent={<Activity size={16} />}>活跃画像</Button>
      </div>

      {activeTab === 'config' && (
        <div className="space-y-6" style={{ maxWidth: 700 }}>
          <div className="p-4 rounded-xl space-y-4" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <div className="text-sm font-semibold">运行模式</div>
            <Select label="模式" selectedKeys={[config.mode]} onSelectionChange={keys => setConfig({ ...config, mode: [...keys][0] as string || 'protect' })}>
              <SelectItem key="monitor">监控（只记录，不处置）</SelectItem>
              <SelectItem key="protect">防护（记录 + 通知管理员）</SelectItem>
              <SelectItem key="strict">严格（自动封禁）</SelectItem>
              <SelectItem key="off">关闭</SelectItem>
            </Select>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="检测窗口（分钟）" value={String(config.window_minutes)} onValueChange={v => setConfig({ ...config, window_minutes: parseInt(v) || 5 })} />
              <Switch isSelected={config.notify_admin} onValueChange={v => setConfig({ ...config, notify_admin: v })}>异常时通知管理员</Switch>
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <div className="text-sm font-semibold">行为阈值</div>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="最大请求数/窗口" value={String(config.max_requests_per_win)} onValueChange={v => setConfig({ ...config, max_requests_per_win: parseInt(v) || 300 })} />
              <Input type="number" label="最大 Quota/窗口（万）" value={String(Math.round(config.max_quota_per_win / 10000))} onValueChange={v => setConfig({ ...config, max_quota_per_win: (parseInt(v) || 1000) * 10000 })} />
              <Input type="number" label="最大模型种数/窗口" value={String(config.max_models_per_win)} onValueChange={v => setConfig({ ...config, max_models_per_win: parseInt(v) || 8 })} />
              <Input type="number" label="最大 IP 段(/24)/窗口" value={String(config.max_ip_cidrs_per_win)} onValueChange={v => setConfig({ ...config, max_ip_cidrs_per_win: parseInt(v) || 15 })} />
              <Input type="number" label="同用户最多 Token 数/窗口" value={String(config.max_tokens_per_user)} onValueChange={v => setConfig({ ...config, max_tokens_per_user: parseInt(v) || 8 })} />
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <div className="text-sm font-semibold">检测模块开关</div>
            <div className="grid grid-cols-2 gap-4">
              <Switch isSelected={config.enable_abuse_detection} onValueChange={v => setConfig({ ...config, enable_abuse_detection: v })}>速率/算力滥用检测</Switch>
              <Switch isSelected={config.enable_jailbreak_detection} onValueChange={v => setConfig({ ...config, enable_jailbreak_detection: v })}>模型破限行为检测</Switch>
              <Switch isSelected={config.enable_llm_abuse} onValueChange={v => setConfig({ ...config, enable_llm_abuse: v })}>LLM 内容滥用检测</Switch>
              <Switch isSelected={config.enable_duplicate_detection} onValueChange={v => setConfig({ ...config, enable_duplicate_detection: v })}>重复探测检测</Switch>
              <Switch isSelected={config.enable_agent_detection} onValueChange={v => setConfig({ ...config, enable_agent_detection: v })}>
                AI 代理攻击检测（初期建议关闭）
              </Switch>
            </div>
          </div>

          <div className="p-4 rounded-xl space-y-4" style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <div className="text-sm font-semibold">自动处置阈值（仅 strict 模式生效）</div>
            <div className="grid grid-cols-2 gap-4">
              <Input type="number" label="自动封 Token 分数阈值" value={String(config.auto_disable_score)} onValueChange={v => setConfig({ ...config, auto_disable_score: parseInt(v) || 90 })} />
              <Input type="number" label="自动封用户分数阈值" value={String(config.auto_ban_score)} onValueChange={v => setConfig({ ...config, auto_ban_score: parseInt(v) || 95 })} />
            </div>
          </div>
        </div>
      )}

      {activeTab === 'events' && (
        <div className="space-y-4">
          <div className="flex gap-2 flex-wrap">
            <Select placeholder="威胁分组" className="w-44" selectedKeys={eventFilter.finding_group ? [eventFilter.finding_group] : []}
              onSelectionChange={keys => setEventFilter({ ...eventFilter, finding_group: [...keys][0] as string || '' })}>
              <SelectItem key="llmjacking">LLM劫持</SelectItem>
              <SelectItem key="jailbreak">模型破限</SelectItem>
              <SelectItem key="malware_gen">恶意内容生成</SelectItem>
              <SelectItem key="reverse_eng">逆向探测</SelectItem>
              <SelectItem key="agent_abuse">AI代理滥用</SelectItem>
            </Select>
            <Input className="w-36" placeholder="最低风险分" value={eventFilter.min_score} onValueChange={v => setEventFilter({ ...eventFilter, min_score: v })} />
            <Button variant="flat" onPress={fetchEvents} startContent={<RefreshCw size={14} />}>刷新</Button>
          </div>
          <div className="data-table-wrap">
            <table className="data-table">
              <thead><tr><th>时间</th><th>分组</th><th>类型</th><th>风险分</th><th>处置</th><th>Token</th><th>IP</th><th>模型</th><th>证据</th></tr></thead>
              <tbody>
                {events.length === 0 ? (
                  <tr><td colSpan={9}><EmptyState icon="🔍" title="暂无风险事件" /></td></tr>
                ) : events.map(e => (
                  <tr key={e.id}>
                    <td className="whitespace-nowrap text-sm">{new Date(e.created_at * 1000).toLocaleString()}</td>
                    <td><Chip size="sm" variant="flat">{groupLabel(e.finding_group)}</Chip></td>
                    <td className="text-xs">{e.finding_type}</td>
                    <td><Chip size="sm" color={e.risk_score >= 85 ? 'danger' : e.risk_score >= 65 ? 'warning' : 'default'} variant="flat">{e.risk_score}</Chip></td>
                    <td><Chip size="sm" color={actionColor(e.action)} variant="flat">{actionLabel(e.action)}</Chip></td>
                    <td className="text-xs">{e.token_name || e.token_id}</td>
                    <td className="text-xs">{e.ip}</td>
                    <td className="text-xs max-w-xs truncate">{e.model}</td>
                    <td className="text-xs max-w-xs truncate">{e.evidence}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'profiles' && (
        <div className="space-y-4">
          <div className="flex gap-2 items-center">
            <Button variant="flat" onPress={fetchProfiles} startContent={<RefreshCw size={14} />} isLoading={loading}>刷新画像</Button>
            {profiles && (
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                活跃 Token：{profiles.stats?.active_tokens || 0} 个 | 活跃用户：{profiles.stats?.active_users || 0} 个
              </span>
            )}
          </div>
          <div className="data-table-wrap">
            <table className="data-table">
              <thead><tr><th>Token ID</th><th>用户 ID</th><th>请求数</th><th>Quota 消耗</th><th>模型种数</th><th>IP 段数</th><th>破限次数</th><th>最后活跃</th><th>操作</th></tr></thead>
              <tbody>
                {loading ? <LoadingRows cols={9} rows={5} /> :
                  !profiles?.top_risk?.length ? (
                    <tr><td colSpan={9}><EmptyState icon="📊" title="暂无高风险画像" /></td></tr>
                  ) : profiles.top_risk.map((p: any) => (
                    <tr key={p.token_id}>
                      <td>{p.token_id}</td>
                      <td>{p.user_id}</td>
                      <td>{p.request_count}</td>
                      <td>{Math.round(p.quota_burned / 1000)}K</td>
                      <td>{p.model_count}</td>
                      <td>{p.ip_cidr_count}</td>
                      <td><Chip size="sm" color={p.jailbreak_count > 3 ? 'danger' : 'default'} variant="flat">{p.jailbreak_count}</Chip></td>
                      <td className="text-sm whitespace-nowrap">{new Date(p.last_seen * 1000).toLocaleString()}</td>
                      <td>
                        <Button size="sm" variant="flat" onPress={() => resetProfile(p.token_id)}>
                          清除画像
                        </Button>
                      </td>
                    </tr>
                  ))
                }
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
