import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import { Input, Button, Divider, Chip } from '../../components/ui';
import {
  User as UserIcon, Mail, Lock, Save, ShieldCheck, ShieldOff,
  Eye, EyeOff, KeyRound, Smartphone, Webhook, Server, Fingerprint,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { formatQuota } from '../../lib/quota';

type TabKey = 'account' | 'security' | 'notifications';

const TABS: { key: TabKey; label: string }[] = [
  { key: 'account',       label: '账号信息' },
  { key: 'security',      label: '安全验证' },
  { key: 'notifications', label: '通知设置' },
];

const NOTIFY_OPTIONS = [
  { key: 'email',   label: '邮件',    icon: Mail,    desc: '通过邮件接收通知' },
  { key: 'webhook', label: 'Webhook', icon: Webhook, desc: '自定义 HTTP 回调' },
  { key: 'bark',    label: 'Bark',    icon: Smartphone, desc: 'iOS 推送通知' },
  { key: 'gotify',  label: 'Gotify',  icon: Server,  desc: '自托管推送服务' },
] as const;

/* ── 小工具 ── */
function FieldLabel({ children, hint }: { children: React.ReactNode; hint?: string }) {
  return (
    <div className="flex items-baseline justify-between mb-1.5">
      <label style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-secondary)' }}>{children}</label>
      {hint && <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>{hint}</span>}
    </div>
  );
}

function Row({ label, desc, hint, children }: { label: string; desc?: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 py-4 border-b" style={{ borderColor: 'var(--border-color)' }}>
      <div className="sm:col-span-1">
        <div className="flex items-baseline justify-between">
          <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>{label}</p>
          {hint && <span style={{ fontSize: '11px', color: 'var(--text-faint)' }}>{hint}</span>}
        </div>
        {desc && <p style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '2px' }}>{desc}</p>}
      </div>
      <div className="sm:col-span-2">{children}</div>
    </div>
  );
}

