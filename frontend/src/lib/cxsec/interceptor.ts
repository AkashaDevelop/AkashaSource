/**
 * 宸汐御安全 fetch 拦截器
 * 挂载后透明替换全局 fetch，只有配置列表内的路径才加密
 */
import { cxEncrypt, cxDecrypt, initSession } from './cxsec';

const _originalFetch = window.fetch.bind(window);
const TEXT_ENCODER = new TextEncoder();
const TEXT_DECODER = new TextDecoder();

// 永远豁免：握手端点本身不参与加密
const ALWAYS_EXEMPT = ['/api/cx/challenge', '/api/cx/ks', '/api/cx/config'];

// 受保护路径列表，由 installCxSecInterceptor 传入（或从 /api/cx/config 拉取后设置）
let _protectedPaths: string[] = [];

// 只有这几个方法才可能带 body，GET/HEAD 天生不该被加密（浏览器 fetch 规范禁止 GET/HEAD 带 body）
const BODY_METHODS = ['POST', 'PUT', 'PATCH'];

function needsEncryption(url: string, method: string): boolean {
  if (!BODY_METHODS.includes(method.toUpperCase())) return false;
  if (ALWAYS_EXEMPT.some(e => url.includes(e))) return false;
  return _protectedPaths.some(p => {
    // 精确前缀匹配（避免 /api/user/login 误匹配 /api/user/login-sso 之类）
    try {
      const parsed = new URL(url, window.location.origin);
      return parsed.pathname === p || parsed.pathname.startsWith(p + '/') || parsed.pathname.startsWith(p + '?');
    } catch {
      return url.includes(p);
    }
  });
}

async function cxFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = typeof input === 'string' ? input
    : input instanceof URL ? input.href
    : (input as Request).url;
  const method = (init?.method) ?? (input instanceof Request ? (input as Request).method : 'GET');

  if (!needsEncryption(url, method)) {
    return _originalFetch(input, init);
  }

  await initSession();

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

  const rawResp = await _originalFetch(url, newInit);

  const ct = rawResp.headers.get('content-type') ?? '';
  if (!ct.includes('application/octet-stream')) {
    return rawResp;
  }

  const respBuf = await rawResp.arrayBuffer();
  const respBytes = new Uint8Array(respBuf);

  try {
    const decrypted = await cxDecrypt(respBytes);
    return new Response(TEXT_DECODER.decode(decrypted), {
      status: rawResp.status,
      headers: new Headers({ 'Content-Type': 'application/json' }),
    });
  } catch {
    return new Response(respBuf, { status: rawResp.status, headers: rawResp.headers });
  }
}

/**
 * installCxSecInterceptor
 * @param paths 受保护的路径列表，不传则沿用上次设置
 */
export function installCxSecInterceptor(paths?: string[]): void {
  if (paths) _protectedPaths = paths;
  (window as any).fetch = cxFetch;
}

export { cxFetch };
