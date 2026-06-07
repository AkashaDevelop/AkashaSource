# 阿卡夏自定义渠道功能

## 功能概述

阿卡夏现已支持**纯可视化配置的自定义渠道**功能，允许管理员无需编写代码，通过 Web 界面配置接入任何兼容 OpenAI 格式或自定义协议的 AI API 提供商。

## 核心特性

✅ **零代码配置** - 所有配置通过可视化表单完成，无需编写 JSON  
✅ **字段映射** - 灵活映射 OpenAI 标准字段到目标 API 字段  
✅ **路径提取** - 支持点路径语法提取嵌套 JSON 字段  
✅ **流式响应** - 支持 SSE (Server-Sent Events) 流式输出  
✅ **自定义认证** - 支持 Bearer、API Key 等多种认证方式  
✅ **请求头扩展** - 支持添加自定义 HTTP Headers  
✅ **权限隔离** - 支持配置私有/公开共享  

---

## 使用流程

### 1. 创建自定义配置模板

管理员进入 **渠道管理 → 自定义配置** 页面，点击"新建配置"：

#### 基础信息
- **配置名称**：如 "DeepSeek 兼容"
- **描述**：配置说明
- **适配器类型**：`openai_compatible`（默认）

#### 请求配置
| 字段 | 说明 | 示例 |
|------|------|------|
| 请求方法 | HTTP 方法 | `POST` |
| 请求端点 | API 路径 | `/v1/chat/completions` |
| Content-Type | 请求内容类型 | `application/json` |
| 认证类型 | 认证方式 | `bearer` |
| 认证头名称 | Header 名称 | `Authorization` |
| 认证头模板 | 认证格式（支持 `{key}` 占位符） | `Bearer {key}` |

#### 字段映射（OpenAI → 目标 API）
| OpenAI 字段 | 目标字段 | 说明 |
|-------------|----------|------|
| model | `model` | 模型名称字段 |
| messages | `messages` | 消息数组字段 |
| temperature | `temperature` | 温度参数 |
| max_tokens | `max_tokens` | 最大 Token 数 |
| stream | `stream` | 是否流式输出 |
| top_p | `top_p` | Top-P 采样 |
| stop | `stop` | 停止词 |

**注意**：如果目标 API 不支持某个字段，填写 `-` 或留空。

#### 响应路径配置（使用点路径语法）
| 配置项 | 路径示例 | 说明 |
|--------|----------|------|
| 内容路径 | `choices.0.message.content` | 提取返回内容 |
| Usage 路径 | `usage` | Token 使用统计根路径 |
| Prompt Tokens | `usage.prompt_tokens` | 输入 Token 数 |
| Completion Tokens | `usage.completion_tokens` | 输出 Token 数 |
| Total Tokens | `usage.total_tokens` | 总 Token 数 |
| 错误路径 | `error.message` | 错误信息路径 |

#### 流式响应配置
- **流式支持**：是/否
- **数据前缀**：`data: `（SSE 标准格式）
- **结束标记**：`[DONE]`
- **流内容路径**：`choices.0.delta.content`

#### 其他配置
- **超时时间**：120 秒（默认）
- **重试次数**：3 次
- **重试间隔**：1 秒
- **公开配置**：是否允许其他管理员使用

---

### 2. 创建自定义渠道

配置模板创建后，在 **渠道管理** 中创建渠道：

1. 选择 **渠道类型**：`🔧 自定义渠道 (100)`
2. 关联 **自定义配置**：选择刚创建的配置模板
3. 填写 **Base URL**：如 `https://api.deepseek.com`
4. 填写 **API Key**：渠道密钥
5. 配置 **支持的模型**：逗号分隔，如 `deepseek-chat,deepseek-coder`
6. 设置 **优先级、权重** 等常规参数

---

## 配置示例

### 示例 1：OpenAI 兼容 API

```
配置名称: OpenAI Compatible
请求端点: /v1/chat/completions
认证模板: Bearer {key}

字段映射: (保持默认即可)
  model → model
  messages → messages
  temperature → temperature
  ...

响应路径:
  内容路径: choices.0.message.content
  Prompt Tokens: usage.prompt_tokens
  Completion Tokens: usage.completion_tokens
```

