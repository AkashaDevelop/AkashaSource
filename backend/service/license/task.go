// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 每 7 天重新确认一次组织成员身份 + Gist 绑定记录，防止"账号被踢出组织"或
// "作者手动去 Gist 里吊销授权"之后本机还傻乎乎地以为自己没事
package license

import (
	"log"
	"time"
)

// StartRevalidateTask 启动周期复核 goroutine；本部署没启用该功能时直接跳过，不占用资源
func StartRevalidateTask() {
	if !FeatureEnabled() {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := Revalidate(); err != nil {
				log.Printf("[license] 7天复核失败: %v", err)
			}
		}
	}()
}
