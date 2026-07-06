package common

// System Constants
var (
	Version      = "公测1.0.0"
	StartTime    = GetTimestamp()
	QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens

	// Image Generation Quota (Fixed for now)
	QuotaDalle3 = 20000 // $0.04
	QuotaDalle2 = 10000 // $0.02

	DefaultModel = "gpt-3.5-turbo"

	// OAuth
	GitHubClientId      = ""
	GitHubClientSecret  = ""
	LinuxDOClientId     = ""
	LinuxDOClientSecret = ""

	// Email
	SMTPServer     = ""
	SMTPPort       = 587
	SMTPAccount    = ""
	SMTPPassword   = ""
	SMTPFrom       = ""
	SMTPSSLEnabled = false

	// Verification
	EmailVerificationEnabled = false
	TurnstileSiteKey         = ""
	TurnstileSecretKey       = ""
	ContentModerationTimeout = 5
	// LinuxDO Level Quota (Map level 0-5 to quota)
	LinuxDOLevelQuota = map[int]int64{
		0: 0,
		1: 0,
		2: 0,
		3: 0,
		4: 0,
		5: 0,
	}

	// Midjourney Quota
	QuotaMJImagine   = 50000 // $0.1
	QuotaMJUpscale   = 5000  // $0.01
	QuotaMJVariation = 5000  // $0.01

	// Turnstile
	TurnstileCheckEnabled = false

	// GeeTest (极验)
	GeeTestEnabled   = false
	GeeTestId        = ""
	GeeTestKey       = ""
	CaptchaProvider  = "" // "turnstile" or "geetest"
	CheckinCaptcha   = false // 签到是否需要人机验证

	// hCaptcha
	HCaptchaSiteKey   = ""
	HCaptchaSecretKey = ""
	HCaptchaEnabled   = false

	// Google reCAPTCHA
	ReCaptchaSiteKey   = ""
	ReCaptchaSecretKey = ""
	ReCaptchaEnabled   = false
	ReCaptchaVersion   = "v2" // "v2" or "v3"

	// Redis
	RedisAddr     = ""
	RedisPassword = ""
	RedisDB       = 0

	// Thinking-to-Content
	ThinkingToContent = false

	// OAuth - Discord
	DiscordClientId     = ""
	DiscordClientSecret = ""

	// OIDC
	OIDCClientId     = ""
	OIDCClientSecret = ""
	OIDCIssuerURL    = ""

	// Telegram
	TelegramBotToken = ""

	// WeChat Open Platform
	WechatAppId     = ""
	WechatAppSecret = ""

	// Email domain restriction
	EmailDomainRestrictionEnabled = false
	EmailDomainWhitelist          = ""

	// Suno
	QuotaSunoGenerate = 50000 // $0.1

	// 易支付
	EpayApiUrl    = ""
	EpayPid       = ""
	EpayKey       = ""
	EpayType      = ""
	EpayNotifyUrl = ""
	EpayReturnUrl = ""

	// 宸汐御安全通讯协议（默认禁用，由超管在系统设置中开启）
	CxSecEnabled        = false
	CxSecProtectedPaths = "/api/user/login,/api/user/register,/api/user/login/2fa,/api/user/password/reset-request,/api/user/password/reset-confirm,/api/user/checkin"

	// 宸汐清源（默认禁用，由超管在系统设置中开启）
	QingyuanEnabled = false
)

// Request Constants
const (
	RequestIdKey = "X-Akasha-Request-Id"
)
