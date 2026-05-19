package models

import "github.com/mindoc-org/mindoc/conf"

// Option struct .
type Option struct {
	OptionId    int    `gorm:"column:option_id;primaryKey;autoIncrement;uniqueIndex" json:"option_id"`
	OptionTitle string `gorm:"column:option_title;size:500" json:"option_title"`
	OptionName  string `gorm:"column:option_name;uniqueIndex;size:80" json:"option_name"`
	OptionValue string `gorm:"column:option_value;type:text" json:"option_value"`
	Remark      string `gorm:"column:remark;type:text" json:"remark"`
}

// TableName 获取对应数据库表名.
func (m *Option) TableName() string {
	return conf.GetDatabasePrefix() + "options"
}

func NewOption() *Option {
	return &Option{}
}

func (p *Option) Find(id int) (*Option, error) {
	if err := DB.First(p, id).Error; err != nil {
		return p, err
	}
	return p, nil
}

func (p *Option) FindByKey(key string) (*Option, error) {
	if err := DB.Table(p.TableName()).Where("option_name = ?", key).First(p).Error; err != nil {
		return p, err
	}
	return p, nil
}

func GetOptionValue(key, def string) string {

	if option, err := NewOption().FindByKey(key); err == nil {
		return option.OptionValue
	}
	return def
}

func (p *Option) InsertOrUpdate() error {

	var err error

	if p.OptionId > 0 {
		err = DB.Save(p).Error
	} else {
		var exist Option
		if DB.Table(p.TableName()).Where("option_name = ?", p.OptionName).First(&exist).Error == nil {
			p.OptionId = exist.OptionId
			err = DB.Save(p).Error
		} else {
			err = DB.Create(p).Error
		}
	}
	return err
}

func (p *Option) InsertMulti(option ...Option) error {
	return DB.Create(&option).Error
}

func (p *Option) All() ([]*Option, error) {
	var options []*Option

	err := DB.Table(p.TableName()).Find(&options).Error

	if err != nil {
		return options, err
	}
	return options, nil
}

func (m *Option) Init() error {

	var count int64
	if DB.Table(m.TableName()).Where("option_name = ?", "ENABLED_REGISTER").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "false"
		option.OptionName = "ENABLED_REGISTER"
		option.OptionTitle = "是否启用注册"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	if DB.Table(m.TableName()).Where("option_name = ?", "ENABLE_DOCUMENT_HISTORY").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "true"
		option.OptionName = "ENABLE_DOCUMENT_HISTORY"
		option.OptionTitle = "是否启用文档历史"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	if DB.Table(m.TableName()).Where("option_name = ?", "ENABLED_CAPTCHA").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "true"
		option.OptionName = "ENABLED_CAPTCHA"
		option.OptionTitle = "是否启用验证码"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	if DB.Table(m.TableName()).Where("option_name = ?", "ENABLE_ANONYMOUS").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "false"
		option.OptionName = "ENABLE_ANONYMOUS"
		option.OptionTitle = "启用匿名访问"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	if DB.Table(m.TableName()).Where("option_name = ?", "SITE_NAME").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "MinDoc文档管理系统"
		option.OptionName = "SITE_NAME"
		option.OptionTitle = "站点名称"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	if DB.Table(m.TableName()).Where("option_name = ?", "site_description").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "MinDoc 是一款针对IT团队开发的简单好用的文档管理系统，可以用来储存日常接口文档，数据库字典，手册说明等文档。内置项目管理，用户管理，权限管理等功能，支持Markdown和富文本两种编辑器，能够满足大部分中小团队的文档管理需求。"
		option.OptionName = "site_description"
		option.OptionTitle = "站点描述"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}

	if DB.Table(m.TableName()).Where("option_name = ?", "site_beian").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = ""
		option.OptionName = "site_beian"
		option.OptionTitle = "域名备案"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}

	if DB.Table(m.TableName()).Where("option_name = ?", "language").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "zh-cn"
		option.OptionName = "language"
		option.OptionTitle = "站点语言"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}

	return nil
}

func (m *Option) Update() error {

	var count int64
	if DB.Table(m.TableName()).Where("option_name = ?", "language").Count(&count); count == 0 {
		option := NewOption()
		option.OptionValue = "zh-cn"
		option.OptionName = "language"
		option.OptionTitle = "站点语言"
		if err := DB.Create(option).Error; err != nil {
			return err
		}
	}
	return nil
}
