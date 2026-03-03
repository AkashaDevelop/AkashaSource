# 功能缺口记录（new-api 对比 backend）

更新时间：2026-03-03

## 一、backend 尚未覆盖的功能入口

### 1) OAuth 绑定公开入口
- new-api：`GET /api/oauth/email/bind`
- backend：当前无对应路由
- 参考：`new-api/router/api-router.go`

### 2) 支付回调（Stripe / Creem）
- new-api：
  - `POST /api/stripe/webhook`
  - `POST /api/creem/webhook`
- backend：当前无对应路由
- 参考：`new-api/router/api-router.go`

### 3) 订阅 Epay 回调与返回
- new-api：
  - `POST /api/subscription/epay/notify`
  - `GET /api/subscription/epay/notify`
  - `GET /api/subscription/epay/return`
  - `POST /api/subscription/epay/return`
- backend：当前无对应路由
- 参考：`new-api/router/api-router.go`

### 4) Telegram OAuth 登录/绑定入口
- new-api：
  - `GET /api/oauth/telegram/login`
  - `GET /api/oauth/telegram/bind`
- backend：当前未提供这两个 `/api` 入口
- 参考：`new-api/router/api-router.go`

### 5) 用户分组查询接口
- new-api：
  - `GET /api/user/groups`
  - `GET /api/user/self/groups`
- backend：当前无对应路由
- 参考：`new-api/router/api-router.go`

### 6) 日志侧渠道亲和缓存统计
- new-api：`GET /api/log/channel_affinity_usage_cache`
- backend：当前无同名日志接口（仅有 option 侧缓存接口）
- 参考：`new-api/router/api-router.go`

---

## 二、已接入但行为仍有差异

### 1) 重置密码语义差异
- new-api：验证 token 后“系统生成新密码并返回”。
- backend：要求提交新密码后直接修改。
- 影响：前端/调用方若依赖“返回随机新密码”的流程，将产生行为不一致。

### 2) ratio_config 暴露策略差异
- new-api：受开关控制，可能返回禁用状态。
- backend：当前默认返回 `model_ratio/completion_ratio`。
- 影响：后台安全策略与暴露范围不完全一致。

### 3) performance 统计深度差异
- new-api：包含磁盘缓存统计、磁盘空间与监控配置。
- backend：当前兼容实现为简化/占位结构。
- 影响：管理端若依赖完整统计字段，展示会有差异。

---

## 三、建议优先级

- P0（优先）
  1. `/api/oauth/email/bind`
  2. `/api/stripe/webhook`、`/api/creem/webhook`
  3. `/api/subscription/epay/notify`、`/api/subscription/epay/return`

- P1（次优先）
  1. `/api/oauth/telegram/login`、`/api/oauth/telegram/bind`
  2. `/api/user/groups`、`/api/user/self/groups`
  3. `/api/log/channel_affinity_usage_cache`

- P2（一致性增强）
  1. 重置密码语义与 new-api 对齐
  2. ratio_config 增加暴露开关行为
  3. performance 统计字段补齐
