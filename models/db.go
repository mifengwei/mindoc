package models

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DB 全局 GORM 数据库实例
var DB *gorm.DB

// SetDB 设置全局数据库实例
func SetDB(db *gorm.DB) {
	DB = db
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	return DB
}

// GORM配置选项
func GORMConfig() *gorm.Config {
	return &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
}
