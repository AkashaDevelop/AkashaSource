import { useState, useEffect, useMemo } from 'react';
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
} from '@heroui/react';
import { Save, Plus, Edit, Trash2, RotateCcw } from 'lucide-react';
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
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const { token } = useAuthStore();
  const [message, setMessage] = useState<{ type: 'success' | 'danger'; text: string } | null>(null);

  // Pricing State
  const [pricingItems, setPricingItems] = useState<PricingItem[]>([]);
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingItem, setEditingItem] = useState<PricingItem | null>(null);
  const [itemForm, setItemForm] = useState<PricingItem>({ model: '', ratio: 0, completion_ratio: 0 });

  const fetchSettings = async () => {
    setLoading(true);
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
      console.error('Failed to fetch settings:', error);
    } finally {
      setLoading(false);
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
      console.error("Error parsing pricing JSON", e);
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
      { key: 'system_name', value: settings['system_name'] || 'STfreApi' },
      { key: 'logo_url', value: settings['logo_url'] || '' },
      { key: 'model_ratio', value: JSON.stringify(modelRatio) },
      { key: 'completion_ratio', value: JSON.stringify(completionRatio) },
      
      // OAuth
      { key: 'github_client_id', value: settings['github_client_id'] || '' },
      { key: 'github_client_secret', value: settings['github_client_secret'] || '' },
      { key: 'linuxdo_client_id', value: settings['linuxdo_client_id'] || '' },
      { key: 'linuxdo_client_secret', value: settings['linuxdo_client_secret'] || '' },
      
      // Email
      { key: 'smtp_server', value: settings['smtp_server'] || '' },
      { key: 'smtp_port', value: settings['smtp_port'] || '587' },
      { key: 'smtp_account', value: settings['smtp_account'] || '' },
      { key: 'smtp_password', value: settings['smtp_password'] || '' },
      { key: 'smtp_from', value: settings['smtp_from'] || '' },

      // Turnstile
      { key: 'turnstile_site_key', value: settings['turnstile_site_key'] || '' },
      { key: 'turnstile_secret_key', value: settings['turnstile_secret_key'] || '' },
      { key: 'turnstile_check_enabled', value: settings['turnstile_check_enabled'] || 'false' },

      // LinuxDO Quota
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
        body: JSON.stringify({ options }),
      });
      const data = await res.json();
      if (res.ok) {
        setMessage({ type: 'success', text: 'Settings saved successfully' });
      } else {
        setMessage({ type: 'danger', text: data.error || 'Failed to save settings' });
      }
    } catch (error: any) {
      setMessage({ type: 'danger', text: error.message });
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
        <CardHeader className="font-bold text-xl">System Settings</CardHeader>
        <CardBody>
          {message && (
            <Alert color={message.type} className="mb-4">
              {message.text}
            </Alert>
          )}
          
          <Form onSubmit={handleSave} className="space-y-4">
            <h3 className="text-lg font-semibold">General</h3>
            <Input
              label="System Name"
              value={settings['system_name'] || ''}
              onValueChange={(val) => updateSetting('system_name', val)}
            />
            <Input
              label="Logo URL"
              value={settings['logo_url'] || ''}
              onValueChange={(val) => updateSetting('logo_url', val)}
            />
            <Input
              label="Notice"
              value={settings['notice'] || ''}
              onValueChange={(val) => updateSetting('notice', val)}
            />
            <Input
              label="Topup Link"
              value={settings['topup_link'] || ''}
              onValueChange={(val) => updateSetting('topup_link', val)}
            />
            <Input
              label="Chat Link"
              value={settings['chat_link'] || ''}
              onValueChange={(val) => updateSetting('chat_link', val)}
            />

            <Divider className="my-2" />
            <h3 className="text-lg font-semibold">GitHub OAuth</h3>
            <Input
              label="Client ID"
              value={settings['github_client_id'] || ''}
              onValueChange={(val) => updateSetting('github_client_id', val)}
            />
            <Input
              label="Client Secret"
              type="password"
              value={settings['github_client_secret'] || ''}
              onValueChange={(val) => updateSetting('github_client_secret', val)}
            />

            <Divider className="my-2" />
            <h3 className="text-lg font-semibold">LinuxDO OAuth</h3>
            <Input
              label="Client ID"
              value={settings['linuxdo_client_id'] || ''}
              onValueChange={(val) => updateSetting('linuxdo_client_id', val)}
            />
            <Input
              label="Client Secret"
              type="password"
              value={settings['linuxdo_client_secret'] || ''}
              onValueChange={(val) => updateSetting('linuxdo_client_secret', val)}
            />
            
            <div className="grid grid-cols-2 gap-4 mt-2">
                {[0, 1, 2, 3, 4, 5].map(level => (
                    <Input
                        key={level}
                        label={`Level ${level} Initial Quota`}
                        type="number"
                        placeholder="500000 = $1"
                        value={settings[`linuxdo_quota_level_${level}`] || '0'}
                        onValueChange={(val) => updateSetting(`linuxdo_quota_level_${level}`, val)}
                    />
                ))}
            </div>

            <Divider className="my-2" />
            <h3 className="text-lg font-semibold">SMTP Email</h3>
            <Input
              label="SMTP Server"
              value={settings['smtp_server'] || ''}
              onValueChange={(val) => updateSetting('smtp_server', val)}
            />
            <div className="flex gap-4">
              <Input
                label="SMTP Port"
                value={settings['smtp_port'] || ''}
                onValueChange={(val) => updateSetting('smtp_port', val)}
                className="w-1/3"
              />
              <Input
                label="SMTP Account"
                value={settings['smtp_account'] || ''}
                onValueChange={(val) => updateSetting('smtp_account', val)}
                className="w-2/3"
              />
            </div>
            <Input
              label="SMTP Password"
              type="password"
              value={settings['smtp_password'] || ''}
              onValueChange={(val) => updateSetting('smtp_password', val)}
            />
            <Input
              label="SMTP From Address"
              value={settings['smtp_from'] || ''}
              onValueChange={(val) => updateSetting('smtp_from', val)}
            />

            <Divider className="my-2" />
            <h3 className="text-lg font-semibold">Security (Turnstile)</h3>
            <div className="flex items-center gap-2">
              <Switch 
                isSelected={settings['turnstile_check_enabled'] === 'true'} 
                onValueChange={(isSelected) => updateSetting('turnstile_check_enabled', String(isSelected))}
              >
                Enable Turnstile Check
              </Switch>
            </div>
            <Input
              label="Site Key"
              value={settings['turnstile_site_key'] || ''}
              onValueChange={(val) => updateSetting('turnstile_site_key', val)}
            />
            <Input
              label="Secret Key"
              type="password"
              value={settings['turnstile_secret_key'] || ''}
              onValueChange={(val) => updateSetting('turnstile_secret_key', val)}
            />

            <Button color="primary" type="submit" isLoading={saving} startContent={<Save size={18} />}>
              Save All Settings
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
      </div>

      {/* Edit Pricing Modal */}
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
                    isDisabled={!!editingItem} // Disable model name edit to simplify logic
                    isRequired
                  />
                  <Input 
                    label="模型倍率 (Model Ratio)" 
                    type="number" 
                    step="0.01"
                    value={itemForm.ratio.toString()}
                    onValueChange={(v) => setItemForm({...itemForm, ratio: parseFloat(v)})}
                  />
                  <Input 
                    label="补全倍率 (Completion Ratio)" 
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
