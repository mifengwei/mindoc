package models

import (
	"errors"
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

type Template struct {
	TemplateId   int    `gorm:"column:template_id;primaryKey;autoIncrement;uniqueIndex" json:"template_id"`
	TemplateName string `gorm:"column:template_name;size:500" json:"template_name"`
	MemberId     int    `gorm:"column:member_id;index" json:"member_id"`
	BookId       int    `gorm:"column:book_id;index" json:"book_id"`
	BookName     string `gorm:"-" json:"book_name"`
	//是否是全局模板：0 否/1 是; 全局模板在所有项目中都可以使用；否则只能在创建模板的项目中使用
	IsGlobal        int       `gorm:"column:is_global;default:0" json:"is_global"`
	TemplateContent string    `gorm:"column:template_content;type:text" json:"template_content"`
	CreateTime      time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	CreateName      string    `gorm:"-" json:"create_name"`
	ModifyTime      time.Time `gorm:"column:modify_time;autoUpdateTime" json:"modify_time"`
	ModifyAt        int       `gorm:"column:modify_at;type:int" json:"-"`
	ModifyName      string    `gorm:"-" json:"modify_name"`
	Version         int64     `gorm:"type:bigint;column:version" json:"version"`
}

// TableName 获取对应数据库表名.
func (m *Template) TableName() string {
	return conf.GetDatabasePrefix() + "templates"
}

func NewTemplate() *Template {
	return &Template{}
}

//查询指定ID的模板
func (t *Template) Find(templateId int) (*Template, error) {
	if templateId <= 0 {
		return t, ErrInvalidParameter
	}

	err := DB.Table(t.TableName()).Where("template_id = ?", templateId).First(t).Error

	if err != nil {
		logger.Error("查询模板时失败 ->%s", err)
	}
	return t, err
}

//查询属于指定项目的模板.
func (t *Template) FindByBookId(bookId int) ([]*Template, error) {
	if bookId <= 0 {
		return nil, ErrInvalidParameter
	}

	var templateList []*Template

	err := DB.Table(t.TableName()).Where("book_id = ?", bookId).Order("template_id DESC").Find(&templateList).Error

	if err != nil {
		logger.Error("查询模板列表失败 ->", err)
	}
	return templateList, err
}

//查询指定项目所有可用模板列表.
func (t *Template) FindAllByBookId(bookId int) ([]*Template, error) {
	if bookId <= 0 {
		return nil, ErrInvalidParameter
	}

	var templateList []*Template

	err := DB.Table(t.TableName()).Where("book_id = ? OR is_global = ?", bookId, 1).Order("template_id DESC").Find(&templateList).Error

	if err != nil {
		logger.Error("查询模板列表失败 ->", err)
	}
	return templateList, err
}

//删除一个模板
func (t *Template) Delete(templateId int, memberId int) error {
	if templateId <= 0 {
		return ErrInvalidParameter
	}

	qs := DB.Table(t.TableName()).Where("template_id = ?", templateId)

	if memberId > 0 {
		qs = qs.Where("member_id = ?", memberId)
	}
	err := qs.Delete(&Template{}).Error

	if err != nil {
		logger.Error("删除模板失败 ->", err)
	}
	return err
}

//添加或更新模板
func (t *Template) Save(cols ...string) (err error) {

	if t.BookId <= 0 {
		return ErrInvalidParameter
	}

	var dummyBook Book
	if DB.Table(NewBook().TableName()).Where("book_id = ?", t.BookId).First(&dummyBook).Error != nil {
		return errors.New("项目不存在")
	}
	var dummyMember Member
	if DB.Table(NewMember().TableName()).Where("member_id = ? AND status = ?", t.MemberId, 0).First(&dummyMember).Error != nil {
		return errors.New("用户已被禁用")
	}
	t.Version = time.Now().Unix()

	if t.TemplateId > 0 {
		t.ModifyTime = time.Now()
		if len(cols) > 0 {
			err = DB.Model(t).Select(cols).Updates(t).Error
		} else {
			err = DB.Save(t).Error
		}
	} else {
		t.CreateTime = time.Now()
		err = DB.Create(t).Error
	}

	return
}

//预加载一些数据
func (t *Template) Preload() *Template {
	if t != nil {
		if t.MemberId > 0 {
			m, err := NewMember().Find(t.MemberId, "account", "real_name")
			if err == nil {
				if m.RealName != "" {
					t.CreateName = m.RealName
				} else {
					t.CreateName = m.Account
				}
			} else {
				logger.Error("加载模板所有者失败 ->", err)
			}
		}
		if t.ModifyAt > 0 {
			if m, err := NewMember().Find(t.ModifyAt, "account", "real_name"); err == nil {
				if m.RealName != "" {
					t.ModifyName = m.RealName
				} else {
					t.ModifyName = m.Account
				}
			}
		}
		if t.BookId > 0 {
			if b, err := NewBook().Find(t.BookId, "book_name"); err == nil {
				t.BookName = b.BookName
			}
		}
	}
	return t
}
