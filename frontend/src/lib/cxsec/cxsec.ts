/**
 * 宸汐御安全 JS 桥接层
 * 本文件只是胶水层，不含任何算法——所有逻辑在 cx_runtime.wasm 内
 */

// ─── WASM 类型声明（WASM 导出的内部 API，混淆命名） ─────────────────────────
declare global {
  function __cx_init(
    serverPubHex: string,
    handshakeResp: Uint8Array,
    clientPrivHex: string,
    fpHex?: string
  ): number;
  function __cx_solve_pow(challengeJSON: string): string;
  function __cx_encrypt(handle: number, plaintext: Uint8Array): Uint8Array | null;
  function __cx_decrypt(handle: number, body: Uint8Array): Uint8Array | null;
  function __cx_destroy(handle: number): void;
  function __cx_fp(rawFpData: string): Uint8Array | null;
}

// ─── 协议配置 ─────────────────────────────────────────────────────────────────

export interface CxSecConfig {
  enabled: boolean;
  paths: string[];
}

// 默认保护路径，与后端 constants.go 默认值保持一致
const DEFAULT_PATHS = [
  '/api/user/login',
  '/api/user/register',
  '/api/user/login/2fa',
  '/api/user/password/reset-request',
  '/api/user/password/reset-confirm',
  '/api/user/checkin',
];

let _cxConfig: CxSecConfig = { enabled: true, paths: DEFAULT_PATHS };

/**
 * fetchCxSecConfig - 从 /api/cx/config 拉取服务端配置
 * 失败时使用内置默认值，保证页面可用
 */
export async function fetchCxSecConfig(): Promise<CxSecConfig> {
  try {
    const resp = await window.fetch('/api/cx/config');
    if (resp.ok) {
      const data = await resp.json() as CxSecConfig;
      if (typeof data.enabled === 'boolean' && Array.isArray(data.paths)) {
        _cxConfig = data;
      }
    }
  } catch {
    // 网络失败就用默认配置，不影响页面加载
  }
  return _cxConfig;
}

export function getCxSecConfig(): CxSecConfig {
  return _cxConfig;
}

// ─── 状态 ─────────────────────────────────────────────────────────────────────
let wasmReady = false;
let wasmLoading = false;
let wasmReadyPromise: Promise<void> | null = null;

let sessionHandle = -1;
let ephemeralPrivHex = '';
let initPending = false;
let _hintHex = ''; // 握手完成后存储，退出时用于通知服务端销毁会话

// ─── WASM 加载 ────────────────────────────────────────────────────────────────

async function loadWasm(): Promise<void> {
  if (wasmReady) return;
  if (wasmLoading && wasmReadyPromise) return wasmReadyPromise;

  wasmLoading = true;
  wasmReadyPromise = (async () => {
    // 加载 Go WASM 胶水
    // 动态加载 wasm_exec.js（非 ES 模块，用 script 标签注入）
    await new Promise<void>((resolve, reject) => {
      if ((window as any).Go) { resolve(); return; }
      const s = document.createElement('script');
      s.src = '/lib/wasm_exec.js';
      s.onload = () => resolve();
      s.onerror = reject;
      document.head.appendChild(s);
    });

    const go = new (window as any).Go();
    const resp = await fetch('/lib/cx_runtime.wasm');
    const buf = await resp.arrayBuffer();
    const result = await WebAssembly.instantiate(buf, go.importObject);
    go.run(result.instance);

    // 等待 WASM 函数注册（__cx_init 等）
    await new Promise<void>((resolve) => {
      const check = () => {
        if (typeof (window as any).__cx_init === 'function') {
          resolve();
        } else {
          setTimeout(check, 20);
        }
      };
      check();
    });

    wasmReady = true;
  })();

  return wasmReadyPromise;
}

// ─── 环境指纹采集 ─────────────────────────────────────────────────────────────

// ～页面加载时生成一次，Tab 关闭即丢弃，解决 Safari/iOS 静态属性固定的问题～
const _tabNonce = Array.from(
  crypto.getRandomValues(new Uint8Array(8)),
  b => b.toString(16).padStart(2, '0')
).join('');

function collectFingerprint(): string {
  const parts = [
    navigator.userAgent,
    screen.width + 'x' + screen.height,
    navigator.language,
    navigator.hardwareConcurrency?.toString() ?? '4',
    new Date().getTimezoneOffset().toString(),
    navigator.platform ?? '',
    (navigator as any).deviceMemory?.toString() ?? '',
    _tabNonce,
  ];
  return parts.join('|');
}

// ─── PoW 求解（Web Worker，非阻塞） ──────────────────────────────────────────

async function solvePoW(challenge: {
  nonce: string;
  diff: number;
  ts: number;
}): Promise<{ solution: string; argon_verify: string }> {
  await loadWasm();
  const resultStr = (window as any).__cx_solve_pow(JSON.stringify(challenge));
  if (!resultStr) throw new Error('宸汐: PoW solving failed');
  return JSON.parse(resultStr);
}

// ─── 握手流程 ─────────────────────────────────────────────────────────────────

