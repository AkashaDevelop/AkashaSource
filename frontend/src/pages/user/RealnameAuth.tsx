import { useState, useEffect, useRef, useCallback } from 'react';
import PageHeader from '../../components/PageHeader';
import { Input, Button, Divider, Chip, Alert } from '../../components/ui';
import {
  ShieldCheck, ShieldAlert, ShieldOff, User as UserIcon,
  CreditCard, RefreshCw, ExternalLink, Eye, EyeOff,
  CheckCircle2, XCircle, Clock, Lock, ScanFace, Loader2,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

/* ── 常量 ── */
// 认证状态：0=待认证 1=已通过 2=未通过 3=已过期
const STATUS_PENDING = 0;
const STATUS_PASSED = 1;
const STATUS_FAILED = 2;

// 双盲场景标识
const SCENARIO_DOUBLE_BLIND = 'double_blind';

// 轮询间隔与上限
const POLL_INTERVAL = 3000;
const POLL_MAX_TIMES = 40; // 最长约 2 分钟

/* ── 小工具 ── */
// 身份证号校验：18 位，末位可为 X
const validateIdCard = (id: string): boolean => /^\d{17}[\dXx]$/.test(id.trim());
// 姓名校验：2-20 位中文/字母/中间点
const validateName = (name: string): boolean => /^[\u4e00-\u9fa5·a-zA-Z]{2,20}$/.test(name.trim());

// 格式化时间戳（秒 → 本地时间）
const fmtTime = (ts?: number) => (ts && ts > 0 ? new Date(ts * 1000).toLocaleString('zh-CN') : '-');

export default function RealnameAuth() {
  const { token } = useAuthStore();

  // 认证状态
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<number>(STATUS_PENDING);
  const [verified, setVerified] = useState(false);
  const [enabled, setEnabled] = useState(true);
  const [scenarios, setScenarios] = useState<string[]>([]);
  const [realName, setRealName] = useState(''); // 脱敏姓名（已认证时后端返回）
  const [certifyId, setCertifyId] = useState('');
  const [verifiedAt, setVerifiedAt] = useState<number | undefined>(undefined);

  // 发起认证表单
  const [formName, setFormName] = useState('');
  const [formIdCard, setFormIdCard] = useState('');
  const [showIdCard, setShowIdCard] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // 认证流程：发起后展示 certify_url 并轮询结果
  const [stage, setStage] = useState<'idle' | 'polling'>('idle');
  const [certifyUrl, setCertifyUrl] = useState('');
  const [polling, setPolling] = useState(false);
  const pollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollCount = useRef(0);

  const isDoubleBlind = scenarios.includes(SCENARIO_DOUBLE_BLIND);

  /* ── 拉取认证状态 ── */
  const fetchStatus = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch('/api/user/realname/status', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setEnabled(data.data.enabled !== false);
        setVerified(!!data.data.verified);
        setStatus(data.data.status ?? STATUS_PENDING);
        setScenarios(data.data.scenarios || []);
        setRealName(data.data.real_name || '');
        setCertifyId(data.data.certify_id || '');
        setVerifiedAt(data.data.verified_at);
      }
    } catch (e) {
      console.error('获取实名认证状态失败:', e);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    fetchStatus();
    return () => {
      if (pollTimer.current) clearTimeout(pollTimer.current);
    };
  }, [fetchStatus]);

  /* ── 发起认证 ── */
  const handleInit = async () => {
    const name = formName.trim();
    const idCard = formIdCard.trim();

    if (!validateName(name)) {
      toast.error('请输入正确的姓名（2-20 位中文或字母）');
      return;
    }
    if (!validateIdCard(idCard)) {
      toast.error('请输入正确的 18 位身份证号码');
      return;
    }

    setSubmitting(true);
    try {
      const res = await fetch('/api/user/realname/init', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ real_name: name, id_card: idCard }),
      });
      const data = await res.json();
      if (data.code !== 0) {
        toast.error(data.msg || '发起认证失败');
        return;
      }
      // 展示人脸核验链接并开始轮询
      setCertifyId(data.data.certify_id || '');
      setCertifyUrl(data.data.certify_url || '');
      setStage('polling');
      setPolling(true);
      pollCount.current = 0;
      toast.success('认证已发起，请完成人脸核验');
      startPolling(data.data.certify_id);
    } catch (e) {
      toast.error('请求失败');
    } finally {
      setSubmitting(false);
    }
  };

  /* ── 轮询认证结果 ── */
  const startPolling = (cId: string) => {
    if (pollTimer.current) clearTimeout(pollTimer.current);

    const tick = async () => {
      pollCount.current += 1;
      try {
        const res = await fetch('/api/user/realname/query', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify({ certify_id: cId }),
        });
        const data = await res.json();
        if (data.code === 0) {
          if (data.data.passed) {
            // 认证通过
            setPolling(false);
            setStage('idle');
            toast.success('实名认证通过');
            fetchStatus();
            return;
          }
          // passed 为 false 时不一定是最终失败，可能仍在核验中
          // 仅当返回了明确失败原因时才认为失败
          if (data.data.passed === false && data.data.reason) {
            setPolling(false);
            setStage('idle');
            toast.error(`认证未通过：${data.data.reason}`);
            fetchStatus();
            return;
          }
        }
      } catch (e) {
        console.error('轮询认证结果失败:', e);
      }

      // 继续轮询
      if (pollCount.current < POLL_MAX_TIMES) {
        pollTimer.current = setTimeout(tick, POLL_INTERVAL);
      } else {
        // 超时停止
        setPolling(false);
        setStage('idle');
        toast.error('认证结果查询超时，请稍后刷新页面查看');
      }
    };

    pollTimer.current = setTimeout(tick, POLL_INTERVAL);
  };

  // 手动刷新结果
  const handleQuery = async () => {
    if (!certifyId) {
      toast.error('未找到认证流水号');
      return;
    }
    setPolling(true);
    try {
      const res = await fetch('/api/user/realname/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ certify_id: certifyId }),
      });
      const data = await res.json();
      if (data.code === 0) {
        if (data.data.passed) {
          toast.success('实名认证通过');
          fetchStatus();
        } else if (data.data.reason) {
          toast.error(`认证未通过：${data.data.reason}`);
          fetchStatus();
        } else {
          toast.info('认证仍在进行中，请稍后再试');
        }
      } else {
        toast.error(data.msg || '查询失败');
      }
    } catch (e) {
      toast.error('请求失败');
    } finally {
      setPolling(false);
    }
  };

  const cancelPolling = () => {
    if (pollTimer.current) clearTimeout(pollTimer.current);
    setPolling(false);
    setStage('idle');
  };

  /* ── 渲染 ── */

  // 加载中骨架
  if (loading) {
    return (
      <div className="max-w-2xl mx-auto">
        <PageHeader title="实名认证" description="完成实名认证以使用相关功能" />
        <div className="flex items-center justify-center py-20" style={{ color: 'var(--text-muted)' }}>
          <Loader2 className="animate-spin" size={20} />
          <span className="ml-2 text-sm">加载中…</span>
        </div>
      </div>
    );
  }

  // 功能未启用
  if (!enabled) {
    return (
      <div className="max-w-2xl mx-auto">
        <PageHeader title="实名认证" description="完成实名认证以使用相关功能" />
        <Alert color="default" title="功能未开启" description="实名认证功能当前未开启，如有需要请联系管理员。" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <PageHeader
        title="实名认证"
        description="完成实名认证以使用相关功能"
        actions={
          <Button variant="flat" size="sm" startContent={<RefreshCw size={14} />} onPress={fetchStatus}>
            刷新状态
          </Button>
        }
      />

      {/* ── 双盲模式提示（醒目） ── */}
      {isDoubleBlind && (
        <div className="mb-4 flex items-start gap-3 px-4 py-3 rounded-xl"
          style={{
            background: 'linear-gradient(135deg, rgba(168,85,247,0.12), rgba(99,102,241,0.10))',
            border: '1px solid rgba(168,85,247,0.35)',
          }}>
          <Lock size={18} className="flex-shrink-0 mt-0.5" style={{ color: '#a855f7' }} />
          <div>
            <p style={{ fontSize: '13px', fontWeight: 700, color: '#c084fc' }}>双盲模式已启用</p>
            <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '2px' }}>
              系统只保留认证结果，不存储您的实名信息（姓名、身份证号），最大程度保护您的隐私。
            </p>
          </div>
        </div>
      )}

      {/* ── 认证状态卡片 ── */}
      <div className="px-5 py-5 mb-4" style={{
        borderRadius: 'var(--radius-lg)', background: 'var(--bg-surface)',
        border: '1px solid var(--border-color)',
      }}>
        {/* 已认证 */}
        {verified && status === STATUS_PASSED && (
          <div className="flex flex-col items-center text-center py-2">
            <div style={{
              width: '52px', height: '52px', borderRadius: '50%',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'rgba(16,185,129,0.12)', marginBottom: '12px',
            }}>
              <ShieldCheck size={26} style={{ color: '#10b981' }} />
            </div>
            <Chip color="success" variant="flat" startContent={<CheckCircle2 size={12} />}>已认证</Chip>
            <div className="mt-4 w-full" style={{ borderTop: '1px solid var(--border-color)', paddingTop: '14px' }}>
              <div className="grid grid-cols-2 gap-4 text-left">
                <div>
                  <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>认证姓名</p>
                  <p style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                    {realName || '已认证'}
                  </p>
                </div>
                <div>
                  <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>认证时间</p>
                  <p style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                    {fmtTime(verifiedAt)}
                  </p>
                </div>
              </div>
            </div>
            <p className="mt-4" style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
              您已完成实名认证，可正常使用相关功能。
            </p>
          </div>
        )}

        {/* 认证失败 */}
        {!verified && status === STATUS_FAILED && (
          <div className="flex flex-col items-center text-center py-2">
            <div style={{
              width: '52px', height: '52px', borderRadius: '50%',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'rgba(239,68,68,0.12)', marginBottom: '12px',
            }}>
              <ShieldAlert size={26} style={{ color: '#ef4444' }} />
            </div>
            <Chip color="danger" variant="flat" startContent={<XCircle size={12} />}>认证未通过</Chip>
            <p className="mt-3" style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
              您的上一次认证未通过，请重新发起认证。
            </p>
            <Button className="mt-4" color="danger" variant="flat" size="sm"
              startContent={<RefreshCw size={14} />} onPress={() => { setStatus(STATUS_PENDING); }}>
              重新认证
            </Button>
          </div>
        )}

        {/* 未认证 / 重新认证中 */}
        {!verified && status !== STATUS_PASSED && status !== STATUS_FAILED && (
          <div className="flex flex-col items-center text-center py-2">
            <div style={{
              width: '52px', height: '52px', borderRadius: '50%',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'var(--bg-elevated)', marginBottom: '12px',
              border: '1px solid var(--border-color)',
            }}>
              <ShieldOff size={26} style={{ color: 'var(--text-muted)' }} />
            </div>
            <Chip variant="flat" startContent={<Clock size={12} />}>未认证</Chip>
            <p className="mt-3" style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
              请填写实名信息并完成人脸核验
            </p>
          </div>
        )}
      </div>

      {/* ── 发起认证表单（未通过时展示） ── */}
      {!verified && status !== STATUS_PASSED && (
        <div className="px-5 py-5" style={{
          borderRadius: 'var(--radius-lg)', background: 'var(--bg-surface)',
          border: '1px solid var(--border-color)',
        }}>
          {/* 人脸核验中：展示链接 + 二维码 + 轮询 */}
          {stage === 'polling' ? (
            <div className="flex flex-col items-center py-2">
              <div className="flex items-center gap-2 mb-4">
                <ScanFace size={18} style={{ color: 'var(--accent-primary)' }} />
                <span style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)' }}>完成人脸核验</span>
              </div>

              {/* 二维码 */}
              {certifyUrl && (
                <div className="flex flex-col items-center gap-2 mb-4">
                  <div style={{ padding: '8px', borderRadius: 'var(--radius-md)', background: 'white', border: '1px solid var(--border-color)' }}>
                    <img
                      src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(certifyUrl)}`}
                      alt="人脸核验二维码"
                      style={{ width: '170px', height: '170px' }}
                    />
                  </div>
                  <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>使用手机扫码完成人脸核验</p>
                </div>
              )}

              {/* 链接按钮 */}
              {certifyUrl && (
                <a href={certifyUrl} target="_blank" rel="noopener noreferrer" className="mb-4">
                  <Button color="primary" variant="flat" size="sm" startContent={<ExternalLink size={14} />}>
                    在新窗口打开核验链接
                  </Button>
                </a>
              )}

              {/* 轮询状态 */}
              <div className="flex items-center gap-2 mb-3" style={{ color: 'var(--text-muted)' }}>
                {polling ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    <span style={{ fontSize: '12px' }}>正在等待核验结果…</span>
                  </>
                ) : (
                  <>
                    <Clock size={14} />
                    <span style={{ fontSize: '12px' }}>已停止轮询，可手动查询结果</span>
                  </>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Button color="primary" size="sm" startContent={<RefreshCw size={14} />}
                  isLoading={polling} onPress={handleQuery}>
                  查询结果
                </Button>
                <Button variant="light" size="sm" onPress={cancelPolling} isDisabled={!polling}>
                  停止轮询
                </Button>
              </div>
            </div>
          ) : (
            /* 表单 */
            <div className="py-1">
              <p className="mb-4" style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>
                填写实名信息
              </p>

              <div className="flex flex-col gap-4">
                <div>
                  <Input
                    label="真实姓名"
                    placeholder="请输入真实姓名"
                    value={formName}
                    onValueChange={setFormName}
                    startContent={<UserIcon size={14} className="text-default-400" />}
                    maxLength={20}
                    description="2-20 位中文或字母"
                  />
                </div>

                <div>
                  <Input
                    label="身份证号"
                    placeholder="请输入 18 位身份证号码"
                    type={showIdCard ? 'text' : 'password'}
                    value={formIdCard}
                    onValueChange={setFormIdCard}
                    startContent={<CreditCard size={14} className="text-default-400" />}
                    maxLength={18}
                    endContent={
                      <button type="button" onClick={() => setShowIdCard(!showIdCard)}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px' }}>
                        {showIdCard ? <EyeOff size={14} /> : <Eye size={14} />}
                      </button>
                    }
                    description="18 位身份证号码，末位可为 X"
                  />
                </div>
              </div>

              <Divider className="my-4" />

              <Alert color="primary" className="mb-4">
                <p style={{ fontSize: '12px', lineHeight: '1.6' }}>
                  提交后将跳转至阿里云实人认证完成人脸核验。{isDoubleBlind
                    ? '当前为双盲模式，您的实名信息不会被存储。'
                    : '系统仅存储姓名与身份证号哈希值，不存储身份证号明文。'}
                </p>
              </Alert>

              <Button color="primary" className="w-full"
                startContent={<ShieldCheck size={16} />}
                isLoading={submitting}
                onPress={handleInit}>
                发起实名认证
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
