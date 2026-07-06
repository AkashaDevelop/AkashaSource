 import { useState, useEffect, type ReactNode } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import {
  Card, CardBody, CardHeader,
  Input, Button, Divider,
  Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure,
  Tooltip, Switch, Textarea, Chip,
  Tabs, Tab,
} from '../../components/ui';
import {
  Save, Plus, Edit, Trash2,
  Globe, CreditCard, Shield, Link2, Mail, Gift, CalendarCheck, Bell, Settings, BarChart3, Fingerprint, Lock, ShieldCheck, Check,
} from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Option { key: string; value: string; }
interface PricingItem { model: string; ratio: number; completion_ratio: number; }

// 实名认证场景选项
const REALNAME_SCENARIOS = [
  { key: 'model_call', label: '模型调用', desc: '调用 AI 模型前需完成实名认证' },
  { key: 'recharge', label: '账户充值', desc: '充值额度前需完成实名认证' },
  { key: 'double_blind', label: '双盲类型', desc: '仅保留认证结果，不存储任何实名信息' },
] as const;

// ── 辅助组件 ──────────────────────────────────────────────────────────────

/** 开关行：左侧标题+描述，右侧开关 */
function SettingRow({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <div className="min-w-0">
        <p className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>{title}</p>
        {description && <p className="text-xs mt-0.5 leading-relaxed" style={{ color: 'var(--text-muted)' }}>{description}</p>}
      </div>
      <div className="flex-shrink-0">{children}</div>
    </div>
  );
}

/** 自动生成的 URL 展示块 */
function UrlBlock({ label, url }: { label: string; url: string | null }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>{label}</span>
      <div className="px-3 py-2 rounded-lg border text-sm font-mono break-all select-all transition-colors"
        style={{
          background: 'var(--bg-elevated)',
          borderColor: 'var(--border-color)',
          color: url ? 'var(--text-secondary)' : 'var(--text-faint)',
        }}>
        {url || '请先设置系统 URL'}
      </div>
    </div>
  );
}

/** 分区标题图标 */
function TabIcon({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <Icon size={15} />
      {label}
    </span>
  );
}

// ── 主组件 ────────────────────────────────────────────────────────────────

