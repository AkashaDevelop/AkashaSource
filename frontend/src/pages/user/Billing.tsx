import { useState, useEffect } from 'react';
import { Input, Button } from '../../components/ui';
import {
  Wallet, Gift, CreditCard, Crown, Zap, Users, Package,
  CheckCircle, Clock, TrendingUp, Sparkles, X, ChevronRight,
  ArrowRight, ReceiptText,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import PageHeader from '../../components/PageHeader';
import { formatQuota, getQuotaDisplayHint } from '../../lib/quota';

/* ───── 类型 ───── */
interface Plan {
  id: number; name: string; description: string; price: number;
  duration_days: number; type: string; group_name: string; quota: number; rpm: number;
}
interface Sub {
  id: number; plan_id: number; plan: Plan;
  status: number; started_at: number; expired_at: number; created_at: number;
}

/* ───── 常量 ───── */
const TABS = [
  { key: 'topup',    label: '在线充值', icon: CreditCard  },
  { key: 'redeem',   label: '兑换码',   icon: Gift        },
  { key: 'plans',    label: '订阅套餐', icon: Crown       }, 
  { key: 'history',  label: '订阅记录', icon: ReceiptText },
  { key: 'payments', label: '充值记录', icon: Clock       },
] as const;
type TabKey = typeof TABS[number]['key'];

const TOPUP_PRESETS = [5, 10, 20, 50, 100];

const PLAN_STYLES: Record<string, { gradient: string; light: string; icon: React.ReactNode; accent: string }> = {
  group: { gradient: 'linear-gradient(135deg,#0891b2,#22d3ee)', light: 'rgba(8,145,178,0.08)', icon: <Users size={18} />,   accent: '#0891b2' },
  quota: { gradient: 'linear-gradient(135deg,#059669,#34d399)', light: 'rgba(5,150,105,0.08)',  icon: <Package size={18} />, accent: '#059669' },
  rpm:   { gradient: 'linear-gradient(135deg,#d97706,#fbbf24)', light: 'rgba(217,119,6,0.08)',  icon: <Zap size={18} />,     accent: '#d97706' },
  combo: { gradient: 'linear-gradient(135deg,#7c3aed,#a78bfa)', light: 'rgba(124,58,237,0.08)', icon: <Crown size={18} />,   accent: '#7c3aed' },
};

const STATUS_BADGE: Record<number, { text: string; color: string; bg: string }> = {
  0: { text: '待支付', color: '#f59e0b', bg: 'rgba(245,158,11,0.1)'  },
  1: { text: '生效中', color: 'var(--color-success-fg)', bg: 'var(--color-success-bg)' },
  2: { text: '已过期', color: 'var(--text-muted)', bg: 'var(--bg-elevated)' },
  3: { text: '已取消', color: 'var(--color-danger-fg)', bg: 'var(--color-danger-bg)' },
};

function fmtDate(ts: number) {
  return ts > 0 ? new Date(ts * 1000).toLocaleDateString('zh-CN') : '永久';
}

/* ══════════════════════════════════════
   支付确认弹窗
══════════════════════════════════════ */
function PayModal({ plan, onClose, onConfirm, paying }: {
  plan: Plan; onClose: () => void;
  onConfirm: (payType: string) => void; paying: boolean;
}) {
  const [payType, setPayType] = useState<'alipay' | 'wxpay'>('alipay');
  const cfg = PLAN_STYLES[plan.type] || PLAN_STYLES.combo;

  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '16px', backdropFilter: 'blur(4px)' }}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="confirm-dialog" style={{ background: 'var(--bg-surface)', borderRadius: 'var(--radius-2xl)', border: '1px solid var(--border-color)', width: '100%', maxWidth: '380px', overflow: 'hidden', boxShadow: 'var(--shadow-hover)' }}>
        {/* 顶部色条 */}
        <div style={{ height: '4px', background: cfg.gradient }} />

        <div style={{ padding: '24px' }}>
          {/* 标题 */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{ width: '40px', height: '40px', borderRadius: 'var(--radius-lg)', background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', flexShrink: 0 }}>
                {cfg.icon}
              </div>
              <div>
                <p style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>{plan.name}</p>
                <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>
                  {plan.duration_days > 0 ? `有效期 ${plan.duration_days} 天` : '永久有效'}
                </p>
              </div>
            </div>
            <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-faint)', padding: '4px' }}>
              <X size={18} />
            </button>
          </div>

          {/* 金额 */}
          <div style={{ padding: '16px', borderRadius: 'var(--radius-xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>应付金额</span>
            <span style={{ fontSize: '30px', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1 }}>
              {plan.price === 0 ? <span style={{ fontSize: '20px' }}>免费</span> : `¥${plan.price.toFixed(2)}`}
            </span>
          </div>

          {/* 支付方式 */}
          <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-muted)', marginBottom: '10px', textTransform: 'uppercase', letterSpacing: '0.05em' }}>支付方式</p>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '20px' }}>
            {([
              { key: 'alipay', label: '支付宝', color: '#1677ff', emoji: '💙' },
              { key: 'wxpay',  label: '微信支付', color: '#07c160', emoji: '💚' },
            ] as const).map(m => (
              <button
                key={m.key}
                onClick={() => setPayType(m.key)}
                style={{
                  padding: '12px', borderRadius: 'var(--radius-lg)', cursor: 'pointer', textAlign: 'center',
                  border: `2px solid ${payType === m.key ? m.color : 'var(--border-color)'}`,
                  background: payType === m.key ? `${m.color}12` : 'var(--bg-elevated)',
                  color: payType === m.key ? m.color : 'var(--text-secondary)',
                  fontWeight: 600, fontSize: '13px', transition: 'all 0.15s',
                }}
              >
                <div style={{ fontSize: '18px', marginBottom: '4px' }}>{m.emoji}</div>
                {m.label}
              </button>
            ))}
          </div>

          {/* 操作 */}
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onClose} style={{ flex: 1, padding: '10px', borderRadius: 'var(--radius-lg)', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: 'var(--text-secondary)', fontSize: '13px', fontWeight: 500, cursor: 'pointer' }}>
              取消
            </button>
            <button
              onClick={() => onConfirm(payType)}
              disabled={paying}
              style={{ flex: 2, padding: '10px', borderRadius: 'var(--radius-lg)', border: 'none', background: cfg.gradient, color: '#fff', fontSize: '13px', fontWeight: 700, cursor: paying ? 'not-allowed' : 'pointer', opacity: paying ? 0.7 : 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}
            >
              {paying ? '处理中...' : '确认支付'}
              {!paying && <ArrowRight size={14} />}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════
   主页面
══════════════════════════════════════ */
export default function BillingPage() {
  const { token, user, updateUser } = useAuthStore();
  const [activeTab, setActiveTab] = useState<TabKey>('topup');

  /* 充值状态 */
  const [amount,     setAmount]     = useState('');
  const [topupPaying,setTopupPaying]= useState(false);
  const [enableTopup,setEnableTopup]= useState(false);
  const [minTopup,   setMinTopup]   = useState(0);

  /* 兑换码状态 */
  const [code,        setCode]        = useState('');
  const [redeemLoading,setRedeemLoading]= useState(false);

  /* 订阅状态 */
  const [plans,       setPlans]       = useState<Plan[]>([]);
  const [mySubs,      setMySubs]      = useState<Sub[]>([]);
  const [selectedPlan,setSelectedPlan]= useState<Plan | null>(null);
  const [subPaying,   setSubPaying]   = useState(false);

  /* 计费偏好 */
  const [billingPreference, setBillingPreference] = useState('subscription_first');

  /* 充值历史 */
  interface PayOrder { id: number; amount: number; quota_added: number; status: number; provider: string; created_at: number; completed_at: number; trade_no: string; }
  const [payOrders,   setPayOrders]   = useState<PayOrder[]>([]);
  const [payTotal,    setPayTotal]    = useState(0);
  const [payPage,     setPayPage]     = useState(1);
  const [payLoading,  setPayLoading]  = useState(false);
  const fetchPayOrders = async (page = 1) => {
    setPayLoading(true);
    try {
      const r = await fetch(`/api/payment/history?page=${page}&size=10`, { headers: { Authorization: `Bearer ${token}` } });
      const d = await r.json();
      if (d.code === 0) { setPayOrders(d.data.data || []); setPayTotal(d.data.total || 0); setPayPage(page); }
    } finally { setPayLoading(false); }
  };

  /* 衍生值 */
  const quota     = user?.quota ?? 0;
  const usedQuota = (user as any)?.used_quota ?? 0;
  const totalEver = quota + usedQuota;
  const usedPct   = totalEver > 0 ? Math.min((usedQuota / totalEver) * 100, 100) : 0;
  const usd       = formatQuota(quota, 4);
  const usedUsd   = formatQuota(usedQuota, 4);
  const activeSubs = mySubs.filter(s => s.status === 1);

  useEffect(() => {
    if (!token) return;
    // 用户信息
    fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0) updateUser(d.data); }).catch(() => {});
    // 系统配置
    fetch('/api/system/status').then(r => r.json()).then(d => {
      const p = d.code === 0 ? d.data : d;
      setEnableTopup(p.options?.enable_topup === 'true');
      const min = parseFloat(p.options?.min_topup || '0');
      if (min > 0) setMinTopup(min);
    }).catch(() => {});
    // 订阅计划
    fetch('/api/subscription/plans', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0) setPlans(d.data || []); }).catch(() => {});
    // 我的订阅
    fetch('/api/subscription/my', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0) setMySubs(d.data || []); }).catch(() => {});
    // 计费偏好
    fetch('/api/subscription/self', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0 && d.data.billing_preference) setBillingPreference(d.data.billing_preference); }).catch(() => {});
  }, [token]);

  // 切换到充值记录 Tab 时加载数据
  useEffect(() => {
    if (activeTab === 'payments' && token) fetchPayOrders(1);
  }, [activeTab, token]); // eslint-disable-line react-hooks/exhaustive-deps

  /* 在线充值 */
  const handleTopup = async () => {
    const amt = parseFloat(amount);
    if (!amt || amt <= 0) { toast.error('请输入有效金额'); return; }
    if (minTopup > 0 && amt < minTopup) { toast.error(`最低充值金额为 $${minTopup}`); return; }
    setTopupPaying(true);
    try {
      const res  = await fetch('/api/payment/create', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ amount: amt }) });
      const data = await res.json();
      if (data.code === 0 && data.data?.pay_url) window.location.href = data.data.pay_url;
      else toast.error(data.msg || '创建订单失败');
    } catch { toast.error('请求失败'); }
    finally { setTopupPaying(false); }
  };

  /* 兑换码 */
  const handleRedeem = async () => {
    if (!code.trim()) return;
    setRedeemLoading(true);
    try {
      const res  = await fetch('/api/user/redemption', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ code: code.trim() }) });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`兑换成功！增加额度 ${formatQuota(data.data.quota, 4)}`);
        setCode('');
        fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } }).then(r => r.json()).then(d => { if (d.code === 0) updateUser(d.data); });
      } else toast.error(data.msg || '兑换失败');
    } catch { toast.error('请求失败'); }
    finally { setRedeemLoading(false); }
  };

  /* 订阅 */
  const handleSubscribe = async (payType: string) => {
    if (!selectedPlan) return;
    setSubPaying(true);
    try {
      const res  = await fetch('/api/subscription/subscribe', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` }, body: JSON.stringify({ plan_id: selectedPlan.id, pay_type: payType }) });
      const data = await res.json();
      if (data.code === 0 && data.data?.order?.pay_url) window.location.href = data.data.order.pay_url;
      else toast.error(data.msg || '创建订单失败');
    } catch { toast.error('请求失败'); }
    finally { setSubPaying(false); }
  };

  /* 计费偏好 */
  const handleBillingPreferenceChange = async (pref: string) => {
    try {
      const res = await fetch('/api/subscription/self/preference', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ billing_preference: pref }),
      });
      const data = await res.json();
      if (data.code === 0) { setBillingPreference(pref); toast.success('计费偏好已更新'); }
      else toast.error(data.msg || '更新失败');
    } catch { toast.error('请求失败'); }
  };

  /* ── 渲染 ── */
  return (
    <div className="space-y-6 max-w-[960px] mx-auto">
      <PageHeader
        title="账单中心"
        description="管理余额、订阅套餐与支付记录"
      />

      {/* ══ 账户余额卡 ══ */}
      <div style={{
        borderRadius: 'var(--radius-2xl)',
        background: 'linear-gradient(135deg, rgba(124,58,237,0.12) 0%, rgba(8,145,178,0.1) 60%, rgba(16,185,129,0.06) 100%)',
        border: '1px solid var(--border-strong)',
        padding: '24px 28px',
        position: 'relative', overflow: 'hidden',
      }}>
        {/* 装饰光晕 */}
        <div style={{ position: 'absolute', top: '-60px', right: '-60px', width: '200px', height: '200px', borderRadius: '50%', background: 'radial-gradient(circle, var(--accent-glow) 0%, transparent 70%)', pointerEvents: 'none' }} />
        <div style={{ position: 'absolute', bottom: '-40px', left: '30%', width: '160px', height: '160px', borderRadius: '50%', background: 'radial-gradient(circle, rgba(8,145,178,0.1) 0%, transparent 70%)', pointerEvents: 'none' }} />

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 items-center">
          {/* 余额主数字 */}
          <div>
            <div className="flex items-center gap-2 mb-2">
              <div style={{ width: '36px', height: '36px', borderRadius: 'var(--radius-lg)', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 12px var(--accent-glow)' }}>
                <Wallet size={18} color="white" />
              </div>
              <div>
                <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0 }}>账户余额</p>
                <p style={{ fontSize: '10px', color: 'var(--text-faint)', margin: 0 }}>{getQuotaDisplayHint()}</p>
              </div>
              {activeSubs.length > 0 && (
                <div style={{ marginLeft: '8px', display: 'flex', alignItems: 'center', gap: '4px', padding: '2px 8px', borderRadius: 'var(--radius-full)', background: 'var(--color-success-bg)', border: '1px solid var(--color-success-bg)' }}>
                  <Sparkles size={10} style={{ color: 'var(--color-success-fg)' }} />
                  <span style={{ fontSize: '11px', color: 'var(--color-success-fg)', fontWeight: 600 }}>订阅中</span>
                </div>
              )}
            </div>
            <div className="flex items-baseline gap-1.5">
              <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>$</span>
              <span style={{ fontSize: '42px', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1, letterSpacing: '-1.5px' }}>{usd}</span>
            </div>
            <p style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '4px' }}>
              已消耗 <span style={{ color: 'var(--color-danger-fg)', fontWeight: 600 }}>${usedUsd}</span>
            </p>
          </div>

          {/* 使用率 + 统计 */}
          <div className="md:col-span-2">
            <div className="flex items-center justify-between mb-2">
              <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>额度使用率</span>
              <span style={{ fontSize: '12px', fontWeight: 600, color: usedPct > 80 ? 'var(--color-danger-fg)' : 'var(--text-secondary)' }}>{usedPct.toFixed(1)}%</span>
            </div>
            <div style={{ height: '6px', borderRadius: 'var(--radius-full)', background: 'rgba(0,0,0,0.15)', overflow: 'hidden', marginBottom: '16px' }}>
              <div style={{ height: '100%', borderRadius: 'var(--radius-full)', width: `${usedPct}%`, background: usedPct > 80 ? 'linear-gradient(90deg,var(--color-danger-fg),var(--color-danger))' : 'linear-gradient(90deg,var(--accent-primary),var(--accent-cosmic))', transition: 'width 0.6s ease' }} />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {[
                { label: '剩余额度', value: quota.toLocaleString(), icon: <Wallet size={13} />, color: 'var(--accent-cosmic)' },
                { label: '累计消耗', value: usedQuota.toLocaleString(), icon: <TrendingUp size={13} />, color: 'var(--color-danger-fg)' },
                { label: '活跃订阅', value: activeSubs.length || '无', icon: <Crown size={13} />, color: 'var(--accent-star)' },
                { label: '订阅总数', value: mySubs.length, icon: <ReceiptText size={13} />, color: 'var(--text-muted)' },
              ].map(item => (
                <div key={item.label} style={{ padding: '10px 12px', borderRadius: 'var(--radius-lg)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', backdropFilter: 'blur(8px)' }}>
                  <div className="flex items-center gap-1.5 mb-1" style={{ color: 'var(--text-muted)' }}>
                    {item.icon}
                    <span style={{ fontSize: '10px' }}>{item.label}</span>
                  </div>
                  <p style={{ fontSize: '14px', fontWeight: 700, color: item.color, margin: 0 }}>{item.value}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ══ 计费偏好 ══ */}
      <div style={{ borderRadius: 'var(--radius-2xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', padding: '20px 24px' }}>
        <div className="flex items-center gap-2 mb-3">
          <Wallet size={16} style={{ color: 'var(--accent-primary)' }} />
          <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>计费偏好</span>
        </div>
        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>资金来源优先级</span>
          <div className="flex gap-2 flex-wrap">
            {([
              { key: 'subscription_first', label: '订阅优先' },
              { key: 'wallet_first', label: '余额优先' },
              { key: 'subscription_only', label: '仅订阅' },
              { key: 'wallet_only', label: '仅余额' },
            ] as const).map(item => {
              const active = billingPreference === item.key;
              return (
                <button key={item.key} type="button"
                  onClick={() => handleBillingPreferenceChange(item.key)}
                  className={`px-3 py-1.5 rounded-lg border text-sm transition-all ${active
                    ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                    : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] cursor-pointer'}`}
                >
                  {item.label}
                </button>
              );
            })}
          </div>
          <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
            订阅优先：先扣订阅额度，不足再扣余额；余额优先：先扣余额，不足再扣订阅；仅订阅/仅余额：只从对应来源扣减，不足则请求失败
          </p>
        </div>
      </div>

      {/* ══ Tab 切换器 ══ */}
      <div style={{ display: 'flex', gap: '2px', background: 'var(--bg-elevated)', borderRadius: 'var(--radius-lg)', padding: '4px', width: 'fit-content' }}>
        {TABS.map(tab => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.key;
          // 订阅记录 tab 显示数量徽章
          const badge = tab.key === 'history' ? mySubs.length : tab.key === 'plans' ? plans.length : 0;
          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150"
              style={{
                background: isActive ? 'var(--bg-surface)' : 'transparent',
                color: isActive ? 'var(--accent-primary)' : 'var(--text-secondary)',
                border: 'none', cursor: 'pointer',
                boxShadow: isActive ? 'var(--shadow-card)' : 'none',
              }}
            >
              <Icon size={14} />
              <span>{tab.label}</span>
              {badge > 0 && (
                <span style={{ fontSize: '10px', fontWeight: 700, padding: '1px 5px', borderRadius: 'var(--radius-full)', background: isActive ? 'var(--color-info-bg)' : 'var(--bg-elevated)', color: isActive ? 'var(--accent-primary)' : 'var(--text-faint)', minWidth: '18px', textAlign: 'center' }}>
                  {badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* ══ 在线充值 Tab ══ */}
      {activeTab === 'topup' && (
        enableTopup ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {/* 快捷金额 */}
            <div style={{ borderRadius: 'var(--radius-2xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', padding: '24px' }}>
              <div className="flex items-center gap-3 mb-5">
                <div style={{ width: '38px', height: '38px', borderRadius: 'var(--radius-lg)', background: 'var(--color-info-bg)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <CreditCard size={17} style={{ color: 'var(--accent-cosmic)' }} />
                </div>
                <div>
                  <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>在线充值</p>
                  <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>通过易支付充值{minTopup > 0 ? `，最低 $${minTopup}` : ''}</p>
                </div>
              </div>

              {/* 快捷金额按钮 */}
              <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '8px' }}>快捷金额</p>
              <div className="grid grid-cols-5 gap-2 mb-4">
                {TOPUP_PRESETS.map(p => (
                  <button
                    key={p}
                    onClick={() => setAmount(String(p))}
                    style={{
                      padding: '8px 4px', borderRadius: 'var(--radius-md)', fontSize: '13px', fontWeight: 600,
                      border: `1px solid ${amount === String(p) ? 'var(--accent-primary)' : 'var(--border-color)'}`,
                      background: amount === String(p) ? 'var(--nav-active-bg)' : 'var(--bg-elevated)',
                      color: amount === String(p) ? 'var(--accent-primary)' : 'var(--text-secondary)',
                      cursor: 'pointer', transition: 'all 0.12s', textAlign: 'center',
                    }}
                  >${p}</button>
                ))}
              </div>

              {/* 自定义金额 */}
              <div style={{ display: 'flex', gap: '10px' }}>
                <Input
                  placeholder={minTopup > 0 ? `最低 $${minTopup}` : '自定义金额'}
                  value={amount}
                  onValueChange={setAmount}
                  type="number"
                  startContent={<span style={{ fontSize: '13px', color: 'var(--text-muted)' }}>$</span>}
                  onKeyDown={e => { if (e.key === 'Enter') handleTopup(); }}
                  size="lg"
                  style={{ flex: 1 }}
                />
                <Button size="lg" onPress={handleTopup} isLoading={topupPaying} isDisabled={!amount}
                  style={{ borderRadius: 'var(--radius-lg)', minWidth: '90px', background: 'linear-gradient(135deg,var(--accent-cosmic),var(--accent-primary))', color: 'white', border: 'none', fontWeight: 600 }}>
                  充值
                </Button>
              </div>
            </div>

            {/* 充值说明 */}
            <div style={{ borderRadius: 'var(--radius-2xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', padding: '24px' }}>
              <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>充值说明</p>
              {[
                { icon: '⚡', text: '充值即时到账，无需等待审核' },
                { icon: '🔒', text: '支持支付宝、微信支付，安全可靠' },
                { icon: '💰', text: getQuotaDisplayHint() },
                { icon: '📋', text: '如遇问题请联系客服，保留支付凭证' },
              ].map((item, i) => (
                <div key={i} style={{ display: 'flex', gap: '10px', marginBottom: i < 3 ? '12px' : 0 }}>
                  <span style={{ fontSize: '16px', flexShrink: 0, lineHeight: 1.4 }}>{item.icon}</span>
                  <p style={{ fontSize: '13px', color: 'var(--text-secondary)', margin: 0, lineHeight: 1.6 }}>{item.text}</p>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: '60px', color: 'var(--text-muted)', borderRadius: 'var(--radius-2xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <CreditCard size={32} style={{ margin: '0 auto 12px', opacity: 0.3 }} />
            <p style={{ fontSize: '14px' }}>在线充值功能暂未开放</p>
            <p style={{ fontSize: '12px', marginTop: '4px', opacity: 0.7 }}>请使用兑换码或联系管理员充值</p>
          </div>
        )
      )}

      {/* ══ 兑换码 Tab ══ */}
      {activeTab === 'redeem' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
          <div style={{ borderRadius: 'var(--radius-2xl)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', padding: '24px' }}>
            <div className="flex items-center gap-3 mb-5">
              <div style={{ width: '38px', height: '38px', borderRadius: 'var(--radius-lg)', background: 'var(--color-warning-bg)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Gift size={17} style={{ color: 'var(--accent-star)' }} />
              </div>
              <div>
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>兑换码</p>
                <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>输入兑换码，额度立即到账</p>
              </div>
            </div>

            <div style={{ display: 'flex', gap: '10px' }}>
              <Input
                placeholder="请输入兑换码..."
                value={code}
                onValueChange={setCode}
                onKeyDown={e => { if (e.key === 'Enter') handleRedeem(); }}
                startContent={<Gift size={14} className="text-default-400" />}
                size="lg"
                style={{ flex: 1 }}
              />
              <Button size="lg" onPress={handleRedeem} isLoading={redeemLoading} isDisabled={!code.trim()}
                style={{ borderRadius: 'var(--radius-lg)', minWidth: '80px', background: 'linear-gradient(135deg,var(--accent-star),var(--accent-primary))', color: 'white', border: 'none', fontWeight: 600 }}>
                兑换
              </Button>
            </div>
            <p style={{ fontSize: '11px', color: 'var(--text-faint)', marginTop: '10px' }}>
              兑换码区分大小写，请确保输入完整 · 按 Enter 快速兑换
            </p>
          </div>

          <div style={{ borderRadius: 'var(--radius-2xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', padding: '24px' }}>
            <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '16px' }}>使用说明</p>
            {[
              { icon: '🎁', text: '兑换码由管理员发放，每个码限用一次' },
              { icon: '⏱', text: '兑换码可能有有效期限制，请及时使用' },
              { icon: '✅', text: '成功兑换后额度立即增加至账户余额' },
            ].map((item, i) => (
              <div key={i} style={{ display: 'flex', gap: '10px', marginBottom: i < 2 ? '12px' : 0 }}>
                <span style={{ fontSize: '16px', flexShrink: 0, lineHeight: 1.4 }}>{item.icon}</span>
                <p style={{ fontSize: '13px', color: 'var(--text-secondary)', margin: 0, lineHeight: 1.6 }}>{item.text}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ══ 订阅套餐 Tab ══ */}
      {activeTab === 'plans' && (
        <div>
          {plans.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '60px', color: 'var(--text-muted)', borderRadius: 'var(--radius-2xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
              <Crown size={32} style={{ margin: '0 auto 12px', opacity: 0.3 }} />
              <p style={{ fontSize: '14px' }}>暂无可用套餐</p>
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px,1fr))', gap: '16px' }}>
              {plans.map(plan => {
                const cfg = PLAN_STYLES[plan.type] || PLAN_STYLES.combo;
                const isActive = activeSubs.some(s => s.plan_id === plan.id);
                return (
                  <div
                    key={plan.id}
                    style={{
                      borderRadius: 'var(--radius-2xl)', background: 'var(--bg-surface)',
                      border: `1px solid ${isActive ? 'var(--color-success-fg)' : 'var(--border-color)'}`,
                      overflow: 'hidden', display: 'flex', flexDirection: 'column',
                      transition: 'transform 0.2s, box-shadow 0.2s',
                    }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(-3px)'; (e.currentTarget as HTMLElement).style.boxShadow = 'var(--shadow-hover)'; }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'translateY(0)'; (e.currentTarget as HTMLElement).style.boxShadow = 'none'; }}
                  >
                    {/* 顶部色条 */}
                    <div style={{ height: '4px', background: cfg.gradient }} />

                    <div style={{ padding: '20px', flex: 1, display: 'flex', flexDirection: 'column', gap: '16px' }}>
                      {/* 标题行 */}
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{ width: '42px', height: '42px', borderRadius: 'var(--radius-lg)', background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', flexShrink: 0 }}>
                          {cfg.icon}
                        </div>
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <p style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>{plan.name}</p>
                          <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: '2px 0 0' }}>
                            {plan.duration_days > 0 ? `有效期 ${plan.duration_days} 天` : '永久有效'}
                          </p>
                        </div>
                        {isActive && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '4px', padding: '2px 8px', borderRadius: 'var(--radius-full)', background: 'var(--color-success-bg)', flexShrink: 0 }}>
                            <CheckCircle size={11} style={{ color: 'var(--color-success-fg)' }} />
                            <span style={{ fontSize: '10px', color: 'var(--color-success-fg)', fontWeight: 600 }}>生效中</span>
                          </div>
                        )}
                      </div>

                      {/* 描述 */}
                      {plan.description && (
                        <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0, lineHeight: 1.6 }}>{plan.description}</p>
                      )}

                      {/* 权益列表 */}
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                        {(plan.type === 'group' || plan.type === 'combo') && plan.group_name && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                            <div style={{ width: '18px', height: '18px', borderRadius: '50%', background: cfg.light, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                              <Users size={10} style={{ color: cfg.accent }} />
                            </div>
                            <span>切换至 <strong style={{ color: 'var(--text-primary)' }}>{plan.group_name}</strong> 分组</span>
                          </div>
                        )}
                        {(plan.type === 'quota' || plan.type === 'combo') && plan.quota > 0 && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                            <div style={{ width: '18px', height: '18px', borderRadius: '50%', background: cfg.light, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                              <Package size={10} style={{ color: cfg.accent }} />
                            </div>
                            <span>赠送 <strong style={{ color: 'var(--text-primary)' }}>{formatQuota(plan.quota, 2)}</strong> 额度</span>
                          </div>
                        )}
                        {(plan.type === 'rpm' || plan.type === 'combo') && plan.rpm > 0 && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                            <div style={{ width: '18px', height: '18px', borderRadius: '50%', background: cfg.light, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                              <Zap size={10} style={{ color: cfg.accent }} />
                            </div>
                            <span>RPM 提升至 <strong style={{ color: 'var(--text-primary)' }}>{plan.rpm}</strong></span>
                          </div>
                        )}
                      </div>

                      {/* 价格 + CTA */}
                      <div style={{ marginTop: 'auto', paddingTop: '12px', borderTop: '1px solid var(--border-color)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <div>
                          <span style={{ fontSize: '28px', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1 }}>
                            {plan.price === 0 ? <span style={{ fontSize: '20px', color: 'var(--color-success-fg)' }}>免费</span> : `¥${plan.price.toFixed(2)}`}
                          </span>
                          {plan.price > 0 && plan.duration_days > 0 && (
                            <span style={{ fontSize: '11px', color: 'var(--text-faint)', marginLeft: '4px' }}>/{plan.duration_days}天</span>
                          )}
                        </div>
                        <button
                          onClick={() => setSelectedPlan(plan)}
                          style={{
                            padding: '8px 16px', borderRadius: 'var(--radius-lg)', border: 'none',
                            background: cfg.gradient, color: '#fff', fontSize: '13px', fontWeight: 600,
                            cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '4px',
                            transition: 'opacity 0.15s',
                          }}
                          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.opacity = '0.85'; }}
                          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.opacity = '1'; }}
                        >
                          {isActive ? '续订' : '订阅'}
                          <ChevronRight size={13} />
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ══ 订阅记录 Tab ══ */}
      {activeTab === 'history' && (
        mySubs.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '60px', color: 'var(--text-muted)', borderRadius: 'var(--radius-2xl)', background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}>
            <ReceiptText size={32} style={{ margin: '0 auto 12px', opacity: 0.3 }} />
            <p style={{ fontSize: '14px' }}>暂无订阅记录</p>
          </div>
        ) : (
          <div style={{ borderRadius: 'var(--radius-xl)', border: '1px solid var(--border-color)', overflow: 'hidden', background: 'var(--bg-surface)' }}>
            {/* 表头 */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 100px 120px 120px 80px', gap: '16px', padding: '11px 18px', background: 'var(--bg-elevated)', borderBottom: '1px solid var(--border-color)' }}>
              {['套餐', '类型', '购买时间', '到期时间', '状态'].map(h => (
                <p key={h} style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', margin: 0, textTransform: 'uppercase', letterSpacing: '0.04em' }}>{h}</p>
              ))}
            </div>
            {mySubs.map((s, i) => {
              const st  = STATUS_BADGE[s.status] || STATUS_BADGE[2];
              const cfg = PLAN_STYLES[s.plan?.type] || PLAN_STYLES.combo;
              return (
                <div
                  key={s.id}
                  style={{ display: 'grid', gridTemplateColumns: '1fr 100px 120px 120px 80px', gap: '16px', padding: '13px 18px', borderBottom: i < mySubs.length - 1 ? '1px solid var(--border-color)' : 'none', alignItems: 'center', transition: 'background 0.12s' }}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'var(--table-hover)'; }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: 'var(--radius-md)', flexShrink: 0, background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
                      <span style={{ transform: 'scale(0.7)', display: 'flex' }}>{cfg.icon}</span>
                    </div>
                    <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {s.plan?.name || `套餐 #${s.plan_id}`}
                    </p>
                  </div>
                  <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: 'var(--radius-full)', background: cfg.light, color: cfg.accent, fontWeight: 600, width: 'fit-content' }}>
                    {{ group: '分组', quota: '额度', rpm: 'RPM', combo: '组合' }[s.plan?.type] || '套餐'}
                  </span>
                  <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0 }}>{fmtDate(s.created_at)}</p>
                  <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0, display: 'flex', alignItems: 'center', gap: '4px' }}>
                    {s.expired_at > 0 ? <><Clock size={11} />{fmtDate(s.expired_at)}</> : '永久'}
                  </p>
                  <span style={{ fontSize: '11px', fontWeight: 700, padding: '3px 8px', borderRadius: 'var(--radius-full)', background: st.bg, color: st.color, width: 'fit-content' }}>
                    {st.text}
                  </span>
                </div>
              );
            })}
          </div>
        )
      )}

      {/* ══ 充值记录 Tab ══ */}
      {activeTab === 'payments' && (
        <div>
          <div style={{ borderRadius: 'var(--radius-xl)', border: '1px solid var(--border-color)', overflow: 'hidden', background: 'var(--bg-surface)' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '80px 1fr 80px 100px 120px', gap: '12px', padding: '11px 18px', background: 'var(--bg-elevated)', borderBottom: '1px solid var(--border-color)' }}>
              {['ID', '支付渠道', '金额', '到账额度', '状态'].map(h => (
                <p key={h} style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-muted)', margin: 0, textTransform: 'uppercase' }}>{h}</p>
              ))}
            </div>
            {payLoading ? (
              <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-muted)' }}>加载中...</div>
            ) : payOrders.length === 0 ? (
              <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-muted)' }}>
                <Clock size={28} style={{ margin: '0 auto 8px', opacity: 0.3 }} />
                <p style={{ fontSize: '13px' }}>暂无充值记录</p>
              </div>
            ) : payOrders.map(o => {
              const statusMap: Record<number, {text:string,color:string,bg:string}> = {
                0: { text: '处理中', color: '#d97706', bg: 'rgba(217,119,6,0.1)' },
                1: { text: '成功', color: '#059669', bg: 'rgba(5,150,105,0.1)' },
                2: { text: '失败', color: '#dc2626', bg: 'rgba(220,38,38,0.1)' },
                3: { text: '已过期', color: '#6b7280', bg: 'rgba(107,114,128,0.1)' },
              };
              const st = statusMap[o.status] || statusMap[0];
              return (
                <div key={o.id} style={{ display: 'grid', gridTemplateColumns: '80px 1fr 80px 100px 120px', gap: '12px', padding: '12px 18px', borderBottom: '1px solid var(--border-color)', alignItems: 'center' }}>
                  <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: 0 }}>#{o.id}</p>
                  <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0, textTransform: 'capitalize' }}>{o.provider}</p>
                  <p style={{ fontSize: '13px', fontWeight: 600, margin: 0 }}>${o.amount.toFixed(2)}</p>
                  <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>
                    {o.quota_added > 0 ? formatQuota(o.quota_added, 4) : '-'}
                  </p>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <span style={{ fontSize: '11px', fontWeight: 700, padding: '2px 8px', borderRadius: 'var(--radius-full)', background: st.bg, color: st.color, width: 'fit-content' }}>{st.text}</span>
                    <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: 0 }}>{new Date(o.created_at * 1000).toLocaleDateString()}</p>
                  </div>
                </div>
              );
            })}
          </div>
          {payTotal > 10 && (
            <div style={{ display: 'flex', justifyContent: 'center', gap: '8px', marginTop: '12px' }}>
              <Button size="sm" variant="flat" isDisabled={payPage <= 1} onPress={() => fetchPayOrders(payPage - 1)}>上一页</Button>
              <span style={{ fontSize: '13px', color: 'var(--text-muted)', lineHeight: '32px' }}>{payPage} / {Math.ceil(payTotal / 10)}</span>
              <Button size="sm" variant="flat" isDisabled={payPage >= Math.ceil(payTotal / 10)} onPress={() => fetchPayOrders(payPage + 1)}>下一页</Button>
            </div>
          )}
        </div>
      )}

      {/* ══ 支付确认弹窗 ══ */}
      {selectedPlan && (
        <PayModal
          plan={selectedPlan}
          onClose={() => setSelectedPlan(null)}
          onConfirm={handleSubscribe}
          paying={subPaying}
        />
      )}
    </div>
  );
}
