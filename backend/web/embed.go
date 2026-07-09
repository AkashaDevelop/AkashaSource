package web

// ～单文件部署：前端静态产物嵌入后端二进制～
// 构建流水线在编译后端前，会把 frontend/dist 的内容拷进 embedded/ 目录覆盖占位文件，
// go:embed 于是把整份前端打进二进制，运行时无需任何外部静态文件，真正开箱即跑。
//
// 路由策略（挂在 gin 的 NoRoute 上，即所有已注册路由都没命中时才轮到这里）：
//   - 命中真实存在的静态文件（/assets/xxx.js、/lib/cx_runtime.wasm 等）→ 直接返回该文件
//   - 其余 GET 请求 → 回退到 index.html，交给前端路由（SPA history 模式）
//   - API 前缀（/api、/v1 等）未命中 → 返回 JSON 404，不能吐 HTML 污染接口语义

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:embedded
var embeddedFiles embed.FS

// apiPrefixes 这些前缀属于后端接口/中转，未命中时应返回 JSON 404，绝不回退 index.html
var apiPrefixes = []string{
	"/api/", "/v1/", "/v1beta/", "/mj/", "/kling/", "/jimeng/",
	"/suno/", "/ping",
}

// RegisterFrontend 把嵌入的前端挂到 gin 的 NoRoute 兜底处理上。
// 必须在所有 API 路由注册完成后调用（NoRoute 只在无路由命中时触发，顺序其实不敏感，
// 但语义上应在 SetApiRouter 之后）。
func RegisterFrontend(r *gin.Engine) {
	// 取 embedded 子目录作为站点根（去掉 embedded/ 前缀，让 /assets/... 能直接对上）
	sub, err := fs.Sub(embeddedFiles, "embedded")
	if err != nil {
		panic("web: 无法定位嵌入的前端目录: " + err.Error())
	}
	staticFS := http.FS(sub)

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 前缀未命中 → JSON 404，不吐 HTML
		if isAPIPath(path) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{"message": "接口不存在", "type": "not_found"},
			})
			return
		}

		// 非 GET/HEAD 的非 API 请求（比如误发的 POST）→ 405，不回退页面
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusMethodNotAllowed)
			return
		}

		// 命中真实静态文件则直接返回，否则回退 index.html（SPA history 路由）
		if servableFile(sub, path) {
			c.FileFromFS(path, staticFS)
			return
		}
		c.FileFromFS("/", staticFS) // "/" → index.html
	})
}

func isAPIPath(path string) bool {
	for _, p := range apiPrefixes {
		if path == strings.TrimSuffix(p, "/") || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// servableFile 判断请求路径是否对应嵌入 FS 里一个真实存在的文件
func servableFile(sub fs.FS, urlPath string) bool {
	clean := strings.TrimPrefix(urlPath, "/")
	if clean == "" {
		return false // 根路径交给 index.html 回退分支
	}
	f, err := sub.Open(clean)
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	return true
}
