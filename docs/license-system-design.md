# 阿卡夏授权管理系统 · 设计文档

> 版本：v1 设计稿　|　日期：2026-08-04
> 代号：**流光** (Ruliu) — 让授权像流水一样顺畅，不再卡在网络上

---

## 一、为什么要做这个系统

现有 AKasha 的授权门禁把三件事全绑在 GitHub 一条外网链路上：

| 需求 | 现在的做法 | 问题 |
|---|---|---|
| 身份认证 | GitHub Device Flow | 国内访问不稳 |
| 唯一绑定 | Gist 当账本 | 无事务、无 CAS、要下发写权限 token |
| 远程吊销 | 查组织成员 + Gist | 网络失败 3 次即**自毁全站** |

本系统把这三件事拆开，各自用最合适的机制承载：

- **身份认证** → GitHub OAuth（只在用户登录后台时用，客户实例完全不碰）
- **唯一绑定** → 自建数据库（有事务、有约束、可审计）
- **授权验证** → Ed25519 离线签名许可证（客户实例零网络依赖）
- **远程吊销** → 签名吊销表（拉不到就跳过，绝不影响业务）

**核心原则：授权系统挂掉，客户的生意照做。**

---

## 二、技术选型

| 层 | 选择 | 说明 |
|---|---|---|
| 框架 | Next.js 15 (App Router) | 前后端一体，Route Handlers 当 API |
| 语言 | TypeScript (strict) | |
| UI | HeroUI v3 + Tailwind CSS v4 | v3 要求 Tailwind v4，不是 v3 |
| ORM | Prisma | 同时支持 PostgreSQL / MySQL |
| 数据库 | PostgreSQL 或 MySQL | 二选一，schema 取最小公分母 |
| 签名 | Ed25519 (node:crypto) | 二进制里只放公钥 |
| 密码 | Argon2id | |
| 会话 | 数据库 Session + httpOnly Cookie | 便于管理员强制下线 |
| 校验 | Zod | 所有入参 |

### 双数据库兼容策略

Prisma 的 `provider` 在 schema 里是静态值，用构建脚本切换：

```
prisma/
├── schema.base.prisma      # 模型定义（共用）
├── schema.postgresql.prisma # datasource + generator
├── schema.mysql.prisma
└── build-schema.mjs         # 按 DB_PROVIDER 拼装 schema.prisma
```

**schema 必须遵守的最小公分母约束：**

- ❌ 不用 PG 的 `jsonb` 特有操作符 → 只做整体读写，用 Prisma `Json`
- ❌ 不用部分索引（MySQL 不支持）→「每用户活跃实例唯一」在应用层事务内保证
- ❌ 不用数组类型 → 用关联表或 Json
- ❌ 不依赖 `citext` → GitHub login **统一转小写存储**
- ⚠️ MySQL 需 `utf8mb4` + `utf8mb4_unicode_ci`
- ⚠️ 字符串索引字段长度显式声明（MySQL 索引长度限制）

---

## 三、系统组成

```
┌──────────────────────────────────────────────────────┐
│                   流光 授权管理系统                     │
│                                                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────────┐  │
│  │  管理员后台  │  │  用户后台   │  │  实例接口(机器)  │  │
│  │  账密登录    │  │ GitHub登录  │  │   无 Session    │  │
│  └────────────┘  └────────────┘  └────────────────┘  │
│         │               │                  │          │
│         └───────────────┴──────────────────┘          │
│                         │                             │
│              ┌──────────▼──────────┐                  │
│              │   核心服务层          │                  │
│              │  签发 / 吊销 / 审计   │                  │
│              └──────────┬──────────┘                  │
│                         │                             │
│              ┌──────────▼──────────┐                  │
│              │  PostgreSQL / MySQL │                  │
│              └─────────────────────┘                  │
└───────────────────────┬──────────────────────────────┘
                        │ 只读、无凭证、可 CDN 加速
                        ▼
              ┌──────────────────────┐
              │   AKasha 客户实例      │
              │  本地验签 + 定期拉吊销  │
              └──────────────────────┘
```

