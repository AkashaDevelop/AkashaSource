# 宸汐御安全（CxSec）v2 通讯协议规范

## 目录

1. [设计目标](#设计目标)
2. [协议总览](#协议总览)
3. [密码学原语](#密码学原语)
4. [握手流程](#握手流程)
5. [密钥派生体系](#密钥派生体系)
6. [纪元棘轮（前向安全）](#纪元棘轮前向安全)
7. [请求加密格式](#请求加密格式)
8. [响应加密格式](#响应加密格式)
9. [内部明文块格式](#内部明文块格式)
10. [安全机制汇总](#安全机制汇总)
11. [会话生命周期](#会话生命周期)
12. [错误码与降级策略](#错误码与降级策略)
13. [版本标记与兼容性](#版本标记与兼容性)

---

## 设计目标

| 目标 | 实现手段 |
|---|---|
| 无明文密钥落盘 | 所有密钥均为运行时派生，不写入数据库或配置文件 |
| 密钥启动时随机生成 | 每次握手 `crypto/rand` 生成临时 ECDH 密钥对 |
| 前向安全 / 动态轮换 | 纪元棘轮：每 600 秒单向推进，旧密钥字节显式清零 |
| 防重放攻击 | Nonce 黑名单（120s TTL）+ 单调递增计数器 |
| 防批量爬取 | SHA3+Argon2id 双重 PoW 挑战（难度20位，约百万次尝试） |
| 防会话伪造 | HMAC-SHA3-256 御印签名，每 30 秒轮换签名密钥 |
| 防路径泄露 | AES-256-GCM 加密 + XOR 白化层 |
| 逆向困难 | garble 混淆构建（`-literals -tiny -seed=random`） |

---

## 协议总览

```
客户端                                              服务端
  │                                                   │
  │  GET /api/cx/challenge                            │
  │──────────────────────────────────────────────────>│
  │                                                   │  生成临时 ECDH P-256 密钥对
  │                                                   │  生成 PoW 挑战 nonce (32B)
  │  { cid, nonce, srv_pub, ts, diff, ttl }           │
  │<──────────────────────────────────────────────────│
  │                                                   │
  │  生成客户端临时 ECDH P-256 密钥对                  │
  │  用 Web Crypto API (浏览器) 生成                   │
  │  求解 PoW 挑战（WASM 内执行）                      │
  │  采集环境指纹并 SHA3-256 哈希                       │
  │                                                   │
  │  POST /api/cx/ks                                  │
  │  body (binary, 137B):                             │
  │  [ cid(32) | cli_pub(33) | sol(8) | argon(32)     │
  │    | fp(32) ]                                     │
  │──────────────────────────────────────────────────>│
  │                                                   │  验证 PoW
  │                                                   │  验证指纹格式
  │                                                   │  ECDH: shared = srv_priv × cli_pub
  │                                                   │  派生 epoch0_key
  │                                                   │  分配 hint(8B) + sessionID
  │                                                   │
  │  body (binary, 16B):                              │
  │  [ hint(8) | counter_init(4) | ttl(4) ]           │
  │<──────────────────────────────────────────────────│
  │                                                   │
  │  ECDH: shared = cli_priv × srv_pub                │
  │  派生 epoch0_key (与服务端相同)                    │
  │  === 握手完成，双方持有相同纪元密钥 ===             │
  │                                                   │
  │  POST /api/xxx  (加密请求)                         │
  │  [ hint(8) | nonce(16) | ciphertext | tag(16)      │
  │    | sig(32) ]                                    │
  │──────────────────────────────────────────────────>│
  │                                                   │  查找会话
  │                                                   │  验御印签名
  │                                                   │  nonce 去重
  │                                                   │  解密 payload
  │                                                   │  处理业务逻辑
  │  [ nonce(16) | ciphertext | tag(16) | sig(32) ]   │
  │<──────────────────────────────────────────────────│
  │  验签 + 解密响应                                   │
```

---

## 密码学原语

| 原语 | 算法 | 用途 |
|---|---|---|
| 密钥协商 | ECDH P-256 | 握手阶段建立共享密钥 |
| 密钥派生 | HKDF-SHA3-256 (Extract+Expand) | 所有子密钥派生 |
| 对称加密 | AES-256-GCM | 请求/响应内容加密 |
| 认证签名 | HMAC-SHA3-256 | 御印签名，覆盖整个报文 |
| PoW 初筛 | SHA3-256 | 快速过滤无效解 |
| PoW 内存难度 | Argon2id (64KB, 1iter, 1thread) | 防 ASIC/脚本批量破解 |
| 随机数生成 | `crypto/rand` | 所有密钥、nonce、挑战 |

---

## 握手流程

### Phase 1：获取挑战

**请求**

```
GET /api/cx/challenge
```

**响应** (JSON)

```json
{
  "cid":   "hex(32B)",   // 挑战 ID，UUID 级随机
  "nonce": "hex(32B)",   // PoW 挑战随机数
  "srv":   "hex(33B)",   // 服务端临时 ECDH P-256 压缩公钥
  "ts":    1234567890,   // 当前 timeHint = floor(unix/60)
  "diff":  20,           // PoW 难度位数（20 ≈ 百万次尝试）
  "algo":  "sha3+argon2id",
  "ttl":   60            // 挑战有效秒数
}
```

> 服务端生成一次性 ECDH 密钥对，私钥仅存在于内存 `ChallengeEntry` 中，TTL 结束后自动清理。

---

### Phase 2：客户端准备

客户端（WASM）依次执行：

1. 使用浏览器 `crypto.subtle.generateKey` 生成临时 P-256 密钥对
2. 导出私钥 `d` 分量（32B），传入 WASM 会话
3. 将未压缩公钥（65B）压缩为 33B 格式
4. 用 WASM 求解 PoW（`jsSolvePow`）
5. 采集环境指纹字符串，经 WASM `__cx_fp` 计算 SHA3-256(32B)

**PoW 求解算法**

```
input = nonce(32B) || solution_uint64_LE(8B) || timeHint_uint64_LE(8B)
h1    = SHA3-256(input)

if BigEndian.Uint32(h1[:4]) >= (1 << (32 - diff)):
    continue  // SHA3 初筛失败

argout = Argon2id(h1, nonce[:8], iter=1, mem=64KB, threads=1, outLen=8)
looseThres = 1 << (32 - diff + 2)

if BigEndian.Uint32(argout[:4]) < looseThres:
    found!  // 宽松阈值（比 SHA3 高2位），防止过慢
```

---

### Phase 3：发送握手请求

**请求**

```
POST /api/cx/ks
Content-Type: application/octet-stream
Body (137B binary):

Offset  Len  Field
  0     32   cid              (挑战 ID，hex decode)
 32     33   cli_pub          (客户端临时 ECDH P-256 压缩公钥)
 65      8   solution         (PoW 解答, uint64 小端)
 73     32   argon_verify     (Argon2id 预计算输出，32B，服务端可选核验)
105     32   fp               (环境指纹哈希，SHA3-256)
```

**响应** (Binary, 16B)

```
Offset  Len  Field
  0      8   hint             (会话索引，后续请求必须携带)
  8      4   counter_init     (计数器初值，通常为 0, uint32 小端)
 12      4   ttl              (会话有效秒数, uint32 小端, 通常 28800=8h)
```

---

### Phase 4：ECDH 密钥协商与 epoch0 派生

双方各自独立执行，结果必须相同：

```
shared = ECDH(my_priv, peer_pub)   // P-256 共享 x 坐标，32B

epoch0 = HKDF-Extract-Expand(
    hash  = SHA3-256,
    IKM   = shared,
    salt  = fpHash,             // 环境指纹哈希，绑定会话到浏览器环境
    info  = "宸汐御安全v2\x00epoch0\x00"
)  // 输出 32B

// ECDH 共享密钥字节立即清零
```

---

## 密钥派生体系

所有密钥派生均使用同一个底层函数 `hkdfSum32`：

```
hkdfSum32(secret, salt, info) -> [32]byte
= HKDF-Extract(hash=SHA3-256, IKM=secret, salt=salt)
  然后 HKDF-Expand(PRK, info, len=32)
```

**域分隔符（kdfDomain）**：`"宸汐御安全v2\x00"` — v2 标记协议破坏性升级，与 v1 不兼容。

### 派生树

```
epoch_key  (32B)
├── DeriveRequestKey(epoch_key, timeBlock, counter, nonce)
│   └── info = kdfDomain + "reqkey\x00" || timeBlock_LE8 || counter_LE4 || nonce
│   └── AES-256-GCM 加密密钥，绑定单次请求
│
├── DeriveHMACKey(epoch_key, timeBlock)
│   └── info = kdfDomain + "hmackey\x00" || timeBlock_LE8
│   └── HMAC-SHA3-256 御印签名密钥，30s 时间窗
│
└── DeriveWhiteningKey(epoch_key, sessionID)
    ├── pre  → info = kdfDomain + "whiten_pre\x00" + sessionID
    └── post → info = kdfDomain + "whiten_post\x00" + sessionID
            → AES-GCM 前后 XOR 白化，增加流量混淆
```

---

## 纪元棘轮（前向安全）

会话建立后，纪元密钥不再长期持有不变，而是按挂钟时间单向棘轮推进：

```
纪元编号 = floor(unix_timestamp / 600)

epoch(n+1) = HKDF(
    IKM  = epoch(n),
    salt = nil,
    info = kdfDomain + "ratchet\x00"
)

// epoch(n-1) 的字节在推进后立即 memset(0)
```

**效果**：
- 攻击者即使拿到某一时刻的内存快照，只能看见当前纪元和上一纪元的密钥
- 拿到 `epoch(n)` 无法算出 `epoch(n-1)`（单向哈希不可逆）
- 上一纪元密钥仅用于 **边界容错**（10 分钟切换点附近的跨纪元请求），随时间自动失效

**客户端同步**：客户端 WASM 根据 `Date.now()` 独立计算纪元编号，**无需额外握手消息同步**，服务端和客户端在相同挂钟时间派生相同纪元密钥。

---

## 请求加密格式

### Wire Format（Binary Body）

```
Offset        Len   Field
  0            8    hint          (会话索引，明文，用于查找 session)
  8           16    nonce         (本次请求随机 nonce，crypto/rand)
 24         var.    ciphertext    (加密后的 inner block，见下文)
 24+len(ct)  16    tag           (AES-256-GCM 认证标签)
 40+len(ct)  32    sig           (HMAC-SHA3-256 御印签名)
```

总计：56 + len(inner_block) 字节

### 加密过程

```
// 1. 派生本次请求所有密钥
epoch_key       = session.EpochKeys().cur
time_block      = floor(unix / 30)
req_key         = DeriveRequestKey(epoch_key, time_block, counter, nonce)
(pre, post)     = DeriveWhiteningKey(epoch_key, sessionID)
hmac_key        = DeriveHMACKey(epoch_key, time_block)

// 2. 构造 inner block（含随机 padding）
inner = counter_LE4 || ts_ms_LE8 || payload_len_LE4 || payload || padding || padlen_1B

// 3. XOR 前白化
whitened = inner XOR (pre 循环)

// 4. AES-256-GCM 加密
// IV = nonce[:12]（nonce 前 12 字节）
sealed = AES-256-GCM(key=req_key, IV=nonce[:12], plaintext=whitened)
ciphertext = sealed[:len-16]
tag        = sealed[len-16:]

// 5. XOR 后白化
ciphertext_final = ciphertext XOR (post 循环)

// 6. 组装 body（不含 sig）
body = hint || nonce || ciphertext_final || tag

// 7. 御印签名（覆盖全部 body）
sig = HMAC-SHA3-256(hmac_key, body)

// 8. 最终 wire body
wire_body = body || sig
```

---

## 响应加密格式

### Wire Format

```
Offset        Len   Field
  0           16    nonce         (响应随机 nonce，服务端生成)
 16         var.    ciphertext    (加密后的响应 inner block)
 16+len(ct)  16    tag
 32+len(ct)  32    sig
```

### 解密过程（客户端）

```
// 取当前 + 上一纪元密钥
(cur_epoch, prev_epoch) = session.EpochKeys()

// 尝试当前纪元 + 当前/前一时间窗；失败则尝试上一纪元
for epoch_key in [cur_epoch, prev_epoch]:
    for time_block in [floor(unix/30), floor(unix/30)-1]:
        hmac_key = DeriveHMACKey(epoch_key, time_block)
        if HMAC-SHA3-256(hmac_key, body_without_sig) == sig:
            verified = true
            break

req_key     = DeriveRequestKey(verified_epoch, time_block, counter, nonce)
(pre, post) = DeriveWhiteningKey(verified_epoch, sessionID)

// 逆向白化 + GCM 解密
de_post   = ciphertext XOR (post 循环)
inner     = AES-256-GCM-Decrypt(key=req_key, IV=nonce[:12], combined=de_post||tag)
de_pre    = inner XOR (pre 循环)

// 解析 inner block
counter = LE4(de_pre[0:4])
ts_ms   = LE8(de_pre[4:12])
pay_len = LE4(de_pre[12:16])
payload = de_pre[16 : 16+pay_len]
```

---

## 内部明文块格式

```
Offset  Len      Field
  0      4       counter          (uint32 LE，防重放，必须严格递增)
  4      8       ts_ms            (int64 LE，毫秒时间戳，±45s 偏差检测)
 12      4       payload_len      (uint32 LE)
 16    pay_len   payload          (原始 JSON bytes)
 16+n  0~127    random_padding   (随机填充，隐藏真实长度，最大 127B)
 last   1       pad_len          (padding 长度，uint8)
```

---

## 安全机制汇总

### 多层防御

```
请求到达
  │
  ├─ [1] 御印签名校验
  │      HMAC-SHA3-256 覆盖 hint+nonce+ciphertext+tag
  │      30s 时间窗（当前 + 前一窗口均可）
  │      签名失败 → 401
  │
  ├─ [2] Nonce 去重
  │      内存黑名单，120s TTL
  │      重放 nonce → 429
  │
  ├─ [3] AES-256-GCM 解密
  │      GCM tag 验证密文完整性
  │      篡改密文 → 401
  │
  ├─ [4] 内层时间戳校验
  │      |now - ts_ms| > 45000ms → 401
  │
  ├─ [5] 计数器校验
  │      counter != session.counter → 401
  │      通过后 counter++
  │
  └─ 注入业务 JSON → handler
```

### 时间窗容错矩阵

为避免时间边界处的合法请求被误拒，中间件同时接受：

| 纪元密钥 | 时间窗 |
|---|---|
| 当前纪元 | 当前 30s 窗口 |
| 当前纪元 | 前一个 30s 窗口 |
| 上一纪元 | 当前 30s 窗口 |
| 上一纪元 | 前一个 30s 窗口 |

共 4 种组合逐一尝试，首个通过即接受。

---

## 会话生命周期

```
创建: POST /api/cx/ks 成功后
  ├── SessionTTL = 8 小时
  ├── 纪元密钥: 每 10 分钟单向推进，旧密钥清零
  ├── Nonce 黑名单: 每次请求清理过期条目（>120s）
  ├── 计数器: 每次请求 +1，严格单调递增
  └── 最大 Nonce 数: 200,000 条（防内存爆满）

销毁:
  ├── 显式: 客户端调用 __cx_destroy(handle)
  └── 隐式: TTL 超时后，下次 Get() 返回 "session expired" 并删除

后台清理: PurgeExpiredChallenges() goroutine 每 2 分钟扫描
```

---

## 错误码与降级策略

| HTTP 状态码 | 触发条件 | 客户端行为 |
|---|---|---|
| 400 | 请求格式错误（body 过短/字段缺失） | 检查客户端序列化 |
| 401 | 签名错误 / 解密失败 / 计数器不匹配 / 时间戳超窗 | 重新握手（`initSession`） |
| 403 | PoW 验证失败 / 指纹格式校验失败 | 重新获取挑战并求解 |
| 429 | Nonce 重放 / Nonce 存储满 | 短暂退避后重试 |
| 200 (非 octet-stream) | 握手端点外的未加密响应 | 按原始 JSON 处理 |

---

## 版本标记与兼容性

| 版本 | 域分隔符 | 特性 |
|---|---|---|
| v1（已废弃） | `"宸汐御安全v1\x00"` | 自制 SHA3 KDF，无棘轮，MasterKey 长期持有 |
| **v2（当前）** | `"宸汐御安全v2\x00"` | 标准 HKDF-SHA3，纪元棘轮前向安全，并发安全 |

v1 与 v2 **完全不兼容**：任何用 v1 KDF 派生的密钥均无法通过 v2 的验签和解密，握手立即失败。由于 `CxSecMiddleware` 在协议升级至 v2 时尚未接入路由，无存量流量迁移负担。

---

## 附录：关键常量

```go
const (
    timeWindowSec      = 30           // HMAC 签名时间窗（秒）
    RatchetIntervalSec = 600          // 纪元棘轮周期（秒）
    kdfDomain          = "宸汐御安全v2\x00"

    PowDifficultyDefault = 20         // PoW 难度位数
    PowArgon2Memory      = 64         // Argon2id 内存 (KB)
    PowArgon2Iterations  = 1
    PowArgon2Threads     = 1
    PowTTLSec            = 120        // PoW 挑战有效期（秒）

    SessionTTL           = 8 * time.Hour
    NonceBlacklistTTL    = 120 * time.Second
    MaxNoncesPerSession  = 200_000

    allowedSkewMS        = 45_000     // 时间戳最大偏差（毫秒）

    // Wire format 各字段长度
    BodyHintLen  = 8
    BodyNonceLen = 16
    BodyTagLen   = 16
    BodySigLen   = 32
)
```
