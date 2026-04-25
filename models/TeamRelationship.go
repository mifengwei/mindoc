package models

import (
	"errors"
	"strings"
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

type TeamRelationship struct {
	TeamRelationshipId int       `gorm:"primaryKey;autoIncrement;column:team_relationship_id;uniqueIndex" json:"team_relationship_id"`
	BookId             int       `gorm:"column:book_id;comment:项目id" json:"book_id"`
	TeamId             int       `gorm:"column:team_id;comment:团队id" json:"team_id"`
	CreateTime         time.Time `gorm:"type:datetime;autoCreateTime;column:create_time;comment:创建时间" json:"create_time"`
	TeamName           string    `gorm:"-" json:"team_name"`
	MemberCount        int       `gorm:"-" json:"member_count"`
	BookMemberId       int       `gorm:"-" json:"book_member_id"`
	BookMemberName     string    `gorm:"-" json:"book_member_name"`
	BookName           string    `gorm:"-" json:"book_name"`
}

// TableName 获取对应数据库表名.
func (m *TeamRelationship) TableName() string {
	return conf.GetDatabasePrefix() + "team_relationship"
}

func NewTeamRelationship() *TeamRelationship {
	return &TeamRelationship{}
}

func (m *TeamRelationship) First(teamId int, cols ...string) (*TeamRelationship, error) {
	if teamId <= 0 {
		return nil, ErrInvalidParameter
	}
	query := DB.Table(m.TableName()).Where("team_id = ?", teamId)
	if len(cols) > 0 {
		query = query.Select(strings.Join(cols, ","))
	}
	err := query.First(m).Error
	if err != nil {
		logger.Error("查询项目团队失败 ->", err)
	}
	return m, err
}

//查找指定项目的指定团队.
func (m *TeamRelationship) FindByBookId(bookId int, teamId int) (*TeamRelationship, error) {
	if teamId <= 0 || bookId <= 0 {
		return nil, ErrInvalidParameter
	}
	err := DB.Table(m.TableName()).Where("team_id = ? AND book_id = ?", teamId, bookId).First(m).Error
	if err != nil {
		logger.Error("查询项目团队失败 ->", err)
	}
	return m, err
}

//删除指定项目的指定团队.
func (m *TeamRelationship) DeleteByBookId(bookId int, teamId int) error {
	err := DB.Table(m.TableName()).Where("team_id = ? AND book_id = ?", teamId, bookId).First(m).Error
	if err != nil {
		logger.Error("查询项目团队失败 ->", err)
		return err
	}
	m.Include()
	return m.Delete(m.TeamRelationshipId)
}

//保存团队项目.
func (m *TeamRelationship) Save(cols ...string) (err error) {
	if m.TeamId <= 0 || m.BookId <= 0 {
		return ErrInvalidParameter
	}

	var count int64
	if m.TeamRelationshipId > 0 {
		DB.Table(m.TableName()).Where("book_id = ? AND team_id = ? AND team_relationship_id != ?", m.BookId, m.TeamId, m.TeamRelationshipId).Count(&count)
	} else {
		DB.Table(m.TableName()).Where("book_id = ? AND team_id = ?", m.BookId, m.TeamId).Count(&count)
	}
	if count > 0 {
		return errors.New("当前团队已加入该项目")
	}

	if m.TeamRelationshipId > 0 {
		err = DB.Save(m).Error
	} else {
		err = DB.Create(m).Error
	}
	if err != nil {
		logger.Error("保存团队项目时出错 ->", err)
	}
	return
}

func (m *TeamRelationship) Delete(teamRelId int) (err error) {
	if teamRelId <= 0 {
		return ErrInvalidParameter
	}
	err = DB.Table(m.TableName()).Where("team_relationship_id = ?", teamRelId).Delete(nil).Error

	if err != nil {
		logger.Error("删除团队项目失败 ->", err)
	}
	return
}

//分页查询团队项目.
func (m *TeamRelationship) FindToPager(teamId, pageIndex, pageSize int) (list []*TeamRelationship, totalCount int, err error) {
	if teamId <= 0 {
		err = ErrInvalidParameter
		return
	}
	offset := (pageIndex - 1) * pageSize

	err = DB.Table(m.TableName()).Where("team_id = ?", teamId).Order("team_relationship_id desc").Offset(offset).Limit(pageSize).Find(&list).Error

	if err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return
	}
	var count int64
	if err = DB.Table(m.TableName()).Where("team_id = ?", teamId).Count(&count).Error; err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return
	}
	totalCount = int(count)
	for _, item := range list {
		item.Include()
	}
	return
}

