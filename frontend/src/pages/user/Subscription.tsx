import { useState, useEffect } from 'react';
import { Button } from '../../components/ui';
import { Crown, Zap, Users, Package, CheckCircle, Clock, Sparkles, X } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Plan {
  id: number; name: string; description: string; price: number;
  duration_days: number; type: string; group_name: string; quota: number; rpm: number;
}
interface Sub {
  id: number; plan_id: number; plan: Plan;
  status: number; started_at: number; expired_at: number; created_at: number;
}

const typeConfig: Record<string, { icon: React.ReactNode; gradient: string; accent: string }> = {
  group: { icon: <Users size={20} />,   gradient: 'linear-gradient(135deg,#0891b2,#06b6d4)', accent: '#0891b2' },
  quota: { icon: <Package size={20} />, gradient: 'linear-gradient(135deg,#059669,#34d399)', accent: '#059669' },
  rpm:   { icon: <Zap size={20} />,     gradient: 'linear-gradient(135deg,#d97706,#fbbf24)', accent: '#d97706' },
  combo: { icon: <Crown size={20} />,   gradient: 'linear-gradient(135deg,#7c3aed,#a78bfa)', accent: '#7c3aed' },
};

const statusLabel: Record<number, { text: string; color: string }> = {
  0: { text: '待支付', color: '#f59e0b' },
  1: { text: '生效中', color: '#34d399' },
  2: { text: '已过期', color: '#9ca3af' },
  3: { text: '已取消', color: '#f87171' },
};

function fmtDate(ts: number) {
  if (!ts) return '永久';
  return new Date(ts * 1000).toLocaleDateString('zh-CN');
}

