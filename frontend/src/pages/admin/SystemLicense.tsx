// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 站长在这里用 GitHub Device Flow 完成组织成员授权，验证通过后系统才会解除拦截喵～
import { useState, useEffect, useRef, useCallback } from 'react';
import { Card, CardBody, Button, Chip } from '../../components/ui';
import { ShieldAlert, ShieldCheck, Unlink, Smartphone, Copy, ExternalLink, Loader2 } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { confirm } from '../../store/confirm';

interface LicenseStatus {
  feature_enabled: boolean;
  authorized: boolean;
  github_login: string;
  bound_at: number;
  last_check: number;
  org: string;
}

interface DeviceCodeInfo {
  device_code: string;
  user_code: string;
  verification_uri: string;
  expires_in: number;
  interval: number;
}

export default function SystemLicense() {
  const { token } = useAuthStore();
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [unbinding, setUnbinding] = useState(false);

  // Device Flow 状态
  const [deviceCode, setDeviceCode] = useState<DeviceCodeInfo | null>(null);
  const [polling, setPolling] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const expiredRef = useRef(false); // 防止过期后继续轮询

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/system-license/status', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) setStatus(data.data);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  }, [token]);

  useEffect(() => {
    fetchStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 清理定时器
  const clearTimers = useCallback(() => {
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
    if (countdownRef.current) {
      clearInterval(countdownRef.current);
      countdownRef.current = null;
    }
  }, []);

  useEffect(() => () => clearTimers(), [clearTimers]);

  // 发起 Device Flow
  const startDeviceFlow = async () => {
    clearTimers();
    expiredRef.current = false;
    setPolling(true);
    setDeviceCode(null);
    try {
      const res = await fetch('/api/system-license/device-code', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      });
      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '发起授权失败');
        setPolling(false);
        return;
      }
      const info: DeviceCodeInfo = data.data;
      setDeviceCode(info);
      setCountdown(info.expires_in);

      // 倒计时
      countdownRef.current = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            expiredRef.current = true;
            clearTimers();
            setPolling(false);
            toast.error('设备码已过期，请重新发起');
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      // 开始轮询
      startPolling(info.device_code, info.interval);
    } catch (e) {
      console.error(e);
      toast.error('请求失败');
      setPolling(false);
    }
  };

  // 轮询授权结果
  const startPolling = (code: string, interval: number) => {
    const poll = async () => {
      if (expiredRef.current) return;
      try {
        const res = await fetch('/api/system-license/poll', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ device_code: code }),
        });
        const data = await res.json();
        if (data.code !== 0) {
          clearTimers();
          setPolling(false);
          toast.error(data.msg || '授权失败');
          return;
        }
        if (data.data.completed) {
          clearTimers();
          setPolling(false);
          toast.success('系统授权成功');
          setDeviceCode(null);
          fetchStatus();
        } else {
          // 继续轮询
          pollTimerRef.current = setTimeout(poll, (interval || 5) * 1000);
        }
      } catch (e) {
        console.error(e);
        // 网络错误，继续重试
        pollTimerRef.current = setTimeout(poll, (interval || 5) * 1000);
      }
    };
    // 首次等待 interval 秒
    pollTimerRef.current = setTimeout(poll, (interval || 5) * 1000);
  };

  // 取消授权
  const cancelDeviceFlow = () => {
    clearTimers();
    expiredRef.current = true;
    setPolling(false);
    setDeviceCode(null);
    setCountdown(0);
  };

  // 复制用户码
  const copyUserCode = () => {
    if (deviceCode?.user_code) {
      navigator.clipboard.writeText(deviceCode.user_code);
      toast.success('已复制设备码');
    }
  };

  const handleUnbind = async () => {
    if (!await confirm({
      title: '解绑系统授权',
      message: '解绑后系统会立即恢复拦截，需要重新授权才能继续使用，确定要解绑吗？',
      danger: true,
    })) return;
    setUnbinding(true);
    try {
      const res = await fetch('/api/system-license/unbind', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('已解绑');
        fetchStatus();
      } else {
        toast.error(data.msg || '解绑失败');
      }
    } catch (e) {
      console.error(e);
      toast.error('解绑请求失败');
    } finally {
      setUnbinding(false);
    }
  };

  if (loading && !status) {
    return (
      <div className="max-w-lg mx-auto">
        <Card><CardBody><p className="text-sm text-center py-6" style={{ color: 'var(--text-secondary)' }}>加载中...</p></CardBody></Card>
      </div>
    );
  }

  if (!status?.feature_enabled) {
    return (
      <div className="max-w-lg mx-auto">
        <Card>
          <CardBody className="text-center py-12 space-y-3">
            <div
              className="w-14 h-14 mx-auto rounded-2xl flex items-center justify-center"
              style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
            >
              <ShieldAlert className="w-6 h-6" style={{ color: 'var(--text-secondary)', opacity: 0.6 }} />
            </div>
            <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>本部署未启用系统授权功能</p>
          </CardBody>
        </Card>
      </div>
    );
  }

  const authorized = status.authorized;
  const accent = authorized ? '#10b981' : 'var(--accent-star)';

  // 格式化倒计时
  const fmtCountdown = (s: number) => {
    const m = Math.floor(s / 60);
    const sec = s % 60;
    return `${m}:${sec.toString().padStart(2, '0')}`;
  };

  return (
    <div className="max-w-lg mx-auto">
      <Card
        className="overflow-hidden"
        style={{
          background: authorized
            ? 'linear-gradient(180deg, rgba(16,185,129,0.07), transparent 55%)'
            : 'linear-gradient(180deg, rgba(251,191,36,0.09), transparent 55%)',
        }}
      >
        <CardBody className="space-y-5 p-8 text-center items-center flex flex-col">
          <div
            className="w-16 h-16 rounded-2xl flex items-center justify-center"
            style={{
              background: authorized ? 'rgba(16,185,129,0.12)' : 'rgba(251,191,36,0.15)',
              boxShadow: `0 4px 20px ${authorized ? 'rgba(16,185,129,0.18)' : 'rgba(251,191,36,0.22)'}`,
            }}
          >
            {authorized
              ? <ShieldCheck className="w-7 h-7" style={{ color: accent }} />
              : <ShieldAlert className="w-7 h-7" style={{ color: accent }} />}
          </div>

          <div className="flex items-center justify-center gap-2 flex-wrap">
            <h3 className="text-lg font-bold" style={{ color: 'var(--text-primary)' }}>
              {authorized ? '系统已授权' : '系统未授权'}
            </h3>
            <Chip size="sm" color={authorized ? 'success' : 'warning'} variant="flat">
              {authorized ? '已激活' : '功能受限'}
            </Chip>
          </div>

          <p className="text-sm max-w-sm" style={{ color: 'var(--text-secondary)' }}>
            {authorized
              ? '本实例已通过 GitHub 组织成员验证，功能不受限制'
              : <>需要使用 GitHub 账号登录，并且是 <Chip size="sm" variant="bordered" className="mx-0.5">{status.org}</Chip> 组织成员，验证通过后系统才会解除拦截</>}
          </p>

          {authorized ? (
            <>
              <div
                className="w-full rounded-xl p-4 grid grid-cols-1 sm:grid-cols-3 gap-3 text-left"
                style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
              >
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.7 }}>绑定账号</p>
                  <p className="text-sm font-mono font-medium mt-0.5" style={{ color: 'var(--text-primary)' }}>{status.github_login}</p>
                </div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.7 }}>绑定时间</p>
                  <p className="text-sm mt-0.5" style={{ color: 'var(--text-primary)' }}>
                    {status.bound_at > 0 ? new Date(status.bound_at * 1000).toLocaleString() : '-'}
                  </p>
                </div>
                <div>
                  <p className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.7 }}>上次复核</p>
                  <p className="text-sm mt-0.5" style={{ color: 'var(--text-primary)' }}>
                    {status.last_check > 0 ? new Date(status.last_check * 1000).toLocaleString() : '-'}
                  </p>
                </div>
              </div>
              <Button
                color="danger" variant="flat" size="sm"
                startContent={<Unlink size={14} />}
                isLoading={unbinding}
                onPress={handleUnbind}
              >
                解绑
              </Button>
            </>
          ) : deviceCode ? (
            // Device Flow 进行中 — 展示设备码和二维码
            <div className="w-full space-y-4">
              <div
                className="w-full rounded-xl p-5 space-y-3"
                style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
              >
                <div className="flex items-center justify-center gap-2">
                  <Smartphone className="w-4 h-4" style={{ color: accent }} />
                  <span className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>在 GitHub 输入以下设备码</span>
                </div>

                {/* 设备码展示 */}
                <div className="flex items-center justify-center gap-2">
                  <code
                    className="text-2xl font-bold tracking-widest px-4 py-2 rounded-lg"
                    style={{
                      background: 'var(--bg-base)',
                      border: '1px solid var(--border-color)',
                      color: accent,
                    }}
                  >
                    {deviceCode.user_code}
                  </code>
                  <button
                    onClick={copyUserCode}
                    className="p-2 rounded-lg transition-colors"
                    style={{ background: 'var(--bg-base)', border: '1px solid var(--border-color)' }}
                    title="复制"
                  >
                    <Copy className="w-4 h-4" style={{ color: 'var(--text-secondary)' }} />
                  </button>
                </div>

                {/* 二维码 */}
                <div className="flex justify-center">
                  <img
                    src={`https://api.qrserver.com/v1/create-qr-code/?size=160x160&data=${encodeURIComponent(deviceCode.verification_uri)}`}
                    alt="QR Code"
                    className="rounded-lg"
                    style={{ border: '1px solid var(--border-color)' }}
                  />
                </div>

                {/* 打开链接 */}
                <div className="flex justify-center">
                  <a
                    href={deviceCode.verification_uri}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-sm hover:underline"
                    style={{ color: accent }}
                  >
                    {deviceCode.verification_uri}
                    <ExternalLink className="w-3 h-3" />
                  </a>
                </div>

                {/* 倒计时 + 轮询状态 */}
                <div className="flex items-center justify-center gap-3 pt-1">
                  {polling && (
                    <span className="inline-flex items-center gap-1.5 text-xs" style={{ color: 'var(--text-secondary)' }}>
                      <Loader2 className="w-3 h-3 animate-spin" />
                      等待授权完成...
                    </span>
                  )}
                  {countdown > 0 && (
                    <span className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.6 }}>
                      有效期剩 {fmtCountdown(countdown)}
                    </span>
                  )}
                </div>
              </div>

              <Button variant="flat" size="sm" onPress={cancelDeviceFlow}>
                取消
              </Button>
            </div>
          ) : (
            // 未授权 — 发起按钮
            <Button
              color="primary" size="md"
              className="shadow-lg"
              startContent={<Smartphone size={16} />}
              isLoading={polling}
              onPress={startDeviceFlow}
            >
              发起 GitHub 设备授权
            </Button>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