---

## 四、数据模型

### 4.1 AdminUser 管理员

```prisma
model AdminUser {
  id           String    @id @default(cuid())
  username     String    @unique @db.VarChar(64)
  passwordHash String    @db.VarChar(255)   // Argon2id
  displayName  String    @db.VarChar(64)
  role         AdminRole @default(ADMIN)     // OWNER / ADMIN / VIEWER
  totpSecret   String?   @db.VarChar(255)    // 加密存储，可选二步验证
  isActive     Boolean   @default(true)
  lastLoginAt  DateTime?
  lastLoginIp  String?   @db.VarChar(64)
  createdAt    DateTime  @default(now())
  updatedAt    DateTime  @updatedAt
}
```

- `OWNER` 唯一，不可删除，可管理其他管理员
- 首次启动走**初始化向导**创建 OWNER（类似 AKasha 的 setup 流程）

### 4.2 SystemConfig 系统配置

单行表（`id` 固定为 `"singleton"`），敏感字段 AES-256-GCM 加密：

```prisma
model SystemConfig {
  id                    String   @id @default("singleton")

  // GitHub 配置
  githubClientId        String?  @db.VarChar(255)
  githubClientSecret    String?  @db.Text      // 🔒 加密
  githubOrg             String?  @db.VarChar(128)
  githubTeamSlug        String?  @db.VarChar(128)
  githubApiBase         String   @default("https://api.github.com")  // 可指向中转
  githubWebBase         String   @default("https://github.com")
  githubProxyUrl        String?  @db.VarChar(255)  // 可选 HTTP 代理

  // 签名密钥
  licensePublicKey      String?  @db.Text
  licensePrivateKey     String?  @db.Text      // 🔒 加密
  keyGeneratedAt        DateTime?
  keyVersion            Int      @default(1)

  // 默认策略
  defaultInstanceQuota  Int      @default(3)
  defaultLicenseTtlDays Int      @default(365)
  defaultVerifyMode     VerifyMode @default(OFFLINE)
  heartbeatIntervalSec  Int      @default(21600)   // 6h
  strictGracePeriodSec  Int      @default(604800)  // 7d

  // 激活码策略
  activationCodeTtlMin  Int      @default(60)
  authCodeTtlSec        Int      @default(600)
  allowedCallbackHosts  Json?    // string[]，空=允许任意 https

  // 站点信息
  siteName              String   @default("流光授权中心")
  siteLogoUrl           String?  @db.VarChar(500)
  contactInfo           String?  @db.Text

  updatedAt             DateTime @updatedAt
  updatedBy             String?  @db.VarChar(64)
}
```

**注意**：`githubApiBase` / `githubProxyUrl` 就是为「自建中转」预留的口子——填了就走中转，不填走官方。

### 4.3 User 用户

```prisma
model User {
  id            String     @id @default(cuid())
  githubId      String     @unique @db.VarChar(64)
  githubLogin   String     @unique @db.VarChar(128)  // 统一小写
  displayName   String?    @db.VarChar(128)
  email         String?    @db.VarChar(255)
  avatarUrl     String?    @db.VarChar(500)

  status        UserStatus @default(ACTIVE)   // ACTIVE / SUSPENDED
  instanceQuota Int?                          // null = 用全局默认
  note          String?    @db.Text           // 管理员备注

  lastLoginAt   DateTime?
  lastLoginIp   String?    @db.VarChar(64)
  teamCheckedAt DateTime?                     // 上次校验 team 成员身份的时间
  createdAt     DateTime   @default(now())
  updatedAt     DateTime   @updatedAt

  instances       Instance[]
  activationCodes ActivationCode[]
  sessions        UserSession[]

  @@index([status])
}
```

### 4.4 Instance 实例