### 示例 2：自定义协议 API

假设目标 API 请求格式：
```json
{
  "input": {
    "prompt": "...",
    "config": {
      "temp": 0.7,
      "max_len": 2048
    }
  }
}
```

响应格式：
```json
{
  "output": {
    "text": "...",
    "stats": {
      "input_tokens": 10,
      "output_tokens": 50
    }
  }
}
```

**配置方式**：
```
字段映射:
  messages → input.prompt  (需要额外处理)
  temperature → input.config.temp
  max_tokens → input.config.max_len

响应路径:
  内容路径: output.text
  Prompt Tokens: output.stats.input_tokens
  Completion Tokens: output.stats.output_tokens
```

**注意**：当前版本字段映射仅支持一级路径，复杂嵌套需要联系开发扩展。

---

## 技术架构

### 后端实现

#### 1. 数据库结构

**custom_channel_configs 表**：
- 基础信息：`name`, `description`, `adapter_type`
- 请求配置：`request_method`, `request_endpoint`, `auth_type` 等
- 字段映射：`field_model`, `field_messages` 等
- 响应路径：`response_content_path`, `response_usage_path` 等
- 流式配置：`stream_enabled`, `stream_data_prefix` 等

**channels 表新增字段**：
- `is_custom`：标识是否为自定义渠道
- `custom_config_id`：关联配置 ID

#### 2. 适配器实现

**`adapter/custom/adaptor.go`**：
- `ConvertRequest()`：OpenAI 格式 → 目标格式转换
- `DoRequest()`：构建 HTTP 请求（认证、Headers）
- `DoResponse()`：解析响应并提取字段
- `handleStreamResponse()`：处理 SSE 流式响应

#### 3. 工厂集成

`adapter/factory.go` 中优先检查 `is_custom` 标志，动态加载配置：

```go
if channel.IsCustom == 1 && channel.CustomConfigId > 0 {
    var config model.CustomChannelConfig
    common.DB.First(&config, channel.CustomConfigId)
    return &custom.Adaptor{Config: &config}
}
```

---

## API 接口

### 自定义配置管理

**获取所有配置**  
`GET /api/custom-channel-config`

**获取单个配置**  
`GET /api/custom-channel-config/:id`

**创建配置**  
`POST /api/custom-channel-config`

**更新配置**  
`PUT /api/custom-channel-config`

**删除配置**  
`DELETE /api/custom-channel-config/:id`

**测试配置**  
`POST /api/custom-channel-config/:id/test`

---

## 限制与注意事项

1. **字段映射限制**：当前仅支持一级路径映射，复杂嵌套结构需扩展
2. **响应路径**：仅支持点路径语法（如 `a.b.c`），不支持数组索引（`a[0].b`）
3. **认证方式**：主要支持 Bearer Token，其他方式需自定义 Headers
4. **流式响应**：仅支持 SSE 格式，不支持 WebSocket
5. **Token 估算**：流式响应的 Token 统计为简单估算，非精确值

---

## 未来扩展方向

- [ ] 前端可视化配置界面（当前后端已完成）
- [ ] 支持数组路径提取（如 `choices[0].message.content`）
- [ ] 支持 Lua/JavaScript 自定义转换脚本
- [ ] 配置模板市场（社区共享）
- [ ] WebSocket 流式支持
- [ ] 请求/响应日志记录与调试工具

---

## 故障排查

### 问题 1：渠道测试失败

**检查项**：
1. Base URL 是否正确（不要带尾部斜杠）
2. API Key 是否有效
3. 请求端点是否正确
4. 认证头模板是否正确配置

### 问题 2：响应内容为空

**检查项**：
1. 响应内容路径是否正确
2. 查看目标 API 实际返回格式
3. 确认路径语法（使用 `.` 分隔）

### 问题 3：Token 统计不准确

**原因**：流式响应的 Token 统计为估算值  
**解决**：在配置中正确设置 `response_prompt_tokens_path` 等字段

---

## 贡献与反馈

如需帮助或发现问题，请提交 Issue 或 PR。

**当前版本**：v1.0.0  
**最后更新**：2026-06-07
