package models

import (
	"time"

	"github.com/mindoc-org/mindoc/conf"
)

type MemberToken struct {
	TokenId   int       `gorm:"column:token_id;primaryKey;autoIncrement;uniqueIndex" json:"token_id"`
	MemberId  int       `gorm:"column:member_id;type:int" json:"member_id"`
	Token     string    `gorm:"column:token;size:150;index" json:"token"`
	Email     string    `gorm:"column:email;size:255" json:"email"`
	IsValid   bool      `gorm:"column:is_valid" json:"is_valid"`
	ValidTime time.Time `gorm:"column:valid_time" json:"valid_time"`
	SendTime  time.Time `gorm:"column:send_time;type:datetime;autoCreateTime" json:"send_time"`
}

// TableName 获取对应数据库表名.
func (m *MemberToken) TableName() string {
	return conf.GetDatabasePrefix() + "member_token"
}

func NewMemberToken() *MemberToken {
	return &MemberToken{}
}

func (m *MemberToken) InsertOrUpdate() (*MemberToken, error) {

	if m.TokenId > 0 {
		err := DB.Save(m).Error
		return m, err
	}
	err := DB.Create(m).Error

	return m, err
}

func (m *MemberToken) FindByFieldFirst(field string, value interface{}) (*MemberToken, error) {
	err := DB.Table(m.TableName()).Where(field+" = ?", value).Order("token_id DESC").First(m).Error

	return m, err
}

func (m *MemberToken) FindSendCount(mail string, start_time time.Time, end_time time.Time) (int, error) {
	var c int64

	err := DB.Table(m.TableName()).
		Where("send_time >= ? AND send_time <= ?", start_time.Format("2006-01-02 15:04:05"), end_time.Format("2006-01-02 15:04:05")).
		Count(&c).Error

	if err != nil {
		return 0, err
	}
	return int(c), nil
}
