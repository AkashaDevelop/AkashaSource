import { useState, useEffect, useCallback } from 'react';
import { Button, Progress, Chip } from '../components/ui';
import {
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  AreaChart, Area, PieChart, Pie, Cell,
} from 'recharts';
import {
  Activity, CreditCard, Key, RefreshCw, Zap,
  FlaskConical, Tag, History, Cpu, Hash, Copy, Check, Shield,
} from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useNavigate } from 'react-router-dom';
import { toast } from '../store/toast';
import StatCard from '../components/StatCard';
import { SkeletonCard } from '../components/SkeletonCard';
import { formatQuota } from '../lib/quota';

/* ───── 类型 ───── */
interface UserStats  { token_count: number; request_count: number; }
interface UserInfo   { username: string; quota: number; used_quota: number; role: number; group?: string; }
interface ChartData  { date: string; usage: number; }
interface ModelStat  { model_name: string; total_quota: number; prompt_tokens: number; completion_tokens: number; request_count: number; }
interface RecentLog  { id: number; created_at: number; model_name: string; quota: number; prompt_tokens: number; completion_tokens: number; type: number; token_name: string; }
interface PeriodStats { requests: number; quota: number; tokens: number; promptTokens: number; completionTokens: number; }

/* ───── 常量 ───── */
const TIME_RANGES = [
  { key: 'today' as const, label: '今日',    days: 0  },
  { key: '7d'   as const, label: '近 7 天', days: 6  },
  { key: '30d'  as const, label: '近 30 天',days: 29 },
];

const QUICK_ACTIONS = [
  { icon: Key,          label: '令牌管理',   desc: '创建 API 密钥', path: '/token',      color: 'var(--accent-star)',      bg: 'var(--color-warning-bg)' },
  { icon: History,      label: '调用日志',   desc: '查看历史记录', path: '/log',        color: 'var(--accent-primary)',   bg: 'var(--color-info-bg)'    },
  { icon: FlaskConical, label: 'Playground', desc: '在线测试接口', path: '/playground', color: 'var(--accent-cosmic)',    bg: 'var(--color-info-bg)'    },
  { icon: Tag,          label: '模型定价',   desc: '查看价格表',  path: '/pricing',    color: 'var(--color-success-fg)', bg: 'var(--color-success-bg)' },
];

const LOG_TYPES: Record<number, { color: string }> = {
  1: { color: 'var(--accent-primary)' },
  2: { color: 'var(--color-success-fg)' },
  3: { color: 'var(--accent-cosmic)' },
  4: { color: 'var(--color-danger-fg)' },
};

/* 饼图用实际 hex（recharts SVG 不支持 CSS 变量） */
const PIE_COLORS = ['#a78bfa', '#22d3ee', '#fbbf24', '#34d399', '#f87171', '#94a3b8'];
const MODEL_COLORS = [
  'var(--accent-primary)', 'var(--accent-cosmic)', 'var(--accent-star)',
  'var(--color-success-fg)', 'var(--color-warning-fg)', 'var(--text-muted)',
];

/* ───── 辅助：RPM 计算 ───── */
function calcRPM(requests: number, timeRange: 'today' | '7d' | '30d') {
  const mins = { today: 1440, '7d': 7 * 1440, '30d': 30 * 1440 }[timeRange];
  const v = requests / mins;
  return v < 1 ? v.toFixed(3) : v.toFixed(2);
}