```prisma
model Instance {
  id            String         @id @default(cuid())
  userId        String
  user          User           @relation(fields: [userId], references: [id], onDelete: Cascade)

  name          String         @db.VarChar(128)     // 用户自定义，如"生产环境"
  fingerprint   String         @unique @db.VarChar(128)
  status        InstanceStatus @default(ACTIVE)     // ACTIVE / REVOKED / SUSPENDED

  verifyMode    VerifyMode     @default(OFFLINE)    // OFFLINE / STRICT
  licenseTtlDays Int?                               // null = 用全局默认

  // 运行时观测
  lastSeenAt    DateTime?
  lastSeenIp    String?        @db.VarChar(64)
  appVersion    String?        @db.VarChar(64)
  hostInfo      Json?                               // OS / 架构 / 主机名等

  activatedAt   DateTime       @default(now())
  revokedAt     DateTime?
  revokedReason String?        @db.Text
  revokedBy     String?        @db.VarChar(64)

  createdAt     DateTime       @default(now())
  updatedAt     DateTime       @updatedAt

  licenses      License[]

  @@index([userId, status])
  @@index([status])
  @@index([lastSeenAt])
}
```

**配额约束**：`ACTIVE` 实例数 ≤ 用户配额。MySQL 无部分索引，所以在**事务内 `SELECT ... FOR UPDATE` 用户行 + 计数**来保证。

### 4.5 License 许可证

```prisma
model License {
  id          String    @id @default(cuid())
  serial      String    @unique @db.VarChar(64)   // LIC-XXXX-XXXX
  instanceId  String
  instance    Instance  @relation(fields: [instanceId], references: [id], onDelete: Cascade)

  keyVersion  Int       @default(1)
  payload     Json                                 // 签名前的完整载荷
  signature   String    @db.Text                   // base64url(Ed25519)

  issuedAt    DateTime  @default(now())
  expiresAt   DateTime
  revokedAt   DateTime?
  supersededBy String?  @db.VarChar(64)            // 被哪张新证替代

  @@index([instanceId, issuedAt])
  @@index([expiresAt])
}
```

保留历史签发记录，便于审计和排查「客户说他的证书不对」这类问题。

### 4.6 ActivationCode 激活码

```prisma
model ActivationCode {
  id           String     @id @default(cuid())
  userId       String
  user         User       @relation(fields: [userId], references: [id], onDelete: Cascade)

  codeHash     String     @unique @db.VarChar(128)  // SHA-256，明文只显示一次
  codePrefix   String     @db.VarChar(16)           // 前几位，用于列表展示
  instanceName String     @db.VarChar(128)          // 预设的实例名

  status       CodeStatus @default(UNUSED)          // UNUSED / USED / EXPIRED / CANCELLED
  expiresAt    DateTime
  usedAt       DateTime?
  usedByInstanceId String? @db.VarChar(64)
  usedFromIp   String?    @db.VarChar(64)

  createdAt    DateTime   @default(now())

  @@index([userId, status])
  @@index([expiresAt])
}
```

### 4.7 AuthorizationCode 授权码（OAuth 跳转模式）

```prisma
model AuthorizationCode {
  id           String   @id @default(cuid())
  userId       String
  codeHash     String   @unique @db.VarChar(128)
  fingerprint  String   @db.VarChar(128)      // 🔑 绑死指纹，防 callback 劫持
  instanceName String?  @db.VarChar(128)
  callbackUrl  String   @db.VarChar(500)
  state        String?  @db.VarChar(255)

  expiresAt    DateTime                        // 默认 10 分钟
  usedAt       DateTime?
  createdAt    DateTime @default(now())

  @@index([expiresAt])
}
```

### 4.8 Session 会话

```prisma
model UserSession {
  id         String   @id @default(cuid())
  userId     String
  user       User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  tokenHash  String   @unique @db.VarChar(128)
  ip         String?  @db.VarChar(64)
  userAgent  String?  @db.VarChar(500)
  expiresAt  DateTime
  createdAt  DateTime @default(now())
  @@index([expiresAt])
}

model AdminSession {
  id         String   @id @default(cuid())
  adminId    String
  tokenHash  String   @unique @db.VarChar(128)
  ip         String?  @db.VarChar(64)
  userAgent  String?  @db.VarChar(500)
  expiresAt  DateTime
  createdAt  DateTime @default(now())
  @@index([expiresAt])
}
```

