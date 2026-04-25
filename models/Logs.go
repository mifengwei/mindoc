package models

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/mindoc-org/mindoc/conf"
)

var loggerQueue = &logQueue{channel: make(chan *Logger, 100), isRuning: 0}

type logQueue struct {
	channel  chan *Logger
	isRuning int32
}

// Logger struct .
type Logger struct {
	LoggerId int64 `gorm:"primaryKey;autoIncrement;uniqueIndex;column:log_id" json:"log_id"`
	MemberId int   `gorm:"column:member_id;type:int" json:"member_id"`
	// 日志类别：operate 操作日志/ system 系统日志/ exception 异常日志 / document 文档操作日志
	Category     string    `gorm:"column:category;size:255;default:operate" json:"category"`
	Content      string    `gorm:"column:content;type:text" json:"content"`
	OriginalData string    `gorm:"column:original_data;type:text" json:"original_data"`
	PresentData  string    `gorm:"column:present_data;type:text" json:"present_data"`
	CreateTime   time.Time `gorm:"type:datetime;column:create_time;autoCreateTime" json:"create_time"`
	UserAgent    string    `gorm:"column:user_agent;size:500" json:"user_agent"`
	IPAddress    string    `gorm:"column:ip_address;size:255" json:"ip_address"`
}

// TableName 获取对应数据库表名.
func (m *Logger) TableName() string {
	return conf.GetDatabasePrefix() + "logs"
}

func NewLogger() *Logger {
	return &Logger{}
}

func (m *Logger) Add() error {
	if m.MemberId <= 0 {
		return errors.New("用户ID不能为空")
	}
	if m.Category == "" {
		m.Category = "system"
	}
	if m.Content == "" {
		return errors.New("日志内容不能为空")
	}
	loggerQueue.channel <- m
	if atomic.LoadInt32(&(loggerQueue.isRuning)) <= 0 {
		atomic.AddInt32(&(loggerQueue.isRuning), 1)
		go addLoggerAsync()
	}
	return nil
}

func addLoggerAsync() {
	defer atomic.AddInt32(&(loggerQueue.isRuning), -1)

	for {
		logger := <-loggerQueue.channel

		DB.Create(logger)
	}
}