export default function SystemSettings() {
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const { token } = useAuthStore();

  const [pricingItems, setPricingItems] = useState<PricingItem[]>([]);
  const { isOpen, onOpen, onOpenChange } = useDisclosure();
  const [editingItem, setEditingItem] = useState<PricingItem | null>(null);
  const [itemForm, setItemForm] = useState<PricingItem>({ model: '', ratio: 1, completion_ratio: 1 });

  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/option', { headers: { Authorization: `Bearer ${token}` } });
      const data = await res.json();
      if (data.code === 0) {
        const map: Record<string, string> = {};
        data.data.forEach((opt: Option) => { map[opt.key] = opt.value; });
        if (!map['system_url']) map['system_url'] = window.location.origin;
        setSettings(map);
        parsePricing(map);
      }
    } catch (e) { console.error('获取配置失败:', e); }
  };

  const parsePricing = (map: Record<string, string>) => {
    try {
      const mr = JSON.parse(map['model_ratio'] || '{}');
      const cr = JSON.parse(map['completion_ratio'] || '{}');
      const all = new Set([...Object.keys(mr), ...Object.keys(cr)]);
      setPricingItems(
        Array.from(all).map(m => ({ model: m, ratio: mr[m] || 0, completion_ratio: cr[m] || 0 }))
          .sort((a, b) => a.model.localeCompare(b.model))
      );
    } catch (e) { console.error('解析倍率失败', e); }
  };

  useEffect(() => { fetchSettings(); }, []);

  const set = (key: string, value: string) => setSettings(prev => ({ ...prev, [key]: value }));
  const get = (key: string, fallback = '') => settings[key] ?? fallback;
  const systemUrl = get('system_url');

  // 解析实名认证场景（JSON 数组字符串）
  let realnameScenarios: string[] = [];
  try { realnameScenarios = JSON.parse(get('realname_scenarios', '[]')) || []; } catch { realnameScenarios = []; }
  const toggleRealnameScenario = (key: string) => {
    const next = realnameScenarios.includes(key)
      ? realnameScenarios.filter(s => s !== key)
      : [...realnameScenarios, key];
    set('realname_scenarios', JSON.stringify(next));
  };

  const handleSave = async () => {
    setSaving(true);
    const mr: Record<string, number> = {};
    const cr: Record<string, number> = {};
    pricingItems.forEach(i => { mr[i.model] = i.ratio; cr[i.model] = i.completion_ratio; });

    const keys = [
      'system_name','system_url','logo_url','notice','footer_html','chat_link','chat_link2',
      'price','min_topup',
      'payment_provider','epay_api_url','epay_pid','epay_key','epay_type','enable_topup',
      'stripe_secret_key','stripe_webhook_secret','stripe_currency','stripe_success_url','stripe_cancel_url',
      'creem_api_key','creem_webhook_secret','creem_product_id','creem_products','creem_success_url','creem_test_mode',
      'content_moderation_enabled','content_moderation_keywords',
      'content_moderation_timeout','content_moderation_whitelist_users',
      'content_moderation_whitelist_models','content_moderation_whitelist_ips',
      'tencent_moderation_secret_id','tencent_moderation_secret_key',
      'tencent_moderation_region','tencent_moderation_biz_type',
      'redis_addr','redis_password','redis_db',
      'github_client_id','github_client_secret',
      'linuxdo_client_id','linuxdo_client_secret',
      'linuxdo_quota_level_0','linuxdo_quota_level_1','linuxdo_quota_level_2',
      'linuxdo_quota_level_3','linuxdo_quota_level_4','linuxdo_quota_level_5',
      'discord_client_id','discord_client_secret',
      'oidc_client_id','oidc_client_secret','oidc_issuer_url',
      'telegram_bot_token','wechat_app_id','wechat_app_secret',
      'smtp_server','smtp_port','smtp_account','smtp_password','smtp_from',
      'smtp_ssl_enabled','email_verification_enabled',
      'cxsec_enabled','cxsec_protected_paths','qingyuan_enabled',
      'turnstile_site_key','turnstile_secret_key','turnstile_check_enabled',
      'captcha_provider','geetest_enabled','geetest_id','geetest_key',
      'hcaptcha_enabled','hcaptcha_site_key','hcaptcha_secret_key',
      'recaptcha_enabled','recaptcha_version','recaptcha_site_key','recaptcha_secret_key',
      'invitation_enabled','invitation_cost','invitation_reward','new_user_reward',
      'register_enabled','password_login_enabled','password_register_enabled',
      'checkin_enabled','checkin_min_reward','checkin_max_reward','checkin_captcha',
      'low_balance_threshold','channel_alert_enabled',
      'thinking_to_content','model_rpm',
      'email_domain_restriction_enabled','email_domain_whitelist',
      'log_retention_days',
      'quota_display_type', 'quota_display_symbol', 'quota_display_rate',
      'passkey_enabled','passkey_rp_id','passkey_display_name','passkey_origins','passkey_allow_insecure','passkey_user_verification','passkey_attachment',
      'about','payment_notify_secret','epay_notify_url','epay_return_url',
      'model_price',
      'realname_enabled','realname_scenarios','realname_provider',
      'realname_aliyun_access_key_id','realname_aliyun_access_key_secret',
      'realname_aliyun_region','realname_aliyun_scene_id',
    ];
    const options = [
      ...keys.map(k => ({ key: k, value: get(k) })),
      { key: 'model_ratio', value: JSON.stringify(mr) },
      { key: 'completion_ratio', value: JSON.stringify(cr) },
    ];

    try {
      const res = await fetch('/api/option', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(options),
      });
      const data = await res.json();
      data.code === 0 ? toast.success('配置保存成功') : toast.error(data.msg || '配置保存失败');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '配置保存失败');
    } finally { setSaving(false); }
  };

  const handleEditPricing = (item: PricingItem) => { setEditingItem(item); setItemForm({ ...item }); onOpen(); };
  const handleAddPricing = () => { setEditingItem(null); setItemForm({ model: '', ratio: 1, completion_ratio: 1 }); onOpen(); };
  const handleDeletePricing = (model: string) => setPricingItems(prev => prev.filter(p => p.model !== model));
  const handleSavePricingItem = (onClose: () => void) => {
    if (!itemForm.model) return;
    setPricingItems(prev =>
      [...prev.filter(p => p.model !== (editingItem?.model || itemForm.model)), itemForm]
        .sort((a, b) => a.model.localeCompare(b.model))
    );
    onClose();
  };

  return (
    <div className="space-y-5 max-w-5xl mx-auto pb-10">
      <PageHeader
        title="系统设置"
        description="管理系统全局配置、支付、OAuth 和安全选项"
        actions={
          <Button color="primary" startContent={<Save size={18} />} isLoading={saving} onPress={handleSave}>
            保存全部配置
          </Button>
        }
      />

      <Tabs aria-label="系统设置" variant="underlined" color="primary">

        {/* ════════ 基础 ════════ */}
        <Tab key="basic" title={<TabIcon icon={Globe} label="基础" />}>
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Lock size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>登录与注册</span>
                </div>
              </CardHeader>
              <CardBody className="gap-3">
                <SettingRow title="允许密码登录" description="关闭后用户只能通过第三方账号或 Passkey 登录">
                  <Switch isSelected={get('password_login_enabled', 'true') === 'true'} onValueChange={v => set('password_login_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <SettingRow title="允许新用户注册" description="关闭后密码注册和第三方自动注册均不可用">
                  <Switch isSelected={get('register_enabled', 'true') === 'true'} onValueChange={v => set('register_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <SettingRow title="允许密码注册" description="关闭后仅允许第三方账号注册新用户">
                  <Switch isSelected={get('password_register_enabled', 'true') === 'true'} onValueChange={v => set('password_register_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <SettingRow title="要求邮箱验证" description="注册时需要输入邮箱验证码">
                  <Switch isSelected={get('email_verification_enabled') === 'true'} onValueChange={v => set('email_verification_enabled', String(v))} />
                </SettingRow>
                <p className="text-xs" style={{ color: 'var(--text-muted)' }}>邮箱验证需要在「邮件」标签页配置 SMTP 服务器</p>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Globe size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>站点信息</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="系统名称" value={get('system_name')} onValueChange={v => set('system_name', v)} placeholder="Akasha" />
                  <Input label="系统地址" value={systemUrl} onValueChange={v => set('system_url', v)} placeholder={window.location.origin} description="用于生成回调地址，务必填写正确" />
                  <Input label="Logo 地址" value={get('logo_url')} onValueChange={v => set('logo_url', v)} placeholder="https://..." />
                  <Input label="对话链接" value={get('chat_link')} onValueChange={v => set('chat_link', v)} placeholder="ChatGPT Next Web 等前端地址" />
                  <Input label="对话链接 2" value={get('chat_link2')} onValueChange={v => set('chat_link2', v)} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Bell size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>公告与页脚</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <Textarea label="公告内容（Markdown）" value={get('notice')} onValueChange={v => set('notice', v)} minRows={3} placeholder="向用户展示的公告信息" />
                <Textarea label="页脚 HTML" value={get('footer_html')} onValueChange={v => set('footer_html', v)} minRows={2} placeholder="自定义页脚内容，支持 HTML" />
                <Textarea label="关于页内容" value={get('about')} onValueChange={v => set('about', v)} minRows={3} description="Markdown 格式，显示在关于页面" />
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <CreditCard size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>充值与计费</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="最低充值金额" type="number" value={get('min_topup')} onValueChange={v => set('min_topup', v)} description="单位：元" />
                  <Input label="默认价格" type="number" value={get('price')} onValueChange={v => set('price', v)} description="每 500000 Quota 对应的金额" />
                </div>
                <Divider />
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>货币展示方式</span>
                  <div className="flex gap-2 flex-wrap">
                    {([
                      { key: 'usd', label: '美元 ($)' },
                      { key: 'cny', label: '人民币 (¥)' },
                      { key: 'tokens', label: '额度点数' },
                      { key: 'custom', label: '自定义' },
                    ] as const).map(item => {
                      const active = (get('quota_display_type') || 'usd') === item.key;
                      return (
                        <button key={item.key} type="button"
                          onClick={() => set('quota_display_type', item.key)}
                          className={`px-3 py-1.5 rounded-lg border text-sm transition-all ${active
                            ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                            : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] cursor-pointer'}`}
                        >
                          {item.label}
                        </button>
                      );
                    })}
                  </div>
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>控制前端所有额度余额的展示格式</p>
                </div>
                {(get('quota_display_type') || 'usd') === 'custom' && (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input label="货币符号" value={get('quota_display_symbol', '¤')} onValueChange={v => set('quota_display_symbol', v)} placeholder="如 ¥、$、€" />
                    <Input label="汇率（相对美元）" type="number" value={get('quota_display_rate', '1')} onValueChange={v => set('quota_display_rate', v)} description="1 美元 = 此汇率 货币" />
                  </div>
                )}
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ════════ 支付 ════════ */}
        <Tab key="payment" title={<TabIcon icon={CreditCard} label="支付" />}>
          <div className="space-y-4">
            {/* 支付总控 */}
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <CreditCard size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>支付总控</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <SettingRow title="开放充值" description="允许用户在前端充值额度">
                  <Switch isSelected={get('enable_topup') === 'true'} onValueChange={v => set('enable_topup', v ? 'true' : 'false')} />
                </SettingRow>
                <Divider />
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>支付渠道</span>
                  <div className="flex gap-2 flex-wrap">
                    {([
                      { key: 'epay', label: '易支付' },
                      { key: 'stripe', label: 'Stripe' },
                      { key: 'creem', label: 'Creem' },
                    ] as const).map(item => {
                      const active = get('payment_provider') === item.key;
                      return (
                        <button
                          key={item.key}
                          type="button"
                          onClick={() => set('payment_provider', item.key)}
                          className={`flex items-center gap-2 px-4 py-2 rounded-xl border text-sm transition-all duration-200
                            ${active
                              ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                              : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] hover:text-[var(--text-primary)] cursor-pointer'
                            }`}
                        >
                          {item.label}
                        </button>
                      );
                    })}
                  </div>
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>选择启用的支付渠道，仅该渠道的配置会生效</p>
                </div>
                <Divider />
                <Input label="支付回调签名密钥" type="password" value={get('payment_notify_secret')} onValueChange={v => set('payment_notify_secret', v)} description="用于验证支付异步通知合法性，留空则不校验" />
              </CardBody>
            </Card>

            {/* 渠道配置详情 */}
            {get('payment_provider') === 'epay' && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>易支付</span>
                    <Chip size="sm" color="primary" variant="flat">EPay</Chip>
                  </div>
                </CardHeader>
                <CardBody className="gap-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input label="API 地址" value={get('epay_api_url')} onValueChange={v => set('epay_api_url', v)} />
                    <Input label="PID" value={get('epay_pid')} onValueChange={v => set('epay_pid', v)} />
                    <Input label="KEY" type="password" value={get('epay_key')} onValueChange={v => set('epay_key', v)} />
                    <Input label="通道类型" value={get('epay_type')} onValueChange={v => set('epay_type', v)} placeholder="alipay / wxpay" />
                  </div>
                  <UrlBlock label="异步回调地址（在易支付后台填写）" url={systemUrl ? `${systemUrl}/api/payment/notify` : null} />
                  <UrlBlock label="同步返回地址" url={systemUrl || null} />
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input label="异步通知地址" value={get('epay_notify_url')} onValueChange={v => set('epay_notify_url', v)} description="留空则自动使用 系统地址/api/payment/notify" />
                    <Input label="同步返回地址" value={get('epay_return_url')} onValueChange={v => set('epay_return_url', v)} description="留空则自动使用系统地址" />
                  </div>
                </CardBody>
              </Card>
            )}

            {get('payment_provider') === 'stripe' && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Stripe</span>
                    <Chip size="sm" color="success" variant="flat">Card</Chip>
                  </div>
                </CardHeader>
                <CardBody className="gap-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input label="Secret Key" type="password" value={get('stripe_secret_key')} onValueChange={v => set('stripe_secret_key', v)} placeholder="sk_live_..." />
                    <Input label="Webhook Secret" type="password" value={get('stripe_webhook_secret')} onValueChange={v => set('stripe_webhook_secret', v)} placeholder="whsec_..." description="Stripe Dashboard → Webhooks 获取" />
                    <Input label="货币代码" value={get('stripe_currency')} onValueChange={v => set('stripe_currency', v)} placeholder="usd" description="留空默认 usd" />
                    <Input label="支付成功跳转" value={get('stripe_success_url')} onValueChange={v => set('stripe_success_url', v)} placeholder="留空自动生成" />
                    <Input label="支付取消跳转" value={get('stripe_cancel_url')} onValueChange={v => set('stripe_cancel_url', v)} placeholder="留空同成功地址" />
                  </div>
                  <UrlBlock label="Webhook 端点（在 Stripe Dashboard 注册）" url={systemUrl ? `${systemUrl}/api/stripe/webhook` : null} />
                </CardBody>
              </Card>
            )}

            {get('payment_provider') === 'creem' && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Creem</span>
                    <Chip size="sm" color="warning" variant="flat">SaaS</Chip>
                  </div>
                </CardHeader>
                <CardBody className="gap-4">
                  <SettingRow title="测试模式" description="使用 test-api.creem.io 而非生产环境">
                    <Switch isSelected={get('creem_test_mode') === 'true'} onValueChange={v => set('creem_test_mode', String(v))} />
                  </SettingRow>
                  <Divider />
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input label="API Key" type="password" value={get('creem_api_key')} onValueChange={v => set('creem_api_key', v)} placeholder="creem_..." />
                    <Input label="Webhook Secret" type="password" value={get('creem_webhook_secret')} onValueChange={v => set('creem_webhook_secret', v)} description="Creem Dashboard → Developers → Webhook" />
                    <Input label="Product ID（旧版单产品）" value={get('creem_product_id')} onValueChange={v => set('creem_product_id', v)} placeholder="prod_..." description="仅当未配置产品目录时使用" />
                    <Input label="支付成功跳转" value={get('creem_success_url')} onValueChange={v => set('creem_success_url', v)} placeholder="留空自动生成" />
                  </div>
                  <UrlBlock label="Webhook 端点（在 Creem Dashboard 注册）" url={systemUrl ? `${systemUrl}/api/creem/webhook` : null} />
                  <Textarea
                    label="产品目录 JSON（多产品，优先级高于单 Product ID）"
                    value={get('creem_products')}
                    onValueChange={v => set('creem_products', v)}
                    minRows={3}
                    placeholder={'[{"product_id":"prod_xxx","name":"10元套餐","price":10.0,"quota":5000000}]'}
                    description="每个产品需包含 product_id、name、price（货币金额）、quota（充值额度）"
                  />
                </CardBody>
              </Card>
            )}
          </div>
        </Tab>

        {/* ════════ 安全 ════════ */}
        <Tab key="security" title={<TabIcon icon={Shield} label="安全" />}>
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Shield size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>宸汐安全套件</span>
                </div>
              </CardHeader>
              <CardBody className="gap-3">
                <SettingRow title="宸汐御安全（CxSec）" description="ECDH+AES-256-GCM 加密通讯，保护登录/注册等敏感接口">
                  <Switch isSelected={get('cxsec_enabled', 'false') === 'true'} onValueChange={v => set('cxsec_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <Input
                  label="受保护的 API 路径"
                  value={get('cxsec_protected_paths', '/api/user/login,/api/user/register,/api/user/login/2fa,/api/user/password/reset-request,/api/user/password/reset-confirm,/api/user/checkin')}
                  onValueChange={v => set('cxsec_protected_paths', v)}
                  description="前端拦截器只对这些路径启用加密，多个路径用英文逗号分隔"
                  isDisabled={get('cxsec_enabled', 'false') !== 'true'}
                />
                <Divider />
                <SettingRow title="宸汐清源" description="提示词注入检测 / 工具调用校验 / 响应内容净化">
                  <Switch isSelected={get('qingyuan_enabled', 'false') === 'true'} onValueChange={v => set('qingyuan_enabled', String(v))} />
                </SettingRow>
                <p className="text-xs leading-relaxed" style={{ color: 'var(--text-muted)' }}>
                  清源检测策略请前往「安全中心 → 宸汐清源」配置；宸汐玄鉴行为风控请在「安全中心 → 宸汐玄鉴」中开启（仅超级管理员可见）。
                </p>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Shield size={16} style={{ color: 'var(--accent-cosmic)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>内容审查</span>
                  <Chip size="sm" color="primary" variant="flat">腾讯云天御</Chip>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <SettingRow title="启用内容审查" description="对用户请求进行关键词和腾讯云 TMS/IMS 双重检测">
                  <Switch isSelected={get('content_moderation_enabled') === 'true'} onValueChange={v => set('content_moderation_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="腾讯云 SecretId" value={get('tencent_moderation_secret_id')} onValueChange={v => set('tencent_moderation_secret_id', v)} />
                  <Input label="腾讯云 SecretKey" type="password" value={get('tencent_moderation_secret_key')} onValueChange={v => set('tencent_moderation_secret_key', v)} />
                  <Input label="地域" value={get('tencent_moderation_region')} onValueChange={v => set('tencent_moderation_region', v)} placeholder="ap-guangzhou" />
                  <Input label="审核策略 BizType" value={get('tencent_moderation_biz_type')} onValueChange={v => set('tencent_moderation_biz_type', v)} description="腾讯云控制台配置的策略编号，留空用默认" />
                  <Input label="审查超时（秒）" type="number" value={get('content_moderation_timeout')} onValueChange={v => set('content_moderation_timeout', v)} />
                </div>
                <Textarea label="敏感词（逗号分隔）" value={get('content_moderation_keywords')} onValueChange={v => set('content_moderation_keywords', v)} minRows={2} />
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Textarea label="白名单用户 ID" value={get('content_moderation_whitelist_users')} onValueChange={v => set('content_moderation_whitelist_users', v)} minRows={2} />
                  <Textarea label="白名单模型" value={get('content_moderation_whitelist_models')} onValueChange={v => set('content_moderation_whitelist_models', v)} minRows={2} />
                  <Textarea label="白名单 IP" value={get('content_moderation_whitelist_ips')} onValueChange={v => set('content_moderation_whitelist_ips', v)} minRows={2} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Shield size={16} style={{ color: 'var(--color-warning)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>人机验证</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>验证码提供商</span>
                  <div className="flex gap-2 flex-wrap">
                    {([
                      { key: '', label: '关闭' },
                      { key: 'turnstile', label: 'Turnstile' },
                      { key: 'geetest', label: '极验' },
                      { key: 'hcaptcha', label: 'hCaptcha' },
                      { key: 'recaptcha', label: 'reCAPTCHA' },
                    ] as const).map(item => {
                      const active = get('captcha_provider') === item.key;
                      return (
                        <button
                          key={item.key || 'none'}
                          type="button"
                          onClick={() => set('captcha_provider', item.key)}
                          className={`flex items-center gap-2 px-4 py-2 rounded-xl border text-sm transition-all duration-200
                            ${active
                              ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                              : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] hover:text-[var(--text-primary)] cursor-pointer'
                            }`}
                        >
                          {item.label}
                        </button>
                      );
                    })}
                  </div>
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>选择启用的验证码提供商，仅该提供商的配置会生效</p>
                </div>
                <Divider />
                {get('captcha_provider') === 'turnstile' && (
                  <>
                    <SettingRow title="Cloudflare Turnstile" description="无需用户交互的无感人机验证">
                      <Switch isSelected={get('turnstile_check_enabled') === 'true'} onValueChange={v => set('turnstile_check_enabled', String(v))} />
                    </SettingRow>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Input label="站点密钥" value={get('turnstile_site_key')} onValueChange={v => set('turnstile_site_key', v)} />
                      <Input label="私钥" type="password" value={get('turnstile_secret_key')} onValueChange={v => set('turnstile_secret_key', v)} />
                    </div>
                  </>
                )}
                {get('captcha_provider') === 'geetest' && (
                  <>
                    <SettingRow title="极验 GeeTest" description="滑动拼图/点选文字等交互式验证">
                      <Switch isSelected={get('geetest_enabled') === 'true'} onValueChange={v => set('geetest_enabled', String(v))} />
                    </SettingRow>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Input label="Captcha ID" value={get('geetest_id')} onValueChange={v => set('geetest_id', v)} />
                      <Input label="Captcha Key" type="password" value={get('geetest_key')} onValueChange={v => set('geetest_key', v)} />
                    </div>
                  </>
                )}
                {get('captcha_provider') === 'hcaptcha' && (
                  <>
                    <SettingRow title="hCaptcha" description="hCaptcha 人机验证">
                      <Switch isSelected={get('hcaptcha_enabled') === 'true'} onValueChange={v => set('hcaptcha_enabled', String(v))} />
                    </SettingRow>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Input label="站点密钥" value={get('hcaptcha_site_key')} onValueChange={v => set('hcaptcha_site_key', v)} />
                      <Input label="私钥" type="password" value={get('hcaptcha_secret_key')} onValueChange={v => set('hcaptcha_secret_key', v)} />
                    </div>
                  </>
                )}
                {get('captcha_provider') === 'recaptcha' && (
                  <>
                    <SettingRow title="Google reCAPTCHA" description="Google reCAPTCHA 人机验证">
                      <Switch isSelected={get('recaptcha_enabled') === 'true'} onValueChange={v => set('recaptcha_enabled', String(v))} />
                    </SettingRow>
                    <div className="flex flex-col gap-2">
                      <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>版本</span>
                      <div className="flex gap-2 flex-wrap">
                        {([
                          { key: 'v2', label: 'v2（复选框）' },
                          { key: 'v3', label: 'v3（不可见）' },
                        ] as const).map(item => {
                          const active = (get('recaptcha_version') || 'v2') === item.key;
                          return (
                            <button
                              key={item.key}
                              type="button"
                              onClick={() => set('recaptcha_version', item.key)}
                              className={`flex items-center gap-2 px-4 py-2 rounded-xl border text-sm transition-all duration-200
                                ${active
                                  ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                                  : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] hover:text-[var(--text-primary)] cursor-pointer'
                                }`}
                            >
                              {item.label}
                            </button>
                          );
                        })}
                      </div>
                      <span className="text-xs" style={{ color: 'var(--text-muted)' }}>v2 显示复选框组件，v3 不可见基于评分验证</span>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Input label="站点密钥" value={get('recaptcha_site_key')} onValueChange={v => set('recaptcha_site_key', v)} />
                      <Input label="私钥" type="password" value={get('recaptcha_secret_key')} onValueChange={v => set('recaptcha_secret_key', v)} />
                    </div>
                  </>
                )}
                {get('captcha_provider') === '' && (
                  <p className="text-sm text-center py-4" style={{ color: 'var(--text-muted)' }}>未启用验证码，用户无需通过人机验证即可注册和登录</p>
                )}
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Fingerprint size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>Passkey / WebAuthn</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <SettingRow title="启用 Passkey" description="允许用户注册指纹/面容/安全密钥作为无密码登录凭证">
                  <Switch isSelected={get('passkey_enabled') === 'true'} onValueChange={v => set('passkey_enabled', String(v))} />
                </SettingRow>
                <Divider />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="RP ID（域名）" value={get('passkey_rp_id')} onValueChange={v => set('passkey_rp_id', v)} placeholder="自动检测（如 localhost 或 example.com）" description="通常为站点域名（不含端口和协议）" />
                  <Input label="显示名称" value={get('passkey_display_name')} onValueChange={v => set('passkey_display_name', v)} placeholder="留空则使用系统名称" />
                </div>
                <Input label="允许的 Origins" value={get('passkey_origins')} onValueChange={v => set('passkey_origins', v)} placeholder="https://example.com,http://localhost:5173" description="多个用英文逗号分隔，留空则自动检测当前请求来源" />
                <SettingRow title="允许 HTTP" description="开发环境下允许非 HTTPS 来源（生产环境不建议开启）">
                  <Switch isSelected={get('passkey_allow_insecure') === 'true'} onValueChange={v => set('passkey_allow_insecure', String(v))} />
                </SettingRow>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="用户验证" value={get('passkey_user_verification')} onValueChange={v => set('passkey_user_verification', v)} placeholder="preferred" description="preferred / required / discouraged" />
                  <Input label="认证器类型" value={get('passkey_attachment')} onValueChange={v => set('passkey_attachment', v)} placeholder="不限" description="platform（设备内置）/ cross-platform（安全密钥）/ 留空不限" />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Settings size={16} style={{ color: 'var(--accent-cosmic)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>基础设施</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Input label="Redis 地址" value={get('redis_addr')} onValueChange={v => set('redis_addr', v)} placeholder="127.0.0.1:6379" />
                  <Input label="Redis 密码" type="password" value={get('redis_password')} onValueChange={v => set('redis_password', v)} />
                  <Input label="Redis 数据库" type="number" value={get('redis_db')} onValueChange={v => set('redis_db', v)} placeholder="0" />
                </div>
                <Divider />
                <Input
                  label="日志留存天数"
                  type="number"
                  value={get('log_retention_days', '180')}
                  onValueChange={v => set('log_retention_days', v)}
                  description="《网络安全法》第 21 条要求不少于 6 个月（180 天），低于 180 将被强制按 180 执行"
                />
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ════════ 实名认证 ════════ */}
        <Tab key="realname" title={<TabIcon icon={ShieldCheck} label="实名认证" />}>
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <ShieldCheck size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>实名认证</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <SettingRow title="启用实名认证" description="开启后用户可进行实名认证，按场景要求强制验证">
                  <Switch isSelected={get('realname_enabled') === 'true'} onValueChange={v => set('realname_enabled', String(v))} />
                </SettingRow>
                <Divider />
                {/* 认证场景多选 */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>认证场景</span>
                  <p className="text-xs" style={{ color: 'var(--text-muted)' }}>勾选需要实名认证的场景，用户在对应操作前需先完成认证</p>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mt-1">
                    {REALNAME_SCENARIOS.map(sc => {
                      const active = realnameScenarios.includes(sc.key);
                      return (
                        <button key={sc.key} type="button"
                          onClick={() => toggleRealnameScenario(sc.key)}
                          className="flex items-start gap-2.5 p-3 text-left rounded-xl border transition-all duration-150"
                          style={{
                            borderColor: active ? 'var(--accent-primary)' : 'var(--border-color)',
                            background: active ? 'var(--nav-active-bg)' : 'var(--bg-elevated)',
                            cursor: 'pointer',
                          }}>
                          <div style={{
                            width: '16px', height: '16px', borderRadius: '4px', flexShrink: 0, marginTop: '1px',
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                            background: active ? 'var(--accent-primary)' : 'transparent',
                            border: `1px solid ${active ? 'var(--accent-primary)' : 'var(--border-strong)'}`,
                          }}>
                            {active && <Check size={11} className="text-white" strokeWidth={3} />}
                          </div>
                          <div className="min-w-0">
                            <p style={{ fontSize: '13px', fontWeight: 600, color: active ? 'var(--accent-primary)' : 'var(--text-primary)' }}>{sc.label}</p>
                            <p style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>{sc.desc}</p>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                  {realnameScenarios.includes('double_blind') && (
                    <p className="text-xs mt-1 px-3 py-2 rounded-lg" style={{
                      background: 'rgba(168,85,247,0.10)', border: '1px solid rgba(168,85,247,0.25)',
                      color: '#c084fc',
                    }}>
                      双盲模式已启用：系统只保留认证结果与流水号，不存储用户姓名和身份证号，做信息合规处理。
                    </p>
                  )}
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>认证服务商配置</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>服务商</span>
                  <div className="flex gap-2 flex-wrap">
                    {([
                      { key: 'aliyun', label: '阿里云实人认证' },
                    ] as const).map(item => {
                      const active = (get('realname_provider') || 'aliyun') === item.key;
                      return (
                        <button
                          key={item.key}
                          type="button"
                          onClick={() => set('realname_provider', item.key)}
                          className={`flex items-center gap-2 px-4 py-2 rounded-xl border text-sm transition-all duration-200
                            ${active
                              ? 'border-[var(--accent-primary)] bg-[var(--nav-active-bg)] text-[var(--accent-primary)] font-semibold'
                              : 'border-[var(--border-color)] text-[var(--text-secondary)] hover:border-[var(--border-strong)] hover:text-[var(--text-primary)] cursor-pointer'
                            }`}
                        >
                          {item.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
                <Divider />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="阿里云 AccessKey ID" value={get('realname_aliyun_access_key_id')} onValueChange={v => set('realname_aliyun_access_key_id', v)} placeholder="LTAI..." description="阿里云 RAM 用户 AccessKey" />
                  <Input label="阿里云 AccessKey Secret" type="password" value={get('realname_aliyun_access_key_secret')} onValueChange={v => set('realname_aliyun_access_key_secret', v)} description="对应的 AccessKey Secret" />
                  <Input label="地域" value={get('realname_aliyun_region', 'cn-hangzhou')} onValueChange={v => set('realname_aliyun_region', v)} placeholder="cn-hangzhou" description="默认 cn-hangzhou" />
                  <Input label="认证场景 ID" value={get('realname_aliyun_scene_id')} onValueChange={v => set('realname_aliyun_scene_id', v)} description="阿里云控制台创建的实人认证场景 ID" />
                </div>
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ════════ OAuth ════════ */}
        <Tab key="oauth" title={<TabIcon icon={Link2} label="OAuth" />}>
          <div className="space-y-4">
            {[
              { key: 'github', label: 'GitHub', icon: '🐙', fields: [
                { k: 'github_client_id', l: '客户端 ID' },
                { k: 'github_client_secret', l: '客户端密钥', pw: true },
              ]},
              { key: 'linuxdo', label: 'LinuxDO', icon: '🐧', fields: [
                { k: 'linuxdo_client_id', l: '客户端 ID' },
                { k: 'linuxdo_client_secret', l: '客户端密钥', pw: true },
              ], extra: (
                <>
                  <Divider />
                  <p className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>各等级初始额度（500000 = $1）</p>
                  <div className="grid grid-cols-2 md:grid-cols-6 gap-3">
                    {[0, 1, 2, 3, 4, 5].map(level => (
                      <Input key={level} label={`L${level}`} type="number" placeholder="0"
                        value={get(`linuxdo_quota_level_${level}`, '0')}
                        onValueChange={v => set(`linuxdo_quota_level_${level}`, v)} />
                    ))}
                  </div>
                </>
              )},
              { key: 'discord', label: 'Discord', icon: '🎮', fields: [
                { k: 'discord_client_id', l: '客户端 ID' },
                { k: 'discord_client_secret', l: '客户端密钥', pw: true },
              ]},
              { key: 'telegram', label: 'Telegram', icon: '✈️', fields: [
                { k: 'telegram_bot_token', l: 'Bot Token', pw: true, ph: '从 @BotFather 获取' },
              ]},
              { key: 'wechat', label: '微信扫码', icon: '💬', fields: [
                { k: 'wechat_app_id', l: 'AppID', ph: '微信开放平台 AppID' },
                { k: 'wechat_app_secret', l: 'AppSecret', pw: true, ph: '微信开放平台 AppSecret' },
              ]},
              { key: 'oidc', label: 'OIDC', icon: '🔐', fields: [
                { k: 'oidc_client_id', l: '客户端 ID' },
                { k: 'oidc_client_secret', l: '客户端密钥', pw: true },
                { k: 'oidc_issuer_url', l: 'Issuer URL', ph: 'https://accounts.example.com', span2: true },
              ]},
            ].map(({ key, label, icon, fields, extra }) => (
              <Card key={key}>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <span className="text-base">{icon}</span>
                    <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>{label}</span>
                  </div>
                </CardHeader>
                <CardBody className="gap-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {fields.map(f => (
                      <Input
                        key={f.k}
                        label={f.l}
                        type={(f as any).pw ? 'password' : 'text'}
                        value={get(f.k)}
                        onValueChange={v => set(f.k, v)}
                        placeholder={(f as any).ph || ''}
                        className={(f as any).span2 ? 'md:col-span-2' : ''}
                      />
                    ))}
                  </div>
                  {extra}
                </CardBody>
              </Card>
            ))}
          </div>
        </Tab>

        {/* ════════ 邮件 ════════ */}
        <Tab key="smtp" title={<TabIcon icon={Mail} label="邮件" />}>
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Mail size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>SMTP 配置</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="SMTP 服务器" value={get('smtp_server')} onValueChange={v => set('smtp_server', v)} placeholder="smtp.gmail.com" />
                  <Input label="端口" value={get('smtp_port')} onValueChange={v => set('smtp_port', v)} placeholder="587" />
                  <Input label="账号" value={get('smtp_account')} onValueChange={v => set('smtp_account', v)} />
                  <Input label="密码" type="password" value={get('smtp_password')} onValueChange={v => set('smtp_password', v)} />
                  <Input label="发件人地址" value={get('smtp_from')} onValueChange={v => set('smtp_from', v)} className="md:col-span-2" />
                </div>
                <Divider />
                <SettingRow title="SSL/TLS" description="启用加密连接">
                  <Switch isSelected={get('smtp_ssl_enabled') === 'true'} onValueChange={v => set('smtp_ssl_enabled', String(v))} />
                </SettingRow>
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Shield size={16} style={{ color: 'var(--accent-cosmic)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>邮箱验证与域名限制</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <SettingRow title="限制注册邮箱域名" description="仅允许白名单内域名的邮箱注册">
                  <Switch isSelected={get('email_domain_restriction_enabled') === 'true'} onValueChange={v => set('email_domain_restriction_enabled', String(v))} />
                </SettingRow>
                <Textarea label="允许的邮箱域名（逗号分隔）" value={get('email_domain_whitelist')} onValueChange={v => set('email_domain_whitelist', v)} minRows={2} placeholder="example.com,company.org" />
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ════════ 邀请 ════════ */}
        <Tab key="invitation" title={<TabIcon icon={Gift} label="邀请" />}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Gift size={16} style={{ color: 'var(--accent-primary)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>邀请奖励</span>
              </div>
            </CardHeader>
            <CardBody className="gap-4">
              <SettingRow title="注册必须使用邀请码" description="开启后新用户注册需填写有效邀请码">
                <Switch isSelected={get('invitation_enabled') === 'true'} onValueChange={v => set('invitation_enabled', String(v))} />
              </SettingRow>
              <Divider />
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Input label="邀请码成本" type="number" placeholder="0" value={get('invitation_cost')} onValueChange={v => set('invitation_cost', v)} description="生成时扣除额度" />
                <Input label="邀请者奖励" type="number" placeholder="0" value={get('invitation_reward')} onValueChange={v => set('invitation_reward', v)} description="被邀请人注册时发放" />
                <Input label="新用户初始额度" type="number" placeholder="0" value={get('new_user_reward')} onValueChange={v => set('new_user_reward', v)} description="注册即赠" />
              </div>
            </CardBody>
          </Card>
        </Tab>

        {/* ════════ 签到 ════════ */}
        <Tab key="checkin" title={<TabIcon icon={CalendarCheck} label="签到" />}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <CalendarCheck size={16} style={{ color: 'var(--accent-primary)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>每日签到</span>
              </div>
            </CardHeader>
            <CardBody className="gap-4">
              <SettingRow title="启用每日签到" description="允许用户每日签到获取随机额度奖励">
                <Switch isSelected={get('checkin_enabled') === 'true'} onValueChange={v => set('checkin_enabled', String(v))} />
              </SettingRow>
              <Divider />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="最小奖励额度" type="number" placeholder="1000" value={get('checkin_min_reward')} onValueChange={v => set('checkin_min_reward', v)} description="Quota 单位" />
                <Input label="最大奖励额度" type="number" placeholder="10000" value={get('checkin_max_reward')} onValueChange={v => set('checkin_max_reward', v)} description="Quota 单位" />
              </div>
              <Divider />
              <SettingRow title="签到需要人机验证" description="防止自动脚本刷签到">
                <Switch isSelected={get('checkin_captcha') === 'true'} onValueChange={v => set('checkin_captcha', String(v))} />
              </SettingRow>
            </CardBody>
          </Card>
        </Tab>

        {/* ════════ 通知 ════════ */}
        <Tab key="notify" title={<TabIcon icon={Bell} label="通知" />}>
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Bell size={16} style={{ color: 'var(--color-warning)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>余额预警</span>
                </div>
              </CardHeader>
              <CardBody className="gap-4">
                <Input
                  label="余额预警阈值"
                  type="number"
                  value={get('low_balance_threshold', '500000')}
                  onValueChange={v => set('low_balance_threshold', v)}
                  description="用户充值入账后，若余额低于此值则推送提醒。默认 500000 ≈ $1"
                />
              </CardBody>
            </Card>

            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <Bell size={16} style={{ color: 'var(--color-danger)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>渠道异常告警</span>
                </div>
              </CardHeader>
              <CardBody className="gap-3">
                <SettingRow title="渠道检查失败时通知管理员" description="渠道健康检查失败时自动推送告警">
                  <Switch isSelected={get('channel_alert_enabled') === 'true'} onValueChange={v => set('channel_alert_enabled', String(v))} />
                </SettingRow>
                <p className="text-xs leading-relaxed" style={{ color: 'var(--text-faint)' }}>
                  各管理员的通知渠道（邮件 / Webhook / Bark / Gotify）需在「个人资料 → 通知设置」中单独配置
                </p>
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ════════ 系统 ════════ */}
        <Tab key="system" title={<TabIcon icon={Settings} label="系统" />}>
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <Settings size={16} style={{ color: 'var(--accent-primary)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>系统参数</span>
              </div>
            </CardHeader>
            <CardBody className="gap-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="全局 RPM 限制" type="number" value={get('model_rpm')} onValueChange={v => set('model_rpm', v)} description="每分钟最大请求数，0 表示不限制" />
              </div>
              <Divider />
              <SettingRow title="thinking_to_content" description="将思考链（thinking）内容转换为普通消息输出">
                <Switch isSelected={get('thinking_to_content') === 'true'} onValueChange={v => set('thinking_to_content', String(v))} />
              </SettingRow>
            </CardBody>
          </Card>
        </Tab>

        {/* ════════ 倍率 ════════ */}
        <Tab key="pricing" title={<TabIcon icon={BarChart3} label="倍率" />}>
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  <BarChart3 size={16} style={{ color: 'var(--accent-primary)' }} />
                  <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>模型计费倍率</span>
                </div>
                <Button size="sm" color="primary" variant="flat" startContent={<Plus size={15} />} onPress={handleAddPricing}>
                  添加模型
                </Button>
              </div>
            </CardHeader>
            <CardBody style={{ padding: 0 }}>
              <div className="data-table-wrap" style={{ borderRadius: 0, border: 'none', boxShadow: 'none' }}>
                <table className="data-table">
                  <thead>
                    <tr><th>模型名称</th><th>模型倍率</th><th>补全倍率</th><th>操作</th></tr>
                  </thead>
                  <tbody>
                    {pricingItems.length === 0 ? (
                      <tr><td colSpan={4}><EmptyState icon="📊" title="暂无倍率配置" description="添加模型以自定义计费倍率，未配置的模型使用默认价格" /></td></tr>
                    ) : pricingItems.map(item => (
                      <tr key={item.model}>
                        <td style={{ fontWeight: 600, fontFamily: 'monospace' }}>{item.model}</td>
                        <td><Chip size="sm" variant="flat">{item.ratio}x</Chip></td>
                        <td><Chip size="sm" variant="flat">{item.completion_ratio}x</Chip></td>
                        <td>
                          <div className="flex items-center gap-2">
                            <Tooltip content="编辑"><span className="text-default-400 cursor-pointer active:opacity-50" onClick={() => handleEditPricing(item)}><Edit size={16} /></span></Tooltip>
                            <Tooltip content="删除"><span className="text-danger cursor-pointer active:opacity-50" onClick={() => handleDeletePricing(item.model)}><Trash2 size={16} /></span></Tooltip>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardBody>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <BarChart3 size={16} style={{ color: 'var(--accent-cosmic)' }} />
                <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>按次计费价格</span>
              </div>
            </CardHeader>
            <CardBody className="gap-3">
              <Textarea
                label="按次计费价格 JSON"
                value={get('model_price', '{}')}
                onValueChange={v => set('model_price', v)}
                minRows={6}
                placeholder='{"dall-e-2":0.02,"dall-e-3":0.04,"mj_imagine":0.1}'
                description='每个模型每次调用的价格（美元/次）。配置后优先于按量计费，quota = price × QuotaPerUnit × groupRatio + 工具附加费。'
              />
            </CardBody>
          </Card>
        </Tab>

      </Tabs>

      <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>{editingItem ? '编辑倍率' : '添加倍率'}</ModalHeader>
              <ModalBody>
                <div className="flex flex-col gap-4">
                  <Input label="模型名称" placeholder="gpt-4o" value={itemForm.model}
                    onValueChange={v => setItemForm({ ...itemForm, model: v })}
                    isDisabled={!!editingItem} isRequired />
                  <Input label="模型倍率" type="number" value={itemForm.ratio.toString()}
                    onValueChange={v => setItemForm({ ...itemForm, ratio: parseFloat(v) })}
                    description="相对于基础价格的倍数" />
                  <Input label="补全倍率" type="number" value={itemForm.completion_ratio.toString()}
                    onValueChange={v => setItemForm({ ...itemForm, completion_ratio: parseFloat(v) })}
                    description="输出 token 相对于输入的倍数" />
                </div>
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
