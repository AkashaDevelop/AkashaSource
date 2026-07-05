// ～货币展示工具～ 参考新阿比货币系统设计
// 支持四种展示模式：美元 / 人民币 / 额度点数 / 自定义
// 全局所有额度展示统一通过此模块格式化

type DisplayType = 'usd' | 'cny' | 'tokens' | 'custom';

interface QuotaConfig {
  quotaPerUnit: number;   // 1 USD = quotaPerUnit Quota（默认 500000）
  displayType: DisplayType;
  displaySymbol: string;  // 自定义模式下的货币符号
  displayRate: number;    // 自定义/人民币模式下相对美元的汇率
}

// 默认配置
let config: QuotaConfig = {
  quotaPerUnit: 500000,
  displayType: 'usd',
  displaySymbol: '$',
  displayRate: 7.3,
};

/** 从系统配置初始化（由 system store 调用） */
export function initQuotaConfig(opts: Record<string, string>) {
  if (opts.quota_display_type) {
    config.displayType = opts.quota_display_type as DisplayType;
  }
  if (opts.quota_display_symbol) {
    config.displaySymbol = opts.quota_display_symbol;
  }
  if (opts.quota_display_rate) {
    config.displayRate = parseFloat(opts.quota_display_rate) || 7.3;
  }
  // 根据展示类型设置默认符号
  if (config.displayType === 'usd') config.displaySymbol = '$';
  else if (config.displayType === 'cny') {
    config.displaySymbol = '¥';
    if (!opts.quota_display_rate) config.displayRate = 7.3;
  } else if (config.displayType === 'tokens') {
    config.displaySymbol = '';
  }
}

/** 格式化 Quota 为展示字符串 */
export function formatQuota(quota: number, decimals = 2): string {
  if (quota === 0) return formatAmount(0, decimals);
  switch (config.displayType) {
    case 'tokens':
      return `${quota} 点`;
    case 'cny':
      return `¥${(quota / config.quotaPerUnit * config.displayRate).toFixed(decimals)}`;
    case 'custom':
      return `${config.displaySymbol}${(quota / config.quotaPerUnit * config.displayRate).toFixed(decimals)}`;
    default: // usd
      return `$${(quota / config.quotaPerUnit).toFixed(decimals)}`;
  }
}

/** 仅格式化金额（不带 Quota 转换，直接是美元金额） */
export function formatAmount(usdAmount: number, decimals = 2): string {
  switch (config.displayType) {
    case 'tokens':
      return `${Math.round(usdAmount * config.quotaPerUnit)} 点`;
    case 'cny':
      return `¥${(usdAmount * config.displayRate).toFixed(decimals)}`;
    case 'custom':
      return `${config.displaySymbol}${(usdAmount * config.displayRate).toFixed(decimals)}`;
    default:
      return `$${usdAmount.toFixed(decimals)}`;
  }
}

/** 将用户输入的金额（以当前展示货币为单位）转换为 Quota */
export function moneyToQuota(money: number): number {
  switch (config.displayType) {
    case 'tokens':
      return Math.round(money); // 直接就是 Quota 点数
    case 'cny':
      return Math.round((money / config.displayRate) * config.quotaPerUnit);
    case 'custom':
      return Math.round((money / config.displayRate) * config.quotaPerUnit);
    default: // usd
      return Math.round(money * config.quotaPerUnit);
  }
}

/** 获取当前货币符号 */
export function getCurrencySymbol(): string {
  return config.displaySymbol || '$';
}

/** 获取 QuotaPerUnit */
export function getQuotaPerUnit(): number {
  return config.quotaPerUnit;
}

/** 获取展示类型说明文字（用于 UI 提示） */
export function getQuotaDisplayHint(): string {
  switch (config.displayType) {
    case 'tokens':
      return '额度以点数显示';
    case 'cny':
      return `1 USD = ${config.displayRate} CNY`;
    case 'custom':
      return `1 USD = ${config.displayRate} ${config.displaySymbol}`;
    default:
      return '1 USD = 500,000 额度';
  }
}
