
<div align="center">

# Akasha (阿卡夏)

**汇聚全宇宙智慧的 AI 网关系统**

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-7-646CFF?logo=vite&logoColor=white)](https://vitejs.dev/)
[![TailwindCSS](https://img.shields.io/badge/Tailwind-4-06B6D4?logo=tailwindcss&logoColor=white)](https://tailwindcss.com/)

</div>

---

Akasha 是一个统一的 AI API 网关，将多家大模型供应商聚合在 OpenAI 兼容接口之后，提供渠道负载均衡、用户管理、额度计费、速率限制和运营管理等能力。

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
- 任意 OpenAI SDK 客户端 — 标准 Chat Completions API (`/v1/chat/completions`)

**渠道管理**
- 多渠道负载均衡（优先级 + 权重）、自动重试
- 自动熔断与半开恢复机制，保障服务高可用
- 多 Key 轮换（每行一个 Key，随机选取）
- 渠道标签分类与筛选
- 批量渠道测试、模型列表自动拉取
- 支持为渠道单独配置 HTTP/SOCKS5 代理与自定义 Headers

**用户与安全**
- 邮箱注册、GitHub OAuth、LinuxDO OAuth（自动同步信任等级）
- Discord OAuth、OIDC 统一认证（支持企业 SSO）、Telegram Login
- 双因素认证 (2FA/TOTP)，支持 Google Authenticator 等验证器应用
- 邮箱密码重置（验证码方式）
- 人机验证：Cloudflare Turnstile 与极验 (GeeTest4) 双引擎，后台一键切换
- 四级速率限制（IP / Token / 用户 / 模型），支持 Redis 分布式限流
- 关键操作限流（登录、注册、密码重置）

**运营能力**
- 精细化额度管理（令牌级），自定义模型倍率
- 缓存计费 — 支持 OpenAI cached_tokens 和 Claude cache_read 折扣计费（默认 0.5 倍率）
- Reasoning Effort 后缀 — 模型名追加 `-high/-medium/-low` 自动设置推理强度，`-thinking` 启用思考模式
- Thinking-to-Content — 将 Claude 思考过程转为 `<thinking>` 包裹的普通内容（后台开关）
- 模型配置管理（分类、倍率、上下文长度）与定价同步
- 用户分组系统（分组级倍率覆盖、渠道权限、QPM 限制）
- 每日签到系统（可配置随机奖励范围，可选人机验证）
- 兑换码、邀请码系统
- 易支付对接，请求日志与 CSV 导出，日志定期清理
- 公开定价页面
- 数据看板：DAU、错误率趋势、多维筛选、系统性能监控
- 内容审核（关键词过滤 + 外部审核 API + 白名单）

**现代化前端**
- React 19 + HeroUI + TailwindCSS 响应式界面
- Playground — 在线 API 测试页面，支持模型选择、流式输出、参数调节
- 系统性能监控 — Goroutines、内存、GC、运行时间实时查看
- 每日签到 — 用户仪表盘一键签到领取随机额度奖励

## 🛠️ 技术栈

| 层级 | 技术 |
|:---|:---|
| 后端 | Go 1.24, Gin, GORM |
| 数据库 | SQLite (默认) / MySQL / PostgreSQL |
| 缓存 | Redis (可选，用于分布式限流与缓存) |
| 前端 | React 19, TypeScript, Vite 7, TailwindCSS 4, HeroUI |
| 状态管理 | Zustand + TanStack React Query |
| 认证 | JWT, OAuth (GitHub, LinuxDO, Discord, OIDC, Telegram) |

## 🏗️ 系统架构

```
┌─────────────┐     ┌──────────────────────────────────────────────┐
│   Client    │     │                  Akasha                      │
│  (OpenAI    │────▶│  Rate Limiter ──▶ Relay Controller           │
│  Compatible)│     │                       │                      │
└─────────────┘     │                 Adapter Factory               │
                    │                 ┌─────┼─────┐                │
                    │                 ▼     ▼     ▼                │
                    │              OpenAI Claude Gemini ...         │
                    │                 │     │     │                │
                    └─────────────────┼─────┼─────┼────────────────┘
                                      ▼     ▼     ▼
                                  Upstream Providers
```

## 🚀 快速开始

### 前置要求

- Go 1.24+
- Node.js 18+ / pnpm
- MySQL 5.7+ 或 PostgreSQL 9.6+ (可选，默认 SQLite)

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
| `--rpm` | `60` | 全局速率限制 (请求/分钟) |

**2. 启动前端**

```bash
cd frontend
pnpm install
pnpm run dev
```

访问 `http://localhost:5173`，开发服务器会自动将 `/api` 请求代理到后端 `localhost:8080`。

**3. 首次使用**

首个注册用户自动成为 Root 管理员。

## ⚙️ 配置说明

登录管理后台，在「系统设置」页面完成以下配置：

| 配置项 | 说明 |
|:---|:---|
| GitHub OAuth | 填入 Client ID 和 Secret |
| LinuxDO OAuth | 填入 Client ID 和 Secret，设置各信任等级初始额度 |
| Discord OAuth | 填入 Client ID 和 Secret |
| OIDC (SSO) | 填入 Client ID、Secret 和 Issuer URL，支持企业统一认证 |
| Telegram Login | 填入 Bot Token，启用 Telegram 登录小组件 |
| 邮件服务 (SMTP) | 配置 SMTP 服务器以启用注册验证与通知 |
| Cloudflare Turnstile | 填入 Site Key 和 Secret Key 以启用人机验证 |
| 极验 (GeeTest4) | 填入 Captcha ID 和 Key，设置验证码提供商为 `geetest` |
| Redis | 配置连接地址以启用分布式限流与缓存 (可选) |
| 易支付 | 配置商户信息以启用在线充值 (可选) |

### 添加渠道

在「渠道管理」中添加 AI 供应商渠道：

| 渠道类型 | 类型 ID | 说明 |
|:---|:---:|:---|
| OpenAI | 1 | 官方 API 或兼容接口 |
| Azure OpenAI | 3 | Azure 部署的 OpenAI 模型 |
| Custom | 8 | 任意 OpenAI 兼容接口 |
| Anthropic Claude | 14 | Claude 系列模型 |
| Google Gemini | 18 | Gemini 系列模型 |
| Midjourney | 30 | 需填写 mj-proxy 地址 |
| 阿里通义千问 | 40 | Qwen 系列模型 |
| 腾讯混元 | 41 | Hunyuan 系列模型 |
| 百度文心一言 | 42 | ERNIE 系列模型 |
| 讯飞星火 | 43 | Spark 系列模型 |
| Deepseek | 44 | Deepseek 系列模型 |
| 智谱 ChatGLM | 45 | GLM 系列模型 (JWT 认证) |
| Moonshot (Kimi) | 46 | Moonshot 系列模型 |
| Ollama | 47 | 本地部署模型 (Key 可为空) |
| Suno | 50 | AI 音乐生成 (需填写 suno-api 地址) |
| Dify | 51 | Dify ChatFlow 工作流代理 |

## 📡 API 接口

所有接口兼容 OpenAI 格式，可直接对接现有 OpenAI SDK 和工具。

```bash
# Chat Completions
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}'
```

支持的端点：

| 端点 | 说明 |
|:---|:---|
| `POST /v1/chat/completions` | 对话补全 (支持流式) |
| `POST /v1/completions` | 旧版文本补全 |
| `POST /v1/messages` | Anthropic Messages API (Claude Code CLI 兼容) |
| `POST /v1/responses` | OpenAI Responses API (Codex CLI 兼容) |
| `POST /v1/embeddings` | 文本向量化 |
| `POST /v1/images/generations` | 图片生成 (DALL-E 等) |
| `POST /v1/audio/speech` | 文本转语音 (TTS) |
| `POST /v1/audio/transcriptions` | 语音转文本 (STT) |
| `POST /v1/moderations` | 内容审核 |
| `POST /v1/rerank` | 文档重排序 |
| `GET /v1/models` | 可用模型列表 |
| `GET /v1/models/:model` | 模型详情 |
| `GET/POST/DELETE /v1/files` | 文件管理 |
| `GET /v1/realtime` | OpenAI Realtime API (WebSocket 双向代理) |
| `POST /gemini/v1beta/models/:model` | Gemini 原生 REST 格式代理 |
| `POST /suno/submit/*` | Suno AI 音乐生成 |
| `GET /api/key/info` | API Key 额度查询 (neko-api-key-tool 兼容) |

### CLI 工具配置

**Claude Code CLI**

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_API_KEY=sk-your-token
```

**OpenAI Codex CLI**

```bash
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-token
```

**通用 OpenAI SDK**

```bash
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-token
```

## 📂 项目结构

```
├── backend/                 # Go 后端服务
│   ├── main.go              # 入口，CLI 参数解析
│   ├── adapter/             # LLM 供应商适配器 (Adaptor 接口)
│   │   ├── openai/          # OpenAI / Azure / Custom
│   │   ├── claude/          # Anthropic Claude
│   │   ├── gemini/          # Google Gemini
│   │   ├── deepseek/        # Deepseek
│   │   ├── zhipu/           # 智谱 ChatGLM
│   │   ├── moonshot/        # Moonshot (Kimi)
│   │   ├── ollama/          # Ollama (本地部署)
│   │   ├── ali/             # 阿里通义千问
│   │   ├── baidu/           # 百度文心一言
│   │   ├── tencent/         # 腾讯混元
│   │   └── xunfei/          # 讯飞星火
│   ├── controller/          # HTTP 请求处理器
│   ├── model/               # GORM 数据模型
│   ├── router/              # 路由定义
│   ├── service/             # 后台服务 (健康检查、日志队列、渠道测试)
│   ├── middleware/           # 中间件 (多级速率限制)
│   ├── migrations/          # SQL 迁移脚本
│   ├── dto/                 # 数据传输对象
│   └── common/              # 公共工具 (配置、常量、Redis)
└── frontend/                # React 前端
    └── src/
        ├── pages/           # 页面组件
        │   ├── admin/       # 管理后台 (渠道、用户、分组、模型、设置等)
        │   └── user/        # 用户面板 (令牌、充值、个人资料等)
        ├── layouts/         # 布局组件 (AdminLayout, UserLayout)
        └── store/           # Zustand 状态管理
```

## 📜 开源协议

本项目采用 [AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0) 协议开源。

- 修改代码并对外提供网络服务时，必须公开修改后的源码
- 必须保留原作者的版权声明

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

<div align="center">

Powered by Akasha Team

</div>
