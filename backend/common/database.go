package common

import (
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
func InitDB(driver string, dsn string) {
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
		log.Fatalf("不支持的数据库驱动: %s", driver)
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

	DB, err = gorm.Open(dialector, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Connection pool settings
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DBDriver = driver
	log.Printf("Database connected successfully (%s)", driver)
}
