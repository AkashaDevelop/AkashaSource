
<div align="center">

# Akasha (阿卡夏)

**汇聚全宇宙智慧的 AI 网关系统**

[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-7-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![TailwindCSS](https://img.shields.io/badge/Tailwind-4-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)

</div>

---

Akasha 是一个统一的 AI API 网关，将多家大模型供应商聚合在 OpenAI 兼容接口之后，提供渠道负载均衡、用户管理、额度计费、速率限制、内容安全审核和运营管理等能力。

## ✨ 核心特性

**多模型聚合**
- 原生支持 OpenAI / Azure、Anthropic Claude、Google Gemini、Midjourney、Suno
- 国产模型：阿里通义千问、腾讯混元、百度文心一言、讯飞星火
- 第三方模型：Deepseek、智谱 ChatGLM、Moonshot (Kimi)、Ollama (本地部署)
- 工作流引擎：Dify ChatFlow 代理（自动转换 OpenAI 格式）
- 统一 OpenAI 兼容接口，支持 Chat、Completions、Embeddings、Audio (TTS/STT)、Images、Moderations、Rerank、Files

**CLI 工具兼容**
- Claude Code CLI — 原生 Anthropic Messages API (`/v1/messages`)，支持流式、tool_use、thinking
- OpenAI Codex CLI — Responses API (`/v1/responses`)，支持流式 SSE 事件
- OpenAI Realtime API — WebSocket 双向代理 (`/v1/realtime`)，支持语音对话
- Gemini 原生格式 — 直连 Gemini REST API (`/gemini/v1beta/models/:model`)
- neko-api-key-tool 兼容 — `GET /api/key/info` 查询 Key 额度信息

**渠道管理**
- 多渠道负载均衡（优先级 + 权重）、自动重试、熔断与半开恢复
- 多 Key 轮换、渠道标签分类、批量测试、模型列表自动拉取
- 渠道账号自动签到与余额监控（对接上游网关账号 API）
- 渠道亲和性路由（会话粘滞，可配置规则与缓存）
- 支持为渠道单独配置 HTTP/SOCKS5 代理与自定义 Headers

**用户与认证**
- 邮箱注册、GitHub OAuth、LinuxDO OAuth（自动同步信任等级）
- Discord OAuth、OIDC 统一认证（企业 SSO）、Telegram Login、微信 OAuth
- Passkey / WebAuthn 无密码登录
- 双因素认证 (2FA/TOTP)，支持 Google Authenticator 等验证器应用
- 邮箱密码重置，邮箱绑定变更
- 人机验证：Cloudflare Turnstile 与极验 (GeeTest4) 双引擎

**内容安全**
- 腾讯云天御（TMS）文本内容安全审核，真实对接官方 Go SDK
- 关键词本地过滤（黑名单逗号分隔）
- 多维白名单（用户 ID / 模型名 / IP）
- 上下文净化模块（ContextSanitizer）：四级策略作用域、五级防护模式、Prompt Injection 检测、工具调用结构防护

**计费与支付**
- 精细化额度管理（令牌级），自定义模型倍率与补全倍率
- 缓存计费 — 支持 OpenAI cached_tokens 和 Claude cache_read 折扣计费
- Reasoning Effort 后缀、Thinking-to-Content 转换
- 订阅套餐管理（分组 / 额度 / RPM 订阅计划）
- **易支付**（epay 协议，支持支付宝/微信）
- **Stripe** — 官方 Go SDK，Checkout Session，支持延迟支付方式（SEPA、银行转账）
- **Creem** — 国际收款，支持多产品目录，HMAC-SHA256 Webhook 验签
- 订单并发安全保护（订单级内存锁 + 幂等检查）
- 管理员手动补单（卡在 pending 状态的订单一键入账）
- 充值历史分页查询（用户 & 管理员双视角）

**安全审计与日志**
- 请求日志记录 IP 与 Request ID（全局链路追踪）
- 管理员操作审计日志（所有非 GET 写操作自动记录，仅超管可查）
- 可配置日志留存天数（不低于 180 天，满足《网络安全法》要求），定时自动清理
- JWT 密钥首次启动自动生成，支持环境变量覆盖

**运营能力**
- 每日签到（可配置随机奖励，可选人机验证）
- 兑换码、邀请码系统
- 请求日志 + CSV 导出
- 公开定价页面
- 数据看板：DAU、错误率趋势、系统性能监控
- 通知系统：邮件 / Webhook / Bark / Gotify，余额不足、渠道异常、签到结果触发

## 🛠️ 技术栈

| 层级 | 技术 |
|:---|:---|
| 后端 | Go 1.26+, Gin, GORM |
| 数据库 | SQLite (默认) / MySQL / PostgreSQL |
| 缓存 | Redis (可选，分布式限流) |
| 前端 | React 19, TypeScript, Vite 7, TailwindCSS 4, 自建组件库（HeroUI 风格 API） |
| 状态管理 | Zustand + TanStack React Query |
| 认证 | JWT, OAuth (GitHub/LinuxDO/Discord/OIDC/Telegram/微信), Passkey/WebAuthn |
| 支付 | 易支付(epay), Stripe v86, Creem |
| 内容安全 | 腾讯云天御 TMS/IMS (tencentcloud-sdk-go) |

## 🏗️ 系统架构

```
┌─────────────┐     ┌──────────────────────────────────────────────┐
│   Client    │     │                  Akasha                      │
│  (OpenAI    │────▶│  Rate Limiter ──▶ Content Moderation         │
│  Compatible)│     │                       │                      │
└─────────────┘     │               Context Sanitizer              │
                    │                       │                      │
                    │               Relay Controller               │
                    │                       │                      │
                    │               Adapter Factory                │
                    │            ┌─────┬────┴───┬─────┐           │
                    │            ▼     ▼        ▼     ▼           │
                    │         OpenAI Claude  Gemini  ...           │
                    └───────────────────────────────────────────── ┘
                                  │      │       │
                                  ▼      ▼       ▼
                              Upstream Providers
```

## 🚀 快速开始

### 前置要求

- Go 1.26+
- Node.js 18+ / pnpm
- MySQL 5.7+ 或 PostgreSQL 9.6+（可选，默认 SQLite）

### 本地开发

**1. 启动后端**

```bash
cd backend
go mod tidy
go run main.go --port 8080 --driver sqlite --dsn "akasha.db"
```

可用启动参数：

| 参数 | 默认值 | 说明 |
|:---|:---|:---|
| `--port` | `8080` | 服务端口 |
| `--driver` | `sqlite` | 数据库驱动 (`sqlite` / `mysql` / `postgres`) |
| `--dsn` | `akasha.db` | 数据库连接字符串 |
| `--rpm` | `60` | 全局速率限制（请求/分钟） |

JWT 密钥首次启动时自动生成，保存至 `backend/jwt_secret.key`。多实例部署时请通过环境变量 `AKASHA_JWT_SECRET` 统一指定密钥。

**2. 启动前端**

```bash
cd frontend
pnpm install
pnpm run dev
```

访问 `http://localhost:5173`，开发服务器会自动将 `/api` 请求代理到后端 `localhost:8080`。

**3. 首次使用**

首个注册用户自动成为超级管理员（RoleRoot）。

## 👤 角色权限

| 角色 | 说明 | 典型权限 |
|:---|:---|:---|
| 普通用户 (1) | 终端使用者 | 管理自己的 Token、充值、查看用量 |
| 管理员 (10) | 运营人员 | 渠道管理、用户管理、日志查看、订单只读查看 |
| 超级管理员 (100) | 系统所有者 | 系统设置、补单操作、操作审计、支付配置 |

## ⚙️ 配置说明

登录管理后台，在「系统设置」页面完成配置：

**基础**

| 配置项 | 说明 |
|:---|:---|
| 系统名称 / URL | 基础信息 |
| 邮件服务 (SMTP) | 注册验证与通知 |
| Redis | 分布式限流与缓存（可选） |

**OAuth 认证**

| 配置项 | 说明 |
|:---|:---|
| GitHub / LinuxDO / Discord / OIDC / Telegram / 微信 | 填入对应 Client ID 和 Secret |
| Turnstile / 极验 | 人机验证提供商配置 |

**支付**

设置 `payment_provider` 为 `epay`、`stripe` 或 `creem`，然后填写对应渠道的密钥信息：

| 渠道 | 必填项 |
|:---|:---|
| 易支付 | API 地址、PID、KEY |
| Stripe | Secret Key、Webhook Secret（在 Stripe Dashboard 注册 Webhook 端点：`/api/stripe/webhook`） |
| Creem | API Key、Webhook Secret、产品目录 JSON（`/api/creem/webhook`） |

**内容安全**

| 配置项 | 说明 |
|:---|:---|
| 腾讯云天御 | SecretId、SecretKey、地域（默认 ap-guangzhou）、审核策略 BizType |
| 敏感词 | 逗号分隔，本地即时过滤 |
| 白名单 | 用户 ID / 模型名 / IP 白名单（逗号分隔） |

**日志**

| 配置项 | 说明 |
|:---|:---|
| 日志留存天数 | 不低于 180 天（《网络安全法》要求），定时任务自动清理过期记录 |

## 📡 API 接口

所有接口兼容 OpenAI 格式，可直接对接现有 OpenAI SDK 和工具。

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

| 端点 | 说明 |
|:---|:---|
| `POST /v1/chat/completions` | 对话补全（支持流式） |
| `POST /v1/messages` | Anthropic Messages API（Claude Code CLI） |
| `POST /v1/responses` | OpenAI Responses API（Codex CLI） |
| `POST /v1/embeddings` | 文本向量化 |
| `POST /v1/images/generations` | 图片生成 |
| `POST /v1/audio/speech` | 文本转语音 |
| `POST /v1/audio/transcriptions` | 语音转文本 |
| `GET /v1/realtime` | Realtime API（WebSocket） |
| `POST /gemini/v1beta/models/:model` | Gemini 原生格式代理 |
| `GET /v1/models` | 可用模型列表 |

### CLI 工具配置

**Claude Code CLI**
```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_API_KEY=sk-your-token
```

**OpenAI Codex CLI / 通用 SDK**
```bash
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-token
```

## 📂 项目结构

```
├── backend/
│   ├── main.go
│   ├── adapter/             # LLM 供应商适配器 + 渠道账号适配器
│   ├── controller/          # HTTP 请求处理器
│   │   ├── payment.go       # 支付核心：创建订单、epay 回调、管理员补单
│   │   ├── user_extra.go    # 用户扩展：Stripe/Creem Webhook、充值信息
│   │   └── system_utils.go  # 系统工具：性能监控、缓存统计
│   ├── model/               # GORM 数据模型（含 PaymentOrder、AdminAuditLog）
│   ├── router/              # 路由定义
│   ├── service/
│   │   ├── payment/         # 支付服务：订单锁、统一入账、Stripe/Creem 客户端
│   │   ├── moderation/      # 腾讯云天御内容安全 SDK 封装
│   │   └── context_sanitizer/ # 上下文净化模块
│   ├── middleware/          # 中间件（限流、审计、Request ID）
│   └── common/              # 公共工具（配置、JWT、Redis）
└── frontend/
    └── src/
        ├── pages/
        │   ├── admin/       # 管理后台（渠道、用户、支付管理、审计日志、设置等）
        │   └── user/        # 用户面板（令牌、充值、充值记录、订阅等）
        └── layouts/         # AdminLayout / UserLayout
```

## 📜 许可协议

本项目为闭源专有软件（Proprietary Software），版权归 Akasha Team 所有，保留所有权利。

- 未经书面授权，禁止复制、修改、反编译、转售或对外提供本软件的全部或部分
- 如需商业授权、私有部署或定制开发，请联系版权所有者

详见 [LICENSE](LICENSE) 文件。

---

<div align="center">

Powered by Akasha Team

</div>
