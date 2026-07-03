/**
 * 宸汐御安全 fetch 拦截器
 * 挂载后透明替换全局 fetch，所有 /api/* 请求自动加解密
 */
import { cxEncrypt, cxDecrypt, initSession } from './cxsec';

const _originalFetch = window.fetch.bind(window);
const TEXT_ENCODER = new TextEncoder();
const TEXT_DECODER = new TextDecoder();

const EXEMPT = ['/api/cx/challenge', '/api/cx/ks'];

function needsEncryption(url: string): boolean {
  if (EXEMPT.some(e => url.includes(e))) return false;
  return url.includes('/api/');
}

async function cxFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = typeof input === 'string' ? input
    : input instanceof URL ? input.href
    : (input as Request).url;

  if (!needsEncryption(url)) {
    return _originalFetch(input, init);
  }

  // ── 确保会话已建立 ────────────────────────────────────────────────
  await initSession();

  // ── 序列化并加密请求体 ────────────────────────────────────────────
  let plainBytes: Uint8Array;
  const body = init?.body ?? (input instanceof Request ? (input as Request).body : null);
  if (body == null) {
    plainBytes = TEXT_ENCODER.encode('');
  } else if (typeof body === 'string') {
    plainBytes = TEXT_ENCODER.encode(body);
  } else if (body instanceof Uint8Array) {
    plainBytes = body;
  } else if (body instanceof ArrayBuffer) {
    plainBytes = new Uint8Array(body);
  } else {
    plainBytes = TEXT_ENCODER.encode(String(body));
  }

  const cipherBody = await cxEncrypt(plainBytes);

  const newInit: RequestInit = {
    ...(init ?? {}),
    method: (init?.method) ?? (input instanceof Request ? (input as Request).method : 'GET'),
    headers: {
      ...(init?.headers as Record<string, string> ?? {}),
      'Content-Type': 'application/octet-stream',
    },
    body: cipherBody.buffer,
  };

  // ── 发送加密请求 ──────────────────────────────────────────────────
  const rawResp = await _originalFetch(url, newInit);

  // ── 解密响应 ──────────────────────────────────────────────────────
  const ct = rawResp.headers.get('content-type') ?? '';
  if (!ct.includes('application/octet-stream')) {
    // 非加密响应（如404、握手错误等），直接透传
    return rawResp;
  }

  const respBuf = await rawResp.arrayBuffer();
  const respBytes = new Uint8Array(respBuf);

  let plainText: string;
  try {
    const decrypted = await cxDecrypt(respBytes);
    plainText = TEXT_DECODER.decode(decrypted);
  } catch {
    // 解密失败时返回原始响应（以防服务端未启用加密中间件的路由）
    return new Response(respBuf, {
      status: rawResp.status,
      headers: rawResp.headers,
    });
  }

  return new Response(plainText, {
    status: rawResp.status,
    headers: new Headers({
      'Content-Type': 'application/json',
    }),
  });
}

/**
 * installCxSecInterceptor - 调用一次即可全局生效
 */
export function installCxSecInterceptor(): void {
  (window as any).fetch = cxFetch;
}

export { cxFetch };
