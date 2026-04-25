package models

import (
	"strings"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"gorm.io/gorm"
)

type Label struct {
	LabelId    int    `gorm:"column:label_id;primaryKey;autoIncrement;uniqueIndex;description:项目标签id" json:"label_id"`
	LabelName  string `gorm:"column:label_name;size:50;uniqueIndex;description:项目标签名称" json:"label_name"`
	BookNumber int    `gorm:"column:book_number;description:包涵项目数量" json:"book_number"`
}

// TableName 获取对应数据库表名.
func (m *Label) TableName() string {
	return conf.GetDatabasePrefix() + "label"
}

func NewLabel() *Label {
	return &Label{}
}

func (m *Label) FindFirst(field string, value interface{}) (*Label, error) {
	err := DB.Table(m.TableName()).Where(field+" = ?", value).First(m).Error

	return m, err
}

//插入或更新标签.
func (m *Label) InsertOrUpdate(labelName string) error {
	err := DB.Table(m.TableName()).Where("label_name = ?", labelName).First(m).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	var count int64
	DB.Table(NewBook().TableName()).Where("label LIKE ?", "%"+labelName+"%").Count(&count)
	m.BookNumber = int(count)
	m.LabelName = labelName

	if err == gorm.ErrRecordNotFound {
		err = nil
		m.LabelName = labelName
		err = DB.Create(m).Error
	} else {
		err = DB.Save(m).Error
	}
	return err
}

//批量插入或更新标签.
func (m *Label) InsertOrUpdateMulti(labels string) {
	if labels != "" {
		labelArray := strings.Split(labels, ",")

		for _, label := range labelArray {
			if label != "" {
				NewLabel().InsertOrUpdate(label)
			}
		}
	}
}

//删除标签
func (m *Label) Delete() error {
	err := DB.Exec("DELETE FROM "+m.TableName()+" WHERE label_id = ?", m.LabelId).Error

	if err != nil {
		return err
	}
	return nil
}

//分页查找标签.
func (m *Label) FindToPager(pageIndex, pageSize int) (labels []*Label, totalCount int, err error) {

	var count int64
	err = DB.Table(m.TableName()).Count(&count).Error

	if err != nil {
		return
	}
	totalCount = int(count)

	offset := (pageIndex - 1) * pageSize

	err = DB.Table(m.TableName()).Order("book_number DESC").Offset(offset).Limit(pageSize).Find(&labels).Error

	if err == gorm.ErrRecordNotFound {
		logger.Info("没有查询到标签 ->", err)
		err = nil
		return
	}
	return
}
