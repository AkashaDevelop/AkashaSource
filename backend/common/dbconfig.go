package common

// 宸汐清源姊妹功能～引导式初始化的小记事本 (｡•ᴗ•｡)
// 向导填好数据库连接之后，把它悄悄记在运行目录的 dbconfig.json 里，
// 下次启动就不用再走一遍向导啦～和 sqlite 默认的 akasha.db 放在同一层，
// 沿用"配置文件就摆在运行目录"这个项目一直以来的习惯。

import (
	"encoding/json"
	"os"
)

const dbConfigPath = "dbconfig.json"

// DBConfig 持久化保存的数据库连接配置
type DBConfig struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

// LoadDBConfig 读取已持久化的数据库配置，不存在时返回 (零值, false)
func LoadDBConfig() (DBConfig, bool) {
	data, err := os.ReadFile(dbConfigPath)
	if err != nil {
		return DBConfig{}, false
	}
	var cfg DBConfig
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Driver == "" {
		return DBConfig{}, false
	}
	return cfg, true
}

// SaveDBConfig 把数据库配置写进运行目录，供下次启动直接读取
// 用 0600 权限，因为 mysql/postgres 的密码会明文写在里面
func SaveDBConfig(cfg DBConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dbConfigPath, data, 0600)
}