### 4.9 AuditLog 审计日志

```prisma
model AuditLog {
  id         String    @id @default(cuid())
  actorType  ActorType                     // ADMIN / USER / INSTANCE / SYSTEM
  actorId    String?   @db.VarChar(64)
  actorLabel String?   @db.VarChar(128)    // 冗余存名字，便于查询

  action     String    @db.VarChar(64)     // instance.activate / instance.revoke ...
  targetType String?   @db.VarChar(32)
  targetId   String?   @db.VarChar(64)

  result     String    @db.VarChar(16)     // success / failure
  detail     Json?
  ip         String?   @db.VarChar(64)
  userAgent  String?   @db.VarChar(500)
  createdAt  DateTime  @default(now())

  @@index([actorType, actorId])
  @@index([action])
  @@index([createdAt])
}
```

### 4.10 枚举

```prisma
enum AdminRole      { OWNER ADMIN VIEWER }
enum UserStatus     { ACTIVE SUSPENDED }
enum InstanceStatus { ACTIVE REVOKED SUSPENDED }
enum VerifyMode     { OFFLINE STRICT }
enum CodeStatus     { UNUSED USED EXPIRED CANCELLED }
enum ActorType      { ADMIN USER INSTANCE SYSTEM }
```

---

## 五、核心流程

### 5.1 用户登录（GitHub OAuth + Team 校验）

```
用户点「用 GitHub 登录」
  ↓
跳转 {githubWebBase}/login/oauth/authorize
     ?client_id=...&scope=read:org&state=<CSRF随机串>
  ↓
GitHub 回调 /api/auth/github/callback?code=...&state=...
  ↓
① 校验 state（防 CSRF）
② code → access_token
③ GET /user 拿 login + id + avatar
④ 🔑 校验 Team 成员身份：
   GET /orgs/{org}/teams/{team_slug}/memberships/{login}
   用【用户自己的 token】查 —— 查自己不受"隐藏成员身份"影响
   要求 state === "active"（"pending" 是受邀未接受，不算）
  ↓
⑤ 不在 team → 拒绝，友好提示"你还不在 xxx 团队里哦"
   在 team  → upsert User → 建 Session → 写审计日志
```

**关键点：access_token 用完即弃，不存库。** 我们不需要长期持有用户的 GitHub 凭证。

Team 成员身份会在每次登录时重新校验，所以「踢出 team」下次登录自动失效。已登录的 session 由管理员手动处理或等自然过期。

### 5.2 激活方式 A：激活码模式

```
【用户后台】
点「新建实例」→ 填实例名
  ↓
后端事务：
  SELECT user FOR UPDATE
  校验 status = ACTIVE
  校验 活跃实例数 < 配额
  生成 32 位激活码（Crockford Base32，去掉易混字符）
  存 SHA-256 hash，明文只在响应里返回一次
  ↓
前端弹窗展示：AKS-7F2K-9QX4-M3P8
（大字号 + 一键复制 + 提示"只显示这一次哦"）

【AKasha 实例】
后台粘贴激活码 → POST /api/v1/activate
{
  "grant_type": "activation_code",
  "code": "AKS-7F2K-9QX4-M3P8",
  "fingerprint": "fp_a3f9...",
  "app_version": "1.2.0",
  "host_info": { "os": "linux", "arch": "amd64", "hostname": "prod-1" }
}
  ↓
后端事务：
  查码 → 校验 UNUSED / 未过期
  查用户 → 校验 ACTIVE
  校验配额（再查一次，防并发）
  校验 fingerprint 未被其他实例占用
  创建 Instance
  签发 License
  标记码 USED
  写审计日志
  ↓
返回 { license: {...}, signature: "...", public_key: "..." }
```

### 5.3 激活方式 B：OAuth 跳转模式

