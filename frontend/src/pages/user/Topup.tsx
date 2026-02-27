import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import { Input, Button } from '../../components/ui';
import { Gift, Wallet, TrendingUp, Zap, CreditCard } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

export default function TopupPage() {
  const [code, setCode] = useState('');
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);
  const [paying, setPaying] = useState(false);
  const [enableTopup, setEnableTopup] = useState(false);
  const [minTopup, setMinTopup] = useState(0);
  const { token, user, updateUser } = useAuthStore();

  const fetchUser = async () => {
    try {
      const res = await fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) updateUser(data.data);
    } catch (e) { console.error(e); }
  };

  useEffect(() => {
    fetchUser();
    fetch('/api/system/status')
      .then(r => r.json())
      .then(d => {
        const payload = d.code === 0 ? d.data : d;
        setEnableTopup(payload.options?.enable_topup === 'true');
        const min = parseFloat(payload.options?.min_topup || '0');
        if (min > 0) setMinTopup(min);
      })
      .catch(() => {});
  }, []);

  const handleRedeem = async () => {
    if (!code.trim()) return;
    setLoading(true);
    try {
      const res = await fetch('/api/user/redemption', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ code: code.trim() }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success(`兑换成功！增加额度 $${(data.data.quota / 500000).toFixed(4)}`);
        setCode('');
        fetchUser();
      } else {
        toast.error(data.msg || '兑换失败');
      }
    } catch {
      toast.error('请求失败，请检查网络');
    } finally {
      setLoading(false);
    }
  };

  const handleTopup = async () => {
    const amt = parseFloat(amount);
    if (!amt || amt <= 0) { toast.error('请输入有效金额'); return; }
    if (minTopup > 0 && amt < minTopup) { toast.error(`最低充值金额为 $${minTopup}`); return; }
    setPaying(true);
    try {
      const res = await fetch('/api/payment/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ amount: amt }),
      });
      const data = await res.json();
      if (data.code === 0 && data.data?.pay_url) {
        window.location.href = data.data.pay_url;
      } else {
        toast.error(data.msg || '创建订单失败');
      }
    } catch {
      toast.error('请求失败，请检查网络');
    } finally {
      setPaying(false);
    }
  };

  const quota = user?.quota ?? 0;
  const usedQuota = (user as any)?.used_quota ?? 0;
  const totalEver = quota + usedQuota;
  const usedPct = totalEver > 0 ? Math.min((usedQuota / totalEver) * 100, 100) : 0;
  const usd = (quota / 500000).toFixed(4);
  const usedUsd = (usedQuota / 500000).toFixed(4);

  return (
    <div className="space-y-6 max-w-2xl mx-auto">
      <PageHeader title="充值中心" description="管理您的账户余额与兑换码" />

      {/* 余额主卡 */}
      <div style={{
        borderRadius: '20px',
        background: 'linear-gradient(135deg, rgba(124,58,237,0.18) 0%, rgba(8,145,178,0.14) 100%)',
        border: '1px solid rgba(124,58,237,0.25)',
        padding: '28px',
        position: 'relative',
        overflow: 'hidden',
      }}>
        <div style={{ position: 'absolute', top: '-40px', right: '-40px', width: '160px', height: '160px', borderRadius: '50%', background: 'radial-gradient(circle, rgba(124,58,237,0.2) 0%, transparent 70%)', pointerEvents: 'none' }} />
        <div style={{ position: 'absolute', bottom: '-30px', left: '20%', width: '120px', height: '120px', borderRadius: '50%', background: 'radial-gradient(circle, rgba(8,145,178,0.15) 0%, transparent 70%)', pointerEvents: 'none' }} />

        <div className="flex items-start justify-between mb-6">
          <div className="flex items-center gap-3">
            <div style={{ width: '46px', height: '46px', borderRadius: '14px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 16px rgba(124,58,237,0.35)' }}>
              <Wallet size={22} color="white" />
            </div>
            <div>
              <p style={{ fontSize: '13px', color: 'var(--text-secondary)', margin: 0 }}>账户余额</p>
              <p style={{ fontSize: '11px', color: 'var(--text-faint)', margin: 0 }}>1 USD = 500,000 额度</p>
            </div>
          </div>
          <div style={{ padding: '4px 10px', borderRadius: '20px', background: 'rgba(52,211,153,0.15)', border: '1px solid rgba(52,211,153,0.3)', display: 'flex', alignItems: 'center', gap: '5px' }}>
            <Zap size={11} color="#34d399" />
            <span style={{ fontSize: '11px', color: '#34d399', fontWeight: 600 }}>活跃</span>
          </div>
        </div>

        <div className="mb-6">
          <div className="flex items-baseline gap-2">
            <span style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '2px' }}>$</span>
            <span style={{ fontSize: '48px', fontWeight: 800, color: 'var(--text-primary)', lineHeight: 1, letterSpacing: '-1px' }}>{usd}</span>
          </div>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginTop: '4px' }}>
            已消耗 <span style={{ color: '#f87171', fontWeight: 600 }}>${usedUsd}</span>
          </p>
        </div>

        <div>
          <div className="flex justify-between mb-2">
            <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>额度使用率</span>
            <span style={{ fontSize: '12px', color: 'var(--text-secondary)', fontWeight: 600 }}>{usedPct.toFixed(1)}%</span>
          </div>
          <div style={{ height: '6px', borderRadius: '99px', background: 'var(--bg-elevated)', overflow: 'hidden' }}>
            <div style={{ height: '100%', borderRadius: '99px', width: `${usedPct}%`, background: usedPct > 80 ? 'linear-gradient(90deg, #f87171, #ef4444)' : 'linear-gradient(90deg, var(--accent-primary), var(--accent-cosmic))', transition: 'width 0.6s ease' }} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 mt-5">
          {[
            { label: '剩余额度', value: quota.toLocaleString(), color: 'var(--accent-cosmic)', icon: <Wallet size={14} /> },
            { label: '累计消耗', value: usedQuota.toLocaleString(), color: '#f87171', icon: <TrendingUp size={14} /> },
          ].map(item => (
            <div key={item.label} style={{ padding: '12px 14px', borderRadius: '12px', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', backdropFilter: 'blur(8px)' }}>
              <div className="flex items-center gap-1.5 mb-1" style={{ color: 'var(--text-secondary)' }}>
                {item.icon}
                <span style={{ fontSize: '11px' }}>{item.label}</span>
              </div>
              <p style={{ fontSize: '15px', fontWeight: 700, color: item.color, margin: 0 }}>{item.value}</p>
            </div>
          ))}
        </div>
      </div>

      {/* 在线充值 */}
      {enableTopup && (
        <div style={{ borderRadius: '20px', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', padding: '24px' }}>
          <div className="flex items-center gap-3 mb-5">
            <div style={{ width: '40px', height: '40px', borderRadius: '12px', background: 'rgba(8,145,178,0.12)', border: '1px solid rgba(8,145,178,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <CreditCard size={18} style={{ color: 'var(--accent-cosmic)' }} />
            </div>
            <div>
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>在线充值</p>
              <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>
                通过易支付充值{minTopup > 0 ? `，最低 $${minTopup}` : ''}
              </p>
            </div>
          </div>
          <div style={{ display: 'flex', gap: '10px' }}>
            <Input
              placeholder={minTopup > 0 ? `最低 $${minTopup}` : '输入充值金额（USD）'}
              value={amount}
              onValueChange={setAmount}
              type="number"
              startContent={<span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>$</span>}
              onKeyDown={e => { if (e.key === 'Enter') handleTopup(); }}
              style={{ flex: 1 }}
              size="lg"
            />
            <Button
              color="primary"
              size="lg"
              onPress={handleTopup}
              isLoading={paying}
              isDisabled={!amount}
              style={{ borderRadius: '12px', minWidth: '96px', background: 'linear-gradient(135deg, var(--accent-cosmic), var(--accent-primary))' }}
            >
              立即充值
            </Button>
          </div>
        </div>
      )}

      {/* 兑换码 */}
      <div style={{ borderRadius: '20px', background: 'var(--bg-surface)', border: '1px solid var(--border-color)', padding: '24px' }}>
        <div className="flex items-center gap-3 mb-5">
          <div style={{ width: '40px', height: '40px', borderRadius: '12px', background: 'rgba(124,58,237,0.12)', border: '1px solid rgba(124,58,237,0.2)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Gift size={18} style={{ color: 'var(--accent-primary)' }} />
          </div>
          <div>
            <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)', margin: 0 }}>兑换码充值</p>
            <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>输入兑换码，额度立即到账</p>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <Input
            placeholder="请输入兑换码..."
            value={code}
            onValueChange={setCode}
            onKeyDown={e => { if (e.key === 'Enter') handleRedeem(); }}
            startContent={<Gift size={15} className="text-default-400" />}
            style={{ flex: 1 }}
            size="lg"
          />
          <Button
            color="primary"
            size="lg"
            onPress={handleRedeem}
            isLoading={loading}
            isDisabled={!code.trim()}
            style={{ borderRadius: '12px', minWidth: '96px', background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))' }}
          >
            兑换
          </Button>
        </div>
        <p style={{ fontSize: '12px', color: 'var(--text-faint)', marginTop: '10px' }}>
          兑换码区分大小写，请确保输入正确 · 按 Enter 快速兑换
        </p>
      </div>
    </div>
  );
}