export async function initSession(): Promise<void> {
  if (sessionHandle >= 0) return; // 已有会话
  if (initPending) {
    // 等待并发握手完成
    await new Promise<void>((resolve) => {
      const check = () => { sessionHandle >= 0 ? resolve() : setTimeout(check, 50); };
      check();
    });
    return;
  }
  initPending = true;

  try {
    await loadWasm();

    // Phase 1: 获取挑战
    const challResp = await fetch('/api/cx/challenge');
    if (!challResp.ok) throw new Error('宸汐: challenge fetch failed');
    const chall = await challResp.json() as {
      cid: string; nonce: string; srv: string; ts: number; diff: number; ttl: number;
    };

    // Phase 2: 生成客户端临时 ECDH 密钥对（纯 JS，避免 WASM 暴露私钥生成）
    const clientKeyPair = await crypto.subtle.generateKey(
      { name: 'ECDH', namedCurve: 'P-256' },
      true,
      ['deriveKey']
    );
    const clientPubRaw = await crypto.subtle.exportKey('raw', clientKeyPair.publicKey);
    const clientPrivJwk = await crypto.subtle.exportKey('jwk', clientKeyPair.privateKey);
    // 私钥 d 分量 => 32B
    const privBytes = base64urlToBytes(clientPrivJwk.d!);
    ephemeralPrivHex = bytesToHex(privBytes);

    // 压缩公钥（33B）
    const pubUncompressed = new Uint8Array(clientPubRaw);
    const compressedPub = compressP256(pubUncompressed);

    // Phase 3: 求解 PoW（使用服务端下发的 nonce，不是 cid）
    const pow = await solvePoW({ nonce: chall.nonce, diff: chall.diff, ts: chall.ts });

    // Phase 4: 采集指纹
    const fpRaw = collectFingerprint();
    const fpBytes = (window as any).__cx_fp(fpRaw) as Uint8Array;

    // Phase 5: 发送握手请求 (binary body: cid32B + pub33B + sol8B + argon32B + fp32B)
    const cidBytes = hexToBytes(chall.cid);
    const solBytes = hexToBytes(pow.solution);
    const argonBytes = hexToBytes(pow.argon_verify);

    const body = new Uint8Array(32 + 33 + 8 + 32 + 32);
    let off = 0;
    body.set(cidBytes.slice(0, 32), off); off += 32;
    body.set(compressedPub, off); off += 33;
    body.set(solBytes, off); off += 8;
    body.set(argonBytes, off); off += 32;
    body.set(fpBytes.slice(0, 32), off);

    const ksResp = await fetch('/api/cx/ks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/octet-stream' },
      body: body.buffer,
    });
    if (!ksResp.ok) throw new Error('宸汐: handshake failed');
    const ksBody = new Uint8Array(await ksResp.arrayBuffer());

    // Phase 6: 初始化 WASM 会话
    const handle = (window as any).__cx_init(
      chall.srv,
      ksBody,
      ephemeralPrivHex,
      bytesToHex(fpBytes)
    ) as number;

    if (handle < 0) throw new Error('宸汐: session init failed');
    sessionHandle = handle;
    // 存储 hint（ksBody 前8字节），退出时通知服务端销毁会话
    _hintHex = Array.from(ksBody.slice(0, 8), b => b.toString(16).padStart(2, '0')).join('');
  } finally {
    initPending = false;
  }
}

// ─── 加解密公开 API ──────────────────────────────────────────────────────────

export async function cxEncrypt(plaintext: Uint8Array): Promise<Uint8Array> {
  await initSession();
  const result = (window as any).__cx_encrypt(sessionHandle, plaintext) as Uint8Array | null;
  if (!result) throw new Error('宸汐: encrypt failed');
  return result;
}

export async function cxDecrypt(body: Uint8Array): Promise<Uint8Array> {
  await initSession();
  const result = (window as any).__cx_decrypt(sessionHandle, body) as Uint8Array | null;
  if (!result) throw new Error('宸汐: decrypt failed');
  return result;
}

export async function destroySession(): Promise<void> {
  if (sessionHandle >= 0) {
    // 主动通知服务端销毁会话，切断密钥链，best-effort
    if (_hintHex) {
      try {
        await window.fetch('/api/cx/session', {
          method: 'DELETE',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ hint: _hintHex }),
        });
      } catch { /* 网络失败也无妨，服务端8小时TTL兜底 */ }
    }
    (window as any).__cx_destroy(sessionHandle);
    sessionHandle = -1;
    _hintHex = '';
  }
}

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

function hexToBytes(hex: string): Uint8Array {
  const len = hex.length >> 1;
  const out = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    out[i] = parseInt(hex.substr(i * 2, 2), 16);
  }
  return out;
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

function base64urlToBytes(b64: string): Uint8Array {
  const b64std = b64.replace(/-/g, '+').replace(/_/g, '/').padEnd(
    b64.length + (4 - b64.length % 4) % 4, '='
  );
  const bin = atob(b64std);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

// P-256 未压缩公钥 (65B, 0x04前缀) → 压缩公钥 (33B)
function compressP256(raw: Uint8Array): Uint8Array {
  const x = raw.slice(1, 33);
  const y = raw.slice(33, 65);
  const prefix = (y[31] & 1) === 0 ? 0x02 : 0x03;
  const out = new Uint8Array(33);
  out[0] = prefix;
  out.set(x, 1);
  return out;
}