```
【AKasha 实例】
点「去授权中心绑定」→ 浏览器跳转：
  https://license.example.com/activate
    ?fp=<fingerprint>
    &cb=<callback_url>
    &state=<随机串>
    &name=<建议实例名>

【授权系统】
① 未登录 → 先走 GitHub 登录（登录后回到这个页面）
② 展示确认页（萌系确认卡片）：
   ┌────────────────────────────────┐
   │  🌸 有个实例想和你绑定～         │
   │                                │
   │  实例名称：生产环境              │
   │  设备指纹：fp_a3f9...c2e8       │
   │  回调地址：client.example.com   │  ← 域名高亮，让用户能判断
   │  剩余配额：2 / 3                │
   │                                │
   │  [ 确认绑定 ]   [ 取消 ]        │
   └────────────────────────────────┘
③ 校验 callback 合法性：
   - 必须 https（localhost/127.0.0.1 例外，便于开发）
   - 若配置了 allowedCallbackHosts 白名单，必须匹配
④ 用户确认 → 生成一次性授权码（10 分钟有效，🔑绑死 fingerprint）
⑤ 302 → {callback}?code=<授权码>&state=<原样回传>

【AKasha 实例】
收到 code → POST /api/v1/activate
{
  "grant_type": "authorization_code",
  "code": "...",
  "fingerprint": "fp_a3f9...",   // 🔑 必须与授权码里的一致
  ...
}
  ↓
fingerprint 不匹配 → 直接拒绝
```

**安全设计**：即使 callback 被篡改导致 code 泄漏，攻击者手里没有匹配的 fingerprint，换不到许可证。这是这个流程的核心防线。

### 5.4 许可证格式

```json
{
  "v": 1,
  "serial": "LIC-7F2K9QX4-M3P8",
  "key_version": 1,
  "instance": {
    "id": "inst_clx7f2k9",
    "fingerprint": "fp_a3f9c2e8b1d4",
    "name": "生产环境"
  },
  "subject": {
    "user_id": "usr_clx3m8p2",
    "github_login": "someone"
  },
  "issued_at": 1754265600,
  "expires_at": 1785801600,
  "policy": {
    "verify_mode": "offline",
    "heartbeat_interval": 21600,
    "grace_period": 604800,
    "revocation_url": "https://license.example.com/api/v1/revocations",
    "revocation_mirrors": [
      "https://cdn.jsdelivr.net/gh/xxx/yyy@main/revocations.json"
    ]
  },
  "features": ["core"],
  "issuer": "ruliu-license-center"
}
```

签名方式：**JCS 规范化 JSON**（RFC 8785 风格，键排序 + 无多余空白）→ Ed25519 签名 → base64url。

客户端存储格式（单文件 `license.dat`）：
```
<base64url(payload)>.<base64url(signature)>
```

### 5.5 混合验证模式

模式写进**签名过的许可证**里，客户端无法篡改：

**OFFLINE 模式（默认）**
```
启动 → 本地验签 → 检查 expires_at → 通过即可工作
每 heartbeat_interval 拉一次吊销表（失败静默跳过）
命中吊销 → 停止服务
到期前 30 天 → 后台提示续期
```
授权系统整个挂掉 → 客户完全不受影响。

**STRICT 模式（管理员可对单实例开启）**
```
同 OFFLINE，但额外要求：
每 heartbeat_interval 必须成功心跳一次
连续失败超过 grace_period（默认 7 天）→ 进入受限状态
心跳成功即恢复，并顺便续期许可证
```
即使 STRICT，也有 7 天宽限期，不会像现在那样 3 次失败就自毁。

### 5.6 心跳

```
POST /api/v1/heartbeat
{
  "serial": "LIC-...",
  "fingerprint": "fp_...",
  "nonce": "<随机串>",
  "timestamp": 1754265600,
  "signature": "<HMAC(serial+fingerprint+nonce+timestamp, license_signature)>"
}
```

用许可证签名本身当 HMAC 密钥——只有持有合法许可证的实例才能构造出正确签名，无需额外下发凭证。

响应：
```json
{
  "status": "active",          // active / revoked / suspended
  "server_time": 1754265600,
  "renewed_license": { ... },  // 临近到期时自动续签
  "message": null              // 给管理员留的通知通道
}
```

