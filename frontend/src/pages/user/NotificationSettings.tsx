import { useState, useEffect } from 'react';
import { Card, CardBody, CardHeader, Button, Input, Select, Switch } from '../../components/ui';
import { Bell, Mail, Webhook, Smartphone, Save } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import PageHeader from '../../components/PageHeader';

/**
 * 用户通知设置接口
 */
interface NotificationSettings {
  notify_type: string;                              // 通知类型：email/webhook/bark/gotify
  notification_email: string;                       // 邮件地址
  webhook_url: string;                              // Webhook URL
  webhook_secret: string;                           // Webhook 密钥（用于 HMAC 签名）
  bark_url: string;                                 // Bark 推送 URL
  gotify_url: string;                               // Gotify 服务器地址
  gotify_token: string;                             // Gotify Token
  gotify_priority: number;                          // Gotify 优先级 (1-10)
  upstream_model_update_notify_enabled: boolean;    // 上游模型更新通知
}

export default function NotificationSettings() {
  const [settings, setSettings] = useState<NotificationSettings>({
    notify_type: 'email',
    notification_email: '',
    webhook_url: '',
    webhook_secret: '',
    bark_url: '',
    gotify_url: '',
    gotify_token: '',
    gotify_priority: 5,
    upstream_model_update_notify_enabled: false,
  });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { token } = useAuthStore();

  /**
   * 获取通知设置
   */
  const fetchSettings = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/user/notification-settings', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data) {
        setSettings(data.data);
      }
    } catch (error) {
      console.error('获取通知设置失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (token) fetchSettings();
  }, [token]);

  /**
   * 保存通知设置
   */
  const saveSettings = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/user/notification-settings', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
      });
      const data = await res.json();
      if (data.code === 0) {
        alert('保存成功');
      } else {
        alert(`保存失败: ${data.message}`);
      }
    } catch (error) {
      alert('保存失败');
      console.error(error);
    } finally {
      setSaving(false);
    }
  };

  /**
   * 测试通知
   */
  const testNotification = async () => {
    try {
      const res = await fetch('/api/user/notification-test', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        alert('测试通知已发送，请检查接收情况');
      } else {
        alert(`发送失败: ${data.message}`);
      }
    } catch (error) {
      alert('发送失败');
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="通知设置"
        description="配置系统通知接收方式"
        action={
          <div className="flex gap-2">
            <Button onClick={testNotification} variant="outline">
              <Bell className="w-4 h-4 mr-2" />
              测试通知
            </Button>
            <Button onClick={saveSettings} disabled={saving}>
              <Save className="w-4 h-4 mr-2" />
              {saving ? '保存中...' : '保存设置'}
            </Button>
          </div>
        }
      />

      {/* 通知类型选择 */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">通知方式</h3>
        </CardHeader>
        <CardBody>
          <Select
            value={settings.notify_type}
            onChange={(e) => setSettings({ ...settings, notify_type: e.target.value })}
          >
            <option value="email">邮件通知</option>
            <option value="webhook">Webhook</option>
            <option value="bark">Bark (iOS)</option>
            <option value="gotify">Gotify (自托管)</option>
          </Select>
        </CardBody>
      </Card>

      {/* 邮件通知配置 */}
      {settings.notify_type === 'email' && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Mail className="w-5 h-5" />
              <h3 className="text-lg font-semibold">邮件通知</h3>
            </div>
          </CardHeader>
          <CardBody>
            <div>
              <label className="block text-sm font-medium mb-1">邮箱地址</label>
              <Input
                type="email"
                value={settings.notification_email}
                onChange={(e) => setSettings({ ...settings, notification_email: e.target.value })}
                placeholder="your@email.com"
              />
            </div>
          </CardBody>
        </Card>
      )}

      {/* Webhook 配置 */}
      {settings.notify_type === 'webhook' && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Webhook className="w-5 h-5" />
              <h3 className="text-lg font-semibold">Webhook 通知</h3>
            </div>
          </CardHeader>
          <CardBody>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Webhook URL</label>
                <Input
                  value={settings.webhook_url}
                  onChange={(e) => setSettings({ ...settings, webhook_url: e.target.value })}
                  placeholder="https://your-server.com/webhook"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">密钥 (可选)</label>
                <Input
                  type="password"
                  value={settings.webhook_secret}
                  onChange={(e) => setSettings({ ...settings, webhook_secret: e.target.value })}
                  placeholder="用于 HMAC-SHA256 签名验证"
                />
                <p className="text-xs text-gray-500 mt-1">
                  设置后将在请求头 X-Webhook-Signature 中包含 HMAC-SHA256 签名
                </p>
              </div>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Bark 配置 */}
      {settings.notify_type === 'bark' && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Smartphone className="w-5 h-5" />
              <h3 className="text-lg font-semibold">Bark 推送 (iOS)</h3>
            </div>
          </CardHeader>
          <CardBody>
            <div>
              <label className="block text-sm font-medium mb-1">Bark URL</label>
              <Input
                value={settings.bark_url}
                onChange={(e) => setSettings({ ...settings, bark_url: e.target.value })}
                placeholder="https://api.day.app/your_key/"
              />
              <p className="text-xs text-gray-500 mt-1">
                从 Bark App 中获取推送 URL
              </p>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Gotify 配置 */}
      {settings.notify_type === 'gotify' && (
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Bell className="w-5 h-5" />
              <h3 className="text-lg font-semibold">Gotify 推送</h3>
            </div>
          </CardHeader>
          <CardBody>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">Gotify 服务器地址</label>
                <Input
                  value={settings.gotify_url}
                  onChange={(e) => setSettings({ ...settings, gotify_url: e.target.value })}
                  placeholder="https://gotify.example.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">应用 Token</label>
                <Input
                  type="password"
                  value={settings.gotify_token}
                  onChange={(e) => setSettings({ ...settings, gotify_token: e.target.value })}
                  placeholder="从 Gotify 应用中获取"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">优先级 (1-10)</label>
                <Input
                  type="number"
                  min="1"
                  max="10"
                  value={settings.gotify_priority}
                  onChange={(e) => setSettings({ ...settings, gotify_priority: parseInt(e.target.value) || 5 })}
                />
              </div>
            </div>
          </CardBody>
        </Card>
      )}

      {/* 通知选项 */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">通知选项</h3>
        </CardHeader>
        <CardBody>
          <div className="flex items-center justify-between">
            <div>
              <div className="font-medium">上游模型更新通知</div>
              <div className="text-sm text-gray-500">当上游提供商发布新模型时接收通知</div>
            </div>
            <Switch
              checked={settings.upstream_model_update_notify_enabled}
              onChange={(checked) => setSettings({ ...settings, upstream_model_update_notify_enabled: checked })}
            />
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
