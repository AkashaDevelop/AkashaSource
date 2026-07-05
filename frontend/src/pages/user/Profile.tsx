import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import {
  Card, CardBody, Input, Button, Divider, Chip, Select, SelectItem,
} from '../../components/ui';
import { User as UserIcon, Mail, Lock, Save, Shield, ShieldCheck, ShieldOff, Bell } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import { formatQuota } from '../../lib/quota';

export default function Profile() {
  const { token, updateUser } = useAuthStore();
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    username: '',
    display_name: '',
    email: '',
    password: '',
    role: 1,
    group: 'default',
    totp_enabled: false,
    quota: 0,
    used_quota: 0,
  });

  const [totpStep, setTotpStep] = useState<'idle' | 'setup'>('idle');
  const [totpUri, setTotpUri] = useState('');
  const [totpBackupCodes, setTotpBackupCodes] = useState<string[]>([]);
  const [totpCode, setTotpCode] = useState('');
  const [totpLoading, setTotpLoading] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableCode, setDisableCode] = useState('');

  // 通知设置状态
  const [notifySettings, setNotifySettings] = useState({
    notify_type: 'email',
    notification_email: '',
    webhook_url: '',
    webhook_secret: '',
    bark_url: '',
    gotify_url: '',
    gotify_token: '',
    gotify_priority: 5,
  });

  const fetchProfile = async () => {
    if (!token) return;
    try {
      const res = await fetch('/api/user/self', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        setFormData({ ...data.data, password: '' });
        updateUser(data.data);
        // 获取通知设置
        if (data.data.setting) {
          try {
            const settings = typeof data.data.setting === 'string' ? JSON.parse(data.data.setting) : data.data.setting;
            setNotifySettings({ ...notifySettings, ...settings });
          } catch {}
        }
      }
    } catch (e) { console.error(e); }
  };

  useEffect(() => { if (token) fetchProfile(); }, [token]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/user/self', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          display_name: formData.display_name,
          email: formData.email,
          password: formData.password || undefined,
          setting: JSON.stringify(notifySettings),
        }),
      });
      const data = await res.json();
      if (data.code === 0) { toast.success('个人信息已更新'); fetchProfile(); }
      else toast.error(data.msg || '更新失败');
    } catch { toast.error('请求失败'); }
    finally { setSaving(false); }
  };

  const getRoleName = (role: number) => {
    if (role >= 100) return '超级管理员';
    if (role >= 10) return '管理员';
    return '普通用户';
  };

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
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ code: totpCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '启用失败');
      toast.success('两步验证已启用');
      setTotpStep('idle');
      setTotpCode('');
      fetchProfile();
    } catch (err: any) { toast.error(err.message); }
    finally { setTotpLoading(false); }
  };

  const handleTotpDisable = async () => {
    if (!disablePassword || !disableCode) return;
    setTotpLoading(true);
    try {
      const res = await fetch('/api/user/totp/disable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ password: disablePassword, code: disableCode }),
      });
      const data = await res.json();
      if (data.code !== 0) throw new Error(data.msg || '禁用失败');
      toast.success('两步验证已禁用');
      setDisablePassword('');
      setDisableCode('');
      fetchProfile();
    } catch (err: any) { toast.error(err.message); }
    finally { setTotpLoading(false); }
  };

  const initials = (formData.display_name || formData.username || 'U').slice(0, 2).toUpperCase();

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <PageHeader
        title="个人设置"
        description="管理您的个人资料和账户安全"
        actions={
          <Button color="primary" startContent={<Save size={18} />} isLoading={saving} onPress={handleSave}>
            保存更改
          </Button>
        }
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* 左侧概览 */}
        <Card className="h-fit">
          <CardBody className="flex flex-col items-center gap-4 py-8">
            {/* 头像 */}
            <div style={{
              width: '80px', height: '80px', borderRadius: '50%',
              background: 'linear-gradient(135deg, var(--accent-primary), var(--accent-cosmic))',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: '28px', fontWeight: 700, color: 'white',
              boxShadow: '0 0 0 4px var(--color-info-bg), 0 4px 24px var(--accent-glow)',
            }}>
              {initials}
            </div>
            <div className="text-center">
              <h2 className="text-xl font-bold" style={{ color: 'var(--text-primary)' }}>
                {formData.display_name || formData.username}
              </h2>
              <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>@{formData.username}</p>
            </div>
            <div className="flex gap-2 flex-wrap justify-center">
              <Chip size="sm" variant="flat" color="primary">{getRoleName(formData.role)}</Chip>
              <Chip size="sm" variant="flat">{formData.group}</Chip>
            </div>
            <Divider />
            {/* 额度信息 */}
            <div className="w-full space-y-2">
              <div style={{
                display: 'flex', justifyContent: 'space-between',
                padding: '8px 12px', borderRadius: '10px',
                background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
              }}>
                <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>剩余额度</span>
                <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--accent-cosmic)' }}>
                  ${formatQuota(formData.quota, 4)}
                </span>
              </div>
              <div style={{
                display: 'flex', justifyContent: 'space-between',
                padding: '8px 12px', borderRadius: '10px',
                background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
              }}>
                <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>已用额度</span>
                <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--color-danger-fg)' }}>
                  ${formatQuota(formData.used_quota, 4)}
                </span>
              </div>
            </div>
          </CardBody>
        </Card>

        {/* 右侧表单 */}
        <div className="md:col-span-2 space-y-5">
          {/* 基本信息 */}
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="flex items-center gap-2 mb-1">
                <UserIcon size={16} style={{ color: 'var(--accent-primary)' }} />
                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>基本信息</span>
              </div>
              <Divider />
              <Input label="用户名" value={formData.username} isDisabled
                startContent={<UserIcon size={16} className="text-default-400" />} />
              <Input label="显示名称" placeholder="设置您的昵称"
                value={formData.display_name}
                onValueChange={v => setFormData({ ...formData, display_name: v })} />
              <Input label="邮箱地址" placeholder="example@email.com"
                value={formData.email}
                onValueChange={v => setFormData({ ...formData, email: v })}
                startContent={<Mail size={16} className="text-default-400" />} />
            </CardBody>
          </Card>

          {/* 安全设置 */}
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="flex items-center gap-2 mb-1">
                <Lock size={16} style={{ color: 'var(--accent-primary)' }} />
                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>安全设置</span>
              </div>
              <Divider />
              <Input label="新密码" placeholder="不修改请留空" type="password"
                value={formData.password}
                onValueChange={v => setFormData({ ...formData, password: v })}
                startContent={<Lock size={16} className="text-default-400" />}
                description="密码长度至少需要8位" />
            </CardBody>
          </Card>

          {/* 两步验证 */}
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="flex items-center justify-between mb-1">
                <div className="flex items-center gap-2">
                  <Shield size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>两步验证 (2FA)</span>
                </div>
                <Chip color={formData.totp_enabled ? 'success' : 'default'} variant="flat" size="sm">
                  {formData.totp_enabled ? '已启用' : '未启用'}
                </Chip>
              </div>
              <Divider />

              {!formData.totp_enabled && totpStep === 'idle' && (
                <div>
                  <p style={{ fontSize: '13px', color: 'var(--text-secondary)', marginBottom: '12px' }}>
                    启用两步验证可以增强账户安全性，需要验证器应用（如 Google Authenticator）。
                  </p>
                  <Button color="primary" variant="flat" startContent={<ShieldCheck size={18} />}
                    isLoading={totpLoading} onPress={handleTotpSetup}>
                    开始设置
                  </Button>
                </div>
              )}

              {totpStep === 'setup' && (
                <div className="space-y-4">
                  <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                    使用验证器应用扫描下方二维码：
                  </p>
                  <div className="flex justify-center p-4 bg-white rounded-xl">
                    <img
                      src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(totpUri)}`}
                      alt="TOTP QR Code" className="w-48 h-48"
                    />
                  </div>
                  <Input label="密钥 URI" value={totpUri} isReadOnly size="sm" />
                  {totpBackupCodes.length > 0 && (
                    <div>
                      <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '8px' }}>
                        备份码（请妥善保存）：
                      </p>
                      <div style={{
                        display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px',
                        padding: '12px', borderRadius: '10px',
                        background: 'var(--bg-elevated)', border: '1px solid var(--border-color)',
                        fontFamily: 'monospace', fontSize: '13px', color: 'var(--text-primary)',
                      }}>
                        {totpBackupCodes.map((code, i) => <span key={i}>{code}</span>)}
                      </div>
                    </div>
                  )}
                  <div className="flex gap-2 items-end">
                    <Input label="验证码" placeholder="000000" value={totpCode}
                      onValueChange={setTotpCode} maxLength={6} />
                    <Button color="primary" isLoading={totpLoading} onPress={handleTotpEnable}>
                      验证并启用
                    </Button>
                  </div>
                  <Button variant="light" size="sm" onPress={() => setTotpStep('idle')}>取消</Button>
                </div>
              )}

              {formData.totp_enabled && totpStep === 'idle' && (
                <div className="space-y-3">
                  <p style={{ fontSize: '13px', color: 'var(--color-success-fg)' }}>两步验证已启用，每次登录需要输入验证码。</p>
                  <Divider />
                  <p style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>禁用两步验证</p>
                  <Input label="当前密码" type="password" value={disablePassword}
                    onValueChange={setDisablePassword} size="sm" />
                  <Input label="验证码" placeholder="000000" value={disableCode}
                    onValueChange={setDisableCode} maxLength={6} size="sm" />
                  <Button color="danger" variant="flat" startContent={<ShieldOff size={18} />}
                    isLoading={totpLoading} onPress={handleTotpDisable}>
                    禁用两步验证
                  </Button>
                </div>
              )}
            </CardBody>
          </Card>

          {/* 通知设置 */}
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="flex items-center gap-2 mb-1">
                <Bell size={16} style={{ color: 'var(--accent-primary)' }} />
                <span style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-primary)' }}>通知设置</span>
              </div>
              <Divider />

              <Select
                label="通知方式"
                selectedKeys={[notifySettings.notify_type]}
                onSelectionChange={keys => setNotifySettings({ ...notifySettings, notify_type: [...keys][0] as string || 'email' })}
              >
                <SelectItem key="email">邮件通知</SelectItem>
                <SelectItem key="webhook">Webhook</SelectItem>
                <SelectItem key="bark">Bark (iOS)</SelectItem>
                <SelectItem key="gotify">Gotify</SelectItem>
              </Select>

              {notifySettings.notify_type === 'email' && (
                <Input
                  label="邮箱地址"
                  type="email"
                  value={notifySettings.notification_email}
                  onValueChange={(v) => setNotifySettings({ ...notifySettings, notification_email: v })}
                  placeholder="your@email.com"
                />
              )}

              {notifySettings.notify_type === 'webhook' && (
                <div className="space-y-3">
                  <Input
                    label="Webhook URL"
                    value={notifySettings.webhook_url}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, webhook_url: v })}
                    placeholder="https://your-server.com/webhook"
                  />
                  <Input
                    label="密钥 (可选)"
                    type="password"
                    value={notifySettings.webhook_secret}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, webhook_secret: v })}
                    description="用于 HMAC-SHA256 签名验证"
                  />
                </div>
              )}

              {notifySettings.notify_type === 'bark' && (
                <Input
                  label="Bark URL"
                  value={notifySettings.bark_url}
                  onValueChange={(v) => setNotifySettings({ ...notifySettings, bark_url: v })}
                  placeholder="https://api.day.app/your_key/"
                  description="从 Bark App 中获取推送 URL"
                />
              )}

              {notifySettings.notify_type === 'gotify' && (
                <div className="space-y-3">
                  <Input
                    label="Gotify 服务器"
                    value={notifySettings.gotify_url}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, gotify_url: v })}
                    placeholder="https://gotify.example.com"
                  />
                  <Input
                    label="应用 Token"
                    type="password"
                    value={notifySettings.gotify_token}
                    onValueChange={(v) => setNotifySettings({ ...notifySettings, gotify_token: v })}
                  />
                </div>
              )}
            </CardBody>
          </Card>
        </div>
      </div>
    </div>
  );
}