### 5.7 吊销表

```json
{
  "v": 1,
  "generated_at": 1754265600,
  "next_update": 1754287200,
  "key_version": 1,
  "revoked": [
    "sha256:3f9c2e8b1d4a...",
    "sha256:7a1f4e9c2b8d..."
  ],
  "signature": "..."
}
```

- 存**指纹的 SHA-256**而不是明文，避免泄漏客户设备信息
- 整体签名，防篡改
- 强缓存友好，可以直接挂 CDN / 多镜像
- 客户端拉取失败 → **静默跳过，视为未吊销**

---

## 六、API 一览

### 6.1 实例接口（机器，无 Session）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/activate` | 激活（激活码 / 授权码两种 grant_type） |
| POST | `/api/v1/heartbeat` | 心跳 + 自动续期 |
| GET | `/api/v1/revocations` | 吊销表（可 CDN 缓存） |
| GET | `/api/v1/pubkey` | 当前公钥（含历史版本，便于轮换） |

### 6.2 用户后台

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/auth/github` | 发起 OAuth |
| GET | `/api/auth/github/callback` | 回调 + Team 校验 |
| POST | `/api/auth/logout` | 退出 |
| GET | `/api/me` | 当前用户 + 配额 |
| GET | `/api/me/instances` | 我的实例列表 |
| POST | `/api/me/instances` | 新建实例 → 返回激活码 |
| PATCH | `/api/me/instances/:id` | 改名 |
| DELETE | `/api/me/instances/:id` | 解绑 |
| GET | `/api/me/codes` | 我的激活码（未使用的） |
| DELETE | `/api/me/codes/:id` | 作废激活码 |
| POST | `/api/activate/approve` | OAuth 跳转模式的确认动作 |

### 6.3 管理员后台

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/admin/setup` | 首次初始化（创建 OWNER） |
| POST | `/api/admin/login` | 账密登录 |
| POST | `/api/admin/logout` | |
| GET/PUT | `/api/admin/config` | 系统配置 |
| POST | `/api/admin/config/test-github` | 测试 GitHub 连通性（含中转） |
| POST | `/api/admin/keys/generate` | 生成签名密钥对 |
| POST | `/api/admin/keys/rotate` | 轮换密钥（旧证仍可验证） |
| GET | `/api/admin/users` | 用户列表（搜索/筛选/分页） |
| GET | `/api/admin/users/:id` | 用户详情 + 其实例 |
| PATCH | `/api/admin/users/:id` | 改配额 / 停用 / 备注 |
| GET | `/api/admin/instances` | 全部实例（搜索/筛选） |
| PATCH | `/api/admin/instances/:id` | 吊销 / 恢复 / 改验证模式 / 延期 |
| POST | `/api/admin/instances/:id/reissue` | 重新签发许可证 |
| GET | `/api/admin/licenses` | 签发历史 |
| GET | `/api/admin/audit-logs` | 审计日志 |
| GET | `/api/admin/stats` | 仪表盘统计 |
| GET/POST | `/api/admin/admins` | 管理员管理（仅 OWNER） |

---

## 七、页面结构

### 7.1 用户后台

```
/                       落地页（未登录：介绍 + 登录按钮）
/dashboard              我的实例总览
  ├── 配额卡片（环形进度，剩余 2/3）
  ├── 实例卡片列表（状态、最后在线、版本）
  └── [+ 新建实例]
/instances/:id          实例详情（运行信息 / 许可证信息 / 操作）
/activate               OAuth 跳转确认页
/help                   接入指引（怎么在 AKasha 里填激活码）
```

### 7.2 管理员后台

```
/admin/setup            首次初始化向导
/admin/login            登录
/admin                  仪表盘（实例数 / 活跃数 / 今日激活 / 异常提醒）
/admin/users            用户管理
/admin/users/:id        用户详情
/admin/instances        实例管理
/admin/instances/:id    实例详情
/admin/licenses         签发记录
/admin/config           系统配置
  ├── GitHub 配置（含中转地址、代理、连通性测试）
  ├── 签名密钥（生成 / 轮换 / 查看公钥）
  ├── 默认策略（配额 / 有效期 / 验证模式）
  └── 站点信息
/admin/audit            审计日志
/admin/admins           管理员账号（仅 OWNER）
```

