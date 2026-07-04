package common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite" // Pure Go SQLite driver
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB       *gorm.DB
	DBDriver string
)

// InitDB 初始化数据库连接
// driver: sqlite, mysql, postgres
// dsn: 数据源名称 (例如: "file.db", "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
// 连接失败时返回 error 而不是直接终止进程——引导式初始化向导需要在数据库还没配好的时候也能把 API 跑起来
func InitDB(driver string, dsn string) error {
	var err error
	var dialector gorm.Dialector

	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", driver)
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
			ParameterizedQueries:      true,        // 不在 SQL 日志中包含参数
			Colorful:                  true,        // 启用/禁用彩色打印
		},
	)

	config := &gorm.Config{
		Logger: newLogger,
	}

	db, err := gorm.Open(dialector, config)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// Connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("获取数据库连接池失败: " + err.Error())
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库无法访问: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	DBDriver = driver
	log.Printf("Database connected successfully (%s)", driver)
	return nil
}
