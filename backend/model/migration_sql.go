package model

import (
	"STfreApi/common"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SQLMigration struct {
	Id        int   `json:"id" gorm:"primaryKey;autoIncrement"`
	Version   int   `json:"version" gorm:"uniqueIndex"`
	Name      string `json:"name"`
	AppliedAt int64 `json:"applied_at"`
}

type SQLMigrationFile struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

func ApplySQLFileMigrations() error {
	if err := common.DB.AutoMigrate(&SQLMigration{}); err != nil {
		return err
	}
	migrations, err := loadSQLMigrations("migrations")
	if err != nil {
		return err
	}
	for _, item := range migrations {
		var count int64
		common.DB.Model(&SQLMigration{}).Where("version = ?", item.Version).Count(&count)
		if count > 0 {
			continue
		}
		if strings.TrimSpace(item.UpSQL) == "" {
			continue
		}
		if err := common.DB.Exec(item.UpSQL).Error; err != nil {
			return err
		}
		common.DB.Create(&SQLMigration{
			Version:   item.Version,
			Name:      item.Name,
			AppliedAt: time.Now().Unix(),
		})
	}
	return nil
}

func RollbackSQLFileMigrations(steps int) error {
	if steps <= 0 {
		return nil
	}
	var applied []SQLMigration
	common.DB.Order("version desc").Limit(steps).Find(&applied)
	if len(applied) == 0 {
		return nil
	}
	migrations, err := loadSQLMigrations("migrations")
	if err != nil {
		return err
	}
	index := map[int]SQLMigrationFile{}
	for _, item := range migrations {
		index[item.Version] = item
	}
	for _, item := range applied {
		file, ok := index[item.Version]
		if !ok || strings.TrimSpace(file.DownSQL) == "" {
			continue
		}
		if err := common.DB.Exec(file.DownSQL).Error; err != nil {
			return err
		}
		common.DB.Delete(&SQLMigration{}, "version = ?", item.Version)
	}
	return nil
}

func loadSQLMigrations(dir string) ([]SQLMigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SQLMigrationFile{}, nil
		}
		return nil, err
	}
	type pair struct {
		version int
		name    string
		upPath  string
		downPath string
	}
	tmp := map[int]*pair{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		base := strings.TrimSuffix(name, ".sql")
		parts := strings.Split(base, ".")
		if len(parts) < 2 {
			continue
		}
		dirName := parts[len(parts)-1]
		core := strings.TrimSuffix(base, "."+dirName)
		coreParts := strings.SplitN(core, "_", 2)
		if len(coreParts) < 2 {
			continue
		}
		version, err := strconv.Atoi(coreParts[0])
		if err != nil {
			continue
		}
		if _, ok := tmp[version]; !ok {
			tmp[version] = &pair{version: version, name: coreParts[1]}
		}
		path := filepath.Join(dir, name)
		if dirName == "up" {
			tmp[version].upPath = path
		} else if dirName == "down" {
			tmp[version].downPath = path
		}
	}
	var versions []int
	for v := range tmp {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	var result []SQLMigrationFile
	dialect := common.DB.Dialector.Name()
	for _, v := range versions {
		item := tmp[v]
		upSQL := readSQLByDialect(item.upPath, dialect)
		downSQL := readSQLByDialect(item.downPath, dialect)
		result = append(result, SQLMigrationFile{
			Version: v,
			Name:    item.name,
			UpSQL:   upSQL,
			DownSQL: downSQL,
		})
	}
	return result, nil
}

func readSQLByDialect(path string, dialect string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)
	sections := map[string][]string{}
	current := ""
	for _, line := range strings.Split(content, "\n") {
		lineTrim := strings.TrimSpace(line)
		if strings.HasPrefix(lineTrim, "-- +dialect:") {
			current = strings.TrimSpace(strings.TrimPrefix(lineTrim, "-- +dialect:"))
			continue
		}
		if current == "" {
			current = "common"
		}
		sections[current] = append(sections[current], line)
	}
	if sql, ok := sections[dialect]; ok {
		return strings.TrimSpace(strings.Join(sql, "\n"))
	}
	if sql, ok := sections["common"]; ok {
		return strings.TrimSpace(strings.Join(sql, "\n"))
	}
	return strings.TrimSpace(content)
}