---

## 八、萌系治愈风 UI 设计

### 8.1 设计基调

不是「加几个圆角和粉色」，而是**降低用户的操作焦虑**——授权系统天然让人紧张（怕点错、怕失效），治愈风的作用是让每一步都显得温和可逆。

### 8.2 色板

```css
/* 主色：樱花粉 —— 用于主按钮、强调 */
--brand-50:  oklch(0.97 0.015 350);
--brand-200: oklch(0.90 0.055 350);
--brand-500: oklch(0.72 0.13  350);
--brand-600: oklch(0.65 0.14  350);

/* 辅色：薄荷绿 —— 成功、在线状态 */
--mint-500:  oklch(0.75 0.11 165);

/* 辅色：天空蓝 —— 信息、链接 */
--sky-500:   oklch(0.72 0.12 240);

/* 警示：柔化的珊瑚色，不用刺目的正红 */
--coral-500: oklch(0.70 0.15 25);

/* 背景：奶油白 → 极浅灰渐变 */
--bg-base:   oklch(0.99 0.005 90);
--bg-subtle: oklch(0.97 0.008 90);
```

**关键**：危险操作（解绑/吊销）用珊瑚色而非正红——保持警示性但不制造恐慌。

### 8.3 形态语言

| 元素 | 规格 |
|---|---|
| 圆角 | 卡片 `1rem`，按钮 `0.75rem`，输入框 `0.625rem` |
| 阴影 | 极柔，`0 2px 12px oklch(0.7 0.05 350 / 0.08)` |
| 间距 | 宽松，卡片内 padding `1.5rem` 起 |
| 字体 | 中文优先圆体（如「阿里巴巴普惠体 / 站酷快乐体」），英文 Nunito |
| 动效 | hover 上浮 2px + 阴影加深，200ms ease-out |

### 8.4 HeroUI v3 组件映射

| 场景 | 组件 |
|---|---|
| 实例卡片 | `Card` + `Chip`（状态）+ `Tooltip` |
| 配额展示 | `ProgressCircle` |
| 激活码展示 | `Modal` + `InputOTP` 风格分段 + 一键复制 |
| 激活码输入 | `InputOTP` |
| 解绑确认 | `AlertDialog` |
| 操作反馈 | `Toast` |
| 管理表格 | `Table` + `Pagination` + `SearchField` |
| 配置分区 | `Tabs` + `Fieldset` |
| 开关项 | `Switch` |
| 加载态 | `Skeleton`（不用转圈，减少焦虑） |
| 空状态 | 插画 + `Typography` + 引导按钮 |
| 用户头像 | `Avatar` |
| 侧边导航 | `ListBox` |

### 8.5 文案风格

用二次元语气，但**不牺牲信息密度**：

| 场景 | 文案 |
|---|---|
| 空实例列表 | 「还没有绑定任何实例呢～点下面的按钮开始吧 (｡•ᴗ•｡)」 |
| 激活码弹窗 | 「这串激活码只会显示这一次哦，记得先复制～」 |
| 解绑确认 | 「确定要解绑「生产环境」吗？解绑后这个实例会停止工作，但你可以随时重新绑定 ✧」 |
| 配额用尽 | 「配额用完啦（3/3），解绑一个不用的实例，或者找管理员申请更多～」 |
| 非 team 成员 | 「你还不在 xxx 团队里呢，请联系管理员邀请你加入 (｡•́︿•̀｡)」 |
| 操作成功 | 「搞定啦～」 |

---

## 九、安全设计

