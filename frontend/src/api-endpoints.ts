/**
 * 前端页面对应的后端 API 端点清单
 *
 * 本文件列出了新增前端功能所需的后端 API 端点
 * 后端开发者需要实现这些端点以支持前端功能
 */

// ==================== 系统监控 API ====================
/**
 * GET /api/admin/system/monitor
 * 获取系统监控数据
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "data": {
 *     "cpu_usage": 45.2,        // CPU 使用率 (0-100)
 *     "memory_usage": 68.5,     // 内存使用率 (0-100)
 *     "memory_total": 8589934592, // 总内存（字节）
 *     "memory_used": 5879916544,  // 已用内存（字节）
 *     "goroutines": 156,        // Goroutine 数量
 *     "timestamp": 1678123456   // 时间戳
 *   }
 * }
 *
 * 实现位置：backend/controller/system_monitor.go
 * 调用服务：backend/service/system_monitor.go
 */

// ==================== 渠道亲和性 API ====================
/**
 * GET /api/admin/channel-affinity/rules
 * 获取渠道亲和性规则配置
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "data": {
 *     "rules": [
 *       {
 *         "name": "rule_gpt4",
 *         "model_regex": "gpt-4.*",
 *         "path_regex": "",
 *         "user_agent_include": [],
 *         "key_source_type": "context_int",
 *         "key_source_key": "user_id",
 *         "key_source_path": "",
 *         "value_regex": "",
 *         "ttl_seconds": 3600,
 *         "include_rule_name": true,
 *         "include_using_group": true,
 *         "skip_retry_on_fail": false,
 *         "param_override": {}
 *       }
 *     ]
 *   }
 * }
 */

/**
 * POST /api/admin/channel-affinity/rules
 * 保存渠道亲和性规则配置
 *
 * 请求体：
 * {
 *   "rules": [...]  // 规则数组
 * }
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "保存成功"
 * }
 */

/**
 * GET /api/admin/channel-affinity/stats
 * 获取渠道亲和性缓存统计
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "data": {
 *     "enabled": true,
 *     "total": 1234,
 *     "unknown": 0,
 *     "by_rule_name": {
 *       "rule_gpt4": 856,
 *       "rule_claude": 378
 *     },
 *     "cache_capacity": 100000,
 *     "cache_algo": "LRU"
 *   }
 * }
 *
 * 实现位置：backend/controller/channel_affinity.go
 * 调用服务：backend/service/channel_affinity.go -> GetChannelAffinityCacheStats()
 */

/**
 * POST /api/admin/channel-affinity/cache/clear
 * 清空所有渠道亲和性缓存
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "data": {
 *     "count": 1234  // 清空的缓存数量
 *   }
 * }
 *
 * 实现位置：backend/controller/channel_affinity.go
 * 调用服务：backend/service/channel_affinity.go -> ClearAffinityCacheAll()
 */

// ==================== 用户通知设置 API ====================
/**
 * GET /api/user/notification-settings
 * 获取当前用户的通知设置
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "data": {
 *     "notify_type": "email",
 *     "notification_email": "user@example.com",
 *     "webhook_url": "",
 *     "webhook_secret": "",
 *     "bark_url": "",
 *     "gotify_url": "",
 *     "gotify_token": "",
 *     "gotify_priority": 5,
 *     "upstream_model_update_notify_enabled": false
 *   }
 * }
 *
 * 实现位置：backend/controller/user.go
 * 从 user.setting 字段解析 JSON
 */

/**
 * POST /api/user/notification-settings
 * 保存用户通知设置
 *
 * 请求体：
 * {
 *   "notify_type": "email",
 *   "notification_email": "user@example.com",
 *   ...
 * }
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "保存成功"
 * }
 *
 * 实现位置：backend/controller/user.go
 * 更新 user.setting 字段
 */

/**
 * POST /api/user/notification-test
 * 发送测试通知
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "测试通知已发送"
 * }
 *
 * 实现位置：backend/controller/user.go
 * 调用服务：backend/service/user_notify.go -> NotifyUser()
 */

// ==================== 批量渠道操作 API ====================
/**
 * POST /api/admin/channels/batch-status
 * 批量更新渠道状态
 *
 * 请求体：
 * {
 *   "channel_ids": [1, 2, 3],
 *   "status": 1  // 1=启用, 2=禁用
 * }
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "操作成功"
 * }
 *
 * 实现位置：backend/controller/channel.go
 * 调用服务：backend/service/batch_update.go -> BatchUpdateChannelStatus()
 */

/**
 * POST /api/admin/channels/batch-priority
 * 批量更新渠道优先级
 *
 * 请求体：
 * {
 *   "channel_ids": [1, 2, 3],
 *   "priority": 20
 * }
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "操作成功"
 * }
 *
 * 实现位置：backend/controller/channel.go
 * 调用服务：backend/service/batch_update.go -> BatchUpdateChannelPriority()
 */

/**
 * POST /api/admin/channels/batch-delete
 * 批量删除渠道
 *
 * 请求体：
 * {
 *   "channel_ids": [1, 2, 3]
 * }
 *
 * 响应格式：
 * {
 *   "code": 0,
 *   "message": "操作成功"
 * }
 *
 * 实现位置：backend/controller/channel.go
 * 调用服务：backend/service/batch_update.go -> BatchDeleteChannels()
 */

// ==================== 实现优先级 ====================
/**
 * 高优先级（核心功能）：
 * 1. 系统监控 API - 简单，直接调用 service/system_monitor.go
 * 2. 批量渠道操作 API - 简单，直接调用 service/batch_update.go
 * 3. 用户通知设置 API - 中等，需要处理 user.setting JSON 字段
 *
 * 中优先级（复杂功能）：
 * 4. 渠道亲和性 API - 复杂，需要配置存储和缓存管理
 *    - 可以先存储在配置文件或数据库中
 *    - 调用 service/channel_affinity.go 的函数
 */

export {};