//加载附加数据.
func (m *TeamRelationship) Include() (*TeamRelationship, error) {
	if m.BookId > 0 {
		b, err := NewBook().Find(m.BookId, "book_name", "identify", "member_id")
		if err != nil {
			return m, err
		}
		m.BookName = b.BookName
		m.BookMemberId = b.MemberId
		if b.MemberId > 0 {
			member, err := NewMember().Find(b.MemberId, "account", "real_name")
			if err != nil {
				return m, err
			}
			if member.RealName == "" {
				m.BookMemberName = member.Account
			} else {
				m.BookMemberName = member.RealName
			}
		}
	}
	if m.TeamId > 0 {
		team, err := NewTeam().First(m.TeamId)
		if err == nil {
			m.TeamName = team.TeamName
			m.MemberCount = team.MemberCount
		}
	}
	return m, nil
}

//查询未加入团队的项目.
func (m *TeamRelationship) FindNotJoinBookByName(teamId int, bookName string, limit int) (*SelectMemberResult, error) {
	if teamId <= 0 {
		return nil, ErrInvalidParameter
	}

	sql := `select book.book_id,book.book_name
from  md_books as book
where book.book_id not in (select team.book_id from md_team_relationship as team where team_id=?)
and book.book_name like ? order by book_id desc limit ?;`

	books := make([]*Book, 0)

	err := DB.Raw(sql, teamId, "%"+bookName+"%", limit).Scan(&books).Error

	if err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return nil, err
	}

	result := SelectMemberResult{}
	items := make([]KeyValueItem, 0)

	for _, book := range books {
		item := KeyValueItem{}
		item.Id = book.BookId
		item.Text = book.BookName
		items = append(items, item)
	}
	result.Result = items

	return &result, err
}

//查找指定项目中未加入的团队.
func (m *TeamRelationship) FindNotJoinBookByBookIdentify(bookId int, teamName string, limit int) (*SelectMemberResult, error) {
	if bookId <= 0 || teamName == "" {
		return nil, ErrInvalidParameter
	}

	sql := `select *
from md_teams as team
where team.team_id not in (select rel.team_id from md_team_relationship as rel where rel.book_id = ?)
and team.team_name like ?
order by team.team_id desc limit ?;`
	teams := make([]*Team, 0)

	err := DB.Raw(sql, bookId, "%"+teamName+"%", limit).Scan(&teams).Error

	if err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return nil, err
	}

	result := SelectMemberResult{}
	items := make([]KeyValueItem, 0)

	for _, team := range teams {
		item := KeyValueItem{}
		item.Id = team.TeamId
		item.Text = team.TeamName
		items = append(items, item)
	}
	result.Result = items

	return &result, err
}

//查询指定项目的团队.
func (m *TeamRelationship) FindByBookToPager(bookId, pageIndex, pageSize int) (list []*TeamRelationship, totalCount int, err error) {

	if bookId <= 0 {
		err = ErrInvalidParameter
		return
	}

	offset := (pageIndex - 1) * pageSize

	err = DB.Table(m.TableName()).Where("book_id = ?", bookId).Order("team_relationship_id desc").Offset(offset).Limit(pageSize).Find(&list).Error

	if err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return
	}
	var count int64
	if err = DB.Table(m.TableName()).Where("book_id = ?", bookId).Count(&count).Error; err != nil {
		logger.Error("查询团队项目时出错 ->", err)
		return
	}
	totalCount = int(count)
	for _, item := range list {
		item.Include()
	}
	return
}
