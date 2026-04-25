package models

import (
	"errors"
	"strings"

	"github.com/beego/i18n"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"gorm.io/gorm"
)

type TeamMember struct {
	TeamMemberId int `gorm:"primaryKey;autoIncrement;column:team_member_id;uniqueIndex" json:"team_member_id"`
	TeamId       int `gorm:"type:int;column:team_id;comment:团队id" json:"team_id"`
	MemberId     int `gorm:"type:int;column:member_id;comment:成员id" json:"member_id"`
	// RoleId 角色：0 创始人(创始人不能被移除) / 1 管理员/2 编辑者/3 观察者
	RoleId   conf.BookRole `gorm:"type:int;column:role_id;comment:RoleId 角色：0 创始人-创始人不能被移除 / 1 管理员/2 编辑者/3 观察者" json:"role_id"`
	RoleName string        `gorm:"-" json:"role_name"`
	Account  string        `gorm:"-" json:"account"`
	RealName string        `gorm:"-" json:"real_name"`
	Avatar   string        `gorm:"-" json:"avatar"`
	Lang     string        `gorm:"-"`
}

// TableName 获取对应数据库表名.
func (m *TeamMember) TableName() string {
	return conf.GetDatabasePrefix() + "team_member"
}

func NewTeamMember() *TeamMember {
	return &TeamMember{}
}

func (m *TeamMember) SetLang(lang string) *TeamMember {
	m.Lang = lang
	return m
}

func (m *TeamMember) First(id int, cols ...string) (*TeamMember, error) {
	if id <= 0 {
		return nil, errors.New("参数错误")
	}

	query := DB.Table(m.TableName()).Where("team_member_id = ?", id)
	if len(cols) > 0 {
		query = query.Select(strings.Join(cols, ","))
	}
	err := query.First(m).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Error("查询团队成员错误 ->", err)
	}

	return m.Include(), err
}

func (m *TeamMember) ChangeRoleId(teamId int, memberId int, roleId conf.BookRole) (member *TeamMember, err error) {

	if teamId <= 0 || memberId <= 0 || roleId <= 0 || roleId > conf.BookObserver {
		return nil, ErrInvalidParameter
	}

	err = DB.Table(m.TableName()).Where("team_id = ?", teamId).Where("member_id = ?", memberId).Order("team_member_id desc").First(m).Error

	if err != nil {
		logger.Error("查询团队用户时失败 ->", err)
		return m, err
	}
	m.RoleId = roleId

	err = m.Save("role_id")

	if err == nil {
		m.Include()
	}
	return m, err
}

//查询团队中指定的用户.
func (m *TeamMember) FindFirst(teamId, memberId int) (*TeamMember, error) {
	if teamId <= 0 || memberId <= 0 {
		return nil, ErrInvalidParameter
	}
	err := DB.Table(m.TableName()).Where("team_id = ?", teamId).Where("member_id = ?", memberId).First(m).Error

	if err != nil {
		logger.Error("查询团队用户失败 ->", err)
		return nil, err
	}
	return m.Include(), nil
}

//更新或插入团队用户.
func (m *TeamMember) Save(cols ...string) (err error) {

	if m.TeamId <= 0 {
		return errors.New("团队不能为空")
	}
	if m.MemberId <= 0 {
		return errors.New("用户不能为空")
	}

	var count int64
	DB.Table(NewTeam().TableName()).Where("team_id = ?", m.TeamId).Count(&count)
	if count == 0 {
		return errors.New("团队不存在")
	}
	DB.Table(NewMember().TableName()).Where("member_id = ? AND status = 0", m.MemberId).Count(&count)
	if count == 0 {
		return errors.New("用户不存在或已禁用")
	}

	if m.TeamMemberId <= 0 {
		DB.Table(m.TableName()).Where("team_id = ? AND member_id = ?", m.TeamId, m.MemberId).Count(&count)
		if count > 0 {
			return errors.New("团队中已存在该用户")
		}
		err = DB.Create(m).Error
	} else {
		err = DB.Save(m).Error
	}
	if err != nil {
		logger.Error("在保存团队时出错 ->", err)
	}
	return
}

