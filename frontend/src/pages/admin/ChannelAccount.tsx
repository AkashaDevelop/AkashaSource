import { useState, useEffect } from 'react';
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Input,
  Switch,
  Chip,
  Tooltip,
  Progress,
  Tabs,
  Tab,
} from '../../components/ui';
import {
  CalendarCheck,
  Wallet,
  RefreshCw,
  Settings,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Play,
  Pause,
  LogIn,
  Eye,
  EyeOff,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import LoadingRows from '../../components/LoadingRows';

interface Channel {
  id: number;
  name: string;
  type: number;
  base_url: string;
  status: number;
  balance: number;
  checkin_enabled: number;
  last_checkin_at: number;
  balance_refresh_at: number;
  account_username: string;
  account_password: string;
  access_token: string;
  platform_user_id: number;
}

interface CheckinLog {
  id: number;
  channel_id: number;
  channel_name?: string;
  status: string;
  message: string;
  reward: string;
  created_at: number;
}

interface BalanceLog {
  id: number;
  channel_id: number;
  channel_name?: string;
  balance: number;
  used: number;
  quota: number;
  message: string;
  created_at: number;
}

interface SchedulerStatus {
  running: boolean;
  checkin_cron: string;
  balance_refresh_cron: string;
  checkin_interval_hours: number;
}

const CHANNEL_TYPES: Record<number, string> = {
  1: 'OpenAI',
  3: 'Azure',
  8: 'Custom',
  14: 'Claude',
  18: 'Gemini',
  40: '通义千问',
  44: 'Deepseek',
  45: '智谱 ChatGLM',
  46: 'Moonshot',
  52: 'SiliconFlow',
};

export default function ChannelAccountManagement() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [checkinLogs, setCheckinLogs] = useState<CheckinLog[]>([]);
  const [balanceLogs, setBalanceLogs] = useState<BalanceLog[]>([]);
  const [schedulerStatus, setSchedulerStatus] = useState<SchedulerStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [triggering, setTriggering] = useState<string | null>(null);
  const { token } = useAuthStore();

  const { isOpen: isConfigOpen, onOpen: onConfigOpen, onOpenChange: onConfigOpenChange } = useDisclosure();
  const { isOpen: isEditOpen, onOpen: onEditOpen, onOpenChange: onEditOpenChange } = useDisclosure();
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [showToken, setShowToken] = useState(false);

  const [configForm, setConfigForm] = useState({
    checkin_interval_hours: 24,
    checkin_cron: '0 8 * * *',
    balance_refresh_cron: '0 */6 * * *',
  });

  const [editForm, setEditForm] = useState({
    account_username: '',
    account_password: '',
    access_token: '',
    platform_user_id: 0,
    checkin_enabled: false,
  });

  const [activeTab, setActiveTab] = useState<string>('channels');

  const fetchChannels = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/channel', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setChannels(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch channels:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchSchedulerStatus = async () => {
    try {
      const res = await fetch('/api/channel/scheduler/status', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setSchedulerStatus(data.data);
        setConfigForm({
          checkin_interval_hours: data.data.checkin_interval_hours || 24,
          checkin_cron: data.data.checkin_cron || '0 8 * * *',
          balance_refresh_cron: data.data.balance_refresh_cron || '0 */6 * * *',
        });
      }
    } catch (error) {
      console.error('Failed to fetch scheduler status:', error);
    }
  };

  const fetchCheckinLogs = async () => {
    setLogsLoading(true);
    try {
      const res = await fetch('/api/channel/checkin/logs?limit=50', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setCheckinLogs(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch checkin logs:', error);
    } finally {
      setLogsLoading(false);
    }
  };

  const fetchBalanceLogs = async () => {
    setLogsLoading(true);
    try {
      const res = await fetch('/api/channel/balance/logs?limit=50', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        setBalanceLogs(data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch balance logs:', error);
    } finally {
      setLogsLoading(false);
    }
  };

  useEffect(() => {
    fetchChannels();
    fetchSchedulerStatus();
  }, []);

  useEffect(() => {
    if (activeTab === 'checkin-logs') {
      fetchCheckinLogs();
    } else if (activeTab === 'balance-logs') {
      fetchBalanceLogs();
    }
  }, [activeTab]);

  const handleTriggerCheckin = async (channelId?: number) => {
    const key = channelId ? `checkin-${channelId}` : 'checkin-all';
    setTriggering(key);
    try {
      const url = channelId
        ? `/api/channel/checkin/trigger/${channelId}`
        : '/api/channel/checkin/trigger';
      const res = await fetch(url, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        if (channelId) {
          const result = data.data;
          if (result.success) {
            toast.success(result.message || '签到成功');
          } else {
            toast.warning(result.message || '签到失败');
          }
        } else {
          toast.success(`签到完成: 成功 ${data.data.success}, 跳过 ${data.data.skipped}, 失败 ${data.data.failed}`);
        }
        fetchChannels();
      } else {
        toast.error(data.msg || '签到失败');
      }
    } catch (error) {
      console.error('Checkin error:', error);
      toast.error('签到请求失败');
    } finally {
      setTriggering(null);
    }
  };

  const handleTriggerBalanceRefresh = async (channelId?: number) => {
    const key = channelId ? `balance-${channelId}` : 'balance-all';
    setTriggering(key);
    try {
      const url = channelId
        ? `/api/channel/balance/refresh/${channelId}`
        : '/api/channel/balance/refresh';
      const res = await fetch(url, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        if (channelId) {
          toast.success(`余额: $${data.data.balance?.toFixed(2) || '0.00'}`);
        } else {
          toast.success(`余额刷新完成: 成功 ${data.data.success}, 失败 ${data.data.failed}`);
        }
        fetchChannels();
      } else {
        toast.error(data.msg || '余额刷新失败');
      }
    } catch (error) {
      console.error('Balance refresh error:', error);
      toast.error('余额刷新请求失败');
    } finally {
      setTriggering(null);
    }
  };

  const handleTestLogin = async (channelId: number) => {
    setTriggering(`login-${channelId}`);
    try {
      const res = await fetch(`/api/channel/login/${channelId}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0 && data.data?.success) {
        toast.success('登录测试成功');
        fetchChannels();
      } else {
        toast.error(data.msg || '登录测试失败');
      }
    } catch (error) {
      console.error('Login test error:', error);
      toast.error('登录测试请求失败');
    } finally {
      setTriggering(null);
    }
  };

  const handleStartScheduler = async () => {
    setTriggering('scheduler-start');
    try {
      const res = await fetch('/api/channel/scheduler/start', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('调度器已启动');
        fetchSchedulerStatus();
      } else {
        toast.error(data.msg || '启动失败');
      }
    } catch (error) {
      console.error('Start scheduler error:', error);
      toast.error('启动请求失败');
    } finally {
      setTriggering(null);
    }
  };

  const handleStopScheduler = async () => {
    setTriggering('scheduler-stop');
    try {
      const res = await fetch('/api/channel/scheduler/stop', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('调度器已停止');
        fetchSchedulerStatus();
      } else {
        toast.error(data.msg || '停止失败');
      }
    } catch (error) {
      console.error('Stop scheduler error:', error);
      toast.error('停止请求失败');
    } finally {
      setTriggering(null);
    }
  };

  const handleEditChannel = (channel: Channel) => {
    setEditingChannel(channel);
    setEditForm({
      account_username: channel.account_username || '',
      account_password: '',
      access_token: channel.access_token || '',
      platform_user_id: channel.platform_user_id || 0,
      checkin_enabled: channel.checkin_enabled === 1,
    });
    setShowPassword(false);
    setShowToken(false);
    onEditOpen();
  };

  const handleSaveChannel = async () => {
    if (!editingChannel) return;
    try {
      const res = await fetch(`/api/channel/account/${editingChannel.id}`, {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...editForm,
          checkin_enabled: editForm.checkin_enabled ? 1 : 0,
        }),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('保存成功');
        onEditOpenChange();
        fetchChannels();
      } else {
        toast.error(data.msg || '保存失败');
      }
    } catch (error) {
      console.error('Save error:', error);
      toast.error('保存请求失败');
    }
  };

  const handleUpdateSchedulerConfig = async () => {
    try {
      const res = await fetch('/api/channel/scheduler/config', {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(configForm),
      });
      const data = await res.json();
      if (data.code === 0) {
        toast.success('调度器配置已更新');
        onConfigOpenChange();
        fetchSchedulerStatus();
      } else {
        toast.error(data.msg || '更新失败');
      }
    } catch (error) {
      console.error('Update config error:', error);
      toast.error('更新配置请求失败');
    }
  };

  const formatTime = (timestamp: number) => {
    if (!timestamp) return '-';
    return new Date(timestamp * 1000).toLocaleString('zh-CN');
  };

  const formatBalance = (balance: number) => {
    if (balance === undefined || balance === null) return '-';
    return `$${balance.toFixed(2)}`;
  };

  const getStatusColor = (status: number) => {
    switch (status) {
      case 1: return 'success';
      case 2: return 'danger';
      default: return 'default';
    }
  };

  const getStatusLabel = (status: number) => {
    switch (status) {
      case 1: return '启用';
      case 2: return '禁用';
      default: return '未知';
    }
  };

  const getCheckinStatusIcon = (status: string) => {
    switch (status) {
      case 'success': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'already': return <Clock className="w-4 h-4 text-yellow-500" />;
      case 'unsupported': return <AlertTriangle className="w-4 h-4 text-gray-500" />;
      default: return <XCircle className="w-4 h-4 text-red-500" />;
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="渠道账号管理"
        subtitle="管理渠道签到和余额监控"
        actions={
          <div className="flex gap-2">
            <Button
              color="primary"
              variant="flat"
              startContent={<Settings className="w-4 h-4" />}
              onPress={onConfigOpen}
            >
              调度配置
            </Button>
            <Button
              color="success"
              variant="flat"
              startContent={<RefreshCw className={`w-4 h-4 ${triggering === 'balance-all' ? 'animate-spin' : ''}`} />}
              onPress={() => handleTriggerBalanceRefresh()}
              isDisabled={!!triggering}
            >
              刷新全部余额
            </Button>
            <Button
              color="warning"
              variant="flat"
              startContent={<CalendarCheck className={`w-4 h-4 ${triggering === 'checkin-all' ? 'animate-spin' : ''}`} />}
              onPress={() => handleTriggerCheckin()}
              isDisabled={!!triggering}
            >
              全部签到
            </Button>
          </div>
        }
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardBody className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="p-3 rounded-xl bg-success-100 dark:bg-success-900/30">
                <Play className="w-6 h-6 text-success-500" />
              </div>
              <div>
                <p className="text-sm text-default-500">调度器状态</p>
                <p className="text-lg font-semibold">
                  {schedulerStatus?.running ? (
                    <span className="text-success-500">运行中</span>
                  ) : (
                    <span className="text-danger-500">已停止</span>
                  )}
                </p>
              </div>
            </div>
            {schedulerStatus?.running ? (
              <Button
                color="danger"
                variant="flat"
                size="sm"
                startContent={<Pause size={16} />}
                isLoading={triggering === 'scheduler-stop'}
                onPress={handleStopScheduler}
              >
                停止
              </Button>
            ) : (
              <Button
                color="success"
                variant="flat"
                size="sm"
                startContent={<Play size={16} />}
                isLoading={triggering === 'scheduler-start'}
                onPress={handleStartScheduler}
              >
                启动
              </Button>
            )}
          </CardBody>
        </Card>

        <Card>
          <CardBody className="flex items-center gap-4">
            <div className="p-3 rounded-xl bg-warning-100 dark:bg-warning-900/30">
              <CalendarCheck className="w-6 h-6 text-warning-500" />
            </div>
            <div>
              <p className="text-sm text-default-500">签到间隔</p>
              <p className="text-lg font-semibold">{schedulerStatus?.checkin_interval_hours || 24} 小时</p>
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody className="flex items-center gap-4">
            <div className="p-3 rounded-xl bg-primary-100 dark:bg-primary-900/30">
              <Wallet className="w-6 h-6 text-primary-500" />
            </div>
            <div>
              <p className="text-sm text-default-500">余额刷新间隔</p>
              <p className="text-lg font-semibold">6 小时</p>
            </div>
          </CardBody>
        </Card>
      </div>

      <Tabs
        selectedKey={activeTab}
        onSelectionChange={(key) => setActiveTab(key as string)}
        variant="underlined"
        classNames={{
          tabList: 'gap-6',
          tab: 'text-lg',
        }}
      >
        <Tab
          key="channels"
          title={
            <div className="flex items-center gap-2">
              <Server className="w-4 h-4" />
              <span>渠道列表</span>
            </div>
          }
        />
        <Tab
          key="checkin-logs"
          title={
            <div className="flex items-center gap-2">
              <CalendarCheck className="w-4 h-4" />
              <span>签到日志</span>
            </div>
          }
        />
        <Tab
          key="balance-logs"
          title={
            <div className="flex items-center gap-2">
              <Wallet className="w-4 h-4" />
              <span>余额日志</span>
            </div>
          }
        />
      </Tabs>

      {activeTab === 'channels' && (
        <Card>
          <CardHeader className="flex justify-between">
            <h3 className="text-lg font-semibold">渠道账号配置</h3>
            <Button
              size="sm"
              variant="light"
              startContent={<RefreshCw className="w-4 h-4" />}
              onPress={fetchChannels}
            >
              刷新
            </Button>
          </CardHeader>
          <CardBody>
            {loading ? (
              <LoadingRows rows={5} />
            ) : channels.length === 0 ? (
              <EmptyState message="暂无渠道数据" />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-divider">
                      <th className="text-left py-3 px-4 font-medium">渠道名称</th>
                      <th className="text-left py-3 px-4 font-medium">类型</th>
                      <th className="text-left py-3 px-4 font-medium">状态</th>
                      <th className="text-left py-3 px-4 font-medium">余额</th>
                      <th className="text-left py-3 px-4 font-medium">签到</th>
                      <th className="text-left py-3 px-4 font-medium">上次签到</th>
                      <th className="text-left py-3 px-4 font-medium">上次刷新</th>
                      <th className="text-right py-3 px-4 font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {channels.map((channel) => (
                      <tr key={channel.id} className="border-b border-divider hover:bg-default-50 dark:hover:bg-default-100/5">
                        <td className="py-3 px-4">
                          <div className="font-medium">{channel.name}</div>
                          <div className="text-xs text-default-400 truncate max-w-[200px]">{channel.base_url}</div>
                        </td>
                        <td className="py-3 px-4">
                          <Chip size="sm" variant="flat">
                            {CHANNEL_TYPES[channel.type] || '未知'}
                          </Chip>
                        </td>
                        <td className="py-3 px-4">
                          <Chip size="sm" color={getStatusColor(channel.status) as any} variant="flat">
                            {getStatusLabel(channel.status)}
                          </Chip>
                        </td>
                        <td className="py-3 px-4 font-mono">
                          {formatBalance(channel.balance)}
                        </td>
                        <td className="py-3 px-4">
                          <Chip
                            size="sm"
                            color={channel.checkin_enabled ? 'success' : 'default'}
                            variant="flat"
                          >
                            {channel.checkin_enabled ? '已启用' : '未启用'}
                          </Chip>
                        </td>
                        <td className="py-3 px-4 text-default-500">
                          {formatTime(channel.last_checkin_at)}
                        </td>
                        <td className="py-3 px-4 text-default-500">
                          {formatTime(channel.balance_refresh_at)}
                        </td>
                        <td className="py-3 px-4">
                          <div className="flex justify-end gap-1">
                            <Tooltip content="编辑账号">
                              <Button
                                isIconOnly
                                size="sm"
                                variant="light"
                                onPress={() => handleEditChannel(channel)}
                              >
                                <Settings className="w-4 h-4" />
                              </Button>
                            </Tooltip>
                            <Tooltip content="测试登录">
                              <Button
                                isIconOnly
                                size="sm"
                                variant="light"
                                color="primary"
                                isLoading={triggering === `login-${channel.id}`}
                                onPress={() => handleTestLogin(channel.id)}
                              >
                                <LogIn className="w-4 h-4" />
                              </Button>
                            </Tooltip>
                            <Tooltip content="刷新余额">
                              <Button
                                isIconOnly
                                size="sm"
                                variant="light"
                                color="success"
                                isLoading={triggering === `balance-${channel.id}`}
                                onPress={() => handleTriggerBalanceRefresh(channel.id)}
                              >
                                <Wallet className="w-4 h-4" />
                              </Button>
                            </Tooltip>
                            <Tooltip content="签到">
                              <Button
                                isIconOnly
                                size="sm"
                                variant="light"
                                color="warning"
                                isLoading={triggering === `checkin-${channel.id}`}
                                onPress={() => handleTriggerCheckin(channel.id)}
                              >
                                <CalendarCheck className="w-4 h-4" />
                              </Button>
                            </Tooltip>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {activeTab === 'checkin-logs' && (
        <Card>
          <CardHeader className="flex justify-between">
            <h3 className="text-lg font-semibold">签到日志</h3>
            <Button
              size="sm"
              variant="light"
              startContent={<RefreshCw className="w-4 h-4" />}
              onPress={fetchCheckinLogs}
            >
              刷新
            </Button>
          </CardHeader>
          <CardBody>
            {logsLoading ? (
              <LoadingRows rows={5} />
            ) : checkinLogs.length === 0 ? (
              <EmptyState message="暂无签到日志" />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-divider">
                      <th className="text-left py-3 px-4 font-medium">渠道</th>
                      <th className="text-left py-3 px-4 font-medium">状态</th>
                      <th className="text-left py-3 px-4 font-medium">消息</th>
                      <th className="text-left py-3 px-4 font-medium">奖励</th>
                      <th className="text-left py-3 px-4 font-medium">时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {checkinLogs.map((log) => (
                      <tr key={log.id} className="border-b border-divider">
                        <td className="py-3 px-4">{log.channel_name || `渠道 ${log.channel_id}`}</td>
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-2">
                            {getCheckinStatusIcon(log.status)}
                            <span className="capitalize">{log.status}</span>
                          </div>
                        </td>
                        <td className="py-3 px-4 text-default-500">{log.message}</td>
                        <td className="py-3 px-4">
                          {log.reward && <Chip size="sm" color="success" variant="flat">{log.reward}</Chip>}
                        </td>
                        <td className="py-3 px-4 text-default-500">{formatTime(log.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      {activeTab === 'balance-logs' && (
        <Card>
          <CardHeader className="flex justify-between">
            <h3 className="text-lg font-semibold">余额日志</h3>
            <Button
              size="sm"
              variant="light"
              startContent={<RefreshCw className="w-4 h-4" />}
              onPress={fetchBalanceLogs}
            >
              刷新
            </Button>
          </CardHeader>
          <CardBody>
            {logsLoading ? (
              <LoadingRows rows={5} />
            ) : balanceLogs.length === 0 ? (
              <EmptyState message="暂无余额日志" />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-divider">
                      <th className="text-left py-3 px-4 font-medium">渠道</th>
                      <th className="text-left py-3 px-4 font-medium">余额</th>
                      <th className="text-left py-3 px-4 font-medium">已用</th>
                      <th className="text-left py-3 px-4 font-medium">配额</th>
                      <th className="text-left py-3 px-4 font-medium">消息</th>
                      <th className="text-left py-3 px-4 font-medium">时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {balanceLogs.map((log) => (
                      <tr key={log.id} className="border-b border-divider">
                        <td className="py-3 px-4">{log.channel_name || `渠道 ${log.channel_id}`}</td>
                        <td className="py-3 px-4 font-mono text-success-500">${log.balance?.toFixed(2) || '-'}</td>
                        <td className="py-3 px-4 font-mono">${log.used?.toFixed(2) || '-'}</td>
                        <td className="py-3 px-4 font-mono">{log.quota?.toFixed(0) || '-'}</td>
                        <td className="py-3 px-4 text-default-500">{log.message || '-'}</td>
                        <td className="py-3 px-4 text-default-500">{formatTime(log.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardBody>
        </Card>
      )}

      <Modal isOpen={isConfigOpen} onOpenChange={onConfigOpenChange}>
        <ModalContent>
          <ModalHeader>调度器配置</ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <Input
                type="number"
                label="签到间隔（小时）"
                value={configForm.checkin_interval_hours.toString()}
                onChange={(e) => setConfigForm({ ...configForm, checkin_interval_hours: parseInt(e.target.value) || 24 })}
                description="自动签到的间隔时间"
              />
              <Input
                label="签到 Cron 表达式"
                value={configForm.checkin_cron}
                onChange={(e) => setConfigForm({ ...configForm, checkin_cron: e.target.value })}
                description="例如: 0 8 * * * 表示每天早上8点签到"
              />
              <Input
                label="余额刷新 Cron 表达式"
                value={configForm.balance_refresh_cron}
                onChange={(e) => setConfigForm({ ...configForm, balance_refresh_cron: e.target.value })}
                description="例如: 0 */6 * * * 表示每6小时刷新一次"
              />
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onConfigOpenChange}>取消</Button>
            <Button color="primary" onPress={handleUpdateSchedulerConfig}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <Modal isOpen={isEditOpen} onOpenChange={onEditOpenChange}>
        <ModalContent>
          <ModalHeader>编辑渠道账号 - {editingChannel?.name}</ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <Input
                label="账号用户名"
                value={editForm.account_username}
                onChange={(e) => setEditForm({ ...editForm, account_username: e.target.value })}
                description="用于自动登录获取 Token"
              />
              <Input
                type={showPassword ? 'text' : 'password'}
                label="账号密码"
                value={editForm.account_password}
                onChange={(e) => setEditForm({ ...editForm, account_password: e.target.value })}
                endContent={
                  <Button
                    isIconOnly
                    size="sm"
                    variant="light"
                    onPress={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </Button>
                }
                description="用于自动登录获取 Token"
              />
              <Input
                type={showToken ? 'text' : 'password'}
                label="Access Token"
                value={editForm.access_token}
                onChange={(e) => setEditForm({ ...editForm, access_token: e.target.value })}
                endContent={
                  <Button
                    isIconOnly
                    size="sm"
                    variant="light"
                    onPress={() => setShowToken(!showToken)}
                  >
                    {showToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                  </Button>
                }
                description="API 访问令牌，可手动填写或通过登录获取"
              />
              <Input
                type="number"
                label="平台用户 ID"
                value={editForm.platform_user_id.toString()}
                onChange={(e) => setEditForm({ ...editForm, platform_user_id: parseInt(e.target.value) || 0 })}
                description="部分平台需要指定用户 ID"
              />
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">启用自动签到</p>
                  <p className="text-sm text-default-500">开启后将按设定间隔自动签到</p>
                </div>
                <Switch
                  isSelected={editForm.checkin_enabled}
                  onValueChange={(value) => setEditForm({ ...editForm, checkin_enabled: value })}
                />
              </div>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={onEditOpenChange}>取消</Button>
            <Button color="primary" onPress={handleSaveChannel}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}

function Server({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <rect x="2" y="2" width="20" height="8" rx="2" />
      <rect x="2" y="14" width="20" height="8" rx="2" />
      <line x1="6" y1="6" x2="6.01" y2="6" />
      <line x1="6" y1="18" x2="6.01" y2="18" />
    </svg>
  );
}
