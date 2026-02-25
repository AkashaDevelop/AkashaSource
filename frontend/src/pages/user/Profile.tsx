import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Input,
  Button,
  Avatar,
  Divider,
} from '@heroui/react';
import { User as UserIcon, Mail, Lock, Save } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

export default function Profile() {
  const { token, user, updateUser } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    username: '',
    display_name: '',
    email: '',
    password: '',
    role: 1,
    group: 'default',
  });

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
        </div>
      </div>
    </div>
  );
}