//删除一个团队用户.
func (m *TeamMember) Delete(id int) (err error) {

	if id <= 0 {
		return ErrInvalidParameter
	}
	err = DB.Table(m.TableName()).Where("team_member_id = ?", id).Delete(nil).Error

	if err != nil {
		logger.Error("删除团队用户时出错 ->", err)
	}
	return
}

//分页查询团队用户.
func (m *TeamMember) FindToPager(teamId, pageIndex, pageSize int) (list []*TeamMember, totalCount int, err error) {
	if teamId <= 0 {
		err = ErrInvalidParameter
		return
	}
	offset := (pageIndex - 1) * pageSize

	err = DB.Table(m.TableName()).Where("team_id = ?", teamId).Offset(offset).Limit(pageSize).Find(&list).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logger.Error("查询团队成员失败 ->", err)
		}
		return
	}
	var c int64
	if err = DB.Table(m.TableName()).Where("team_id = ?", teamId).Count(&c).Error; err != nil {
		return
	}
	totalCount = int(c)

	//将来优化
	for _, item := range list {
		item.Lang = m.Lang
		item.Include()
	}
	return
}

//查询关联数据.
func (m *TeamMember) Include() *TeamMember {

	if member, err := NewMember().Find(m.MemberId, "account", "real_name", "avatar"); err == nil {
		m.Account = member.Account
		m.RealName = member.RealName
		m.Avatar = member.Avatar
	}
	if m.RoleId == 0 {
		m.RoleName = i18n.Tr(m.Lang, "common.creator") //"创始人"
	} else if m.RoleId == 1 {
		m.RoleName = i18n.Tr(m.Lang, "common.administrator") //"管理员"
	} else if m.RoleId == 2 {
		m.RoleName = i18n.Tr(m.Lang, "common.editor") //"编辑者"
	} else if m.RoleId == 3 {
		m.RoleName = i18n.Tr(m.Lang, "common.observer") //"观察者"
	}
	return m
}

//查询未加入团队的用户。
func (m *TeamMember) FindNotJoinMemberByAccount(teamId int, account string, limit int) (*SelectMemberResult, error) {
	if teamId <= 0 {
		return nil, ErrInvalidParameter
	}

	sql := `select mdmb.member_id,mdmb.account,mdmb.real_name,team.team_member_id
from md_members as mdmb
  left join md_team_member as team on team.team_id = ? and mdmb.member_id = team.member_id
  where mdmb.account like ? or mdmb.real_name like ? AND team_member_id IS NULL
  order by mdmb.member_id desc
limit ?;`

	members := make([]*Member, 0)

	err := DB.Raw(sql, teamId, "%"+account+"%", "%"+account+"%", limit).Scan(&members).Error

	if err != nil {
		logger.Error("查询团队用户时出错 ->", err)
		return nil, err
	}

	result := SelectMemberResult{}
	items := make([]KeyValueItem, 0)

	for _, member := range members {
		item := KeyValueItem{}
		item.Id = member.MemberId
		item.Text = member.Account + "[" + member.RealName + "]"
		items = append(items, item)
	}
	result.Result = items

	return &result, err
}

func (m *TeamMember) FindByBookIdAndMemberId(bookId, memberId int) (*TeamMember, error) {
	if bookId <= 0 || memberId <= 0 {
		return nil, ErrInvalidParameter
	}
	//一个用户可能在多个团队中，且一个项目可能有多个团队参与。因此需要查询用户最大权限。
	sql := `select *
from md_team_member as team
where team.team_id in (select rel.team_id from md_team_relationship as rel where rel.book_id = ?)
and team.member_id = ? order by team.role_id asc limit 1;`

	err := DB.Raw(sql, bookId, memberId).Scan(m).Error

	if err != nil {
		logger.Error("查询用户项目所在团队失败 ->bookId=", bookId, " memberId=", memberId, err)
		return nil, err
	}
	return m, nil
}
