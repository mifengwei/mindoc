// 数据库模型.
package models

import (
	"time"

	"os"

	"strings"

	"gorm.io/gorm"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/utils/filetil"
)

// Attachment struct .
type Attachment struct {
	AttachmentId int       `gorm:"primaryKey;autoIncrement;column:attachment_id" json:"attachment_id"`
	BookId       int       `gorm:"column:book_id;type:int" json:"book_id"`
	DocumentId   int       `gorm:"column:document_id;type:int" json:"doc_id"`
	FileName     string    `gorm:"column:file_name;size:255" json:"file_name"`
	FilePath     string    `gorm:"column:file_path;size:2000" json:"file_path"`
	FileSize     float64   `gorm:"column:file_size;type:float" json:"file_size"`
	HttpPath     string    `gorm:"column:http_path;size:2000" json:"http_path"`
	FileExt      string    `gorm:"column:file_ext;size:50" json:"file_ext"`
	CreateTime   time.Time `gorm:"type:datetime;column:create_time;autoCreateTime" json:"create_time"`
	CreateAt     int       `gorm:"column:create_at;type:int" json:"create_at"`
	ResourceType string    `gorm:"-" json:"resource_type"`
}

// TableName 获取对应上传附件数据库表名.
func (m *Attachment) TableName() string {
	return conf.GetDatabasePrefix() + "attachment"
}

// TableNameWithPrefix 获取带前缀的表名.
func (m *Attachment) TableNameWithPrefix() string {
	return m.TableName()
}

func NewAttachment() *Attachment {
	return &Attachment{}
}

func (m *Attachment) Insert() error {
	return DB.Create(m).Error
}

func (m *Attachment) Update() error {
	return DB.Save(m).Error
}

func (m *Attachment) Delete() error {
	err := DB.Delete(m).Error

	if err == nil {
		if err1 := os.Remove(m.FilePath); err1 != nil {
			logger.Error(err1)
		}
	}

	return err
}

func (m *Attachment) Find(id int) (*Attachment, error) {
	if id <= 0 {
		return m, ErrInvalidParameter
	}

	err := DB.Table(m.TableName()).Where("attachment_id = ?", id).First(m).Error

	return m, err
}

// 查询指定文档的附件列表
func (m *Attachment) FindListByDocumentId(docId int) (attaches []*Attachment, err error) {
	err = DB.Table(m.TableName()).Where("document_id = ? AND book_id > ?", docId, 0).Order("attachment_id DESC").Find(&attaches).Error
	return
}

// 分页查询附件
func (m *Attachment) FindToPager(pageIndex, pageSize int) (attachList []*AttachmentResult, totalCount int, err error) {
	var c int64
	err = DB.Table(m.TableName()).Count(&c).Error

	if err != nil {
		return nil, 0, err
	}
	totalCount = int(c)

	var list []*Attachment

	offset := (pageIndex - 1) * pageSize
	if pageSize == 0 {
		err = DB.Table(m.TableName()).Order("attachment_id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	} else {
		err = DB.Table(m.TableName()).Order("attachment_id DESC").Find(&list).Error
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Info("没有查到附件 ->", err)
			err = nil
		}
		return
	}

	for _, item := range list {
		attach := &AttachmentResult{}
		attach.Attachment = *item
		attach.FileShortSize = filetil.FormatBytes(int64(attach.FileSize))
		//当项目ID为0标识是文章的附件
		if item.BookId == 0 && item.DocumentId > 0 {
			blog := NewBlog()
			if err := DB.Table(blog.TableName()).Select("blog_title").Where("blog_id = ?", item.DocumentId).First(blog).Error; err == nil {
				attach.BookName = blog.BlogTitle
			} else {
				attach.BookName = "[文章不存在]"
			}
		} else {
			book := NewBook()

			if e := DB.Table(book.TableName()).Select("book_name").Where("book_id = ?", item.BookId).First(book).Error; e == nil {
				attach.BookName = book.BookName

				doc := NewDocument()

				if e := DB.Table(doc.TableName()).Select("document_name").Where("document_id = ?", item.DocumentId).First(doc).Error; e == nil {
					attach.DocumentName = doc.DocumentName
				} else {
					attach.DocumentName = "[文档不存在]"
				}

			} else {
				attach.BookName = "[项目不存在]"
			}
		}
		attach.LocalHttpPath = strings.Replace(item.FilePath, "\\", "/", -1)

		attachList = append(attachList, attach)
	}

	return
}
