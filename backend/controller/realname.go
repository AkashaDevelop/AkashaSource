package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/realname"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// InitRealnameVerify 发起实名认证
// POST /api/user/realname/init
func InitRealnameVerify(c *gin.Context) {
	if !realname.IsRealnameEnabled() {
		common.Fail(c, common.CodeForbidden, "实名认证功能未启用")
		return
	}
	if !realname.IsConfigured() {
		common.Fail(c, common.CodeServerError, "实名认证服务未配置，请联系管理员")
		return
	}

	userId := c.GetInt("id")
	var req struct {
		RealName string `json:"real_name" binding:"required"`
		IdCard   string `json:"id_card" binding:"required"`
		MetaInfo string `json:"meta_info"` // 前端 SDK 获取的 metaInfo，纯 API 模式可空
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数不完整：需要 real_name 和 id_card")
		return
	}

	// 检查是否已认证
	if realname.IsUserRealnameVerified(userId) {
		common.Fail(c, common.CodeParamError, "您已完成实名认证")
		return
	}

	// 创建认证记录
	outerOrderNo := fmt.Sprintf("rn_%d_%d", userId, time.Now().Unix())
	record := &model.RealnameAuth{
		UserId:     userId,
		CertifyId:  "", // 发起后填充
		Provider:   common.OptionMap[model.OptionKeyRealnameProvider],
		Status:     model.RealnameStatusPending,
		SceneId:    common.OptionMap[model.OptionKeyRealnameAliyunSceneId],
	}
	if record.Provider == "" {
		record.Provider = "aliyun"
	}

	// 双盲模式：只存流水号和结果，不存任何实名信息（合规处理）
	if !realname.IsDoubleBlindMode() {
		record.RealName = req.RealName
		record.IdCardHash = realname.HashIdCard(req.IdCard)
	}

	// 调用阿里云 InitFaceVerify
	client := realname.GetAliyunClient()
	result, err := client.InitFaceVerify(outerOrderNo, req.RealName, req.IdCard, req.MetaInfo)
	if err != nil {
		common.Failf(c, common.CodeServerError, "发起认证失败: %v", err)
		return
	}

	if result.Code != "200" {
		common.Failf(c, common.CodeServerError, "认证服务返回错误: %s - %s", result.Code, result.Message)
		return
	}

	// 保存认证流水号
	record.CertifyId = result.CertifyId
	if err := record.Create(); err != nil {
		common.Failf(c, common.CodeServerError, "保存认证记录失败: %v", err)
		return
	}

	common.OK(c, gin.H{
		"certify_id":  result.CertifyId,
		"certify_url": result.CertifyUrl,
		"message":     "认证已发起，请引导用户完成人脸核验",
	})
}

