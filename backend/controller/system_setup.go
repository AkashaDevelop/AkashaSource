package controller

// 宸汐清源姊妹功能～引导式初始化向导的数据库连接接口 (｡•ᴗ•｡)
// 只负责"接上数据库"这一件事：账号创建还是走原来的 GetSetup/PostSetup。

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/bootstrap"

	"github.com/gin-gonic/gin"
)

type SetupDatabaseRequest struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

func PostSetupDatabase(c *gin.Context) {
	// 已经有超管账号了，说明向导早就走完过，不允许再偷偷换库
	var rootCount int64
	if common.DB != nil {
		common.DB.Model(&model.User{}).Where("role = ?", model.RoleRoot).Count(&rootCount)
	}
	if rootCount > 0 {
		common.Fail(c, common.CodeConflict, "系统已经初始化完成，不能重新配置数据库")
		return
	}

	var req SetupDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数有误")
		return
	}
	if req.Driver != "sqlite" && req.Driver != "mysql" && req.Driver != "postgres" {
		common.Fail(c, common.CodeParamError, "不支持的数据库驱动")
		return
	}
	if req.DSN == "" {
		common.Fail(c, common.CodeParamError, "数据库连接信息不能为空")
		return
	}

	if err := common.InitDB(req.Driver, req.DSN); err != nil {
		common.Failf(c, common.CodeServerError, "连接数据库失败: %s", err.Error())
		return
	}

	if err := common.SaveDBConfig(common.DBConfig{Driver: req.Driver, DSN: req.DSN}); err != nil {
		common.Failf(c, common.CodeServerError, "数据库已连接，但保存配置失败: %s", err.Error())
		return
	}

	bootstrap.RunPostDBInit()
	common.OKMsg(c, "数据库连接成功", gin.H{"connected": true})
}