| 项 | 措施 |
|---|---|
| 管理员密码 | Argon2id（memory 64MB, iterations 3） |
| 敏感配置 | AES-256-GCM，密钥来自 `ENCRYPTION_KEY` 环境变量 |
| 签名私钥 | 加密存库，永不出现在 API 响应里 |
| 激活码/授权码 | 只存 SHA-256，明文仅返回一次 |
| Session | httpOnly + Secure + SameSite=Lax，存库可强制下线 |
| CSRF | OAuth state + 表单双提交 Cookie |
| 开放重定向 | callback 白名单 + https 强制 + 指纹绑定兜底 |
| 速率限制 | 激活 10/min/IP，登录 5/min/IP，心跳 60/h/实例 |
| 时序攻击 | 码校验用 `timingSafeEqual` |
| 审计 | 所有写操作落 AuditLog |
| 并发 | 配额校验在事务内 `FOR UPDATE` |

### 密钥轮换

`/api/v1/pubkey` 返回**所有历史公钥**：

```json
{
  "current": 2,
  "keys": [
    { "version": 1, "key": "...", "retired_at": 1754265600 },
    { "version": 2, "key": "...", "retired_at": null }
  ]
}
```

客户端按许可证里的 `key_version` 选对应公钥验签，老证书不会因为轮换而失效。

---

## 十、目录结构

```
ruliu-license/
├── prisma/
│   ├── schema.base.prisma
│   ├── schema.postgresql.prisma
│   ├── schema.mysql.prisma
│   ├── build-schema.mjs
│   └── seed.ts
├── src/
│   ├── app/
│   │   ├── (user)/                 用户后台路由组
│   │   ├── (admin)/admin/          管理员后台路由组
│   │   ├── activate/               OAuth 跳转确认页
│   │   └── api/
│   │       ├── v1/                 实例接口
│   │       ├── auth/               GitHub OAuth
│   │       ├── me/                 用户接口
│   │       └── admin/              管理接口
│   ├── components/
│   │   ├── ui/                     HeroUI 二次封装
│   │   ├── user/                   用户后台组件
│   │   ├── admin/                  管理后台组件
│   │   └── mascot/                 萌系插画/空状态
│   ├── lib/
│   │   ├── db/                     Prisma client
│   │   ├── auth/                   session / 密码 / 权限
│   │   ├── github/                 GitHub API 客户端（含中转/代理）
│   │   ├── license/                签发 / 验签 / 吊销表
│   │   ├── crypto/                 加密 / 哈希 / 码生成
│   │   ├── audit/                  审计日志
│   │   └── validation/             Zod schemas
│   └── styles/
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
└── docs/
    ├── DESIGN.md                   本文档
    ├── INTEGRATION.md              AKasha 端接入指引
    └── DEPLOY.md
```

**每个 lib 子目录一个职责**，单文件不超过 300 行。

---

## 十一、实施阶段

| 阶段 | 内容 | 产出 |
|---|---|---|
| **P1 地基** | 项目脚手架、Prisma 双库、加密工具、Session | 能跑起来的空壳 |
| **P2 管理员** | 初始化向导、登录、系统配置、密钥生成 | 管理员能配好 GitHub |
| **P3 用户侧** | GitHub OAuth + Team 校验、实例列表、激活码 | 用户能登录并生成激活码 |
| **P4 实例接口** | 激活（双模式）、心跳、吊销表、签发引擎 | 完整授权闭环 |
| **P5 管理增强** | 用户/实例管理、审计、仪表盘 | 运营能力完备 |
| **P6 美化** | 萌系主题、插画、动效、空状态 | 治愈风落地 |
| **P7 对接** | AKasha 客户端改造 + 接入文档 | 端到端打通 |

每个阶段独立可用，可以随时停下来验收。

---

## 十二、待确认

1. **部署位置** —— 授权系统本身放国内还是境外？影响 GitHub OAuth 的可达性（用户登录时浏览器要访问 github.com，这部分绕不开，但可以配中转）
2. **域名** —— 需要一个固定域名给客户实例回调和拉吊销表
3. **AKasha 端改造** —— 是在现有 `service/license` 上改，还是新起一个 `service/ruliu`？建议新起，保留旧逻辑一段时间做灰度
4. **吉祥物** —— 要不要设计一个看板娘？影响插画工作量
