// 🌸 API 标签页的静态资料：支持的参数表～

export interface ApiParam {
  name: string;
  type: string;
  defaultValue: string;
  range?: string;
  desc: string;
}

/* ～Chat Completions 支持的参数（对齐 OpenAI 兼容协议）～ */
export const CHAT_PARAMS: ApiParam[] = [
  { name: 'temperature', type: 'number', defaultValue: '1', range: '0 ~ 2', desc: '采样温度；越低越稳定' },
  { name: 'top_p', type: 'number', defaultValue: '1', range: '0 ~ 1', desc: '核采样累计概率' },
  { name: 'max_tokens', type: 'integer', defaultValue: '', range: '>= 1', desc: '响应中最大 token 数' },
  { name: 'frequency_penalty', type: 'number', defaultValue: '0', range: '-2 ~ 2', desc: '惩罚高频 token 的重复出现' },
  { name: 'presence_penalty', type: 'number', defaultValue: '0', range: '-2 ~ 2', desc: '鼓励引入新话题' },
  { name: 'stop', type: 'array', defaultValue: '', desc: '最多 4 个停止生成的字符串' },
  { name: 'seed', type: 'integer', defaultValue: '', desc: '尽量保证可复现的采样种子' },
  { name: 'n', type: 'integer', defaultValue: '1', range: '>= 1', desc: '生成的候选条数' },
  { name: 'stream', type: 'boolean', defaultValue: 'false', desc: '通过 SSE 流式返回 token' },
  { name: 'response_format', type: 'object', defaultValue: '', desc: '强制输出 JSON 对象或符合 Schema 的结果' },
  { name: 'tools', type: 'array', defaultValue: '', desc: '模型可调用的工具 / 函数声明' },
  { name: 'tool_choice', type: 'string', defaultValue: '', range: 'auto / none / required', desc: '工具选择策略或具体工具名' },
  { name: 'logprobs', type: 'boolean', defaultValue: 'false', desc: '返回每个 token 的对数概率' },
  { name: 'top_logprobs', type: 'integer', defaultValue: '', range: '0 ~ 20', desc: '每个 token 返回的 top 概率数量' },
  { name: 'logit_bias', type: 'object', defaultValue: '', desc: '按 token 的 logit 偏置映射' },
  { name: 'user', type: 'string', defaultValue: '', desc: '用于风险审计的终端用户标识' },
];

/* ～Embeddings 支持的参数～ */
export const EMBEDDING_PARAMS: ApiParam[] = [
  { name: 'input', type: 'string | array', defaultValue: '', desc: '要向量化的文本或文本数组' },
  { name: 'encoding_format', type: 'string', defaultValue: 'float', range: 'float / base64', desc: '返回向量的编码格式' },
  { name: 'dimensions', type: 'integer', defaultValue: '', desc: '输出向量维度（仅部分模型支持）' },
  { name: 'user', type: 'string', defaultValue: '', desc: '用于风险审计的终端用户标识' },
];

/* ～图像生成支持的参数～ */
export const IMAGE_PARAMS: ApiParam[] = [
  { name: 'prompt', type: 'string', defaultValue: '', desc: '图像描述提示词' },
  { name: 'n', type: 'integer', defaultValue: '1', range: '1 ~ 10', desc: '生成的图像数量' },
  { name: 'size', type: 'string', defaultValue: '1024x1024', desc: '图像尺寸' },
  { name: 'quality', type: 'string', defaultValue: 'standard', range: 'standard / hd', desc: '图像质量' },
  { name: 'style', type: 'string', defaultValue: '', range: 'vivid / natural', desc: '图像风格' },
  { name: 'response_format', type: 'string', defaultValue: 'url', range: 'url / b64_json', desc: '返回格式' },
];

/* ～根据端点选参数表～ */
export function paramsForEndpoint(endpoint: string): ApiParam[] {
  const e = endpoint.toLowerCase();
  if (e.includes('embedding')) return EMBEDDING_PARAMS;
  if (e.includes('image')) return IMAGE_PARAMS;
  return CHAT_PARAMS;
}
