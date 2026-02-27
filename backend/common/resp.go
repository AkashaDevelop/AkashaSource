package common

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code 业务状态码
type Code int

const (
	CodeSuccess    Code = 0
	CodeParamError Code = 10001
	CodeUnauthorized Code = 10002
	CodeForbidden  Code = 10003
	CodeNotFound   Code = 10004
	CodeConflict   Code = 10005
	CodeServerError Code = 50000
)

// R 统一响应结构
type R struct {
	Code Code        `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// OK 成功响应（带数据）
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeSuccess, Msg: "操作成功", Data: data})
}

// OKMsg 成功响应（自定义消息）
func OKMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeSuccess, Msg: msg, Data: data})
}

// Fail 业务错误响应（始终 HTTP 200，业务码非 0）
func Fail(c *gin.Context, code Code, msg string) {
	c.JSON(http.StatusOK, R{Code: code, Msg: msg})
}

// Failf 格式化错误响应
func Failf(c *gin.Context, code Code, format string, args ...interface{}) {
	c.JSON(http.StatusOK, R{Code: code, Msg: fmt.Sprintf(format, args...)})
}