// Base64url helpers for WebAuthn
function b64urlToBuf(b64: string): Uint8Array {
  const pad = '='.repeat((4 - b64.length % 4) % 4);
  const b64s = (b64 + pad).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(b64s);
  return Uint8Array.from(raw, c => c.charCodeAt(0));
}
function bufToB64url(buf: ArrayBuffer | Uint8Array): string {
  const bytes = buf instanceof Uint8Array ? buf : new Uint8Array(buf);
  let bin = '';
  bytes.forEach(b => bin += String.fromCharCode(b));
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

export default function Profile() {
  const { token, updateUser } = useAuthStore();
  const [saving, setSaving] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>('account');
  const [showPassword, setShowPassword] = useState(false);
  const [formData, setFormData] = useState({
    username: '', display_name: '', email: '', password: '',
    role: 1, group: 'default', totp_enabled: false, quota: 0, used_quota: 0,
  });

  const [totpStep, setTotpStep] = useState<'idle' | 'setup'>('idle');
  const [totpUri, setTotpUri] = useState('');
  const [totpBackupCodes, setTotpBackupCodes] = useState<string[]>([]);
  const [totpCode, setTotpCode] = useState('');
  const [totpLoading, setTotpLoading] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableCode, setDisableCode] = useState('');

  const [passkeyEnabled, setPasskeyEnabled] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);

  const [notifySettings, setNotifySettings] = useState({
    notify_type: 'email', notification_email: '', webhook_url: '', webhook_secret: '',
    bark_url: '', gotify_url: '', gotify_token: '', gotify_priority: 5,
  });

  const fetchProfile = async () => {
    if (!token) return;
    try {
      const res = await fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        setFormData({ ...data.data, password: '' });
        updateUser(data.data);
        if (data.data.setting) {
          try {
            const s = typeof data.data.setting === 'string' ? JSON.parse(data.data.setting) : data.data.setting;
            setNotifySettings({ ...notifySettings, ...s });
          } catch {}
        }
      }
    } catch (e) { console.error(e); }

    try {
      const pkRes = await fetch('/api/user/passkey', { headers: { Authorization: `Bearer ${token}` } });
      const pkData = await pkRes.json();
      if (pkData.code === 0) setPasskeyEnabled(pkData.data.enabled);
    } catch {}
  };

  useEffect(() => { if (token) fetchProfile(); }, [token]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/user/self', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          display_name: formData.display_name, email: formData.email,
          password: formData.password || undefined, setting: JSON.stringify(notifySettings),
        }),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success('已保存'); fetchProfile(); }
      else toast.error(data.msg || '保存失败');
    } catch { toast.error('请求失败'); }
    finally { setSaving(false); }
  };

  const getRoleName = (r: number) => r >= 100 ? '超级管理员' : r >= 10 ? '管理员' : '普通用户';

  const handleTotpSetup = async () => {
    setTotpLoading(true);
    try {
      const res = await fetch('/api/user/totp/setup', { method: 'POST', headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '设置失败');
      setTotpUri(data.data.uri);
      setTotpBackupCodes(data.data.backup_codes || []);
      setTotpStep('setup');
    } catch (err: any) { toast.error(err.message); }
    finally { setTotpLoading(false); }
  };

  const handleTotpEnable = async () => {
    if (!totpCode) return;
    setTotpLoading(true);
    try {
      const res = await fetch('/api/user/totp/enable', {
        method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ code: totpCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '启用失败');
      toast.success('两步验证已启用');
      setTotpStep('idle'); setTotpCode(''); fetchProfile();
    } catch (err: any) { toast.error(err.message); }
    finally { setTotpLoading(false); }
  };

  const handleTotpDisable = async () => {
    if (!disablePassword || !disableCode) return;
    setTotpLoading(true);
    try {
      const res = await fetch('/api/user/totp/disable', {
        method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ password: disablePassword, code: disableCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '禁用失败');
      toast.success('两步验证已禁用');
      setDisablePassword(''); setDisableCode(''); fetchProfile();
    } catch (err: any) { toast.error(err.message); }
    finally { setTotpLoading(false); }
  };

  const handlePasskeyRegister = async () => {
    setPasskeyLoading(true);
    try {
      // 1. Begin registration
      const beginRes = await fetch('/api/user/passkey/register/begin', {
        method: 'POST', headers: { Authorization: `Bearer ${token}` },
      });
      const beginData = await beginRes.json();
      if (beginData.code !== 0) throw new Error(beginData.msg || '启动注册失败');
      const { session_id, options } = beginData.data;

      // 2. Convert options for browser API (options is { publicKey: {...} })
      const pk = options.publicKey;
      const publicKey = {
        ...pk,
        challenge: b64urlToBuf(pk.challenge),
        user: { ...pk.user, id: b64urlToBuf(pk.user.id) },
        excludeCredentials: (pk.excludeCredentials || []).map((c: any) => ({ ...c, id: b64urlToBuf(c.id) })),
      };

      // 3. Create credential
      const credential = await navigator.credentials.create({ publicKey }) as any;
      if (!credential) throw new Error('浏览器未返回凭证');

      // 4. Finish registration
      const finishBody = {
        session_id,
        id: credential.id,
        rawId: bufToB64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufToB64url(credential.response.attestationObject),
          clientDataJSON: bufToB64url(credential.response.clientDataJSON),
        },
        clientExtensionResults: credential.getClientExtensionResults ? credential.getClientExtensionResults() : {},
      };
      const finishRes = await fetch('/api/user/passkey/register/finish', {
        method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(finishBody),
      });
      const finishData = await finishRes.json();
      if (finishData.code !== 0) throw new Error(finishData.msg || '注册失败');
      toast.success('Passkey 注册成功');
      setPasskeyEnabled(true);
    } catch (err: any) {
      toast.error(err.message || 'Passkey 注册失败');
    } finally {
      setPasskeyLoading(false);
    }
  };

  const handlePasskeyDelete = async () => {
    setPasskeyLoading(true);
    try {
      const res = await fetch('/api/user/passkey', {
        method: 'DELETE', headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '解绑失败');
      toast.success('Passkey 已解绑');
      setPasskeyEnabled(false);
    } catch (err: any) {
      toast.error(err.message);
    } finally {
      setPasskeyLoading(false);
    }
  };

  const initials = (formData.display_name || formData.username || 'U').slice(0, 2).toUpperCase();

  return (
    <div className="max-w-3xl mx-auto">
      <PageHeader
        title="个人设置"
        description="管理账号信息、安全验证与通知偏好"
        actions={
          <Button color="primary" startContent={<Save size={16} />} isLoading={saving} onPress={handleSave}>
            保存更改
          </Button>
        }
      />

      {/* ── 身份摘要条 ── */}
      <div className="flex items-center gap-4 px-5 py-4 mb-1" style={{
        borderRadius: 'var(--radius-lg)', background: 'var(--bg-surface)',
        border: '1px solid var(--border-color)',
      }}>
        <div style={{
          width: '44px', height: '44px', borderRadius: 'var(--radius-lg)', flexShrink: 0,
          background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: '16px', fontWeight: 700, color: 'white',
        }}>
          {initials}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span style={{ fontSize: '15px', fontWeight: 700, color: 'var(--text-primary)' }}>
              {formData.display_name || formData.username}
            </span>
            <Chip size="sm" variant="flat" color="primary">{getRoleName(formData.role)}</Chip>
            <Chip size="sm" variant="flat" startContent={<KeyRound size={10} />}>{formData.group}</Chip>
          </div>
          <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>@{formData.username} · {formData.email || '未绑定邮箱'}</p>
        </div>
        <div className="hidden sm:flex items-center gap-4 pl-4" style={{ borderLeft: '1px solid var(--border-color)' }}>
          <div className="text-right">
            <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>余额</p>
            <p style={{ fontSize: '14px', fontWeight: 700, color: 'var(--accent-cosmic)' }}>${formatQuota(formData.quota, 2)}</p>
          </div>
          <div className="text-right">
            <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>已用</p>
            <p style={{ fontSize: '14px', fontWeight: 700, color: 'var(--color-danger-fg)' }}>${formatQuota(formData.used_quota, 2)}</p>
          </div>
        </div>
      </div>

      {/* ── Tab 导航 ── */}
      <div className="flex items-center gap-1 px-1 mb-4" style={{ borderBottom: '1px solid var(--border-color)' }}>
        {TABS.map(tab => {
          const isActive = activeTab === tab.key;
          return (
            <button key={tab.key} type="button" onClick={() => setActiveTab(tab.key)}
              style={{
                padding: '10px 16px', fontSize: '13px', fontWeight: 600,
                color: isActive ? 'var(--accent-primary)' : 'var(--text-muted)',
                background: 'none', border: 'none', cursor: 'pointer',
                borderBottom: `2px solid ${isActive ? 'var(--accent-primary)' : 'transparent'}`,
                marginBottom: '-1px', transition: 'all 0.15s',
              }}>
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* ── Tab 内容 ── */}
      <div style={{ borderRadius: 'var(--radius-lg)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>

        {/* ══ 账号信息 ══ */}
        {activeTab === 'account' && (
          <div className="px-5 py-2">
            <Row label="用户名" desc="系统唯一标识，不可修改">
              <Input value={formData.username} isDisabled size="sm"
                startContent={<UserIcon size={14} className="text-default-400" />} />
            </Row>
            <Row label="显示名称" desc="展示在界面上的昵称">
              <Input placeholder="设置昵称" size="sm"
                value={formData.display_name}
                onValueChange={v => setFormData({ ...formData, display_name: v })} />
            </Row>
            <Row label="邮箱地址" desc="用于通知和找回密码">
              <Input placeholder="example@email.com" size="sm"
                value={formData.email}
                onValueChange={v => setFormData({ ...formData, email: v })}
                startContent={<Mail size={14} className="text-default-400" />} />
            </Row>
            <Row label="登录密码" desc="留空则不修改当前密码">
              <Input placeholder="输入新密码" size="sm"
                type={showPassword ? 'text' : 'password'}
                value={formData.password}
                onValueChange={v => setFormData({ ...formData, password: v })}
                startContent={<Lock size={14} className="text-default-400" />}
                endContent={
                  <button type="button" onClick={() => setShowPassword(!showPassword)}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '2px' }}>
                    {showPassword ? <EyeOff size={14} /> : <Eye size={14} />}
                  </button>
                }
                description="至少 8 位字符" />
            </Row>
          </div>
        )}

        {/* ══ 安全验证 ══ */}
        {activeTab === 'security' && (
          <div className="px-5 py-2">
            <Row label="两步验证 (2FA)" desc="登录时需要额外的验证码">
              <div className="space-y-3">
                {/* 状态徽章 */}
                <div className="flex items-center gap-2">
                  {formData.totp_enabled ? (
                    <Chip color="success" variant="flat" size="sm" startContent={<ShieldCheck size={12} />}>已启用</Chip>
                  ) : (
                    <Chip variant="flat" size="sm" startContent={<ShieldOff size={12} />}>未启用</Chip>
                  )}
                </div>

                {/* 未启用 - 引导 */}
                {!formData.totp_enabled && totpStep === 'idle' && (
                  <div className="space-y-3">
                    <p style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
                      启用后，每次登录需要输入验证器应用生成的一次性验证码。
                    </p>
                    <Button color="primary" variant="flat" size="sm" startContent={<ShieldCheck size={15} />}
                      isLoading={totpLoading} onPress={handleTotpSetup}>
                      开始设置
                    </Button>
                  </div>
                )}

                {/* 设置流程 */}
                {totpStep === 'setup' && (
                  <div className="space-y-4 pt-2">
                    {/* 步骤指示 */}
                    <div className="flex items-center gap-2" style={{ fontSize: '12px' }}>
                      {['扫描二维码', '输入验证码', '保存备份码'].map((step, i) => (
                        <span key={i} className="flex items-center gap-2">
                          <span style={{
                            width: '18px', height: '18px', borderRadius: '50%',
                            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
                            fontSize: '10px', fontWeight: 700,
                            background: i === 0 ? 'var(--accent-primary)' : 'var(--bg-elevated)',
                            color: i === 0 ? 'white' : 'var(--text-muted)',
                            border: `1px solid ${i === 0 ? 'var(--accent-primary)' : 'var(--border-color)'}`,
                          }}>{i + 1}</span>
                          <span style={{ color: i === 0 ? 'var(--accent-primary)' : 'var(--text-muted)', fontWeight: i === 0 ? 600 : 400 }}>{step}</span>
                          {i < 2 && <span style={{ color: 'var(--text-faint)' }}>·</span>}
                        </span>
                      ))}
                    </div>

                    <div className="flex flex-col sm:flex-row gap-4">
                      <div className="flex flex-col items-center gap-2">
                        <div style={{ padding: '6px', borderRadius: 'var(--radius-md)', background: 'white', border: '1px solid var(--border-color)' }}>
                          <img src={`https://api.qrserver.com/v1/create-qr-code/?size=140x140&data=${encodeURIComponent(totpUri)}`}
                            alt="QR" style={{ width: '130px', height: '130px' }} />
                        </div>
                        <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>用验证器 App 扫描</p>
                      </div>
                      <div className="flex-1 space-y-3">
                        <Input label="密钥 URI" value={totpUri} isReadOnly size="sm" />
                        <div className="flex gap-2 items-end">
                          <Input label="验证码" placeholder="000000" value={totpCode}
                            onValueChange={setTotpCode} maxLength={6} size="sm" style={{ maxWidth: '120px' }} />
                          <Button color="primary" size="sm" isLoading={totpLoading} onPress={handleTotpEnable}>
                            验证并启用
                          </Button>
                        </div>
                      </div>
                    </div>

                    {totpBackupCodes.length > 0 && (
                      <div>
                        <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--accent-star)', marginBottom: '6px' }}>
                          备份码（妥善保存，丢失无法恢复）
                        </p>
                        <div style={{
                          display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))', gap: '4px',
                          padding: '10px', borderRadius: 'var(--radius-md)',
                          background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                          fontFamily: 'monospace', fontSize: '12px', color: 'var(--text-primary)',
                        }}>
                          {totpBackupCodes.map((code, i) => (
                            <span key={i} style={{ padding: '2px 6px', borderRadius: '4px', background: 'var(--bg-surface)' }}>{code}</span>
                          ))}
                        </div>
                      </div>
                    )}
                    <Button variant="light" size="sm" onPress={() => setTotpStep('idle')}>取消</Button>
                  </div>
                )}

                {/* 已启用 - 禁用流程 */}
                {formData.totp_enabled && totpStep === 'idle' && (
                  <div className="space-y-3 pt-2">
                    <div className="flex items-center gap-2 p-2.5" style={{
                      borderRadius: 'var(--radius-md)', background: 'var(--color-success-bg)',
                    }}>
                      <ShieldCheck size={14} style={{ color: 'var(--color-success-fg)' }} />
                      <span style={{ fontSize: '12px', color: 'var(--color-success-fg)' }}>两步验证已生效，每次登录需输入验证码。</span>
                    </div>
                    <Divider />
                    <FieldLabel hint="需验证身份">禁用两步验证</FieldLabel>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      <Input placeholder="当前密码" type="password" value={disablePassword}
                        onValueChange={setDisablePassword} size="sm"
                        startContent={<Lock size={14} className="text-default-400" />} />
                      <Input placeholder="验证码" value={disableCode}
                        onValueChange={setDisableCode} maxLength={6} size="sm"
                        startContent={<ShieldOff size={14} className="text-default-400" />} />
                    </div>
                    <Button color="danger" variant="flat" size="sm" startContent={<ShieldOff size={14} />}
                      isLoading={totpLoading} onPress={handleTotpDisable}>
                      确认禁用
                    </Button>
                  </div>
                )}
              </div>
            </Row>

            <Row label="Passkey" desc="使用指纹、面容或安全密钥无密码登录">
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <Chip size="sm" variant="flat" color={passkeyEnabled ? 'success' : 'default'}
                    startContent={<Fingerprint size={12} />}>
                    {passkeyEnabled ? '已绑定' : '未绑定'}
                  </Chip>
                </div>
                {!passkeyEnabled ? (
                  <Button color="primary" variant="flat" size="sm" startContent={<Fingerprint size={15} />}
                    isLoading={passkeyLoading} onPress={handlePasskeyRegister}>
                    注册 Passkey
                  </Button>
                ) : (
                  <Button color="danger" variant="flat" size="sm"
                    isLoading={passkeyLoading} onPress={handlePasskeyDelete}>
                    解绑 Passkey
                  </Button>
                )}
                {passkeyEnabled && (
                  <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
                    绑定后可在登录页面使用 Passkey 无密码登录。解绑后需要重新注册。
                  </p>
                )}
              </div>
            </Row>
          </div>
        )}

        {/* ══ 通知设置 ══ */}
        {activeTab === 'notifications' && (
          <div className="px-5 py-2">
            <Row label="通知方式" desc="选择接收系统通知的渠道">
              <div className="space-y-2">
                {NOTIFY_OPTIONS.map(opt => {
                  const isActive = notifySettings.notify_type === opt.key;
                  const Icon = opt.icon;
                  return (
                    <button key={opt.key} type="button"
                      onClick={() => setNotifySettings({ ...notifySettings, notify_type: opt.key })}
                      className="flex items-center gap-3 w-full text-left p-2.5"
                      style={{
                        borderRadius: 'var(--radius-md)', cursor: 'pointer',
                        border: `1px solid ${isActive ? 'var(--accent-primary)' : 'var(--border-color)'}`,
                        background: isActive ? 'var(--nav-active-bg)' : 'var(--bg-elevated)',
                        transition: 'all 0.12s',
                      }}>
                      <div style={{
                        width: '28px', height: '28px', borderRadius: 'var(--radius-sm)', flexShrink: 0,
                        background: isActive ? 'var(--accent-primary)' : 'var(--bg-surface)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        border: `1px solid ${isActive ? 'var(--accent-primary)' : 'var(--border-color)'}`,
                      }}>
                        <Icon size={13} style={{ color: isActive ? 'white' : 'var(--text-muted)' }} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p style={{
                          fontSize: '13px', fontWeight: 600, margin: 0,
                          color: isActive ? 'var(--accent-primary)' : 'var(--text-primary)',
                        }}>{opt.label}</p>
                        <p style={{ fontSize: '11px', color: 'var(--text-muted)', margin: '1px 0 0' }}>{opt.desc}</p>
                      </div>
                      <div style={{
                        width: '16px', height: '16px', borderRadius: '50%', flexShrink: 0,
                        border: `2px solid ${isActive ? 'var(--accent-primary)' : 'var(--border-strong)'}`,
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                      }}>
                        {isActive && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--accent-primary)' }} />}
                      </div>
                    </button>
                  );
                })}
              </div>
            </Row>

            {/* 动态配置字段 */}
            {notifySettings.notify_type === 'email' && (
              <Row label="通知邮箱" desc="接收通知的邮箱地址">
                <Input type="email" size="sm" placeholder="your@email.com"
                  value={notifySettings.notification_email}
                  onValueChange={(v) => setNotifySettings({ ...notifySettings, notification_email: v })}
                  startContent={<Mail size={14} className="text-default-400" />} />
              </Row>
            )}
            {notifySettings.notify_type === 'webhook' && (
              <>
                <Row label="Webhook URL" desc="接收 POST 请求的端点">
                  <Input size="sm" placeholder="https://your-server.com/webhook"
                    value={notifySettings.webhook_url}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, webhook_url: v })}
                    startContent={<Webhook size={14} className="text-default-400" />} />
                </Row>
                <Row label="签名密钥" desc="用于 HMAC-SHA256 签名验证" hint="可选">
                  <Input type="password" size="sm"
                    value={notifySettings.webhook_secret}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, webhook_secret: v })}
                    startContent={<Lock size={14} className="text-default-400" />} />
                </Row>
              </>
            )}
            {notifySettings.notify_type === 'bark' && (
              <Row label="Bark URL" desc="从 Bark App 中获取">
                <Input size="sm" placeholder="https://api.day.app/your_key/"
                  value={notifySettings.bark_url}
                  onValueChange={(v) => setNotifySettings({ ...notifySettings, bark_url: v })}
                  startContent={<Smartphone size={14} className="text-default-400" />} />
              </Row>
            )}
            {notifySettings.notify_type === 'gotify' && (
              <>
                <Row label="Gotify 服务器" desc="自托管 Gotify 实例地址">
                  <Input size="sm" placeholder="https://gotify.example.com"
                    value={notifySettings.gotify_url}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, gotify_url: v })}
                    startContent={<Server size={14} className="text-default-400" />} />
                </Row>
                <Row label="应用 Token" desc="在 Gotify 中创建应用后获取">
                  <Input type="password" size="sm"
                    value={notifySettings.gotify_token}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, gotify_token: v })}
                    startContent={<KeyRound size={14} className="text-default-400" />} />
                </Row>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