/* ───── 主组件 ───── */
export default function Dashboard() {
  const { user, token } = useAuthStore();
  const navigate = useNavigate();

  const [stats,     setStats]     = useState<UserStats   | null>(null);
  const [userInfo,  setUserInfo]  = useState<UserInfo    | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [loading,   setLoading]   = useState(false);

  const [checkedIn,       setCheckedIn]       = useState(false);
  const [checkinLoading,  setCheckinLoading]  = useState(false);
  const [checkinCaptcha,  setCheckinCaptcha]  = useState(false);
  const [captchaProvider, setCaptchaProvider] = useState('');
  const [geetestEnabled,  setGeetestEnabled]  = useState(false);
  const [geetestId,       setGeetestId]       = useState('');
  const [turnstileEnabled,setTurnstileEnabled]= useState(false);
  const [turnstileSiteKey,setTurnstileSiteKey]= useState('');

  const [timeRange,    setTimeRange]    = useState<'today'|'7d'|'30d'>('7d');
  const [modelStats,   setModelStats]   = useState<ModelStat[]>([]);
  const [recentLogs,   setRecentLogs]   = useState<RecentLog[]>([]);
  const [periodStats,  setPeriodStats]  = useState<PeriodStats | null>(null);
  const [statLoading,  setStatLoading]  = useState(false);
  const [apiUrlCopied, setApiUrlCopied] = useState(false);

  const getDateRange = useCallback(() => {
    const now  = new Date();
    const end  = now.toISOString().split('T')[0];
    const days = TIME_RANGES.find(r => r.key === timeRange)?.days ?? 6;
    const start = days === 0
      ? end
      : new Date(now.getTime() - days * 86400000).toISOString().split('T')[0];
    return { start, end };
  }, [timeRange]);

  const fetchCheckinStatus = async () => {
    try {
      const res = await fetch('/api/user/checkin', { headers: { Authorization: `Bearer ${token}` } });
      const d   = await res.json();
      if (d.code === 0) setCheckedIn(d.data.checked_in);
    } catch {}
  };

  const handleCheckin = async () => {
    setCheckinLoading(true);
    try {
      let body: any = {};
      if (checkinCaptcha) {
        const useGeetest   = captchaProvider === 'geetest' && geetestEnabled;
        const useTurnstile = captchaProvider === 'turnstile' ? turnstileEnabled : (!captchaProvider && turnstileEnabled);
        if (useGeetest) {
          const gd = await new Promise<any>((resolve) => {
            if (!(window as any).initGeetest4) { resolve(null); return; }
            (window as any).initGeetest4({ captchaId: geetestId, product: 'bind' }, (o: any) => {
              o.onSuccess(() => resolve(o.getValidate()));
              o.onError(() => resolve(null));
              o.onClose(() => resolve(null));
              o.showCaptcha();
            });
          });
          if (!gd) { setCheckinLoading(false); return; }
          body.geetest = gd;
        } else if (useTurnstile) {
          const tk = await new Promise<string>((resolve) => {
            if (!(window as any).turnstile) { resolve(''); return; }
            (window as any).turnstile.render('#checkin-turnstile', {
              sitekey: turnstileSiteKey,
              callback: (t: string) => resolve(t),
              'error-callback': () => resolve(''),
            });
          });
          if (!tk) { toast.warning('请完成人机验证'); setCheckinLoading(false); return; }
          body.turnstile = tk;
        }
      }
      const res = await fetch('/api/user/checkin', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const d = await res.json();
      if (d.code === 0) {
        setCheckedIn(true);
        toast.success(`签到成功！获得 ${formatQuota(d.data.reward, 4)} 额度`);
        fetchDashboard();
      } else {
        toast.error(d.msg || '签到失败');
      }
    } catch {}
    finally { setCheckinLoading(false); }
  };

  const fetchDashboard = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/dashboard', { headers: { Authorization: `Bearer ${token}` } });
      const d   = await res.json();
      if (d.code === 0) {
        setStats(d.data.stats);
        setUserInfo(d.data.user);
        setChartData(d.data.chart || []);
      }
    } catch {}
    finally { setLoading(false); }
  };

  const fetchPeriodData = useCallback(async () => {
    if (!token) return;
    setStatLoading(true);
    const { start, end } = getDateRange();
    try {
      const params = new URLSearchParams({ start_time: start, end_time: end });
      const [sRes, lRes] = await Promise.all([
        fetch(`/api/log/self/stat?${params}`,       { headers: { Authorization: `Bearer ${token}` } }),
        fetch('/api/log/self/search?page=1&size=8', { headers: { Authorization: `Bearer ${token}` } }),
      ]);
      const [sData, lData] = await Promise.all([sRes.json(), lRes.json()]);
      if (sData.code === 0 && sData.data) {
        const items: ModelStat[] = sData.data.items || [];
        const sorted = [...items].sort((a, b) => b.total_quota - a.total_quota);
        setModelStats(sorted.slice(0, 6));
        const promptTok = items.reduce((s, i) => s + i.prompt_tokens, 0);
        const compTok   = items.reduce((s, i) => s + i.completion_tokens, 0);
        setPeriodStats({
          requests:        items.reduce((s, i) => s + i.request_count, 0),
          quota:           sData.data.total_quota || 0,
          tokens:          promptTok + compTok,
          promptTokens:    promptTok,
          completionTokens:compTok,
        });
      }
      if (lData.code === 0) setRecentLogs((lData.data?.data || []).slice(0, 7));
    } catch {}
    finally { setStatLoading(false); }
  }, [token, getDateRange]);

  useEffect(() => {
    if (!token) return;
    fetchDashboard();
    fetchCheckinStatus();
    fetch('/api/system/status').then(r => r.json()).then(d => {
      const p = d.code === 0 ? d.data : d;
      if (!p.options) return;
      if (p.options.checkin_captcha === 'true') setCheckinCaptcha(true);
      if (p.options.captcha_provider)           setCaptchaProvider(p.options.captcha_provider);
      if (p.options.turnstile_check_enabled === 'true') { setTurnstileEnabled(true); setTurnstileSiteKey(p.options.turnstile_site_key || ''); }
      if (p.options.geetest_enabled === 'true')         { setGeetestEnabled(true);   setGeetestId(p.options.geetest_id || ''); }
    }).catch(() => {});
  }, [token]);

  useEffect(() => { fetchPeriodData(); }, [fetchPeriodData]);

  /* 衍生值 */
  const remainingRaw  = userInfo ? (userInfo.quota - (userInfo.used_quota || 0)) : 0;
  const remainingDisplay = formatQuota(remainingRaw, 4);
  const usedDisplay      = formatQuota((userInfo?.used_quota || 0), 4);
  const remainingPct  = userInfo?.quota ? Math.min((remainingRaw / userInfo.quota) * 100, 100) : 0;
  const totalModelQ   = modelStats.reduce((s, m) => s + m.total_quota, 0);
  const rangeLabel    = { today: '今日', '7d': '近7天', '30d': '近30天' }[timeRange];
  const rpm           = periodStats ? calcRPM(periodStats.requests, timeRange) : '—';
  const initials      = (userInfo?.username || user?.username || 'U').slice(0, 2).toUpperCase();
  const apiBase       = `${window.location.origin}/v1`;

  const handleCopyApi = () => {
    navigator.clipboard.writeText(apiBase).then(() => {
      setApiUrlCopied(true);
      setTimeout(() => setApiUrlCopied(false), 2000);
    });
  };

  const getRoleName = (role: number) => role >= 100 ? '超级管理员' : role >= 10 ? '管理员' : '普通用户';

  /* 饼图数据 */
  const pieData = modelStats.slice(0, 5).map(m => ({
    name: m.model_name.length > 16 ? m.model_name.slice(0, 14) + '…' : m.model_name,
    value: m.total_quota,
  }));

  /* ───── 渲染 ───── */
  return (
    <div className="space-y-5 animate-fade-in-up">

      {/* ══ 头部 ══ */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold gradient-text">仪表盘</h1>
          <p className="text-sm mt-0.5" style={{ color: 'var(--text-muted)' }}>
            欢迎回来，{userInfo?.username || user?.username} ✿
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <div style={{ display:'flex', gap:'3px', background:'var(--bg-elevated)', borderRadius:'var(--radius-md)', padding:'3px' }}>
            {TIME_RANGES.map(r => (
              <button key={r.key} onClick={() => setTimeRange(r.key)} style={{
                padding:'4px 12px', borderRadius:'var(--radius-md)', fontSize:'12px', fontWeight:600,
                cursor:'pointer', border:'none',
                background: timeRange === r.key ? 'var(--bg-surface)' : 'transparent',
                color:      timeRange === r.key ? 'var(--accent-primary)' : 'var(--text-secondary)',
                boxShadow:  timeRange === r.key ? 'var(--shadow-card)' : 'none',
                transition:'all 0.15s',
              }}>{r.label}</button>
            ))}
          </div>
          <Button isIconOnly variant="flat" onPress={() => { fetchDashboard(); fetchPeriodData(); }} isLoading={loading}
            style={{ background:'var(--bg-elevated)', color:'var(--accent-primary)', borderRadius:'var(--radius-lg)', border:'1px solid var(--border-color)' }}>
            <RefreshCw size={16} />
          </Button>
          <Button onPress={() => navigate('/topup')}
            style={{ background:'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color:'white', borderRadius:'var(--radius-lg)', fontSize:'13px' }}>
            充值余额
          </Button>
        </div>
      </div>

      {/* ══ 统计卡片 ══ */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {loading && !stats ? Array.from({length:4}).map((_,i) => <SkeletonCard key={i} lines={1} />) : (<>
          <StatCard
            title="账户余额"
            value={<span style={{ color:'var(--accent-cosmic)' }}>{remainingDisplay}</span>}
            icon={<CreditCard size={20} style={{ color:'var(--accent-cosmic)' }} />}
            iconBg="var(--color-info-bg)"
            footer={<Progress size="sm" value={remainingPct} color="success" aria-label="余额" />}
          />
          <StatCard
            title={`${rangeLabel}调用`}
            value={statLoading ? <span style={{color:'var(--text-muted)'}}>—</span> : (periodStats?.requests ?? 0).toLocaleString()}
            icon={<Activity size={20} style={{ color:'var(--accent-primary)' }} />}
            iconBg="var(--color-info-bg)"
            footer={`历史总计 ${(stats?.request_count || 0).toLocaleString()} 次`}
          />
          <StatCard
            title={`${rangeLabel}消耗`}
            value={statLoading ? <span style={{color:'var(--text-muted)'}}>—</span> : <span style={{color:'var(--color-danger-fg)'}}>{formatQuota(periodStats?.quota||0, 4)}</span>}
            icon={<Zap size={20} style={{ color:'var(--color-danger-fg)' }} />}
            iconBg="var(--color-danger-bg)"
            footer={`累计 ${usedDisplay}`}
          />
          <StatCard
            title={`${rangeLabel} Token`}
            value={statLoading ? <span style={{color:'var(--text-muted)'}}>—</span> : (periodStats?.tokens ?? 0).toLocaleString()}
            icon={<Hash size={20} style={{ color:'var(--accent-star)' }} />}
            iconBg="var(--color-warning-bg)"
            footer={`活跃令牌 ${stats?.token_count || 0} 个`}
          />
        </>)}
      </div>

      {/* ══ 性能指标横条 ══ */}
      <div className="akasha-card px-5 py-3.5">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: '输入 Token',  value: statLoading ? '—' : (periodStats?.promptTokens     ?? 0).toLocaleString(), icon: <Cpu  size={14} />, color:'var(--accent-primary)' },
            { label: '输出 Token',  value: statLoading ? '—' : (periodStats?.completionTokens ?? 0).toLocaleString(), icon: <Hash size={14} />, color:'var(--accent-cosmic)'  },
            { label: '均 RPM',      value: statLoading ? '—' : rpm,                                                    icon: <Activity size={14} />, color:'var(--color-success-fg)' },
            { label: '日均调用',    value: statLoading ? '—' : (timeRange==='today' ? (periodStats?.requests??0) : Math.round((periodStats?.requests??0) / (timeRange==='7d'?7:30))).toLocaleString(),
              icon: <RefreshCw size={14} />, color:'var(--accent-star)' },
          ].map(m => (
            <div key={m.label} className="flex items-center gap-2.5">
              <span style={{ color: m.color, opacity: 0.8 }}>{m.icon}</span>
              <div>
                <p style={{ fontSize:'11px', color:'var(--text-muted)', margin:0 }}>{m.label}</p>
                <p style={{ fontSize:'14px', fontWeight:700, color:'var(--text-primary)', margin:0, lineHeight:1.3 }}>{m.value}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Token 比例条 */}
        {!statLoading && periodStats && periodStats.tokens > 0 && (
          <div className="mt-3 pt-3" style={{ borderTop:'1px solid var(--border-color)' }}>
            <div className="flex items-center justify-between mb-1.5">
              <span style={{ fontSize:'11px', color:'var(--text-muted)' }}>Token 构成</span>
              <div className="flex items-center gap-3">
                <span style={{ fontSize:'11px', color:'var(--accent-primary)' }}>■ 输入 {((periodStats.promptTokens/periodStats.tokens)*100).toFixed(1)}%</span>
                <span style={{ fontSize:'11px', color:'var(--accent-cosmic)'  }}>■ 输出 {((periodStats.completionTokens/periodStats.tokens)*100).toFixed(1)}%</span>
              </div>
            </div>
            <div style={{ height:'5px', borderRadius:'var(--radius-full)', background:'var(--bg-elevated)', overflow:'hidden', display:'flex' }}>
              <div style={{ height:'100%', width:`${(periodStats.promptTokens/periodStats.tokens)*100}%`, background:'var(--accent-primary)', transition:'width 0.6s ease' }} />
              <div style={{ height:'100%', flex:1, background:'var(--accent-cosmic)' }} />
            </div>
          </div>
        )}
      </div>

      {/* ══ 主内容区：趋势图 + 右侧面板 ══ */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">

        {/* 消耗趋势图 */}
        <div className="lg:col-span-2 akasha-card p-5">
          <h3 className="text-subtitle mb-4" style={{ color:'var(--text-primary)' }}>消耗趋势（近 7 日）</h3>
          <div style={{ height:'200px' }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorUsage" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="var(--accent-primary)" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="var(--accent-cosmic)"  stopOpacity={0.05} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
                <XAxis dataKey="date" tick={{ fill:'var(--text-muted)', fontSize:11 }} />
                <YAxis tick={{ fill:'var(--text-muted)', fontSize:11 }} tickFormatter={(v:number) => formatQuota(v, 3)} />
                <Tooltip
                  cursor={{ fill:'var(--bg-elevated)' }}
                  contentStyle={{ background:'var(--bg-elevated)', borderRadius:'var(--radius-lg)', border:'1px solid var(--border-color)', fontSize:'12px' }}
                  formatter={(v:number|undefined) => [formatQuota(v??0, 6), '消耗']}
                />
                <Area type="monotone" dataKey="usage" stroke="var(--accent-primary)" strokeWidth={2} fillOpacity={1} fill="url(#colorUsage)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* 右侧面板 */}
        <div className="flex flex-col gap-4">

          {/* 账户卡 */}
          <div className="akasha-card p-4">
            <div className="flex items-center gap-3 mb-3">
              <div style={{
                width:'44px', height:'44px', borderRadius:'50%', flexShrink:0,
                background:'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
                display:'flex', alignItems:'center', justifyContent:'center',
                fontSize:'16px', fontWeight:700, color:'white',
                boxShadow:'0 0 0 3px var(--color-info-bg)',
              }}>{initials}</div>
              <div className="min-w-0">
                <p style={{ fontSize:'14px', fontWeight:600, color:'var(--text-primary)', margin:0 }}>
                  {userInfo?.username || user?.username}
                </p>
                <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                  <span style={{ fontSize:'11px', padding:'1px 7px', borderRadius:'var(--radius-full)', background:'var(--color-info-bg)', color:'var(--accent-primary)', fontWeight:600 }}>
                    {getRoleName(userInfo?.role || user?.role || 1)}
                  </span>
                  {userInfo?.group && (
                    <span style={{ fontSize:'11px', padding:'1px 7px', borderRadius:'var(--radius-full)', background:'var(--bg-elevated)', color:'var(--text-muted)', border:'1px solid var(--border-color)' }}>
                      {userInfo.group}
                    </span>
                  )}
                </div>
              </div>
            </div>

            {/* 配额进度 */}
            <div className="space-y-1.5 mb-3">
              <div className="flex justify-between">
                <span style={{ fontSize:'11px', color:'var(--text-muted)' }}>余额使用率</span>
                <span style={{ fontSize:'11px', color:'var(--text-muted)' }}>{(100 - remainingPct).toFixed(1)}%</span>
              </div>
              <Progress size="sm" value={100 - remainingPct}
                color={(100 - remainingPct) > 80 ? 'danger' : (100 - remainingPct) > 60 ? 'warning' : 'primary'}
                aria-label="配额使用率" />
              <div className="flex justify-between">
                <span style={{ fontSize:'11px', color:'var(--text-muted)' }}>已消耗 {usedDisplay}</span>
                <span style={{ fontSize:'11px', color:'var(--accent-cosmic)', fontWeight:600 }}>余 {remainingDisplay}</span>
              </div>
            </div>

            {/* API 端点 */}
            <div style={{ borderRadius:'var(--radius-md)', background:'var(--bg-elevated)', border:'1px solid var(--border-color)', padding:'8px 10px' }}>
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <p style={{ fontSize:'10px', color:'var(--text-muted)', margin:'0 0 2px' }}>API Base URL</p>
                  <p style={{ fontSize:'11px', fontFamily:'monospace', color:'var(--accent-primary)', margin:0, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>
                    {apiBase}
                  </p>
                </div>
                <button onClick={handleCopyApi} style={{ background:'none', border:'none', cursor:'pointer', color: apiUrlCopied ? 'var(--color-success-fg)' : 'var(--text-muted)', flexShrink:0, transition:'color 0.2s' }}>
                  {apiUrlCopied ? <Check size={14} /> : <Copy size={14} />}
                </button>
              </div>
            </div>
          </div>

          {/* 签到 */}
          <div className="akasha-card p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div style={{ padding:'7px', borderRadius:'var(--radius-lg)', background: checkedIn ? 'var(--color-success-bg)' : 'var(--bg-elevated)', border:`1px solid ${checkedIn ? 'var(--color-success-bg)' : 'var(--border-color)'}`, fontSize:'17px', lineHeight:1 }}>
                  {checkedIn ? '✅' : '🎁'}
                </div>
                <div>
                  <p style={{ fontSize:'13px', fontWeight:600, color:'var(--text-primary)', margin:0 }}>每日签到</p>
                  <p style={{ fontSize:'11px', color: checkedIn ? 'var(--color-success-fg)' : 'var(--text-muted)', margin:0 }}>
                    {checkedIn ? '今日已签到 ✦' : '领取随机额度'}
                  </p>
                </div>
              </div>
              <Button size="sm" isDisabled={checkedIn} isLoading={checkinLoading} onPress={handleCheckin}
                style={checkedIn
                  ? { background:'var(--bg-elevated)', color:'var(--text-muted)', borderRadius:'var(--radius-md)', fontSize:'12px', minWidth:'60px' }
                  : { background:'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color:'white', borderRadius:'var(--radius-md)', fontSize:'12px', minWidth:'60px' }
                }>
                {checkedIn ? '已签到' : '签到'}
              </Button>
            </div>
          </div>

          {/* 快捷操作 */}
          <div className="akasha-card p-4 flex-1">
            <p style={{ fontSize:'11px', fontWeight:600, color:'var(--text-muted)', textTransform:'uppercase', letterSpacing:'0.06em', marginBottom:'10px' }}>快捷操作</p>
            <div className="grid grid-cols-2 gap-2">
              {QUICK_ACTIONS.map(a => (
                <button key={a.path} onClick={() => navigate(a.path)}
                  className="flex items-center gap-2 p-2.5 rounded-xl text-left"
                  style={{ background:a.bg, border:`1px solid ${a.bg}`, cursor:'pointer', transition:'transform 0.15s, box-shadow 0.15s' }}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform='translateY(-2px)'; (e.currentTarget as HTMLElement).style.boxShadow='var(--shadow-hover)'; }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform='translateY(0)'; (e.currentTarget as HTMLElement).style.boxShadow='none'; }}
                >
                  <a.icon size={14} style={{ color:a.color, flexShrink:0 }} />
                  <div className="min-w-0">
                    <p style={{ fontSize:'12px', fontWeight:600, color:'var(--text-primary)', margin:0 }}>{a.label}</p>
                    <p style={{ fontSize:'10px', color:'var(--text-muted)', margin:0, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>{a.desc}</p>
                  </div>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ══ 底部三栏：饼图 + 模型排行 + 最近调用 ══ */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">

        {/* 模型分布饼图 */}
        <div className="akasha-card p-5">
          <h3 className="text-subtitle mb-2" style={{ color:'var(--text-primary)' }}>{rangeLabel}模型分布</h3>
          {!statLoading && pieData.length === 0 ? (
            <div style={{ padding:'40px 0', textAlign:'center', color:'var(--text-muted)', fontSize:'13px' }}>暂无数据</div>
          ) : statLoading && pieData.length === 0 ? (
            <div style={{ height:'160px' }} className="skeleton-shimmer rounded-xl" />
          ) : (
            <>
              <div style={{ height:'160px' }}>
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={pieData} cx="50%" cy="50%"
                      innerRadius={42} outerRadius={70}
                      paddingAngle={3} dataKey="value"
                    >
                      {pieData.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
                    </Pie>
                    <Tooltip
                      contentStyle={{ background:'var(--bg-elevated)', borderRadius:'var(--radius-lg)', border:'1px solid var(--border-color)', fontSize:'11px' }}
                      formatter={(v: number) => [formatQuota(v, 4), '消耗']}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="space-y-1.5 mt-1">
                {pieData.map((m, i) => (
                  <div key={m.name} className="flex items-center gap-2">
                    <span style={{ width:'8px', height:'8px', borderRadius:'50%', background:PIE_COLORS[i], flexShrink:0 }} />
                    <span style={{ fontSize:'11px', color:'var(--text-muted)', overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap', flex:1 }}>{m.name}</span>
                    <span style={{ fontSize:'11px', fontWeight:600, color:'var(--text-primary)', flexShrink:0 }}>
                      {totalModelQ > 0 ? ((m.value/totalModelQ)*100).toFixed(1) : 0}%
                    </span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* 模型排行 */}
        <div className="akasha-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-subtitle" style={{ color:'var(--text-primary)' }}>{rangeLabel}模型排行</h3>
            {statLoading && <span className="text-xs animate-nebula-pulse" style={{ color:'var(--text-muted)' }}>加载中</span>}
          </div>
          {!statLoading && modelStats.length === 0 ? (
            <div style={{ padding:'40px 0', textAlign:'center', color:'var(--text-muted)', fontSize:'13px' }}>暂无数据</div>
          ) : (
            <div className="space-y-3">
              {(statLoading && modelStats.length === 0 ? Array.from({length:4}) : modelStats).map((m, i) => {
                if (!m) return <div key={i} className="space-y-1"><div className="skeleton-shimmer h-3 w-2/3 rounded" /><div className="skeleton-shimmer h-2 rounded" /></div>;
                const pct   = totalModelQ > 0 ? (m.total_quota / totalModelQ) * 100 : 0;
                const color = MODEL_COLORS[i] || 'var(--text-muted)';
                return (
                  <div key={m.model_name}>
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-1.5 min-w-0 flex-1">
                        <span style={{ width:'16px', height:'16px', borderRadius:'50%', background:color, display:'inline-flex', alignItems:'center', justifyContent:'center', fontSize:'9px', fontWeight:700, color:'white', flexShrink:0 }}>{i+1}</span>
                        <span style={{ fontSize:'11px', fontFamily:'monospace', color:'var(--text-primary)', overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>{m.model_name}</span>
                      </div>
                      <span style={{ fontSize:'11px', fontWeight:600, color:'var(--text-secondary)', flexShrink:0, marginLeft:'6px' }}>{m.request_count.toLocaleString()}</span>
                    </div>
                    <div style={{ height:'3px', borderRadius:'var(--radius-full)', background:'var(--bg-elevated)' }}>
                      <div style={{ height:'100%', borderRadius:'var(--radius-full)', width:`${pct}%`, background:color, transition:'width 0.6s ease' }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 最近调用 */}
        <div className="akasha-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-subtitle" style={{ color:'var(--text-primary)' }}>最近调用</h3>
            <button onClick={() => navigate('/log')} style={{ fontSize:'12px', color:'var(--accent-primary)', background:'none', border:'none', cursor:'pointer' }}>
              全部 →
            </button>
          </div>
          {!statLoading && recentLogs.length === 0 ? (
            <div style={{ padding:'40px 0', textAlign:'center', color:'var(--text-muted)', fontSize:'13px' }}>暂无记录</div>
          ) : (
            <div className="space-y-2">
              {(statLoading && recentLogs.length === 0 ? Array.from({length:5}) : recentLogs).map((log, i) => {
                if (!log) return <div key={i} className="skeleton-shimmer rounded-lg" style={{ height:'48px' }} />;
                const ti = LOG_TYPES[log.type] || { color:'var(--text-muted)' };
                return (
                  <div key={log.id} style={{ display:'flex', alignItems:'center', gap:'9px', padding:'8px 10px', borderRadius:'var(--radius-lg)', background:'var(--bg-elevated)', border:'1px solid var(--border-color)' }}>
                    <div style={{ width:'6px', height:'6px', borderRadius:'50%', background:ti.color, flexShrink:0 }} />
                    <div style={{ flex:1, minWidth:0 }}>
                      <p style={{ fontSize:'12px', fontWeight:500, color:'var(--text-primary)', margin:0, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>
                        {log.model_name || '—'}
                      </p>
                      <p style={{ fontSize:'10px', color:'var(--text-muted)', margin:0 }}>
                        {new Date(log.created_at * 1000).toLocaleString('zh-CN', { month:'numeric', day:'numeric', hour:'2-digit', minute:'2-digit' })}
                      </p>
                    </div>
                    <div style={{ textAlign:'right', flexShrink:0 }}>
                      <p style={{ fontSize:'11px', fontWeight:600, color: log.type === 4 ? 'var(--color-danger-fg)' : 'var(--text-primary)', margin:0 }}>
                        {formatQuota(log.quota, 5)}
                      </p>
                      {log.type === 1 && (log.prompt_tokens + log.completion_tokens) > 0 && (
                        <p style={{ fontSize:'10px', color:'var(--text-muted)', margin:0 }}>
                          {(log.prompt_tokens + log.completion_tokens).toLocaleString()} tok
                        </p>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