// QueryRealnameResult 查询认证结果
// POST /api/user/realname/query
func QueryRealnameResult(c *gin.Context) {
	if !realname.IsRealnameEnabled() {
		common.Fail(c, common.CodeForbidden, "实名认证功能未启用")
		return
	}

	userId := c.GetInt("id")
	var req struct {
		CertifyId string `json:"certify_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CertifyId == "" {
		// 没传 certify_id 就查最新的
		record, err := model.GetRealnameAuthByUserId(userId)
		if err != nil {
			common.Fail(c, common.CodeNotFound, "未找到认证记录")
			return
		}
		req.CertifyId = record.CertifyId
	}

	// 调用阿里云查询
	client := realname.GetAliyunClient()
	result, err := client.DescribeFaceVerify(req.CertifyId)
	if err != nil {
		common.Failf(c, common.CodeServerError, "查询认证结果失败: %v", err)
		return
	}

	if result.Code != "200" {
		common.Failf(c, common.CodeServerError, "查询失败: %s - %s", result.Code, result.Message)
		return
	}

	// 更新数据库记录
	record, err := realname.GetRecordByCertifyId(req.CertifyId)
	if err == nil {
		now := time.Now().Unix()
		if result.Passed == "T" {
			record.Status = model.RealnameStatusPassed
			record.VerifiedAt = now
		} else if result.Passed == "F" {
			record.Status = model.RealnameStatusFailed
			record.Reason = result.Reason
		}
		record.Update()

		// 同步用户状态
		if record.Status == model.RealnameStatusPassed {
			common.DB.Model(&model.User{}).Where("id = ?", record.UserId).
				Update("realname_status", model.RealnameStatusPassed)
		} else if record.Status == model.RealnameStatusFailed {
			common.DB.Model(&model.User{}).Where("id = ?", record.UserId).
				Update("realname_status", model.RealnameStatusFailed)
		}
	}

	passed := result.Passed == "T"
	common.OK(c, gin.H{
		"passed":  passed,
		"reason":  result.Reason,
		"sub_code": result.SubCode,
	})
}

// GetRealnameStatus 获取当前用户实名认证状态
// GET /api/user/realname/status
func GetRealnameStatus(c *gin.Context) {
	userId := c.GetInt("id")
	var user model.User
	if err := common.DB.Select("id, realname_status").First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取用户信息失败")
		return
	}

	// 获取最新认证记录
	record, _ := model.GetRealnameAuthByUserId(userId)

	resp := gin.H{
		"enabled":   realname.IsRealnameEnabled(),
		"verified":  user.RealnameStatus == model.RealnameStatusPassed,
		"status":    user.RealnameStatus,
		"scenarios": realname.GetRealnameScenarios(),
	}
	if record != nil {
		resp["real_name"] = maskRealName(record.RealName)
		resp["certify_id"] = record.CertifyId
		resp["created_at"] = record.CreatedAt
		resp["verified_at"] = record.VerifiedAt
	}

	common.OK(c, resp)
}

// AdminGetRealnameRecords 管理员查看认证记录
// GET /api/admin/realname/records
func AdminGetRealnameRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	statusStr := c.DefaultQuery("status", "-1")
	status, _ := strconv.Atoi(statusStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := model.GetAllRealnameRecords(page, pageSize, status)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询记录失败")
		return
	}

	// 关联用户名
	userIds := make([]int, 0, len(records))
	for _, r := range records {
		userIds = append(userIds, r.UserId)
	}
	userMap := make(map[int]string)
	if len(userIds) > 0 {
		var users []model.User
		common.DB.Where("id IN ?", userIds).Select("id, username").Find(&users)
		for _, u := range users {
			userMap[u.Id] = u.Username
		}
	}

	list := make([]gin.H, 0, len(records))
	for _, r := range records {
		list = append(list, gin.H{
			"id":          r.Id,
			"user_id":     r.UserId,
			"username":    userMap[r.UserId],
			"real_name":   maskRealName(r.RealName),
			"provider":    r.Provider,
			"status":      r.Status,
			"reason":      r.Reason,
			"certify_id":  r.CertifyId,
			"created_at":  r.CreatedAt,
			"verified_at": r.VerifiedAt,
		})
	}

	common.OK(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// AdminGetRealnameStats 管理员获取认证统计
// GET /api/admin/realname/stats
func AdminGetRealnameStats(c *gin.Context) {
	var totalUsers int64
	common.DB.Model(&model.User{}).Count(&totalUsers)

	var verifiedUsers int64
	common.DB.Model(&model.User{}).Where("realname_status = ?", model.RealnameStatusPassed).Count(&verifiedUsers)

	var pendingRecords int64
	common.DB.Model(&model.RealnameAuth{}).Where("status = ?", model.RealnameStatusPending).Count(&pendingRecords)

	var failedRecords int64
	common.DB.Model(&model.RealnameAuth{}).Where("status = ?", model.RealnameStatusFailed).Count(&failedRecords)

	common.OK(c, gin.H{
		"total_users":     totalUsers,
		"verified_users":  verifiedUsers,
		"pending_records": pendingRecords,
		"failed_records":  failedRecords,
		"enabled":         realname.IsRealnameEnabled(),
	})
}

// AdminResetRealname 管理员重置用户实名状态
// DELETE /api/admin/realname/:user_id
func AdminResetRealname(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	// 重置用户状态
	common.DB.Model(&model.User{}).Where("id = ?", userId).
		Update("realname_status", model.RealnameStatusPending)

	// 标记旧记录为过期
	common.DB.Model(&model.RealnameAuth{}).Where("user_id = ? AND status = ?", userId, model.RealnameStatusPassed).
		Update("status", model.RealnameStatusExpired)

	common.OKMsg(c, "已重置用户实名认证状态", nil)
}

// maskRealName 脱敏处理真实姓名
func maskRealName(name string) string {
	if len(name) <= 1 {
		return name
	}
	runes := []rune(name)
	if len(runes) == 2 {
		return string(runes[0]) + "*"
	}
	result := string(runes[0])
	for i := 1; i < len(runes)-1; i++ {
		result += "*"
	}
	result += string(runes[len(runes)-1])
	return result
}
