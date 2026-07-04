import { useState, useEffect } from 'react';
import PageHeader from '../../components/PageHeader';
import EmptyState from '../../components/EmptyState';
import {
  Card,
  CardBody,
  Input,
  Button,
  Divider,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure,
  Tooltip,
  Switch,
  Textarea,
  Tabs,
  Tab,
} from '../../components/ui';
import { Save, Plus, Edit, Trash2 } from 'lucide-react';
import { useAuthStore } from '../../store/auth';
import { toast } from '../../store/toast';

interface Option { key: string; value: string; }
interface PricingItem { model: string; ratio: number; completion_ratio: number; }

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
        // auto-fill system_url from browser origin if not set
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

  const handleSave = async () => {
    setSaving(true);
    const mr: Record<string, number> = {};
    const cr: Record<string, number> = {};
    pricingItems.forEach(i => { mr[i.model] = i.ratio; cr[i.model] = i.completion_ratio; });

    const keys = [
      // 基础
      'system_name','system_url','logo_url','notice','footer_html','chat_link','chat_link2',
      'price','min_topup',
      // 支付
      'payment_provider','epay_api_url','epay_pid','epay_key','epay_type',
      'enable_topup',
      'stripe_secret_key','stripe_webhook_secret','stripe_currency','stripe_success_url','stripe_cancel_url',
      'creem_api_key','creem_webhook_secret','creem_product_id','creem_products','creem_success_url','creem_test_mode',
      // 风控
      'content_moderation_enabled','content_moderation_keywords',
      'content_moderation_timeout','content_moderation_whitelist_users',
      'content_moderation_whitelist_models','content_moderation_whitelist_ips',
      'tencent_moderation_secret_id','tencent_moderation_secret_key',
      'tencent_moderation_region','tencent_moderation_biz_type',
      // 缓存
      'redis_addr','redis_password','redis_db',
      // OAuth
      'github_client_id','github_client_secret',
      'linuxdo_client_id','linuxdo_client_secret',
      'linuxdo_quota_level_0','linuxdo_quota_level_1','linuxdo_quota_level_2',
      'linuxdo_quota_level_3','linuxdo_quota_level_4','linuxdo_quota_level_5',
      'discord_client_id','discord_client_secret',
      'oidc_client_id','oidc_client_secret','oidc_issuer_url',
      'telegram_bot_token',
      'wechat_app_id','wechat_app_secret',
      // 邮件
      'smtp_server','smtp_port','smtp_account','smtp_password','smtp_from',
      'smtp_ssl_enabled','email_verification_enabled',
      // 安全
      'cxsec_enabled','cxsec_protected_paths','qingyuan_enabled',
      'turnstile_site_key','turnstile_secret_key','turnstile_check_enabled',
      'captcha_provider','geetest_enabled','geetest_id','geetest_key',
      // 邀请
      'invitation_enabled','invitation_cost','invitation_reward','new_user_reward',
      // 签到
      'checkin_enabled','checkin_min_reward','checkin_max_reward','checkin_captcha',
      // 通知
      'low_balance_threshold','channel_alert_enabled',
      // 系统
      'thinking_to_content','model_rpm',
      // 邮件域名限制
      'email_domain_restriction_enabled','email_domain_whitelist',
      // 日志留存
      'log_retention_days',
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
    <div className="space-y-5 max-w-4xl mx-auto pb-10">
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

        {/* ── 基础 ── */}
        <Tab key="basic" title="🌐 基础">
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="系统名称" value={get('system_name')} onValueChange={v => set('system_name', v)} />
                <Input label="系统地址" value={get('system_url')} onValueChange={v => set('system_url', v)} placeholder={window.location.origin} />
                <Input label="Logo 地址" value={get('logo_url')} onValueChange={v => set('logo_url', v)} />
                <Input label="对话链接" value={get('chat_link')} onValueChange={v => set('chat_link', v)} />
                <Input label="对话链接 2" value={get('chat_link2')} onValueChange={v => set('chat_link2', v)} />
                <Input label="最低充值" type="number" value={get('min_topup')} onValueChange={v => set('min_topup', v)} />
                <Input label="默认价格" type="number" value={get('price')} onValueChange={v => set('price', v)} />
              </div>
              <Textarea label="公告内容" value={get('notice')} onValueChange={v => set('notice', v)} minRows={2} />
              <Textarea label="页脚 HTML" value={get('footer_html')} onValueChange={v => set('footer_html', v)} minRows={2} />
            </CardBody>
          </Card>
        </Tab>

        {/* ── 支付 ── */}
        <Tab key="payment" title="💳 支付">
          <div className="space-y-4">
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-[var(--text-primary)]">开放充值</p>
                  <p className="text-xs text-[var(--text-secondary)]">允许用户充值额度</p>
                </div>
                <Switch isSelected={get('enable_topup') === 'true'} onValueChange={v => set('enable_topup', v ? 'true' : 'false')} />
              </div>
              <Divider />
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="支付渠道" value={get('payment_provider')} onValueChange={v => set('payment_provider', v)} placeholder="epay / stripe / creem" description="配置哪个渠道就填对应名称，对应配置才会生效" />
              </div>
            </CardBody>
          </Card>

          {/* 易支付 */}
          {(get('payment_provider') === 'epay' || !get('payment_provider')) && (
          <Card>
            <CardBody className="gap-4 p-5">
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>易支付</p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="易支付 API 地址" value={get('epay_api_url')} onValueChange={v => set('epay_api_url', v)} />
                <Input label="易支付 PID" value={get('epay_pid')} onValueChange={v => set('epay_pid', v)} />
                <Input label="易支付 KEY" type="password" value={get('epay_key')} onValueChange={v => set('epay_key', v)} />
                <Input label="易支付通道类型" value={get('epay_type')} onValueChange={v => set('epay_type', v)} placeholder="alipay / wxpay" />
                <div className="flex flex-col gap-1">
                  <span className="text-xs text-[var(--text-secondary)]">易支付回调地址（自动生成）</span>
                  <div className="px-3 py-2 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border-color)] text-sm text-[var(--text-secondary)] font-mono break-all select-all">
                    {get('system_url') ? `${get('system_url')}/api/payment/notify` : <span className="italic opacity-50">请先设置系统 URL</span>}
                  </div>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="text-xs text-[var(--text-secondary)]">易支付同步返回地址（自动生成）</span>
                  <div className="px-3 py-2 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border-color)] text-sm text-[var(--text-secondary)] font-mono break-all select-all">
                    {get('system_url') || <span className="italic opacity-50">请先设置系统 URL</span>}
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
          )}

          {/* Stripe */}
          {get('payment_provider') === 'stripe' && (
          <Card>
            <CardBody className="gap-4 p-5">
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Stripe</p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="Secret Key" type="password" value={get('stripe_secret_key')} onValueChange={v => set('stripe_secret_key', v)} placeholder="sk_live_..." />
                <Input label="Webhook Secret" type="password" value={get('stripe_webhook_secret')} onValueChange={v => set('stripe_webhook_secret', v)} placeholder="whsec_..." description="在 Stripe Dashboard → Webhooks 获取" />
                <Input label="货币代码" value={get('stripe_currency')} onValueChange={v => set('stripe_currency', v)} placeholder="usd" description="留空默认 usd" />
                <Input label="支付成功跳转地址" value={get('stripe_success_url')} onValueChange={v => set('stripe_success_url', v)} placeholder="留空则自动生成" />
                <Input label="支付取消跳转地址" value={get('stripe_cancel_url')} onValueChange={v => set('stripe_cancel_url', v)} placeholder="留空则同成功地址" />
                <div className="flex flex-col gap-1">
                  <span className="text-xs text-[var(--text-secondary)]">Stripe Webhook 地址（在 Dashboard 注册）</span>
                  <div className="px-3 py-2 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border-color)] text-sm text-[var(--text-secondary)] font-mono break-all select-all">
                    {get('system_url') ? `${get('system_url')}/api/stripe/webhook` : <span className="italic opacity-50">请先设置系统 URL</span>}
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
          )}

          {/* Creem */}
          {get('payment_provider') === 'creem' && (
          <Card>
            <CardBody className="gap-4 p-5">
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Creem</p>
              <Switch isSelected={get('creem_test_mode') === 'true'} onValueChange={v => set('creem_test_mode', String(v))}>
                测试模式（使用 test-api.creem.io）
              </Switch>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="API Key" type="password" value={get('creem_api_key')} onValueChange={v => set('creem_api_key', v)} placeholder="creem_..." />
                <Input label="Webhook Secret" type="password" value={get('creem_webhook_secret')} onValueChange={v => set('creem_webhook_secret', v)} description="在 Creem Dashboard → Developers → Webhook 获取" />
                <Input label="Product ID（旧版单产品）" value={get('creem_product_id')} onValueChange={v => set('creem_product_id', v)} placeholder="prod_..." description="仅当未配置产品目录时使用" />
                <Input label="支付成功跳转地址" value={get('creem_success_url')} onValueChange={v => set('creem_success_url', v)} placeholder="留空则自动生成" />
                <div className="flex flex-col gap-1">
                  <span className="text-xs text-[var(--text-secondary)]">Creem Webhook 地址（在 Dashboard 注册）</span>
                  <div className="px-3 py-2 rounded-lg bg-[var(--bg-elevated)] border border-[var(--border-color)] text-sm text-[var(--text-secondary)] font-mono break-all select-all">
                    {get('system_url') ? `${get('system_url')}/api/creem/webhook` : <span className="italic opacity-50">请先设置系统 URL</span>}
                  </div>
                </div>
              </div>
              <Textarea
                label="产品目录 JSON（多产品时使用，优先级高于单 Product ID）"
                value={get('creem_products')}
                onValueChange={v => set('creem_products', v)}
                minRows={3}
                placeholder={'[{"product_id":"prod_xxx","name":"10元套餐","price":10.0,"quota":5000000}]'}
                description="每个产品需包含 product_id、name、price（货币金额）、quota（充值额度）四个字段"
              />
            </CardBody>
          </Card>
          )}
          </div>
        </Tab>

        {/* ── 安全 ── */}
        <Tab key="security" title="🛡️ 安全">
          <div className="space-y-4">
            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>宸汐御安全通讯协议</p>
                <Switch
                  isSelected={get('cxsec_enabled', 'false') === 'true'}
                  onValueChange={v => set('cxsec_enabled', String(v))}
                >
                  启用宸汐御安全（CxSec）加密通讯
                </Switch>
                <Input
                  label="受保护的 API 路径（逗号分隔）"
                  value={get('cxsec_protected_paths', '/api/user/login,/api/user/register,/api/user/login/2fa,/api/user/password/reset-request,/api/user/password/reset-confirm,/api/user/checkin')}
                  onValueChange={v => set('cxsec_protected_paths', v)}
                  description="前端拦截器只对这些路径启用 ECDH+AES-256-GCM 加密，多个路径用英文逗号分隔"
                  isDisabled={get('cxsec_enabled', 'false') !== 'true'}
                />
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>宸汐清源上下文净化</p>
                <Switch
                  isSelected={get('qingyuan_enabled', 'false') === 'true'}
                  onValueChange={v => set('qingyuan_enabled', String(v))}
                >
                  启用宸汐清源（提示词注入检测 / 工具调用校验 / 响应内容净化）
                </Switch>
                <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                  具体的检测策略请前往「宸汐清源」管理页配置，此开关是模块的总闸；关闭后所有策略均不生效。
                </p>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>宸汐玄鉴行为风控</p>
                <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                  用户行为风控、破限与逆向探测拦截默认关闭，需要单独在「宸汐玄鉴」管理页（仅超级管理员可见）中开启并配置检测策略。
                </p>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>日志留存</p>
                <Input
                  label="日志留存天数"
                  type="number"
                  value={get('log_retention_days', '180')}
                  onValueChange={v => set('log_retention_days', v)}
                  description="根据《网络安全法》第 21 条，网络日志留存不少于 6 个月（180 天）；低于 180 的配置将被系统强制按 180 天执行，超期日志由定时任务自动清理"
                />
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>内容审查（腾讯云天御）</p>
                <Switch isSelected={get('content_moderation_enabled') === 'true'} onValueChange={v => set('content_moderation_enabled', String(v))}>
                  启用内容审查
                </Switch>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="腾讯云 SecretId" value={get('tencent_moderation_secret_id')} onValueChange={v => set('tencent_moderation_secret_id', v)} />
                  <Input label="腾讯云 SecretKey" type="password" value={get('tencent_moderation_secret_key')} onValueChange={v => set('tencent_moderation_secret_key', v)} />
                  <Input label="腾讯云地域" value={get('tencent_moderation_region')} onValueChange={v => set('tencent_moderation_region', v)} placeholder="ap-guangzhou" />
                  <Input label="审核策略 BizType" value={get('tencent_moderation_biz_type')} onValueChange={v => set('tencent_moderation_biz_type', v)} description="在腾讯云控制台配置的策略编号，留空则使用账号默认策略" />
                  <Input label="审查超时（秒）" type="number" value={get('content_moderation_timeout')} onValueChange={v => set('content_moderation_timeout', v)} />
                </div>
                <Textarea label="敏感词（逗号分隔）" value={get('content_moderation_keywords')} onValueChange={v => set('content_moderation_keywords', v)} minRows={2} />
                <Textarea label="白名单用户 ID" value={get('content_moderation_whitelist_users')} onValueChange={v => set('content_moderation_whitelist_users', v)} minRows={2} />
                <Textarea label="白名单模型" value={get('content_moderation_whitelist_models')} onValueChange={v => set('content_moderation_whitelist_models', v)} minRows={2} />
                <Textarea label="白名单 IP" value={get('content_moderation_whitelist_ips')} onValueChange={v => set('content_moderation_whitelist_ips', v)} minRows={2} />
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Turnstile 人机验证</p>
                <Switch isSelected={get('turnstile_check_enabled') === 'true'} onValueChange={v => set('turnstile_check_enabled', String(v))}>
                  启用 Turnstile 验证
                </Switch>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="站点密钥" value={get('turnstile_site_key')} onValueChange={v => set('turnstile_site_key', v)} />
                  <Input label="私钥" type="password" value={get('turnstile_secret_key')} onValueChange={v => set('turnstile_secret_key', v)} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>极验 GeeTest 人机验证</p>
                <Switch isSelected={get('geetest_enabled') === 'true'} onValueChange={v => set('geetest_enabled', String(v))}>
                  启用极验验证
                </Switch>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="Captcha ID" value={get('geetest_id')} onValueChange={v => set('geetest_id', v)} placeholder="极验后台获取" />
                  <Input label="Captcha Key" type="password" value={get('geetest_key')} onValueChange={v => set('geetest_key', v)} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>Redis 缓存</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="Redis 地址" value={get('redis_addr')} onValueChange={v => set('redis_addr', v)} placeholder="127.0.0.1:6379" />
                  <Input label="Redis 密码" type="password" value={get('redis_password')} onValueChange={v => set('redis_password', v)} />
                  <Input label="Redis 数据库" type="number" value={get('redis_db')} onValueChange={v => set('redis_db', v)} placeholder="0" />
                </div>
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ── OAuth ── */}
        <Tab key="oauth" title="🔗 OAuth">
          <div className="space-y-4">
            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>🐙 GitHub OAuth</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="客户端 ID" value={get('github_client_id')} onValueChange={v => set('github_client_id', v)} />
                  <Input label="客户端密钥" type="password" value={get('github_client_secret')} onValueChange={v => set('github_client_secret', v)} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>🐧 LinuxDO OAuth</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="客户端 ID" value={get('linuxdo_client_id')} onValueChange={v => set('linuxdo_client_id', v)} />
                  <Input label="客户端密钥" type="password" value={get('linuxdo_client_secret')} onValueChange={v => set('linuxdo_client_secret', v)} />
                </div>
                <Divider />
                <p style={{ fontSize: '13px', color: 'var(--text-muted)' }}>各等级初始额度（500000 = $1）</p>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                  {[0, 1, 2, 3, 4, 5].map(level => (
                    <Input key={level} label={`等级 ${level}`} type="number" placeholder="0"
                      value={get(`linuxdo_quota_level_${level}`, '0')}
                      onValueChange={v => set(`linuxdo_quota_level_${level}`, v)} />
                  ))}
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>🎮 Discord OAuth</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="客户端 ID" value={get('discord_client_id')} onValueChange={v => set('discord_client_id', v)} />
                  <Input label="客户端密钥" type="password" value={get('discord_client_secret')} onValueChange={v => set('discord_client_secret', v)} />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>✈️ Telegram 登录</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="Bot Token" type="password" value={get('telegram_bot_token')} onValueChange={v => set('telegram_bot_token', v)} placeholder="从 @BotFather 获取" />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>💬 微信扫码登录</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="AppID" value={get('wechat_app_id')} onValueChange={v => set('wechat_app_id', v)} placeholder="微信开放平台 AppID" />
                  <Input label="AppSecret" type="password" value={get('wechat_app_secret')} onValueChange={v => set('wechat_app_secret', v)} placeholder="微信开放平台 AppSecret" />
                </div>
              </CardBody>
            </Card>

            <Card>
              <CardBody className="gap-4 p-5">
                <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>🔐 OIDC（通用 OAuth2）</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input label="客户端 ID" value={get('oidc_client_id')} onValueChange={v => set('oidc_client_id', v)} />
                  <Input label="客户端密钥" type="password" value={get('oidc_client_secret')} onValueChange={v => set('oidc_client_secret', v)} />
                  <Input label="Issuer URL" value={get('oidc_issuer_url')} onValueChange={v => set('oidc_issuer_url', v)} placeholder="https://accounts.example.com" className="col-span-2" />
                </div>
              </CardBody>
            </Card>
          </div>
        </Tab>

        {/* ── 邮件 ── */}
        <Tab key="smtp" title="📧 邮件">
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="SMTP 服务器" value={get('smtp_server')} onValueChange={v => set('smtp_server', v)} />
                <Input label="SMTP 端口" value={get('smtp_port')} onValueChange={v => set('smtp_port', v)} placeholder="587" />
                <Input label="SMTP 账号" value={get('smtp_account')} onValueChange={v => set('smtp_account', v)} />
                <Input label="SMTP 密码" type="password" value={get('smtp_password')} onValueChange={v => set('smtp_password', v)} />
                <Input label="发件人地址" value={get('smtp_from')} onValueChange={v => set('smtp_from', v)} />
              </div>
              <Switch isSelected={get('smtp_ssl_enabled') === 'true'} onValueChange={v => set('smtp_ssl_enabled', String(v))}>
                启用 SSL/TLS
              </Switch>
              <Switch isSelected={get('email_verification_enabled') === 'true'} onValueChange={v => set('email_verification_enabled', String(v))}>
                注册时需要邮箱验证码
              </Switch>
              <Divider />
              <Switch isSelected={get('email_domain_restriction_enabled') === 'true'} onValueChange={v => set('email_domain_restriction_enabled', String(v))}>
                限制注册邮箱域名
              </Switch>
              <Textarea label="允许的邮箱域名（逗号分隔）" value={get('email_domain_whitelist')} onValueChange={v => set('email_domain_whitelist', v)} minRows={2} placeholder="example.com,company.org" />
            </CardBody>
          </Card>
        </Tab>

        {/* ── 邀请 ── */}
        <Tab key="invitation" title="🎁 邀请">
          <Card>
            <CardBody className="gap-4 p-5">
              <Switch isSelected={get('invitation_enabled') === 'true'} onValueChange={v => set('invitation_enabled', String(v))}>
                注册必须使用邀请码
              </Switch>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Input label="邀请码成本" type="number" placeholder="0" value={get('invitation_cost')} onValueChange={v => set('invitation_cost', v)} description="生成时扣除额度" />
                <Input label="邀请者奖励" type="number" placeholder="0" value={get('invitation_reward')} onValueChange={v => set('invitation_reward', v)} description="被邀请用户注册时" />
                <Input label="新用户初始额度" type="number" placeholder="0" value={get('new_user_reward')} onValueChange={v => set('new_user_reward', v)} description="注册即赠" />
              </div>
            </CardBody>
          </Card>
        </Tab>

        {/* ── 签到 ── */}
        <Tab key="checkin" title="📅 签到">
          <Card>
            <CardBody className="gap-4 p-5">
              <Switch isSelected={get('checkin_enabled') === 'true'} onValueChange={v => set('checkin_enabled', String(v))}>
                启用每日签到
              </Switch>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input
                  label="最小奖励额度"
                  type="number"
                  placeholder="例如：1000"
                  value={get('checkin_min_reward')}
                  onValueChange={v => set('checkin_min_reward', v)}
                  description="每次签到随机奖励下限（Quota 单位）"
                />
                <Input
                  label="最大奖励额度"
                  type="number"
                  placeholder="例如：10000"
                  value={get('checkin_max_reward')}
                  onValueChange={v => set('checkin_max_reward', v)}
                  description="每次签到随机奖励上限（Quota 单位）"
                />
              </div>
              <Switch isSelected={get('checkin_captcha') === 'true'} onValueChange={v => set('checkin_captcha', String(v))}>
                签到需要人机验证
              </Switch>
            </CardBody>
          </Card>
        </Tab>

        {/* ── 通知 ── */}
        <Tab key="notify" title="🔔 通知">
          <Card>
            <CardBody className="gap-4 p-5">
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>余额预警</p>
              <Input
                label="余额预警阈值（quota 单位）"
                type="number"
                value={get('low_balance_threshold', '500000')}
                onValueChange={v => set('low_balance_threshold', v)}
                description="用户充值入账后，若余额低于此值则推送提醒。默认 500000 ≈ $1"
              />
            </CardBody>
          </Card>
          <Card className="mt-4">
            <CardBody className="gap-4 p-5">
              <p style={{ fontSize: '14px', fontWeight: 600, color: 'var(--text-secondary)' }}>渠道异常告警</p>
              <Switch isSelected={get('channel_alert_enabled') === 'true'} onValueChange={v => set('channel_alert_enabled', String(v))}>
                渠道检查失败时通知所有管理员
              </Switch>
              <p style={{ fontSize: '12px', color: 'var(--text-faint)' }}>
                各管理员的通知渠道（邮件 / Webhook / Bark / Gotify）需在「个人资料」→「通知设置」中单独配置
              </p>
            </CardBody>
          </Card>
        </Tab>

        {/* ── 系统 ── */}
        <Tab key="system" title="⚙️ 系统">
          <Card>
            <CardBody className="gap-4 p-5">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Input label="全局 RPM 限制" type="number" value={get('model_rpm')} onValueChange={v => set('model_rpm', v)} description="每分钟最大请求数，0 表示不限制" />
              </div>
              <Switch isSelected={get('thinking_to_content') === 'true'} onValueChange={v => set('thinking_to_content', String(v))}>
                将思考内容转换为普通消息（thinking_to_content）
              </Switch>
            </CardBody>
          </Card>
        </Tab>

        {/* ── 倍率 ── */}
        <Tab key="pricing" title="📊 倍率">
          <Card>
            <CardBody style={{ padding: 0 }}>
              <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '12px 16px' }}>
                <Button size="sm" color="primary" variant="flat" startContent={<Plus size={15} />} onPress={handleAddPricing}>
                  添加模型
                </Button>
              </div>
              <div className="data-table-wrap" style={{ borderRadius: 0, border: 'none', boxShadow: 'none' }}>
                <table className="data-table">
                  <thead>
                    <tr><th>模型名称</th><th>模型倍率</th><th>补全倍率</th><th>操作</th></tr>
                  </thead>
                  <tbody>
                    {pricingItems.length === 0 ? (
                      <tr><td colSpan={4}><EmptyState icon="📊" title="暂无倍率配置" description="添加模型以自定义计费倍率" /></td></tr>
                    ) : pricingItems.map(item => (
                      <tr key={item.model}>
                        <td style={{ fontWeight: 600, fontFamily: 'monospace' }}>{item.model}</td>
                        <td>{item.ratio}x</td>
                        <td>{item.completion_ratio}x</td>
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
