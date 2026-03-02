# 更新日志

所有重要变更会记录在此文件。

## [1.0.0] - 2026-03-02

### 新增
- 补齐视频兼容控制器与路由：
  - `POST /v1/video/generations`
  - `GET /v1/video/generations/:task_id`
  - `POST /v1/videos`
  - `GET /v1/videos/:task_id`
  - `GET /v1/videos/:task_id/content`
  - `POST /v1/videos/:video_id/remix`
- 新增视频兼容处理器：任务提交、任务查询、结果代理（含 `data:` URL 解码）。

### 对齐与兼容
- Token 管理补齐：`/api/token/search`、`/api/token/:id`、`/api/token/batch`。
- Redemption 补齐：`/api/redemption/search`、`/api/redemption/:id`、`DELETE /api/redemption/invalid`、`DELETE /api/redemption/:id` 等。
- Log 补齐：`/api/log/search`、`/api/log/self/search`，并兼容 `target_timestamp/start_timestamp/end_timestamp` 参数。
- 订阅后台管理对齐：`/api/subscription/admin/*`（plans/bind/users）。
- 运营数据工具对齐：`/api/ratio_sync/*`、`/api/data`、`/api/prefill_group/*`。

### 前端映射回补
- 管理端订阅页面改为使用 `/api/subscription/admin/plans` 及其 `PATCH/PUT/POST` 变体。
- 管理端兑换码列表改为 `/api/redemption/search`。
- 日志页面列表改为 `/api/log/search` 和 `/api/log/self/search`。
- 日志清理改为 query 方式：`DELETE /api/log?target_timestamp=...`。

### 复查结论（阶段A：路由差异清单精确化）
- **核心 P0/P1 路由已完成对齐并通过构建验证**。
- 仍存在少量非阻断差异（主要是 new-api 的扩展生态路由）：
  - `new-api/router/video-router.go` 中 `kling` 与 `jimeng` 专用转换链路。
  - `new-api/router/api-router.go` 中 `uptime/status`、部分第三方支付/绑定流程、部分历史管理接口。
- 以上差异当前不影响本仓库既有前后端联调主链路，后续可按业务优先级逐步纳入。

### 构建与验收
- Backend: `go build ./...` 通过。
- Frontend: `npm run build` 通过。

---

> 当前版本：`1.0.0`
