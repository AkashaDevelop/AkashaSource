package common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
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

// buildSQLiteDSN 给 SQLite DSN 追加并发相关的 pragma 参数喵～
// glebarez/sqlite（底层 modernc.org/sqlite）用的是 `_pragma=NAME(value)` 语法，
// 不是 mattn/go-sqlite3 那种 `_journal_mode=` 写法，别搞混啦
// - journal_mode(WAL): 读写不互相阻塞
// - busy_timeout(5000): 写锁冲突时等待而不是直接报 SQLITE_BUSY
// - synchronous(NORMAL): 搭配 WAL 使用的官方推荐折中项
func buildSQLiteDSN(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn // 用户已经自己配置过 pragma，不重复覆盖
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
}

// InitDB 初始化数据库连接
// driver: sqlite, mysql, postgres
// dsn: 数据源名称 (例如: "file.db", "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
// 连接失败时返回 error 而不是直接终止进程——引导式初始化向导需要在数据库还没配好的时候也能把 API 跑起来
func InitDB(driver string, dsn string) error {
	var err error
	var dialector gorm.Dialector

	switch driver {
	case "sqlite":
		dialector = sqlite.Open(buildSQLiteDSN(dsn))
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

	if driver == "sqlite" {
		// SQLite 写入本质单线程，多连接没有意义，只会增加锁等待/SQLITE_BUSY 概率喵
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	} else {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	DBDriver = driver
	log.Printf("Database connected successfully (%s)", driver)
	return nil
}
