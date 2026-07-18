import { useState } from 'react';
import { Copy, Check, Terminal, KeyRound, Sigma, Gauge } from 'lucide-react';
import { type ModelPrice, parseEndpoints } from './lib';
import { paramsForEndpoint } from './apiParams';
import type { GroupLimit } from './ModelDetailsDrawer';

// 🌸 详情抽屉 - API 标签页：接入信息 + 身份验证 + 调用示例 + 支持的参数～

type Lang = 'curl' | 'python' | 'node';

function buildSnippet(lang: Lang, modelName: string, endpoint: string): string {
  const base = window.location.origin;
  const isChat = endpoint !== 'embeddings' && !endpoint.includes('embedding');

  if (lang === 'curl') {
    if (!isChat) {
      return `curl ${base}/v1/embeddings \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $API_KEY" \\
  -d '{
    "model": "${modelName}",
    "input": "你好呀"
  }'`;
    }
    return `curl ${base}/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $API_KEY" \\
  -d '{
    "model": "${modelName}",
    "messages": [{"role": "user", "content": "你好呀"}]
  }'`;
  }

  if (lang === 'python') {
    return `from openai import OpenAI

client = OpenAI(
    base_url="${base}/v1",
    api_key="<YOUR_API_KEY>",
)

resp = client.chat.completions.create(
    model="${modelName}",
    messages=[{"role": "user", "content": "你好呀"}],
)
print(resp.choices[0].message.content)`;
  }

  return `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: '${base}/v1',
  apiKey: '<YOUR_API_KEY>',
});

const resp = await client.chat.completions.create({
  model: '${modelName}',
  messages: [{ role: 'user', content: '你好呀' }],
});
console.log(resp.choices[0].message.content);`;
}

function SectionTitle({ icon, title }: { icon: React.ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-2 mb-3">
      <span style={{ color: 'var(--accent-primary)', display: 'flex' }}>{icon}</span>
      <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--text-primary)' }}>{title}</span>
    </div>
  );
}

/* ～参数值小胶囊～ */
function Pill({ children, mono = true }: { children: React.ReactNode; mono?: boolean }) {
  return (
    <span style={{
      display: 'inline-block', padding: '1px 8px', borderRadius: '6px', fontSize: '11px', fontWeight: 500,
      fontFamily: mono ? 'SF Mono, Menlo, monospace' : undefined,
      background: 'var(--bg-elevated)', border: '1px solid var(--border-color)', color: 'var(--text-secondary)',
    }}>
      {children}
    </span>
  );
}

