// 更新日志页面喵～记录每个版本都做了什么改动，让超管随时能回顾产品的成长足迹～
import PageHeader from '../../components/PageHeader';
import { Chip } from '../../components/ui';
import { Sparkles, Wrench, Bug } from 'lucide-react';

type ChangeType = 'feature' | 'improve' | 'fix';

interface ChangeItem {
  type: ChangeType;
  text: string;
}

interface ChangelogEntry {
  version: string;
  date: string;
  current?: boolean;
  items: ChangeItem[];
}

// 类型标签的图标与颜色配置（新增/优化/修复三种）
const TYPE_META: Record<ChangeType, { label: string; icon: typeof Sparkles; color: string }> = {
  feature: { label: '新增', icon: Sparkles, color: 'var(--accent-primary)' },
  improve: { label: '优化', icon: Wrench, color: 'var(--accent-cosmic)' },
  fix: { label: '修复', icon: Bug, color: 'var(--accent-star)' },
};

// 硬编码的更新日志列表，发布新版本时在这里追加一条即可～
const CHANGELOG: ChangelogEntry[] = [
  {
    version: '公测 1.1.0',
    date: '2026-07-18',
    current: true,
    items: [
      { type: 'feature', text: '分组系统重构：分组可设公开/隐藏可见性，倍率收归分组表，卡片式布局集中管理' },
      { type: 'feature', text: '特殊分组授权：用户到某基础分组后自动解锁额外分组（含隐藏分组）；用户可直接授予多个额外分组' },
      { type: 'feature', text: '特殊分组倍率：配置本分组用户使用某分组时的专属倍率，覆盖目标分组默认倍率' },
      { type: 'feature', text: '获取模型对话框：渠道拉取上游模型后弹窗挑选（搜索/新旧分区/按供应商勾选）再写回' },
      { type: 'feature', text: '模型广场详情按分组定价：按当前有权限的分组展示价格（游客仅 default，登录按可用分组）' },
      { type: 'feature', text: '新增渠道/模型后自动同步到模型元数据；定价页新增「广场未匹配定价」筛选与待配价标记' },
      { type: 'feature', text: '模型广场全面重构：多维筛选、卡片/表格双视图、计价单位切换、模型详情三标签页（概览/性能/API）' },
      { type: 'feature', text: '模型详情性能页基于真实调用日志：TPS、平均延迟、成功率、近 7 天请求趋势' },
      { type: 'feature', text: '官方元数据仓库一键同步：自动带出模型描述、图标、标签、端点、供应商与官方价格，支持中/英/日多语言' },
      { type: 'feature', text: '分组三维速率限制 RPM/TPM/RPD，五个中继入口统一生效；空分组表自动播种 default/vip/svip' },
      { type: 'feature', text: '模型元数据批量操作与侧边抽屉编辑器：匹配规则选择、端点可视化编辑与模板填充' },
      { type: 'improve', text: '多选下拉统一显示所选项名称，替代「已选 N 项」的模糊计数' },
      { type: 'improve', text: '模型中心合并供应商管理，移除部署管理与 io.net 部署能力' },
      { type: 'improve', text: '所有渠道 ID 手输框改为下拉选择；模型/订阅/分组页面增加分页' },
      { type: 'improve', text: '宸汐清源规则缓存日志降噪：刷新间隔 30 秒调整为 5 分钟，仅变化时输出' },
      { type: 'fix', text: '修复渠道「从上游拉取」按钮点击无反应' },
      { type: 'fix', text: '修复「从上游同步参考价」为上游全部模型新建记录、污染模型广场配置的问题' },
      { type: 'fix', text: '修复匹配规则无法选中、批量操作被 :id 路由拦截、官方元数据中文路径 404 等问题' },
    ],
  },
  {
    version: '公测 1.0.0',
    date: '2026-07-06',
    items: [
      { type: 'feature', text: '新增多验证码方式支持、Passkey 与双因素认证（2FA）登录' },
      { type: 'feature', text: '新增宸汐玄鉴 AI 内容审核模块，支持行为风控与上下文净化' },
      { type: 'feature', text: '新增渠道管理与供应商部署中心，支持渠道商角色体系' },
      { type: 'improve', text: '重构令牌分组架构，统一对齐权限与分组逻辑' },
      { type: 'improve', text: '重构额度计费与展示全流程，计费口径更加清晰准确' },
      { type: 'fix', text: '修复宸汐御、宸汐清源、宸汐玄鉴三大安全模块的若干问题' },
    ],
  },
];

export default function Changelog() {
  return (
    <div className="space-y-6">
      <PageHeader title="更新日志" description="记录 AKasha 每个版本的新功能、优化与修复内容～" />

      <div className="relative pl-6 space-y-6">
        {/* 时间线竖线 */}
        <div
          className="absolute left-[7px] top-2 bottom-2 w-px"
          style={{ background: 'var(--border-color)' }}
          aria-hidden="true"
        />

        {CHANGELOG.map(entry => (
          <div key={entry.version} className="relative">
            {/* 时间线节点 */}
            <span
              className="absolute -left-6 top-1.5 w-3 h-3 rounded-full"
              style={{
                background: entry.current ? 'var(--accent-primary)' : 'var(--bg-elevated)',
                border: '2px solid var(--bg-base)',
                boxShadow: '0 0 0 1px var(--border-color)',
              }}
              aria-hidden="true"
            />

            <div
              className="p-4 rounded-xl"
              style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-color)' }}
            >
              <div className="flex items-center gap-2 mb-3 flex-wrap">
                <h3 className="text-base font-bold" style={{ color: 'var(--text-primary)' }}>
                  {entry.version}
                </h3>
                {entry.current && <Chip color="primary" size="sm">当前版本</Chip>}
                <span className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.6 }}>
                  {entry.date}
                </span>
              </div>

              <ul className="space-y-2">
                {entry.items.map((item, idx) => {
                  const meta = TYPE_META[item.type];
                  const Icon = meta.icon;
                  return (
                    <li key={idx} className="flex items-start gap-2 text-sm">
                      <span
                        className="mt-0.5 flex items-center justify-center w-5 h-5 rounded-md flex-shrink-0"
                        style={{ background: `${meta.color}22`, color: meta.color }}
                      >
                        <Icon size={12} />
                      </span>
                      <span style={{ color: 'var(--text-secondary)' }}>{item.text}</span>
                    </li>
                  );
                })}
              </ul>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
