package email

import "STfreApi/common"

// SendEmail 委托给 common.SendEmail，统一邮件发送实现与安全策略。
// 参数顺序为 (to, subject, body)，与历史调用方保持兼容。
func SendEmail(to string, subject string, body string) error {
	return common.SendEmail(subject, to, body)
}