export default function DrawerApiTab({ model, groupLimits }: { model: ModelPrice; groupLimits: GroupLimit[] }) {
  const [lang, setLang] = useState<Lang>('curl');
  const [copied, setCopied] = useState(false);
  const endpoints = parseEndpoints(model.endpoints);
  const primaryEndpoint = endpoints[0] || 'openai';
  const snippet = buildSnippet(lang, model.model, primaryEndpoint);
  const params = paramsForEndpoint(primaryEndpoint);

  const copyCode = async () => {
    await navigator.clipboard.writeText(snippet);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };

  return (
    <>
      {/* 接入信息 */}
      <SectionTitle icon={<Terminal size={15} />} title="接入信息" />
      <div style={{ borderRadius: '12px', border: '1px solid var(--border-color)', overflow: 'hidden', marginBottom: '20px' }}>
        {[
          ['API Base', `${window.location.origin}/v1`],
          ['模型名称', model.model],
          ['鉴权方式', 'Bearer Token（在令牌页创建）'],
        ].map(([label, value], idx) => (
          <div key={label} className="flex items-center justify-between gap-3" style={{ padding: '10px 16px', fontSize: '13px', background: idx % 2 === 0 ? 'var(--bg-surface)' : 'var(--bg-elevated)' }}>
            <span style={{ color: 'var(--text-muted)', flexShrink: 0 }}>{label}</span>
            <span style={{ color: 'var(--text-primary)', fontWeight: 500, fontFamily: 'SF Mono, Menlo, monospace', fontSize: '12px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{value}</span>
          </div>
        ))}
      </div>

      {/* 🔑 身份验证 */}
      <SectionTitle icon={<KeyRound size={15} />} title="身份验证" />
      <div style={{ padding: '14px 16px', borderRadius: '12px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)', marginBottom: '20px', fontSize: '12px', color: 'var(--text-secondary)', lineHeight: 2 }}>
        所有请求必须携带 <Pill>Authorization: Bearer &lt;TOKEN&gt;</Pill> 请求头。
        Anthropic 格式的端点也接受 <Pill>x-api-key</Pill> 请求头。
        <div style={{ marginTop: '4px' }}>
          在「令牌」页面生成 API Key，可以按模型、分组、IP、额度等维度精细化授权。
        </div>
      </div>

      {/* 调用示例 */}
      <div className="flex items-center justify-between mb-3">
        <SectionTitle icon={<Terminal size={15} />} title="调用示例" />
        <div className="flex items-center" style={{ background: 'var(--bg-elevated)', borderRadius: '9px', border: '1px solid var(--border-color)', padding: '2px', marginBottom: '12px' }}>
          {([['curl', 'cURL'], ['python', 'Python'], ['node', 'Node.js']] as const).map(([k, l]) => (
            <button key={k} onClick={() => setLang(k)} style={{
              padding: '4px 10px', borderRadius: '7px', border: 'none', fontSize: '11px', fontWeight: 500, cursor: 'pointer',
              background: lang === k ? 'var(--bg-surface)' : 'transparent',
              color: lang === k ? 'var(--accent-primary)' : 'var(--text-muted)',
              boxShadow: lang === k ? 'var(--shadow-card)' : 'none',
            }}>{l}</button>
          ))}
        </div>
      </div>

      <div style={{ position: 'relative', borderRadius: '12px', border: '1px solid var(--border-color)', background: 'var(--bg-surface)', overflow: 'hidden', marginBottom: '8px' }}>
        <button
          onClick={copyCode}
          style={{ position: 'absolute', top: '10px', right: '10px', padding: '6px', borderRadius: '8px', border: '1px solid var(--border-color)', background: 'var(--bg-elevated)', color: copied ? '#10b981' : 'var(--text-muted)', cursor: 'pointer', display: 'flex', zIndex: 1 }}
          title="复制代码"
        >
          {copied ? <Check size={13} /> : <Copy size={13} />}
        </button>
        <pre style={{ margin: 0, padding: '16px', fontSize: '12px', lineHeight: 1.7, fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)', overflowX: 'auto', whiteSpace: 'pre' }}>
          {snippet}
        </pre>
      </div>
      <p style={{ fontSize: '11px', color: 'var(--text-faint)', margin: '0 0 20px' }}>
        替换 <Pill>&lt;YOUR_API_KEY&gt;</Pill> 为令牌设置中的 API Key。完全兼容 OpenAI SDK～
      </p>

      {/* Σ 支持的参数 */}
      <SectionTitle icon={<Sigma size={15} />} title="支持的参数" />
      <div style={{ borderRadius: '12px', border: '1px solid var(--border-color)', overflow: 'hidden' }}>
        {/* 表头 */}
        <div className="grid" style={{ gridTemplateColumns: '1.2fr 0.7fr 0.9fr 1.4fr', padding: '10px 16px', fontSize: '11px', fontWeight: 700, color: 'var(--text-muted)', background: 'var(--bg-elevated)', borderBottom: '1px solid var(--border-color)' }}>
          <span>参数</span><span>类型</span><span>默认值 / 范围</span><span>说明信息</span>
        </div>
        {params.map((p, idx) => (
          <div
            key={p.name}
            className="grid items-center"
            style={{
              gridTemplateColumns: '1.2fr 0.7fr 0.9fr 1.4fr', padding: '10px 16px', fontSize: '12px',
              background: idx % 2 === 0 ? 'var(--bg-surface)' : 'var(--bg-elevated)',
              borderBottom: idx < params.length - 1 ? '1px solid var(--border-color)' : 'none',
            }}
          >
            <span style={{ fontFamily: 'SF Mono, Menlo, monospace', fontWeight: 600, color: 'var(--text-primary)', wordBreak: 'break-all', paddingRight: '8px' }}>{p.name}</span>
            <span><Pill>{p.type}</Pill></span>
            <span className="flex items-center gap-1 flex-wrap" style={{ paddingRight: '8px' }}>
              {p.defaultValue !== '' && (
                <span className="flex items-center gap-1">
                  <span style={{ color: 'var(--text-faint)', fontSize: '11px' }}>=</span>
                  <Pill>{p.defaultValue}</Pill>
                </span>
              )}
              {p.range && <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{p.range}</span>}
              {p.defaultValue === '' && !p.range && <span style={{ color: 'var(--text-faint)' }}>—</span>}
            </span>
            <span style={{ color: 'var(--text-secondary)', lineHeight: 1.5 }}>{p.desc}</span>
          </div>
        ))}
      </div>
      <p style={{ fontSize: '11px', color: 'var(--text-faint)', marginTop: '10px' }}>
        参数以 OpenAI 兼容协议为准，具体支持范围取决于上游模型能力～
      </p>

      {/* ⏱️ 速率限制 */}
      {groupLimits.length > 0 && (
        <div style={{ marginTop: '20px' }}>
          <SectionTitle icon={<Gauge size={15} />} title="速率限制" />
          <div style={{ borderRadius: '12px', border: '1px solid var(--border-color)', overflow: 'hidden' }}>
            <div className="grid" style={{ gridTemplateColumns: '1.4fr 1fr 1fr 1fr', padding: '10px 16px', fontSize: '11px', fontWeight: 700, color: 'var(--text-muted)', background: 'var(--bg-elevated)', borderBottom: '1px solid var(--border-color)' }}>
              <span>分组</span><span style={{ textAlign: 'right' }}>RPM</span><span style={{ textAlign: 'right' }}>TPM</span><span style={{ textAlign: 'right' }}>RPD</span>
            </div>
            {groupLimits.map((g, idx) => (
              <div
                key={g.group}
                className="grid items-center"
                style={{
                  gridTemplateColumns: '1.4fr 1fr 1fr 1fr', padding: '10px 16px', fontSize: '12px',
                  background: idx % 2 === 0 ? 'var(--bg-surface)' : 'var(--bg-elevated)',
                  borderBottom: idx < groupLimits.length - 1 ? '1px solid var(--border-color)' : 'none',
                }}
              >
                <span style={{ color: 'var(--accent-primary)', fontWeight: 600 }}>{g.group}</span>
                <span style={{ textAlign: 'right', fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)' }}>{g.rpm > 0 ? g.rpm.toLocaleString() : '不限'}</span>
                <span style={{ textAlign: 'right', fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)' }}>{g.tpm > 0 ? (g.tpm >= 1000 ? `${Math.round(g.tpm / 1000)}K` : g.tpm) : '不限'}</span>
                <span style={{ textAlign: 'right', fontFamily: 'SF Mono, Menlo, monospace', color: 'var(--text-primary)' }}>{g.rpd > 0 ? (g.rpd >= 1000 ? `${Math.round(g.rpd / 1000)}K` : g.rpd) : '不限'}</span>
              </div>
            ))}
          </div>
          <p style={{ fontSize: '11px', color: 'var(--text-faint)', marginTop: '10px' }}>
            RPM = 每分钟请求数，TPM = 每分钟 token 数，RPD = 每日请求数；限制按令牌分组生效。
          </p>
        </div>
      )}
    </>
  );
}
