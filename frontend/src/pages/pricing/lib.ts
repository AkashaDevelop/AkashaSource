// 🌸 模型广场的类型定义和小工具们～

// 🌸 单个分组的定价明细（后端 /api/pricing 的 group_pricing 提供）～
export interface GroupPrice {
  group: string;   // 分组名
  ratio: number;   // 分组倍率（展示用）
  input: number;   // 输入价格（元/百万 token）
  output: number;  // 输出价格（元/百万 token）
  cache: number;   // 缓存输入价格（元/百万 token）
}

export interface ModelPrice {
  model: string;
  input_ratio: number;
  output_ratio: number;
  cache_ratio?: number;
  upstream_input_price: number;
  upstream_output_price: number;
  actual_input_price: number;
  actual_output_price: number;
  // 🎀 元数据增强字段（后端 /api/pricing 提供）～
  description?: string;
  icon?: string;
  tags?: string;
  endpoints?: string;
  vendor_name?: string;
  vendor_icon?: string;
  context_length?: number;
  groups?: string[]; // 🎀 该模型可用的分组～
  group_pricing?: GroupPrice[]; // 🌸 按分组定价明细～
}

export interface VendorInfo {
  name: string;
  icon: string;
  color: string;
}

/* ～名称猜供应商，仅在元数据缺失时兜底用～ */
const VENDOR_FALLBACK: { pattern: RegExp; name: string; icon: string; color: string }[] = [
  { pattern: /^(gpt|o1|o3|o4|dall-e|whisper|tts|text-|chatgpt)/i, name: 'OpenAI', icon: '⚡', color: '#10a37f' },
  { pattern: /^claude/i, name: 'Anthropic', icon: '🟣', color: '#d97706' },
  { pattern: /^gemini/i, name: 'Google', icon: '🔵', color: '#4285f4' },
  { pattern: /^grok/i, name: 'xAI', icon: '✖️', color: '#000000' },
  { pattern: /^deepseek/i, name: 'DeepSeek', icon: '🌊', color: '#4f46e5' },
  { pattern: /^(glm|chatglm)/i, name: '智谱 AI', icon: '🔮', color: '#7c3aed' },
  { pattern: /^qwen/i, name: '阿里云', icon: '☁️', color: '#ff6a00' },
  { pattern: /^hunyuan/i, name: '腾讯混元', icon: '💙', color: '#0052d9' },
  { pattern: /^ernie/i, name: '百度文心', icon: '🔍', color: '#2932e1' },
  { pattern: /^spark/i, name: '讯飞星火', icon: '🔥', color: '#ca0d19' },
  { pattern: /^(moonshot|kimi)/i, name: 'Moonshot', icon: '🌙', color: '#6d28d9' },
  { pattern: /^(llama|ollama)/i, name: 'Ollama', icon: '🦙', color: '#8b5cf6' },
  { pattern: /^mistral/i, name: 'Mistral', icon: '🌬️', color: '#ffb300' },
];

/* ～稳定取色：同名供应商永远同色～ */
const PALETTE = ['#10a37f', '#d97706', '#4285f4', '#4f46e5', '#7c3aed', '#ff6a00', '#0052d9', '#ca0d19', '#6d28d9', '#0891b2', '#db2777', '#65a30d'];

function hashColor(name: string): string {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  return PALETTE[h % PALETTE.length];
}

export function getVendor(m: ModelPrice): VendorInfo {
  if (m.vendor_name) {
    const fb = VENDOR_FALLBACK.find(v => v.name === m.vendor_name);
    return {
      name: m.vendor_name,
      icon: fb?.icon || '🏢',
      color: fb?.color || hashColor(m.vendor_name),
    };
  }
  const hit = VENDOR_FALLBACK.find(v => v.pattern.test(m.model));
  return hit ?? { name: '其他', icon: '🧠', color: '#64748b' };
}

/* ～标签解析：JSON 数组或逗号分隔都吃～ */
export function parseTags(tags?: string): string[] {
  if (!tags) return [];
  try {
    const arr = JSON.parse(tags);
    return Array.isArray(arr) ? arr.map(String).filter(Boolean) : [];
  } catch {
    return tags.split(/[,，]/).map(t => t.trim()).filter(Boolean);
  }
}

/* ～端点解析：对象取键、数组取值、逗号分隔兜底～ */
export function parseEndpoints(endpoints?: string): string[] {
  if (!endpoints) return [];
  try {
    const parsed = JSON.parse(endpoints);
    if (Array.isArray(parsed)) return parsed.map(String).filter(Boolean);
    if (parsed && typeof parsed === 'object') return Object.keys(parsed);
    return [];
  } catch {
    return endpoints.split(',').map(e => e.trim()).filter(Boolean);
  }
}

/* ～价格格式化：unit = 'M' 每百万 / 'K' 每千～ */
export function formatPrice(perM: number, unit: 'M' | 'K'): string {
  if (perM <= 0) return '免费';
  const p = unit === 'K' ? perM / 1000 : perM;
  if (p < 0.0001) return p.toExponential(1);
  if (p < 0.01) return p.toFixed(4);
  return p.toFixed(2);
}

/* ～上下文长度友好显示～ */
export function formatContext(len?: number): string {
  if (!len || len <= 0) return '';
  if (len >= 1000000) return `${(len / 1000000).toFixed(len % 1000000 ? 1 : 0)}M`;
  if (len >= 1000) return `${Math.round(len / 1000)}K`;
  return String(len);
}

/* ～倍率分级配色～ */
export function tierColor(r: number) {
  if (r <= 1) return { fg: '#10b981', bg: 'rgba(16,185,129,0.12)' };
  if (r <= 5) return { fg: '#0891b2', bg: 'rgba(8,145,178,0.12)' };
  if (r <= 15) return { fg: '#d97706', bg: 'rgba(217,119,6,0.12)' };
  return { fg: '#dc2626', bg: 'rgba(220,38,38,0.12)' };
}

export const TIER_OPTIONS = [
  { key: 'free', label: '免费', dot: '#10b981', test: (x: ModelPrice) => x.actual_input_price <= 0 && x.actual_output_price <= 0 },
  { key: 'low', label: '经济 ≤5x', dot: '#0891b2', test: (x: ModelPrice) => (x.actual_input_price > 0 || x.actual_output_price > 0) && x.input_ratio <= 5 && x.output_ratio <= 5 },
  { key: 'mid', label: '标准 5-15x', dot: '#d97706', test: (x: ModelPrice) => Math.max(x.input_ratio, x.output_ratio) > 5 && Math.max(x.input_ratio, x.output_ratio) <= 15 },
  { key: 'high', label: '高级 >15x', dot: '#dc2626', test: (x: ModelPrice) => Math.max(x.input_ratio, x.output_ratio) > 15 },
] as const;