export default function SubscriptionPage() {
  const { token } = useAuthStore();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [mySubs, setMySubs] = useState<Sub[]>([]);
  const [paying, setPaying] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<Plan | null>(null);
  const [payType, setPayType] = useState<'alipay' | 'wxpay'>('alipay');

  useEffect(() => {
    fetch('/api/subscription/plans', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0) setPlans(d.data || []); });
    fetch('/api/subscription/my', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json()).then(d => { if (d.code === 0) setMySubs(d.data || []); });
  }, []);

  const subscribe = async () => {
    if (!selectedPlan) return;
    setPaying(true);
    try {
      const res = await fetch('/api/subscription/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ plan_id: selectedPlan.id, pay_type: payType }),
      });
      const data = await res.json();
      if (data.code === 0 && data.data?.order?.pay_url) {
        window.location.href = data.data.order.pay_url;
      } else {
        toast.error(data.msg || '创建订单失败');
      }
    } catch { toast.error('请求失败'); }
    finally { setPaying(false); }
  };

  const activeSubs = mySubs.filter(s => s.status === 1);

  return (
    <div style={{ maxWidth: '860px', margin: '0 auto' }} className="space-y-8">
      {/* Hero */}
      <div style={{
        borderRadius: '24px', padding: '32px',
        background: 'linear-gradient(135deg, rgba(124,58,237,0.12) 0%, rgba(8,145,178,0.08) 100%)',
        border: '1px solid rgba(124,58,237,0.2)',
        display: 'flex', alignItems: 'center', gap: '20px', flexWrap: 'wrap',
      }}>
        <div style={{
          width: '52px', height: '52px', borderRadius: '16px', flexShrink: 0,
          background: 'linear-gradient(135deg,#7c3aed,#06b6d4)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <Sparkles size={24} color="#fff" />
        </div>
        <div>
          <h1 style={{ fontSize: '20px', fontWeight: 800, color: 'var(--text-primary)', margin: 0 }}>订阅中心</h1>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)', margin: '4px 0 0' }}>升级订阅，解锁更高速率与专属模型权限</p>
        </div>
        {activeSubs.length > 0 && (
          <div style={{ marginLeft: 'auto', display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '5px' }}>
            {activeSubs.map(s => (
              <div key={s.id} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <CheckCircle size={13} color="#34d399" />
                <span style={{ fontSize: '12px', fontWeight: 600, color: '#34d399' }}>{s.plan?.name}</span>
                {s.expired_at > 0 && (
                  <span style={{ fontSize: '11px', color: 'var(--text-faint)', display: 'flex', alignItems: 'center', gap: '3px' }}>
                    <Clock size={10} />{fmtDate(s.expired_at)} 到期
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Plan cards */}
      <div>
        <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '14px', textTransform: 'uppercase', letterSpacing: '0.06em' }}>可用套餐</p>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))', gap: '16px' }}>
          {plans.map(p => {
            const cfg = typeConfig[p.type] || typeConfig.combo;
            const isActive = activeSubs.some(s => s.plan_id === p.id);
            return (
              <div key={p.id} style={{ borderRadius: '20px', background: 'var(--bg-surface)', border: `1px solid ${isActive ? 'rgba(52,211,153,0.4)' : 'var(--border-color)'}`, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
                <div style={{ height: '4px', background: cfg.gradient }} />
                <div style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '14px', flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <div style={{ width: '44px', height: '44px', borderRadius: '14px', background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', flexShrink: 0 }}>{cfg.icon}</div>
                    <div style={{ minWidth: 0 }}>
                      <p style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>{p.name}</p>
                      <p style={{ fontSize: '11px', color: 'var(--text-secondary)', margin: '2px 0 0' }}>{p.duration_days > 0 ? `有效期 ${p.duration_days} 天` : '永久有效'}</p>
                    </div>
                    {isActive && <CheckCircle size={16} color="#34d399" style={{ marginLeft: 'auto', flexShrink: 0 }} />}
                  </div>
                  {p.description && <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0, lineHeight: 1.6 }}>{p.description}</p>}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '7px' }}>
                    {(p.type === 'group' || p.type === 'combo') && p.group_name && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                        <Users size={12} color={cfg.accent} /><span>切换至 <strong style={{ color: 'var(--text-primary)' }}>{p.group_name}</strong> 分组</span>
                      </div>
                    )}
                    {(p.type === 'quota' || p.type === 'combo') && p.quota > 0 && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                        <Package size={12} color={cfg.accent} /><span>赠送 <strong style={{ color: 'var(--text-primary)' }}>${(p.quota / 500000).toFixed(2)}</strong> 额度</span>
                      </div>
                    )}
                    {(p.type === 'rpm' || p.type === 'combo') && p.rpm > 0 && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: '7px', fontSize: '12px', color: 'var(--text-secondary)' }}>
                        <Zap size={12} color={cfg.accent} /><span>RPM 提升至 <strong style={{ color: 'var(--text-primary)' }}>{p.rpm}</strong></span>
                      </div>
                    )}
                  </div>
                  <div style={{ marginTop: 'auto', display: 'flex', alignItems: 'center', justifyContent: 'space-between', paddingTop: '4px' }}>
                    <div>
                      <span style={{ fontSize: '26px', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1 }}>{p.price === 0 ? '免费' : `¥${p.price.toFixed(2)}`}</span>
                      {p.price > 0 && p.duration_days > 0 && <span style={{ fontSize: '11px', color: 'var(--text-faint)', marginLeft: '4px' }}>/ {p.duration_days}天</span>}
                    </div>
                    <Button size="sm" onPress={() => setSelectedPlan(p)} style={{ borderRadius: '10px', background: cfg.gradient, color: '#fff', fontWeight: 600, border: 'none' }}>
                      {isActive ? '续订' : '订阅'}
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
          {plans.length === 0 && <div style={{ gridColumn: '1/-1', textAlign: 'center', padding: '60px', color: 'var(--text-faint)', fontSize: '14px' }}>暂无可用套餐</div>}
        </div>
      </div>

      {mySubs.length > 0 && (
        <div>
          <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '14px', textTransform: 'uppercase', letterSpacing: '0.06em' }}>订阅记录</p>
          <div style={{ borderRadius: '16px', border: '1px solid var(--border-color)', overflow: 'hidden' }}>
            {mySubs.map((s, i) => {
              const st = statusLabel[s.status] || { text: '未知', color: '#9ca3af' };
              const cfg = typeConfig[s.plan?.type] || typeConfig.combo;
              return (
                <div key={s.id} style={{ display: 'flex', alignItems: 'center', gap: '14px', padding: '14px 18px', borderBottom: i < mySubs.length - 1 ? '1px solid var(--border-color)' : undefined, background: 'var(--bg-surface)' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '10px', flexShrink: 0, background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff' }}>
                    <span style={{ transform: 'scale(0.75)', display: 'flex' }}>{cfg.icon}</span>
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>{s.plan?.name || `套餐 #${s.plan_id}`}</p>
                    <p style={{ fontSize: '11px', color: 'var(--text-faint)', margin: '2px 0 0' }}>{new Date(s.created_at * 1000).toLocaleDateString('zh-CN')} 购买</p>
                  </div>
                  {s.expired_at > 0 && <span style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: '4px' }}><Clock size={11} />{fmtDate(s.expired_at)}</span>}
                  <span style={{ fontSize: '11px', fontWeight: 700, color: st.color, padding: '3px 10px', borderRadius: '99px', background: `${st.color}18` }}>{st.text}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Payment modal */}
      {selectedPlan && (() => {
        const cfg = typeConfig[selectedPlan.type] || typeConfig.combo;
        return (
          <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '16px' }}
            onClick={e => { if (e.target === e.currentTarget) setSelectedPlan(null); }}>
            <div style={{ background: 'var(--bg-surface)', borderRadius: '24px', border: '1px solid var(--border-color)', width: '100%', maxWidth: '400px', overflow: 'hidden' }}>
              {/* Header */}
              <div style={{ height: '5px', background: cfg.gradient }} />
              <div style={{ padding: '24px 24px 0' }}>
                <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <div style={{ width: '44px', height: '44px', borderRadius: '14px', background: cfg.gradient, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', flexShrink: 0 }}>
                      {cfg.icon}
                    </div>
                    <div>
                      <p style={{ fontSize: '16px', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>{selectedPlan.name}</p>
                      <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: '2px 0 0' }}>
                        {selectedPlan.duration_days > 0 ? `有效期 ${selectedPlan.duration_days} 天` : '永久有效'}
                      </p>
                    </div>
                  </div>
                  <button onClick={() => setSelectedPlan(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-faint)', padding: '4px' }}>
                    <X size={18} />
                  </button>
                </div>

                {/* Price */}
                <div style={{ padding: '14px 16px', borderRadius: '14px', background: 'var(--bg-elevated)', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>应付金额</span>
                  <span style={{ fontSize: '28px', fontWeight: 800, color: 'var(--text-primary)' }}>
                    {selectedPlan.price === 0 ? '免费' : `¥${selectedPlan.price.toFixed(2)}`}
                  </span>
                </div>

                {/* Payment method */}
                <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-secondary)', marginBottom: '10px' }}>选择支付方式</p>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '20px' }}>
                  {([
                    { key: 'alipay', label: '支付宝', color: '#1677ff', bg: 'rgba(22,119,255,0.08)' },
                    { key: 'wxpay',  label: '微信支付', color: '#07c160', bg: 'rgba(7,193,96,0.08)' },
                  ] as const).map(m => (
                    <button key={m.key} onClick={() => setPayType(m.key)}
                      style={{
                        padding: '12px', borderRadius: '12px', cursor: 'pointer', textAlign: 'center',
                        border: `2px solid ${payType === m.key ? m.color : 'var(--border-color)'}`,
                        background: payType === m.key ? m.bg : 'var(--bg-elevated)',
                        color: payType === m.key ? m.color : 'var(--text-secondary)',
                        fontWeight: 600, fontSize: '13px', transition: 'all 0.15s',
                      }}>
                      {m.label}
                    </button>
                  ))}
                </div>
              </div>

              <div style={{ padding: '0 24px 24px', display: 'flex', gap: '10px' }}>
                <Button variant="flat" onPress={() => setSelectedPlan(null)} style={{ flex: 1, borderRadius: '12px' }}>取消</Button>
                <Button isLoading={paying} onPress={subscribe}
                  style={{ flex: 2, borderRadius: '12px', background: cfg.gradient, color: '#fff', fontWeight: 700, border: 'none' }}>
                  确认支付
                </Button>
              </div>
            </div>
          </div>
        );
      })()}
    </div>
  );
}
