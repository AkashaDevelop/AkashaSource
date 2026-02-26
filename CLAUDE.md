# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Akasha (阿卡夏) — an AI gateway system that aggregates multiple LLM providers behind a unified OpenAI-compatible API. Built with a Go backend and React frontend, it includes user management, quota billing, rate limiting, channel load balancing, and an admin dashboard.

## Common Commands

### Backend
```bash
cd backend
go mod tidy                                              # Install/sync dependencies
go run main.go --port 8080 --driver sqlite --dsn akasha.db  # Run with SQLite (default)
go run main.go --driver mysql --dsn "user:pass@tcp(host)/db" # Run with MySQL
go run main.go --driver postgres --dsn "postgres://..."      # Run with PostgreSQL
go build -o akasha .                                     # Build binary
```

### Frontend
```bash
cd frontend
pnpm install          # Install dependencies (pnpm is the lockfile manager)
pnpm run dev          # Dev server at localhost:5173, proxies /api to localhost:8080
pnpm run build        # Type-check then production build (tsc -b && vite build)
pnpm run lint         # ESLint
```

## Architecture

### Backend (Go, Gin, GORM)

Go module path: `STfreApi`

```
backend/
├── main.go              # Entry point, CLI flags: --port, --driver, --dsn, --rpm
├── router/api.go        # All route definitions (public, auth, admin, OpenAI-compat)
├── adapter/             # LLM provider adapters (adapter pattern)
│   ├── interface.go     # Adaptor interface: ConvertRequest, DoRequest, DoResponse
│   ├── factory.go       # Maps channel type → adapter constructor
│   ├── openai/          # OpenAI/Azure
│   ├── claude/          # Anthropic Claude
│   ├── gemini/          # Google Gemini
│   ├── ali/             # Alibaba Qwen
│   ├── baidu/           # Baidu Wenxin
│   ├── tencent/         # Tencent Hunyuan
│   └── xunfei/          # Xunfei Spark
├── controller/          # HTTP handlers (relay.go is the main LLM proxy handler)
├── model/               # GORM models + DB init (channel, token, user, log, option, etc.)
├── service/             # Background services (channel health check, async log queue)
├── middleware/           # Rate limiting (IP/token/user tiers, Redis-backed)
├── dto/                 # OpenAI-compatible request/response structs
└── common/              # Config, constants, HTTP client, Redis init
```

**Request flow:** Client → `router/api.go` → rate limit middleware → `controller/relay.go` → `adapter/factory.go` selects adapter → adapter converts to provider format → upstream call → response streamed back.

**Multi-format inbound support:**
- `/v1/chat/completions` — standard OpenAI format → relay via adapters
- `/v1/messages` — Anthropic Messages API (Claude Code CLI) → `controller/messages.go` → passthrough to Claude channels or convert to OpenAI for others
- `/v1/responses` — OpenAI Responses API (Codex CLI) → `controller/responses.go` → convert to chat format, relay, convert response back

**Adaptor interface** (each provider implements this):
- `ConvertRequest` — transform OpenAI-format request to provider-native format
- `DoRequest` — execute HTTP call to upstream provider
- `DoResponse` — parse response, extract usage, stream back to client

**Channel types** (integer IDs in `model/channel.go`): 1=OpenAI, 3=Azure, 8=Custom, 14=Claude, 18=Gemini, 30=Midjourney, 40=Qwen, 41=Hunyuan, 42=Wenxin, 43=Spark, 44=Deepseek, 45=Zhipu, 46=Moonshot, 47=Ollama.

### Frontend (React 19, TypeScript, Vite, HeroUI)

```
frontend/src/
├── App.tsx              # Route definitions (public, user, admin)
├── store/auth.ts        # Zustand auth store (user state, JWT token)
├── layouts/             # AdminLayout.tsx, UserLayout.tsx (sidebar nav)
└── pages/               # Page components
    ├── admin/           # Channel, User, Setting, Redemption, Migration management
    └── user/            # Token, Profile, Topup, Invitation
```

State: Zustand. Data fetching: TanStack React Query. UI: HeroUI (NextUI) + TailwindCSS. Charts: Recharts.

Admin routes require role >= 10. Vite dev server proxies `/api` → `http://localhost:8080`.

## Key Conventions

### Database Compatibility
All database code must work with SQLite, MySQL, and PostgreSQL simultaneously. Prefer GORM methods over raw SQL. When raw SQL is unavoidable, handle quoting and boolean differences per driver (use `common.UsingPostgreSQL`, `common.UsingSQLite`, `common.UsingMySQL` flags).

### Adding a New LLM Provider
1. Create `backend/adapter/<provider>/` implementing the `Adaptor` interface
2. Add a channel type constant in `model/channel.go`
3. Register the adapter in `adapter/factory.go`
4. Add frontend UI support in `pages/admin/Channel.tsx`

### Rate Limiting
Three tiers: IP-based (base RPM), token-based (3x), user-based (5x). Supports Redis for distributed deployments; falls back to in-memory.

### Authentication
JWT-based. OAuth providers: GitHub, LinuxDO. First registered user becomes root admin.
