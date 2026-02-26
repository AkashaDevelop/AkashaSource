import { useState, useEffect } from 'react';
import {
  Card,
  CardHeader,
  CardBody,
  Input,
  Button,
  Form,
  Divider,
  Alert,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Tooltip,
  Switch,
  Textarea,
} from '@heroui/react';
import { Save, Plus, Edit, Trash2 } from 'lucide-react';
import { useAuthStore } from '../../store/auth';

interface Option {
  key: string;
  value: string;
}

interface PricingItem {
  model: string;
  ratio: number;
  completion_ratio: number;
}

export default function SystemSettings() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const { token } = useAuthStore();
  const [message, setMessage] = useState<{ type: 'success' | 'danger'; text: string } | null>(null);

  // Pricing State
  const [pricingItems, setPricingItems] = useState<PricingItem[]>([]);
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingItem, setEditingItem] = useState<PricingItem | null>(null);
  const [itemForm, setItemForm] = useState<PricingItem>({ model: '', ratio: 0, completion_ratio: 0 });

  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/option', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (res.ok) {
        const settingsMap: Record<string, string> = {};
        data.data.forEach((opt: Option) => {
          settingsMap[opt.key] = opt.value;
        });
        setSettings(settingsMap);
        parsePricing(settingsMap);
      }
    } catch (error) {
      console.error('获取配置失败:', error);
    }
  };

  const parsePricing = (settingsMap: Record<string, string>) => {
    try {
      const modelRatio = JSON.parse(settingsMap['model_ratio'] || '{}');
      const completionRatio = JSON.parse(settingsMap['completion_ratio'] || '{}');
      
      const allModels = new Set([...Object.keys(modelRatio), ...Object.keys(completionRatio)]);
      const items: PricingItem[] = Array.from(allModels).map(model => ({
        model,
        ratio: modelRatio[model] || 0,
        completion_ratio: completionRatio[model] || 0,
      }));
      setPricingItems(items.sort((a, b) => a.model.localeCompare(b.model)));
    } catch (e) {
      console.error("解析模型倍率失败", e);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);

    // Serialize Pricing
    const modelRatio: Record<string, number> = {};
    const completionRatio: Record<string, number> = {};
    pricingItems.forEach(item => {
      modelRatio[item.model] = item.ratio;
      completionRatio[item.model] = item.completion_ratio;
    });

    const options = [
      { key: 'system_name', value: settings['system_name'] || 'Akasha' },
      { key: 'system_url', value: settings['system_url'] || '' },
      { key: 'logo_url', value: settings['logo_url'] || '' },
      { key: 'notice', value: settings['notice'] || '' },
      { key: 'footer_html', value: settings['footer_html'] || '' },
      { key: 'topup_link', value: settings['topup_link'] || '' },
      { key: 'chat_link', value: settings['chat_link'] || '' },
      { key: 'price', value: settings['price'] || '' },
      { key: 'min_topup', value: settings['min_topup'] || '' },
      { key: 'model_ratio', value: JSON.stringify(modelRatio) },
      { key: 'completion_ratio', value: JSON.stringify(completionRatio) },
      { key: 'payment_provider', value: settings['payment_provider'] || '' },
      { key: 'epay_api_url', value: settings['epay_api_url'] || '' },
      { key: 'epay_pid', value: settings['epay_pid'] || '' },
      { key: 'epay_key', value: settings['epay_key'] || '' },
      { key: 'epay_type', value: settings['epay_type'] || '' },
      { key: 'epay_notify_url', value: settings['epay_notify_url'] || '' },
      { key: 'epay_return_url', value: settings['epay_return_url'] || '' },
      { key: 'content_moderation_enabled', value: settings['content_moderation_enabled'] || 'false' },
      { key: 'content_moderation_keywords', value: settings['content_moderation_keywords'] || '' },
      { key: 'content_moderation_api', value: settings['content_moderation_api'] || '' },
      { key: 'content_moderation_timeout', value: settings['content_moderation_timeout'] || '5' },
      { key: 'content_moderation_whitelist_users', value: settings['content_moderation_whitelist_users'] || '' },
      { key: 'content_moderation_whitelist_models', value: settings['content_moderation_whitelist_models'] || '' },
      { key: 'content_moderation_whitelist_ips', value: settings['content_moderation_whitelist_ips'] || '' },
      { key: 'redis_addr', value: settings['redis_addr'] || '' },
      { key: 'redis_password', value: settings['redis_password'] || '' },
      { key: 'redis_db', value: settings['redis_db'] || '' },
      { key: 'github_client_id', value: settings['github_client_id'] || '' },
      { key: 'github_client_secret', value: settings['github_client_secret'] || '' },
      { key: 'linuxdo_client_id', value: settings['linuxdo_client_id'] || '' },
      { key: 'linuxdo_client_secret', value: settings['linuxdo_client_secret'] || '' },
      { key: 'smtp_server', value: settings['smtp_server'] || '' },
      { key: 'smtp_port', value: settings['smtp_port'] || '587' },
      { key: 'smtp_account', value: settings['smtp_account'] || '' },
      { key: 'smtp_password', value: settings['smtp_password'] || '' },
      { key: 'smtp_from', value: settings['smtp_from'] || '' },
      { key: 'turnstile_site_key', value: settings['turnstile_site_key'] || '' },
      { key: 'turnstile_secret_key', value: settings['turnstile_secret_key'] || '' },
      { key: 'turnstile_check_enabled', value: settings['turnstile_check_enabled'] || 'false' },
      { key: 'invitation_enabled', value: settings['invitation_enabled'] || 'false' },
      { key: 'invitation_cost', value: settings['invitation_cost'] || '0' },
      { key: 'invitation_reward', value: settings['invitation_reward'] || '0' },
      { key: 'new_user_reward', value: settings['new_user_reward'] || '0' },
      { key: 'linuxdo_quota_level_0', value: settings['linuxdo_quota_level_0'] || '0' },
      { key: 'linuxdo_quota_level_1', value: settings['linuxdo_quota_level_1'] || '0' },
      { key: 'linuxdo_quota_level_2', value: settings['linuxdo_quota_level_2'] || '0' },
      { key: 'linuxdo_quota_level_3', value: settings['linuxdo_quota_level_3'] || '0' },
      { key: 'linuxdo_quota_level_4', value: settings['linuxdo_quota_level_4'] || '0' },
      { key: 'linuxdo_quota_level_5', value: settings['linuxdo_quota_level_5'] || '0' },
    ];

    try {
      const res = await fetch('/api/option', {
        method: 'PUT',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}` 
        },
        body: JSON.stringify(options),
      });
      const data = await res.json();
      if (res.ok) {
        setMessage({ type: 'success', text: '配置保存成功' });
      } else {
        setMessage({ type: 'danger', text: data.error || '配置保存失败' });
      }
    } catch (error) {
      const text = error instanceof Error ? error.message : '配置保存失败';
      setMessage({ type: 'danger', text });
    } finally {
      setSaving(false);
    }
  };

  const updateSetting = (key: string, value: string) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  };

  // Pricing Handlers
  const handleEditPricing = (item: PricingItem) => {
    setEditingItem(item);
    setItemForm({ ...item });
    onOpen();
  };

  const handleAddPricing = () => {
    setEditingItem(null);
    setItemForm({ model: '', ratio: 1, completion_ratio: 1 });
    onOpen();
  };

  const handleDeletePricing = (model: string) => {
    setPricingItems(prev => prev.filter(p => p.model !== model));
  };

  const handleSavePricingItem = (onClose: () => void) => {
    if (!itemForm.model) return;
    
    setPricingItems(prev => {
      const filtered = prev.filter(p => p.model !== (editingItem?.model || itemForm.model));
      return [...filtered, itemForm].sort((a, b) => a.model.localeCompare(b.model));
    });
    onClose();
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto pb-10">
      <Card>
        <CardHeader className="font-bold text-xl">系统设置</CardHeader>
        <CardBody>
          {message && (
            <Alert color={message.type} className="mb-4">
              {message.text}
            </Alert>
          )}
          
          <Form onSubmit={handleSave} className="space-y-6">
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">基础设置</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="系统名称"
                  value={settings['system_name'] || ''}
                  onValueChange={(val) => updateSetting('system_name', val)}
                />
                <Input
                  label="系统地址"
                  value={settings['system_url'] || ''}
                  onValueChange={(val) => updateSetting('system_url', val)}
                />
                <Input
                  label="Logo 地址"
                  value={settings['logo_url'] || ''}
                  onValueChange={(val) => updateSetting('logo_url', val)}
                />
                <Input
                  label="最低充值"
                  type="number"
                  value={settings['min_topup'] || ''}
                  onValueChange={(val) => updateSetting('min_topup', val)}
                />
                <Input
                  label="默认价格"
                  type="number"
                  value={settings['price'] || ''}
                  onValueChange={(val) => updateSetting('price', val)}
                />
                <Input
                  label="充值链接"
                  value={settings['topup_link'] || ''}
                  onValueChange={(val) => updateSetting('topup_link', val)}
                />
                <Input
                  label="对话链接"
                  value={settings['chat_link'] || ''}
                  onValueChange={(val) => updateSetting('chat_link', val)}
                />
              </div>
              <Textarea
                label="公告内容"
                value={settings['notice'] || ''}
                onValueChange={(val) => updateSetting('notice', val)}
              />
              <Textarea
                label="页脚内容"
                value={settings['footer_html'] || ''}
                onValueChange={(val) => updateSetting('footer_html', val)}
              />
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">支付配置</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="支付渠道"
                  value={settings['payment_provider'] || ''}
                  onValueChange={(val) => updateSetting('payment_provider', val)}
                  placeholder="例如：epay"
                />
                <Input
                  label="易支付 API 地址"
                  value={settings['epay_api_url'] || ''}
                  onValueChange={(val) => updateSetting('epay_api_url', val)}
                />
                <Input
                  label="易支付 PID"
                  value={settings['epay_pid'] || ''}
                  onValueChange={(val) => updateSetting('epay_pid', val)}
                />
                <Input
                  label="易支付 KEY"
                  type="password"
                  value={settings['epay_key'] || ''}
                  onValueChange={(val) => updateSetting('epay_key', val)}
                />
                <Input
                  label="易支付通道类型"
                  value={settings['epay_type'] || ''}
                  onValueChange={(val) => updateSetting('epay_type', val)}
                  placeholder="例如：alipay"
                />
                <Input
                  label="易支付回调地址"
                  value={settings['epay_notify_url'] || ''}
                  onValueChange={(val) => updateSetting('epay_notify_url', val)}
                />
                <Input
                  label="易支付同步返回地址"
                  value={settings['epay_return_url'] || ''}
                  onValueChange={(val) => updateSetting('epay_return_url', val)}
                />
              </div>
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">内容审查</h3>
              <div className="flex items-center gap-2">
                <Switch
                  isSelected={settings['content_moderation_enabled'] === 'true'}
                  onValueChange={(val) => updateSetting('content_moderation_enabled', String(val))}
                >
                  启用内容审查
                </Switch>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="审查接口地址"
                  value={settings['content_moderation_api'] || ''}
                  onValueChange={(val) => updateSetting('content_moderation_api', val)}
                />
                <Input
                  label="审查超时（秒）"
                  type="number"
                  value={settings['content_moderation_timeout'] || ''}
                  onValueChange={(val) => updateSetting('content_moderation_timeout', val)}
                />
              </div>
              <Textarea
                label="敏感词（逗号分隔）"
                value={settings['content_moderation_keywords'] || ''}
                onValueChange={(val) => updateSetting('content_moderation_keywords', val)}
              />
              <Textarea
                label="白名单用户 ID（逗号分隔）"
                value={settings['content_moderation_whitelist_users'] || ''}
                onValueChange={(val) => updateSetting('content_moderation_whitelist_users', val)}
              />
              <Textarea
                label="白名单模型（逗号分隔）"
                value={settings['content_moderation_whitelist_models'] || ''}
                onValueChange={(val) => updateSetting('content_moderation_whitelist_models', val)}
              />
              <Textarea
                label="白名单 IP（逗号分隔）"
                value={settings['content_moderation_whitelist_ips'] || ''}
                onValueChange={(val) => updateSetting('content_moderation_whitelist_ips', val)}
              />
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">Redis 缓存</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="Redis 地址"
                  value={settings['redis_addr'] || ''}
                  onValueChange={(val) => updateSetting('redis_addr', val)}
                />
                <Input
                  label="Redis 密码"
                  type="password"
                  value={settings['redis_password'] || ''}
                  onValueChange={(val) => updateSetting('redis_password', val)}
                />
                <Input
                  label="Redis 数据库"
                  type="number"
                  value={settings['redis_db'] || ''}
                  onValueChange={(val) => updateSetting('redis_db', val)}
                />
              </div>
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">GitHub OAuth 配置</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="客户端 ID"
                  value={settings['github_client_id'] || ''}
                  onValueChange={(val) => updateSetting('github_client_id', val)}
                />
                <Input
                  label="客户端密钥"
                  type="password"
                  value={settings['github_client_secret'] || ''}
                  onValueChange={(val) => updateSetting('github_client_secret', val)}
                />
              </div>
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">LinuxDO OAuth 配置</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="客户端 ID"
                  value={settings['linuxdo_client_id'] || ''}
                  onValueChange={(val) => updateSetting('linuxdo_client_id', val)}
                />
                <Input
                  label="客户端密钥"
                  type="password"
                  value={settings['linuxdo_client_secret'] || ''}
                  onValueChange={(val) => updateSetting('linuxdo_client_secret', val)}
                />
              </div>
              <div className="grid grid-cols-2 gap-4 mt-2">
                {[0, 1, 2, 3, 4, 5].map(level => (
                  <Input
                    key={level}
                    label={`等级 ${level} 初始额度`}
                    type="number"
                    placeholder="500000 = 1 美元"
                    value={settings[`linuxdo_quota_level_${level}`] || '0'}
                    onValueChange={(val) => updateSetting(`linuxdo_quota_level_${level}`, val)}
                  />
                ))}
              </div>
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">SMTP 邮件</h3>
              <Input
                label="SMTP 服务器"
                value={settings['smtp_server'] || ''}
                onValueChange={(val) => updateSetting('smtp_server', val)}
              />
              <div className="flex gap-4">
                <Input
                  label="SMTP 端口"
                  value={settings['smtp_port'] || ''}
                  onValueChange={(val) => updateSetting('smtp_port', val)}
                  className="w-1/3"
                />
                <Input
                  label="SMTP 账号"
                  value={settings['smtp_account'] || ''}
                  onValueChange={(val) => updateSetting('smtp_account', val)}
                  className="w-2/3"
                />
              </div>
              <Input
                label="SMTP 密码"
                type="password"
                value={settings['smtp_password'] || ''}
                onValueChange={(val) => updateSetting('smtp_password', val)}
              />
              <Input
                label="发件人地址"
                value={settings['smtp_from'] || ''}
                onValueChange={(val) => updateSetting('smtp_from', val)}
              />
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">邀请设置</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="flex flex-col gap-2">
                  <label className="text-sm font-medium">邀请码开关</label>
                  <Switch
                    isSelected={settings['invitation_enabled'] === 'true'}
                    onValueChange={(val) => updateSetting('invitation_enabled', String(val))}
                  >
                    注册必须使用邀请码
                  </Switch>
                </div>
                <Input
                  label="邀请码成本"
                  type="number"
                  placeholder="0"
                  value={settings['invitation_cost'] || ''}
                  onValueChange={(val) => updateSetting('invitation_cost', val)}
                  description="生成邀请码时扣除额度"
                />
                <Input
                  label="邀请者奖励"
                  type="number"
                  placeholder="0"
                  value={settings['invitation_reward'] || ''}
                  onValueChange={(val) => updateSetting('invitation_reward', val)}
                  description="被邀请用户注册时奖励"
                />
                <Input
                  label="新用户奖励"
                  type="number"
                  placeholder="0"
                  value={settings['new_user_reward'] || ''}
                  onValueChange={(val) => updateSetting('new_user_reward', val)}
                  description="新用户初始额度"
                />
              </div>
            </div>

            <Divider className="my-2" />
            <div className="space-y-3">
              <h3 className="text-lg font-semibold">安全验证</h3>
              <div className="flex items-center gap-2">
                <Switch 
                  isSelected={settings['turnstile_check_enabled'] === 'true'} 
                  onValueChange={(isSelected) => updateSetting('turnstile_check_enabled', String(isSelected))}
                >
                  启用 Turnstile 验证
                </Switch>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="Turnstile 站点密钥"
                  value={settings['turnstile_site_key'] || ''}
                  onValueChange={(val) => updateSetting('turnstile_site_key', val)}
                />
                <Input
                  label="Turnstile 私钥"
                  type="password"
                  value={settings['turnstile_secret_key'] || ''}
                  onValueChange={(val) => updateSetting('turnstile_secret_key', val)}
                />
              </div>
            </div>

            <Button color="primary" type="submit" isLoading={saving} startContent={<Save size={18} />}>
              保存全部配置
            </Button>
          </Form>
        </CardBody>
      </Card>

      {/* 模型倍率可视化设置 */}
      <Card>
          <CardHeader className="flex justify-between">
            <h3 className="text-lg font-semibold">模型倍率配置</h3>
            <Button size="sm" color="primary" variant="flat" startContent={<Plus size={16} />} onPress={handleAddPricing}>
              添加模型
            </Button>
          </CardHeader>
          <Divider />
          <CardBody>
            <Table aria-label="Pricing Table" removeWrapper>
              <TableHeader>
                <TableColumn>模型名称</TableColumn>
                <TableColumn>模型倍率</TableColumn>
                <TableColumn>补全倍率</TableColumn>
                <TableColumn>操作</TableColumn>
              </TableHeader>
              <TableBody emptyContent="暂无配置">
                {pricingItems.map((item) => (
                  <TableRow key={item.model}>
                    <TableCell className="font-bold">{item.model}</TableCell>
                    <TableCell>{item.ratio}</TableCell>
                    <TableCell>{item.completion_ratio}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Tooltip content="编辑">
                          <span className="text-lg text-default-400 cursor-pointer active:opacity-50" onClick={() => handleEditPricing(item)}>
                            <Edit size={18} />
                          </span>
                        </Tooltip>
                        <Tooltip color="danger" content="删除">
                          <span className="text-lg text-danger cursor-pointer active:opacity-50" onClick={() => handleDeletePricing(item.model)}>
                            <Trash2 size={18} />
                          </span>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardBody>
        </Card>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editingItem ? '编辑倍率' : '添加倍率'}</ModalHeader>
              <ModalBody>
                <Form className="flex flex-col gap-4">
                  <Input 
                    label="模型名称" 
                    placeholder="gpt-4" 
                    value={itemForm.model}
                    onValueChange={(v) => setItemForm({...itemForm, model: v})}
                    isDisabled={!!editingItem}
                    isRequired
                  />
                  <Input 
                    label="模型倍率" 
                    type="number" 
                    step="0.01"
                    value={itemForm.ratio.toString()}
                    onValueChange={(v) => setItemForm({...itemForm, ratio: parseFloat(v)})}
                  />
                  <Input 
                    label="补全倍率" 
                    type="number" 
                    step="0.01"
                    value={itemForm.completion_ratio.toString()}
                    onValueChange={(v) => setItemForm({...itemForm, completion_ratio: parseFloat(v)})}
                  />
                </Form>
              </ModalBody>
              <ModalFooter>
                <Button color="danger" variant="light" onPress={onClose}>取消</Button>
                <Button color="primary" onPress={() => handleSavePricingItem(onClose)}>确定</Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </div>
  );
}
