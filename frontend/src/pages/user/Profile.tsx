import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Input,
  Button,
  Avatar,
  Divider,
  Chip,
  Alert,
} from '@heroui/react';
import { User as UserIcon, Mail, Lock, Save, Shield, ShieldCheck, ShieldOff } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

export default function Profile() {
  const { token, updateUser } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    username: '',
    display_name: '',
    email: '',
    password: '',
    role: 1,
    group: 'default',
    totp_enabled: false,
  });

  // 2FA state
  const [totpStep, setTotpStep] = useState<'idle' | 'setup' | 'verify'>('idle');
  const [totpUri, setTotpUri] = useState('');
  const [totpBackupCodes, setTotpBackupCodes] = useState<string[]>([]);
  const [totpCode, setTotpCode] = useState('');
  const [totpMsg, setTotpMsg] = useState('');
  const [totpLoading, setTotpLoading] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableCode, setDisableCode] = useState('');

  const fetchProfile = async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch('/api/user/self', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        setFormData({
            ...data.data,
            password: '', // Don't show hash
        });
        updateUser(data.data);
      }
    } catch (error) {
      console.error('Failed to fetch profile:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchProfile();
  }, [token]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/user/self', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
            display_name: formData.display_name,
            email: formData.email,
            password: formData.password || undefined,
        }),
      });

      if (res.ok) {
        alert('个人信息已更新');
        fetchProfile();
      } else {
        alert('更新失败');
      }
    } catch (error) {
      console.error('Save error:', error);
      alert('请求失败');
    } finally {
      setSaving(false);
    }
  };

  const getRoleName = (role: number) => {
      if (role >= 100) return '超级管理员';
      if (role >= 10) return '管理员';
      return '普通用户';
  };

  const handleTotpSetup = async () => {
    setTotpLoading(true);
    setTotpMsg('');
    try {
      const res = await fetch('/api/user/totp/setup', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || '设置失败');
      setTotpUri(data.uri);
      setTotpBackupCodes(data.backup_codes || []);
      setTotpStep('setup');
    } catch (err: any) {
      setTotpMsg(err.message);
    } finally {
      setTotpLoading(false);
    }
  };

  const handleTotpEnable = async () => {
    if (!totpCode) return;
    setTotpLoading(true);
    setTotpMsg('');
    try {
      const res = await fetch('/api/user/totp/enable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ code: totpCode }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || '启用失败');
      setTotpMsg('两步验证已启用');
      setTotpStep('idle');
      setTotpCode('');
      fetchProfile();
    } catch (err: any) {
      setTotpMsg(err.message);
    } finally {
      setTotpLoading(false);
    }
  };

  const handleTotpDisable = async () => {
    if (!disablePassword || !disableCode) return;
    setTotpLoading(true);
    setTotpMsg('');
    try {
      const res = await fetch('/api/user/totp/disable', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ password: disablePassword, code: disableCode }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || '禁用失败');
      setTotpMsg('两步验证已禁用');
      setDisablePassword('');
      setDisableCode('');
      fetchProfile();
    } catch (err: any) {
      setTotpMsg(err.message);
    } finally {
      setTotpLoading(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto p-6">
      {loading && <div className="text-center">Loading...</div>}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">个人设置</h1>
          <p className="text-default-500">管理您的个人资料和账户安全</p>
        </div>
        <Button 
            color="primary" 
            startContent={<Save size={18} />}
            isLoading={saving}
            onPress={handleSave}
        >
            保存更改
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* 左侧概览 */}
        <Card className="h-fit">
            <CardBody className="flex flex-col items-center gap-4 py-8">
                <Avatar 
                    src={`https://i.pravatar.cc/150?u=${formData.username}`} 
                    className="w-24 h-24 text-large"
                />
                <div className="text-center">
                    <h2 className="text-xl font-bold">{formData.display_name || formData.username}</h2>
                    <p className="text-default-500">@{formData.username}</p>
                </div>
                <div className="flex gap-2">
                    <div className="px-3 py-1 bg-primary/10 text-primary rounded-full text-xs font-medium">
                        {getRoleName(formData.role)}
                    </div>
                    <div className="px-3 py-1 bg-secondary/10 text-secondary rounded-full text-xs font-medium">
                        分组: {formData.group}
                    </div>
                </div>
            </CardBody>
        </Card>

        {/* 右侧表单 */}
        <div className="md:col-span-2 space-y-6">
            <Card>
                <CardHeader>
                    <h3 className="text-lg font-semibold">基本信息</h3>
                </CardHeader>
                <Divider />
                <CardBody className="gap-4">
                    <Input
                        label="用户名"
                        value={formData.username}
                        isDisabled
                        startContent={<UserIcon className="text-default-400" size={18} />}
                    />
                    <Input
                        label="显示名称"
                        placeholder="设置您的昵称"
                        value={formData.display_name}
                        onValueChange={(v) => setFormData({...formData, display_name: v})}
                    />
                    <Input
                        label="邮箱地址"
                        placeholder="example@email.com"
                        value={formData.email}
                        onValueChange={(v) => setFormData({...formData, email: v})}
                        startContent={<Mail className="text-default-400" size={18} />}
                    />
                </CardBody>
            </Card>

            <Card>
                <CardHeader>
                    <h3 className="text-lg font-semibold">安全设置</h3>
                </CardHeader>
                <Divider />
                <CardBody className="gap-4">
                    <Input
                        label="新密码"
                        placeholder="不修改请留空"
                        type="password"
                        value={formData.password}
                        onValueChange={(v) => setFormData({...formData, password: v})}
                        startContent={<Lock className="text-default-400" size={18} />}
                        description="密码长度至少需要8位"
                    />
                </CardBody>
            </Card>

            {/* 2FA Card */}
            <Card>
                <CardHeader className="flex justify-between">
                    <h3 className="text-lg font-semibold flex items-center gap-2">
                        <Shield size={20} /> 两步验证 (2FA)
                    </h3>
                    <Chip color={formData.totp_enabled ? 'success' : 'default'} variant="flat" size="sm">
                        {formData.totp_enabled ? '已启用' : '未启用'}
                    </Chip>
                </CardHeader>
                <Divider />
                <CardBody className="gap-4">
                    {totpMsg && (
                        <Alert color={totpMsg.includes('已启用') || totpMsg.includes('已禁用') ? 'success' : 'danger'}>
                            {totpMsg}
                        </Alert>
                    )}

                    {!formData.totp_enabled && totpStep === 'idle' && (
                        <div>
                            <p className="text-default-500 text-sm mb-3">启用两步验证可以增强账户安全性。您需要一个验证器应用（如 Google Authenticator）。</p>
                            <Button
                                color="primary"
                                variant="flat"
                                startContent={<ShieldCheck size={18} />}
                                isLoading={totpLoading}
                                onPress={handleTotpSetup}
                            >
                                开始设置
                            </Button>
                        </div>
                    )}

                    {totpStep === 'setup' && (
                        <div className="space-y-4">
                            <p className="text-sm text-default-500">请使用验证器应用扫描下方二维码，或手动输入密钥URI：</p>
                            <div className="flex justify-center p-4 bg-white rounded-lg">
                                <img src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(totpUri)}`} alt="TOTP QR Code" className="w-48 h-48" />
                            </div>
                            <Input label="密钥 URI" value={totpUri} isReadOnly size="sm" />
                            {totpBackupCodes.length > 0 && (
                                <div>
                                    <p className="text-sm font-medium mb-2">备份码（请妥善保存）：</p>
                                    <div className="grid grid-cols-2 gap-1 p-3 bg-default-100 rounded-lg font-mono text-sm">
                                        {totpBackupCodes.map((code, i) => (
                                            <span key={i}>{code}</span>
                                        ))}
                                    </div>
                                </div>
                            )}
                            <div className="flex gap-2 items-end">
                                <Input label="验证码" placeholder="000000" value={totpCode} onValueChange={setTotpCode} maxLength={6} />
                                <Button color="primary" isLoading={totpLoading} onPress={handleTotpEnable}>验证并启用</Button>
                            </div>
                            <Button variant="light" size="sm" onPress={() => setTotpStep('idle')}>取消</Button>
                        </div>
                    )}

                    {formData.totp_enabled && totpStep === 'idle' && (
                        <div className="space-y-4">
                            <p className="text-sm text-success">两步验证已启用，每次登录需要输入验证码。</p>
                            <Divider />
                            <p className="text-sm font-medium">禁用两步验证</p>
                            <Input label="当前密码" type="password" value={disablePassword} onValueChange={setDisablePassword} size="sm" />
                            <Input label="验证码" placeholder="000000" value={disableCode} onValueChange={setDisableCode} maxLength={6} size="sm" />
                            <Button
                                color="danger"
                                variant="flat"
                                startContent={<ShieldOff size={18} />}
                                isLoading={totpLoading}
                                onPress={handleTotpDisable}
                            >
                                禁用两步验证
                            </Button>
                        </div>
                    )}
                </CardBody>
            </Card>
        </div>
      </div>
    </div>
  );
}
