package models

import (
	"errors"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"gorm.io/gorm"
)

type Relationship struct {
	RelationshipId int `gorm:"primaryKey;autoIncrement;uniqueIndex;column:relationship_id" json:"relationship_id"`
	MemberId       int `gorm:"column:member_id;type:int;description:作者id" json:"member_id"`
	BookId         int `gorm:"column:book_id;type:int;description:所属项目id" json:"book_id"`
	// RoleId 角色：0 创始人(创始人不能被移除) / 1 管理员/2 编辑者/3 观察者
	RoleId conf.BookRole `gorm:"column:role_id;type:int;description:角色-配置文件里写死：0 创始人-不能被移除 / 1 管理员/2 编辑者/3 观察者" json:"role_id"`
}

// TableName 获取对应数据库表名. 用户和项目的关联表
func (m *Relationship) TableName() string {
	return conf.GetDatabasePrefix() + "relationship"
}

func NewRelationship() *Relationship {
	return &Relationship{}
}

func (m *Relationship) Find(id int) (*Relationship, error) {
	err := DB.Table(m.TableName()).Where("relationship_id = ?", id).First(m).Error
	return m, err
}

//查询指定项目的创始人.
func (m *Relationship) FindFounder(book_id int) (*Relationship, error) {
	err := DB.Table(m.TableName()).Where("book_id = ?", book_id).Where("role_id = ?", 0).First(m).Error

	return m, err
}

func (m *Relationship) UpdateRoleId(bookId, memberId int, roleId conf.BookRole) (*Relationship, error) {
	book := NewBook()
	book.BookId = bookId

	if err := DB.First(book, bookId).Error; err != nil {
		logger.Error("UpdateRoleId => ", err)
		return m, errors.New("项目不存在")
	}
	err := DB.Table(m.TableName()).Where("member_id = ?", memberId).Where("book_id = ?", bookId).First(m).Error

	if err == gorm.ErrRecordNotFound {
		m = NewRelationship()
		m.BookId = bookId
		m.MemberId = memberId
		m.RoleId = roleId
	} else if err != nil {
		return m, err
	} else if m.RoleId == conf.BookFounder {
		return m, errors.New("不能变更创始人的权限")
	}
	m.RoleId = roleId

	if m.RelationshipId > 0 {
		err = DB.Save(m).Error
	} else {
		err = DB.Create(m).Error
	}

	return m, err

}

func (m *Relationship) FindForRoleId(bookId, memberId int) (conf.BookRole, error) {
	relationship := NewRelationship()

	err := DB.Table(m.TableName()).Where("book_id = ?", bookId).Where("member_id = ?", memberId).First(relationship).Error

	if err != nil {
		return conf.BookRoleNoSpecific, err
	}
	return relationship.RoleId, nil
}

func (m *Relationship) FindByBookIdAndMemberId(book_id, member_id int) (*Relationship, error) {
	err := DB.Table(m.TableName()).Where("book_id = ?", book_id).Where("member_id = ?", member_id).First(m).Error

	return m, err
}

func (m *Relationship) Insert() error {
	return DB.Create(m).Error
}

func (m *Relationship) Update(tx *gorm.DB) error {
	err := tx.Save(m).Error
	if err != nil {
		tx.Rollback()
	}
	return err
}

func (m *Relationship) DeleteByBookIdAndMemberId(book_id, member_id int) error {
	err := DB.Table(m.TableName()).Where("book_id = ?", book_id).Where("member_id = ?", member_id).First(m).Error

	if err == gorm.ErrRecordNotFound {
		return errors.New("用户未参与该项目")
	}
	if err != nil {
		return err
	}
	if m.RoleId == conf.BookFounder {
		return errors.New("不能删除创始人")
	}
	err = DB.Delete(m).Error

	if err != nil {
		logger.Error("删除项目参与者 => ", err)
		return errors.New("删除失败")
	}
	return nil

}

func (m *Relationship) Transfer(book_id, founder_id, receive_id int) error {
	founder := NewRelationship()

	err := DB.Table(m.TableName()).Where("book_id = ?", book_id).Where("member_id = ?", founder_id).First(founder).Error

	if err != nil {
		return err
	}
	if founder.RoleId != conf.BookFounder {
		return errors.New("转让者不是创始人")
	}
	receive := NewRelationship()

	err = DB.Table(m.TableName()).Where("book_id = ?", book_id).Where("member_id = ?", receive_id).First(receive).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	tx := DB.Begin()

	founder.RoleId = conf.BookAdmin

	receive.MemberId = receive_id
	receive.RoleId = conf.BookFounder
	receive.BookId = book_id

	if err := founder.Update(tx); err != nil {
		return err
	}
	if receive.RelationshipId > 0 {
		if err := tx.Save(receive).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err := tx.Create(receive).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}
