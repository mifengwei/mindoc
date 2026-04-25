package models

import (
	"time"

	"github.com/mindoc-org/mindoc/conf"
)

type Migration struct {
	MigrationId int       `gorm:"column:migration_id;primaryKey;autoIncrement;uniqueIndex" json:"migration_id"`
	Name        string    `gorm:"column:name;size:500" json:"name"`
	Statements  string    `gorm:"column:statements;type:text" json:"statements"`
	Status      string    `gorm:"column:status;default:update" json:"status"`
	CreateTime  time.Time `gorm:"column:create_time;type:datetime;autoCreateTime" json:"create_time"`
	Version     int64     `gorm:"type:bigint;column:version;uniqueIndex" json:"version"`
}

// TableName 获取对应数据库表名.
func (m *Migration) TableName() string {
	return conf.GetDatabasePrefix() + "migrations"
}

func NewMigration() *Migration {
	return &Migration{}
}

func (m *Migration) FindFirst() (*Migration, error) {
	err := DB.Table(m.TableName()).Order("migration_id DESC").First(m).Error

	return m, err
}
