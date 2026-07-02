package context_sanitizer

// 注意: examples.go 包含伪代码示例，不参与编译
// 实际集成代码已在 relay.go 和 adaptor.go 中完成

// 集成点说明:
// 1. backend/main.go - 调用 contextsanitizer.Init()
// 2. backend/controller/relay.go - 调用 ApplyRequest()
// 3. backend/adapter/openai/adaptor.go - 调用 ApplyOpenAIResponse() 和流式处理
