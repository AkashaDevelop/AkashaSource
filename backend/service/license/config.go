// ⚠️ REMOVABLE MODULE — 系统授权门禁（GitHub 组织成员校验 + 单实例激活）
// 这个模块未来可能整体移除，尽量别让它的逻辑渗透进其它包里喵～
package license

import "encoding/base64"

// 下面几个值必须在打包发布前由项目作者手动替换成真实值：
//  1. targetOrgB64 —— 目标 GitHub 组织名的 base64 编码
//  2. gistIdB64    —— 存绑定记录的 Gist ID 的 base64 编码
//  3. gistTokenB64 —— 只需要 gist 权限的 GitHub PAT 的 base64 编码（强烈建议用独立机器人账号的 token，
//     不要用作者主账号的 PAT——泄露了也只会祸害这一个 Gist，不牵连别的账号权限）
//  4. hmacSecretB64 —— 纯内部使用的签名密钥，不对应任何外部账号，随机生成一次即可，
//     用来防止有人直接改数据库里的 authorized 字段绕过授权（见 signature.go）
//  5. licenseGithubClientIdB64 / licenseGithubClientSecretB64 —— 单独给这个授权模块注册的
//     GitHub OAuth App（不要复用站点登录用的那个 OAuth App，两者用途不同）。
//     去 GitHub → 头像 → Settings → Developer settings → OAuth Apps → New OAuth App 创建：
//     Homepage URL：随便填你的站点地址
//     Authorization callback URL：随便填（Device Flow 不需要回调地址）
//     创建后进入 App 详情页 → 勾选 "Enable Device Flow"（必须启用，否则 Device Flow 接口会报 403）
//     然后把 Client ID / Client Secret 分别 base64 编码填进下面
//
// 生成方式：echo -n "真实值" | base64
//
// 不读环境变量、不进 Option 表、不在任何管理页面暴露——这几项只有项目作者自己知道，
// 部署方/站长在后台看不到也改不了，只能老老实实走 GitHub OAuth 走完整个授权流程喵。
const (
	targetOrgB64                 = "QWthc2hhRGV2ZWxvcA=="                                     // base64("AkashaDevelop")
	gistIdB64                    = "Mjk5NDcwYjBiNDEzMmIzZTMwNmExZGNhMjc2ZjI3NjI="             // base64(真实 Gist ID)
	gistTokenB64                 = "Z2hwX3h4UnNOTTFkejNhOGNhRGxXdEh4ZDdMc3RCb3F3MzFhWXk1RA==" // base64(真实 PAT)
	hmacSecretB64                = "hb5vkkT9ToCJxhmIQebPf09pjtniMI0rvNFWk7vqcq0="             // 随机生成的内部签名密钥
	licenseGithubClientIdB64     = "T3YyM2xpckVxZDdoMXhsYXoxbDA="                             // base64(独立 OAuth App 的 Client ID)
	licenseGithubClientSecretB64 = "NDE0YjY2OTAwNTliN2YyZGFiMTRiMWZhZmM0YTljNjBkOTNkZDkxYQ==" // base64(独立 OAuth App 的 Client Secret)
)

var (
	targetOrg                 = decodeB64(targetOrgB64)
	gistId                    = decodeB64(gistIdB64)
	gistToken                 = decodeB64(gistTokenB64)
	hmacSecret                = []byte(decodeB64(hmacSecretB64))
	licenseGithubClientId     = decodeB64(licenseGithubClientIdB64)
	licenseGithubClientSecret = decodeB64(licenseGithubClientSecretB64)
)

func decodeB64(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(b)
}

// FeatureEnabled 几项核心配置都填了真实值才算启用；只要有一项为空（占位符没被替换），
// 整个门禁模块直接跳过，不影响开发/测试环境构建，也不影响没有配置这个功能的部署喵～
func FeatureEnabled() bool {
	return targetOrg != "" && gistId != "" && gistToken != "" && len(hmacSecret) > 0 &&
		licenseGithubClientId != "" && licenseGithubClientSecret != ""
}
