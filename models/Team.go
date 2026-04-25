package models

import (
	"errors"
	"strings"
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"gorm.io/gorm"
)

//团队.
type Team struct {
	TeamId      int       `gorm:"primaryKey;autoIncrement;column:team_id;uniqueIndex" json:"team_id"`
	TeamName    string    `gorm:"size:255;column:team_name;comment:团队名称" json:"team_name"`
	MemberId    int       `gorm:"type:int;column:member_id;comment:创建人id" json:"member_id"`
	IsDelete    bool      `gorm:"column:is_delete;default:false;comment:是否删除 false：否 true：是" json:"is_delete"`
	CreateTime  time.Time `gorm:"type:datetime;autoCreateTime;column:create_time;comment:创建时间" json:"create_time"`
	MemberCount int       `gorm:"-" json:"member_count"`
	BookCount   int       `gorm:"-" json:"book_count"`
	MemberName  string    `gorm:"-" json:"member_name"`
}

// TableName 获取对应数据库表名.
func (t *Team) TableName() string {
	return conf.GetDatabasePrefix() + "teams"
}

func NewTeam() *Team {
	return &Team{}
}

// 查询一个团队.
func (t *Team) First(id int, cols ...string) (*Team, error) {
	if id <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	query := DB.Table(t.TableName()).Where("team_id = ?", id)
	if len(cols) > 0 {
		query = query.Select(strings.Join(cols, ","))
	}
	err := query.First(t).Error

	if err != nil {
		logger.Error("查询团队失败 ->", id, err)
		return nil, err
	}
	t.Include()
	return t, err
}

func (t *Team) Delete(id int) (err error) {
	if id <= 0 {
		return ErrInvalidParameter
	}

	tx := DB.Begin()

	if tx.Error != nil {
		logger.Error("开启事物时出错 ->", tx.Error)
		return tx.Error
	}

	if err = tx.Table(t.TableName()).Where("team_id = ?", id).Delete(nil).Error; err != nil {
		logger.Error("删除团队时出错 ->", err)
		tx.Rollback()
		return
	}

	if err = tx.Exec("delete from md_team_member where team_id=?;", id).Error; err != nil {
		logger.Error("删除团队成员时出错 ->", err)
		tx.Rollback()
		return
	}

	if err = tx.Exec("delete from md_team_relationship where team_id=?;", id).Error; err != nil {
		logger.Error("删除团队项目时出错 ->", err)
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	return
}

//分页查询团队.
func (t *Team) FindToPager(pageIndex, pageSize int) (list []*Team, totalCount int, err error) {

	offset := (pageIndex - 1) * pageSize

	err = DB.Table(t.TableName()).Order("team_id desc").Offset(offset).Limit(pageSize).Find(&list).Error

	if err != nil {
		return
	}

	var c int64
	if err = DB.Table(t.TableName()).Count(&c).Error; err != nil {
		return
	}
	totalCount = int(c)

	for _, item := range list {
		item.Include()
	}
	return
}

func (t *Team) Include() {

	if member, err := NewMember().Find(t.MemberId, "account", "real_name"); err == nil {
		if member.RealName != "" {
			t.MemberName = member.RealName
		} else {
			t.MemberName = member.Account
		}
	}
	var c int64
	if DB.Table(NewTeamRelationship().TableName()).Where("team_id = ?", t.TeamId).Count(&c); c > 0 {
		t.BookCount = int(c)
	}
	if DB.Table(NewTeamMember().TableName()).Where("team_id = ?", t.TeamId).Count(&c); c > 0 {
		t.MemberCount = int(c)
	}
}

//更新或添加一个团队.
func (t *Team) Save(cols ...string) (err error) {
	if t.TeamName == "" {
		return NewError(5001, "团队名称不能为空")
	}

	if t.TeamId <= 0 {
		var count int64
		DB.Table(t.TableName()).Where("team_name = ?", t.TeamName).Count(&count)
		if count > 0 {
			return errors.New("团队名称已存在")
		}
	}
	if t.TeamId <= 0 {
		err = DB.Create(t).Error
	} else {
		err = DB.Save(t).Error
	}
	if err != nil {
		logger.Error("在保存团队时出错 ->", err)
	}
	return
}
