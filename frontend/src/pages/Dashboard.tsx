import { useState, useEffect } from 'react';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Progress,
} from '@heroui/react';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts';
import { Activity, CreditCard, Key, Server, RefreshCw, CalendarCheck } from 'lucide-react';
import { useAuthStore } from '../store/auth';
import { useNavigate } from 'react-router-dom';

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
      if (res.ok) setCheckedIn(data.checked_in);
    } catch (e) { console.error(e); }
  };

  const handleCheckin = async () => {
    setCheckinLoading(true);
    try {
      let body: any = {};

      // Handle captcha if required
      if (checkinCaptcha) {
        const useGeetest = captchaProvider === 'geetest' && geetestEnabled;
        const useTurnstile = captchaProvider === 'turnstile' ? turnstileEnabled : (!captchaProvider && turnstileEnabled);

        if (useGeetest) {
          const geetestData = await new Promise<any>((resolve) => {
            if (!(window as any).initGeetest4) { resolve(null); return; }
            (window as any).initGeetest4({
              captchaId: geetestId,
              product: 'bind',
            }, (captchaObj: any) => {
              captchaObj.onSuccess(() => resolve(captchaObj.getValidate()));
              captchaObj.onError(() => resolve(null));
              captchaObj.onClose(() => resolve(null));
              captchaObj.showCaptcha();
            });
          });
          if (!geetestData) {
            setCheckinLoading(false);
            return;
          }
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
          if (!turnstileToken) {
            alert('请完成人机验证');
            setCheckinLoading(false);
            return;
          }
          body.turnstile = turnstileToken;
        }
      }

      const res = await fetch('/api/user/checkin', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (res.ok) {
        setCheckedIn(true);
        alert(`签到成功！获得 $${(data.reward / 500000).toFixed(4)} 额度`);
        fetchDashboard();
      } else {
        alert(data.error || '签到失败');
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
      if (res.ok) {
        setStats(data.stats);
        setUserInfo(data.user);
        setChartData(data.chart || []);
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
    // Fetch captcha config for check-in
    fetch('/api/system/status')
      .then(res => res.json())
      .then(data => {
        if (data.options) {
          if (data.options.checkin_captcha === 'true') setCheckinCaptcha(true);
          if (data.options.captcha_provider) setCaptchaProvider(data.options.captcha_provider);
          if (data.options.turnstile_check_enabled === 'true') {
            setTurnstileEnabled(true);
            setTurnstileSiteKey(data.options.turnstile_site_key || '');
          }
          if (data.options.geetest_enabled === 'true') {
            setGeetestEnabled(true);
            setGeetestId(data.options.geetest_id || '');
          }
        }
      });
  }, [token]);

  return (
    <div className="p-6 space-y-6 max-w-[1400px] mx-auto">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">仪表盘</h1>
          <p className="text-default-500">欢迎回来, {userInfo?.username || user?.username}</p>
        </div>
        <div className="flex gap-2">
            <Button isIconOnly variant="flat" onPress={fetchDashboard} isLoading={loading}>
                <RefreshCw size={20} />
            </Button>
            <Button color="primary" variant="shadow" onPress={() => navigate('/topup')}>
            充值余额
            </Button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">总调用次数</span>
              <span className="text-2xl font-bold">{stats?.request_count || 0}</span>
            </div>
            <div className="p-2 bg-primary/10 rounded-lg text-primary">
              <Activity size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
            <span className="text-default-400">历史累计</span>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">剩余额度</span>
              <span className="text-2xl font-bold text-success">
                ${userInfo?.quota ? (userInfo.quota - (userInfo.used_quota || 0)).toFixed(2) : '0.00'}
              </span>
            </div>
            <div className="p-2 bg-success/10 rounded-lg text-success">
              <CreditCard size={24} />
            </div>
          </div>
          <Progress 
            size="sm" 
            value={userInfo?.quota ? ((userInfo.quota - (userInfo.used_quota || 0)) / userInfo.quota) * 100 : 0} 
            color="success" 
            className="mt-4"
          />
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">活跃令牌</span>
              <span className="text-2xl font-bold">{stats?.token_count || 0}</span>
            </div>
            <div className="p-2 bg-warning/10 rounded-lg text-warning">
              <Key size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
            <Button size="sm" variant="light" onPress={() => navigate('/token')} className="px-0 h-6">管理令牌</Button>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex justify-between items-start">
            <div className="flex flex-col gap-1">
              <span className="text-small text-default-500">已用额度</span>
              <span className="text-2xl font-bold text-danger">
                 ${(userInfo?.used_quota || 0).toFixed(2)}
              </span>
            </div>
            <div className="p-2 bg-danger/10 rounded-lg text-danger">
              <Server size={24} />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-2 text-small">
             <span className="text-default-400">历史累计消耗</span>
          </div>
        </Card>
      </div>

      {/* 签到卡片 */}
      <Card className="p-4">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-secondary/10 rounded-lg text-secondary">
              <CalendarCheck size={24} />
            </div>
            <div>
              <h3 className="font-semibold">每日签到</h3>
              <p className="text-small text-default-500">
                {checkedIn ? '今日已签到，明天再来吧' : '签到领取随机额度奖励'}
              </p>
            </div>
          </div>
          <Button
            color={checkedIn ? 'default' : 'secondary'}
            variant={checkedIn ? 'flat' : 'shadow'}
            isDisabled={checkedIn}
            isLoading={checkinLoading}
            onPress={handleCheckin}
          >
            {checkedIn ? '已签到' : '立即签到'}
          </Button>
        </div>
      </Card>

      {/* 图表区域 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="p-4 lg:col-span-2">
          <CardHeader>
            <h3 className="text-lg font-semibold">近7日消耗统计 ($)</h3>
          </CardHeader>
          <CardBody className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorUsage" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#17C964" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#17C964" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#333" opacity={0.1} />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip 
                  cursor={{fill: 'transparent'}}
                  contentStyle={{ 
                    backgroundColor: 'var(--heroui-content1)', 
                    borderRadius: '12px',
                    border: 'none',
                  }} 
                />
                <Area 
                  type="monotone"
                  dataKey="usage" 
                  stroke="#17C964" 
                  fillOpacity={1}
                  fill="url(#colorUsage)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
