// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 站长在这里用 GitHub 登录完成组织成员授权，验证通过后系统才会解除拦截喵～
import { useState, useEffect } from 'react';
import { Card, CardBody, Button, Chip } from '../../components/ui';
import { Github, ShieldAlert, ShieldCheck, Unlink } from 'lucide-react';
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

export default function SystemLicense() {
  const { token } = useAuthStore();
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [unbinding, setUnbinding] = useState(false);

  const fetchStatus = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/system-license/status', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) setStatus(data.data);
    } catch (e) { console.error(e); }
    finally { setLoading(false); }
  };

  useEffect(() => {
    fetchStatus();
    // 授权回调重定向回来时会带上这两个查询参数，弹个 toast 就清掉
    const params = new URLSearchParams(window.location.search);
    const license = params.get('license');
    if (license === 'success') {
      toast.success('系统授权成功');
      fetchStatus();
    } else if (license === 'error') {
      toast.error(params.get('reason') || '授权失败');
    }
    if (license) {
      params.delete('license');
      params.delete('reason');
      const rest = params.toString();
      window.history.replaceState({}, '', window.location.pathname + (rest ? `?${rest}` : ''));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
  const accent = authorized ? '#10b981' : 'var(--accent-star)'; // 已授权=翡翠绿，未授权=既有的警示金色

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
          ) : (
            <Button
              color="primary" size="md"
              className="shadow-lg"
              startContent={<Github size={16} />}
              onPress={() => { window.location.href = '/api/system-license/github/start'; }}
            >
              使用 GitHub 登录进行授权
            </Button>
          )}
        </CardBody>
      </Card>
    </div>
  );
}
