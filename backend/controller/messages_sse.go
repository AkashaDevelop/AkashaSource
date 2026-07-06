package controller

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func writeClaudeSSE(c *gin.Context, eventType string, data interface{}) {
	c.Writer.Write(renderClaudeSSE(eventType, data))
	c.Writer.Flush()
}

// renderClaudeSSE 只负责把事件渲染成字节，不负责写出去——供需要"先攒住再决定发不发"的场景使用～
func renderClaudeSSE(eventType string, data interface{}) []byte {
	jsonBytes, _ := json.Marshal(data)
	var buf bytes.Buffer
	buf.WriteString("event: " + eventType + "\n")
	buf.WriteString("data: " + string(jsonBytes) + "\n\n")
	return buf.Bytes()
}

func openAIStopToClaude(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func splitSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
