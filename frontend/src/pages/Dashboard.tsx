import { useState, useEffect } from 'react';
import { Button, Progress } from '../components/ui';
import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { Activity, CreditCard, Key, Server, RefreshCw } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useNavigate } from 'react-router-dom';
import { toast } from '../store/toast';
import PageHeader from '../components/PageHeader';
import StatCard from '../components/StatCard';

interface UserStats {
  token_count: number;
  request_count: number;
}

interface UserInfo {
  username: string;
  quota: number;
  used_quota: number;
  role: number;
}

interface ChartData {
  date: string;
  usage: number;
}

export default function Dashboard() {
  const { user, token } = useAuthStore();
  const navigate = useNavigate();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [loading, setLoading] = useState(false);
  const [checkedIn, setCheckedIn] = useState(false);
  const [checkinLoading, setCheckinLoading] = useState(false);
  const [checkinCaptcha, setCheckinCaptcha] = useState(false);
  const [captchaProvider, setCaptchaProvider] = useState('');
  const [geetestEnabled, setGeetestEnabled] = useState(false);
  const [geetestId, setGeetestId] = useState('');
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');

  const fetchCheckinStatus = async () => {
    try {
      const res = await fetch('/api/user/checkin', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) setCheckedIn(data.data.checked_in);
    } catch (e) { console.error(e); }
  };

  const handleCheckin = async () => {
    setCheckinLoading(true);
    try {
      let body: any = {};

      if (checkinCaptcha) {
        const useGeetest = captchaProvider === 'geetest' && geetestEnabled;
        const useTurnstile = captchaProvider === 'turnstile' ? turnstileEnabled : (!captchaProvider && turnstileEnabled);

        if (useGeetest) {
          const geetestData = await new Promise<any>((resolve) => {
            if (!(window as any).initGeetest4) { resolve(null); return; }
            (window as any).initGeetest4({ captchaId: geetestId, product: 'bind' }, (captchaObj: any) => {
              captchaObj.onSuccess(() => resolve(captchaObj.getValidate()));
              captchaObj.onError(() => resolve(null));
              captchaObj.onClose(() => resolve(null));
              captchaObj.showCaptcha();
            });
          });
          if (!geetestData) { setCheckinLoading(false); return; }
          body.geetest = geetestData;
        } else if (useTurnstile) {
          const turnstileToken = await new Promise<string>((resolve) => {
            if (!(window as any).turnstile) { resolve(''); return; }
            (window as any).turnstile.render('#checkin-turnstile', {
              sitekey: turnstileSiteKey,
              callback: (t: string) => resolve(t),
              'error-callback': () => resolve(''),
            });
          });
          if (!turnstileToken) { toast.warning('请完成人机验证'); setCheckinLoading(false); return; }
          body.turnstile = turnstileToken;
        }
      }

      const res = await fetch('/api/user/checkin', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.code === 0) {
        setCheckedIn(true);
        toast.success(`签到成功！获得 $${(data.data.reward / 500000).toFixed(4)} 额度`);
        fetchDashboard();
      } else {
        toast.error(data.msg || '签到失败');
      }
    } catch (e) { console.error(e); }
    finally { setCheckinLoading(false); }
  };

  const fetchDashboard = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/dashboard', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setStats(data.data.stats);
        setUserInfo(data.data.user);
        setChartData(data.data.chart || []);
      }
    } catch (error) {
      console.error('Failed to fetch dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) {
      fetchDashboard();
      fetchCheckinStatus();
    }
    fetch('/api/system/status')
      .then(res => res.json())
      .then(data => {
        const payload = data.code === 0 ? data.data : data;
        if (payload.options) {
          if (payload.options.checkin_captcha === 'true') setCheckinCaptcha(true);
          if (payload.options.captcha_provider) setCaptchaProvider(payload.options.captcha_provider);
          if (payload.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(payload.options.turnstile_site_key || '');
          }
          if (payload.options.geetest_enabled === 'true') {
            setGeetestEnabled(true);
            setGeetestId(payload.options.geetest_id || '');
          }
        }
      });
  }, [token]);

  const remainingRaw = userInfo ? (userInfo.quota - (userInfo.used_quota || 0)) : 0;
  const remainingUsd = (remainingRaw / 500000).toFixed(4);
  const usedUsd = ((userInfo?.used_quota || 0) / 500000).toFixed(4);
  const remainingPct = userInfo?.quota ? Math.min((remainingRaw / userInfo.quota) * 100, 100) : 0;

  return (
    <div className="space-y-6 max-w-[1400px] mx-auto animate-fade-in-up">
      <PageHeader
        title="仪表盘"
        description={`欢迎回来，${userInfo?.username || user?.username} ✿`}
        actions={
          <>
            <Button isIconOnly variant="flat" onPress={fetchDashboard} isLoading={loading}
              style={{ background: 'var(--bg-elevated)', color: 'var(--accent-primary)', borderRadius: '12px', border: '1px solid var(--border-color)' }}>
              <RefreshCw size={18} />
            </Button>
            <Button onPress={() => navigate('/topup')}
              style={{ background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))', color: 'white', borderRadius: '12px' }}>
              充值余额
            </Button>
          </>
        }
      />

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="总调用次数"
          value={stats?.request_count || 0}
          icon={<Activity size={22} style={{ color: 'var(--accent-primary)' }} />}
          iconBg="rgba(124,58,237,0.12)"
          footer="历史累计"
        />
        <StatCard
          title="剩余额度"
          value={<span style={{ color: 'var(--accent-cosmic)' }}>${remainingUsd}</span>}
          icon={<CreditCard size={22} style={{ color: 'var(--accent-cosmic)' }} />}
          iconBg="rgba(8,145,178,0.12)"
          footer={
            <Progress size="sm" value={remainingPct} color="success" aria-label="剩余额度" />
          }
        />
        <StatCard
          title="活跃令牌"
          value={<span style={{ color: 'var(--accent-star)' }}>{stats?.token_count || 0}</span>}
          icon={<Key size={22} style={{ color: 'var(--accent-star)' }} />}
          iconBg="rgba(217,119,6,0.12)"
          footer={
            <button onClick={() => navigate('/token')} className="transition-colors" style={{ color: 'var(--accent-star)' }}>
              管理令牌 →
            </button>
          }
        />
        <StatCard
          title="已用额度"
          value={<span style={{ color: '#f87171' }}>${usedUsd}</span>}
          icon={<Server size={22} style={{ color: '#f87171' }} />}
          iconBg="rgba(248,113,113,0.12)"
          footer="历史累计消耗"
        />
      </div>

      {/* 签到卡片 */}
      <div className="akasha-card p-5">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl text-2xl" style={{ background: 'var(--bg-elevated)' }}>
              {checkedIn ? '✅' : '🎁'}
            </div>
            <div>
              <h3 className="font-semibold" style={{ color: 'var(--text-primary)' }}>每日签到</h3>
              <p className="text-xs mt-0.5" style={{ color: 'var(--text-secondary)' }}>
                {checkedIn ? '今日已签到，明天再来吧 ✦' : '签到领取随机额度奖励'}
              </p>
            </div>
          </div>
          <Button
            isDisabled={checkedIn}
            isLoading={checkinLoading}
            onPress={handleCheckin}
            style={checkedIn ? {
              background: 'var(--bg-elevated)',
              color: 'var(--text-secondary)',
              borderRadius: '12px',
            } : {
              background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
              color: 'white',
              borderRadius: '12px',
            }}
          >
            {checkedIn ? '已签到' : '立即签到'}
          </Button>
        </div>
      </div>

      {/* 图表 */}
      <div className="akasha-card p-5">
        <h3 className="text-base font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>近7日消耗统计 (USD)</h3>
        <div className="h-[280px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorUsage" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--accent-primary)" stopOpacity={0.4} />
                  <stop offset="95%" stopColor="var(--accent-cosmic)" stopOpacity={0.05} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border-color)" />
              <XAxis dataKey="date" tick={{ fill: 'var(--text-secondary)', fontSize: 12 }} />
              <YAxis
                tick={{ fill: 'var(--text-secondary)', fontSize: 12 }}
                tickFormatter={(v: number) => `$${(v / 500000).toFixed(4)}`}
              />
              <Tooltip
                cursor={{ fill: 'var(--bg-elevated)' }}
                contentStyle={{
                  background: 'var(--bg-surface)',
                  borderRadius: '12px',
                  border: '1px solid var(--border-color)',
                  fontSize: '13px',
                }}
                formatter={(value: number) => [`$${(value / 500000).toFixed(6)}`, '消耗']}
              />
              <Area type="monotone" dataKey="usage" stroke="var(--accent-primary)" strokeWidth={2} fillOpacity={1} fill="url(#colorUsage)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}
