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
    version: '公测 1.0.0',
    date: '2026-07-06',
    current: true,
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
