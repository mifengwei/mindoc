// Package models .
package models

import (
	"errors"
	"github.com/mindoc-org/mindoc/utils/auth2"
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

var (
	_ Auth2Account = (*WorkWeixinAccount)(nil)
	_ Auth2Account = (*DingTalkAccount)(nil)
)

type Auth2Account interface {
	ExistedMember(id string) (*Member, error)
	AddBind(userInfo auth2.UserInfo, member *Member) error
}

func NewWorkWeixinAccount() *WorkWeixinAccount {
	return &WorkWeixinAccount{}
}

type WorkWeixinAccount struct {
	MemberId          int    `gorm:"column:member_id;type:int;default:-1;index" json:"member_id"`
	UserDbId          int    `gorm:"primaryKey;autoIncrement;column:user_db_id" json:"user_db_id"`
	WorkWeixin_UserId string `gorm:"size:100;column:workweixin_user_id;uniqueIndex" json:"workweixin_user_id"`
	// WorkWeixin_Name   string    `gorm:"size:255;column:workweixin_name" json:"workweixin_name"`
	// WorkWeixin_Phone  string    `gorm:"size:25;column:workweixin_phone" json:"workweixin_phone"`
	// WorkWeixin_Email  string    `gorm:"size:255;column:workweixin_email" json:"workweixin_email"`
	// WorkWeixin_Status int       `gorm:"type:int;column:status" json:"status"`
	// WorkWeixin_Avatar string    `gorm:"size:1024;column:avatar" json:"avatar"`
	CreateTime    time.Time `gorm:"type:datetime;column:create_time;autoCreateTime" json:"create_time"`
	CreateAt      int       `gorm:"type:int;column:create_at" json:"create_at"`
	LastLoginTime time.Time `gorm:"type:datetime;column:last_login_time" json:"last_login_time"`
}

// TableName 获取对应数据库表名.
func (m *WorkWeixinAccount) TableName() string {
	return conf.GetDatabasePrefix() + "workweixin_accounts"
}

func (m *WorkWeixinAccount) ExistedMember(workweixin_user_id string) (*Member, error) {
	account := NewWorkWeixinAccount()
	member := NewMember()
	err := DB.Table(m.TableName()).Where("workweixin_user_id = ?", workweixin_user_id).First(account).Error
	if err != nil {
		return member, err
	}

	member, err = member.Find(account.MemberId)
	if err != nil {
		return member, err
	}

	if member.Status != 0 {
		return member, errors.New("receive_account_disabled")
	}

	return member, nil

}

// AddBind 添加一个用户.
func (m *WorkWeixinAccount) AddBind(userInfo auth2.UserInfo, member *Member) error {
	tmpM := NewWorkWeixinAccount()
	err := DB.Table(m.TableName()).Where("workweixin_user_id = ?", userInfo.UserId).First(tmpM).Error
	if err == nil {
		tmpM.MemberId = member.MemberId
		if err := DB.Save(tmpM).Error; err != nil {
			logger.Error("保存用户数据到数据时失败 =>", err)
			return errors.New("用户信息绑定失败, 数据库错误")
		}
		return nil
	}

	m.MemberId = member.MemberId
	m.WorkWeixin_UserId = userInfo.UserId

	var c int64
	DB.Table(m.TableName()).Where("member_id = ?", m.MemberId).Count(&c)
	if c > 0 {
		return errors.New("已绑定，不可重复绑定")
	}

	if err := DB.Create(m).Error; err != nil {
		logger.Error("保存用户数据到数据时失败 =>", err)
		return errors.New("用户信息绑定失败, 数据库错误")
	}

	return nil
}

func NewDingTalkAccount() *DingTalkAccount {
	return &DingTalkAccount{}
}

type DingTalkAccount struct {
	MemberId        int       `gorm:"column:member_id;type:int;default:-1;index" json:"member_id"`
	UserDbId        int       `gorm:"primaryKey;autoIncrement;column:user_db_id" json:"user_db_id"`
	Dingtalk_UserId string    `gorm:"size:100;column:dingtalk_user_id;uniqueIndex" json:"dingtalk_user_id"`
	CreateTime      time.Time `gorm:"type:datetime;column:create_time;autoCreateTime" json:"create_time"`
	CreateAt        int       `gorm:"type:int;column:create_at" json:"create_at"`
	LastLoginTime   time.Time `gorm:"type:datetime;column:last_login_time" json:"last_login_time"`
}

// TableName 获取对应数据库表名.
func (m *DingTalkAccount) TableName() string {
	return conf.GetDatabasePrefix() + "dingtalk_accounts"
}

func (m *DingTalkAccount) ExistedMember(userid string) (*Member, error) {
	account := NewDingTalkAccount()
	member := NewMember()
	err := DB.Table(m.TableName()).Where("dingtalk_user_id = ?", userid).First(account).Error
	if err != nil {
		return member, err
	}

	member, err = member.Find(account.MemberId)
	if err != nil {
		return member, err
	}

	if member.Status != 0 {
		return member, errors.New("receive_account_disabled")
	}

	return member, nil

}

// AddBind 添加一个用户.
func (m *DingTalkAccount) AddBind(userInfo auth2.UserInfo, member *Member) error {
	tmpM := NewDingTalkAccount()
	err := DB.Table(m.TableName()).Where("dingtalk_user_id = ?", userInfo.UserId).First(tmpM).Error
	if err == nil {
		tmpM.MemberId = member.MemberId
		if err := DB.Save(tmpM).Error; err != nil {
			logger.Error("保存用户数据到数据时失败 =>", err)
			return errors.New("用户信息绑定失败, 数据库错误")
		}
		return nil
	}

	m.Dingtalk_UserId = userInfo.UserId
	m.MemberId = member.MemberId

	var c int64
	DB.Table(m.TableName()).Where("member_id = ?", m.MemberId).Count(&c)
	if c > 0 {
		return errors.New("已绑定，不可重复绑定")
	}

	if err := DB.Create(m).Error; err != nil {
		logger.Error("保存用户数据到数据时失败 =>", err)
		return errors.New("用户信息绑定失败, 数据库错误")
	}

	return nil
}
