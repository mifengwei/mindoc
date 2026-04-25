package models

import (
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

type DocumentHistory struct {
	HistoryId    int       `gorm:"primaryKey;autoIncrement;uniqueIndex;column:history_id" json:"history_id"`
	Action       string    `gorm:"size:255;column:action;comment:modify" json:"action"`
	ActionName   string    `gorm:"size:255;column:action_name;comment:修改文档" json:"action_name"`
	DocumentId   int       `gorm:"type:int;index;column:document_id;comment:关联文档id" json:"doc_id"`
	DocumentName string    `gorm:"size:500;column:document_name;comment:关联文档id" json:"doc_name"`
	ParentId     int       `gorm:"type:int;index;column:parent_id;default:0;comment:父级文档id" json:"parent_id"`
	Markdown     string    `gorm:"type:text;column:markdown;comment:文档内容" json:"markdown"`
	Content      string    `gorm:"type:text;column:content;comment:文档内容" json:"content"`
	MemberId     int       `gorm:"type:int;column:member_id;comment:作者id" json:"member_id"`
	ModifyTime   time.Time `gorm:"type:datetime;autoUpdateTime;column:modify_time;comment:修改时间" json:"modify_time"`
	ModifyAt     int       `gorm:"type:int;column:modify_at;comment:修改人id" json:"-"`
	Version      int64     `gorm:"type:bigint;column:version;comment:版本" json:"version"`
	IsOpen       int       `gorm:"type:int;column:is_open;default:0;comment:是否展开子目录 0：阅读时关闭节点 1：阅读时展开节点 2：空目录 单击时会展开下级节点" json:"is_open"`
}

type DocumentHistorySimpleResult struct {
	HistoryId  int       `json:"history_id"`
	ActionName string    `json:"action_name"`
	MemberId   int       `json:"member_id"`
	Account    string    `json:"account"`
	ModifyAt   int       `json:"modify_at"`
	ModifyName string    `json:"modify_name"`
	ModifyTime time.Time `json:"modify_time"`
	Version    int64     `json:"version"`
}

// TableName 获取对应数据库表名.
func (m *DocumentHistory) TableName() string {
	return conf.GetDatabasePrefix() + "document_history"
}

func NewDocumentHistory() *DocumentHistory {
	return &DocumentHistory{}
}

func (m *DocumentHistory) Find(id int) (*DocumentHistory, error) {
	err := DB.Table(m.TableName()).Where("history_id = ?", id).First(m).Error

	return m, err
}

//清空指定文档的历史.
func (m *DocumentHistory) Clear(docId int) error {

	err := DB.Exec("DELETE FROM md_document_history WHERE document_id = ?", docId).Error

	return err
}

//删除历史.
func (m *DocumentHistory) Delete(historyId, docId int) error {

	err := DB.Table(m.TableName()).Where("history_id = ? AND document_id = ?", historyId, docId).Delete(nil).Error

	return err
}

//恢复指定历史的文档.
func (m *DocumentHistory) Restore(historyId, docId, uid int) error {

	err := DB.Table(m.TableName()).Where("history_id = ? AND document_id = ?", historyId, docId).First(m).Error

	if err != nil {
		return err
	}
	doc, err := NewDocument().Find(m.DocumentId)

	if err != nil {
		return err
	}
	history := NewDocumentHistory()
	history.DocumentId = docId
	history.Content = doc.Content
	history.Markdown = doc.Markdown
	history.DocumentName = doc.DocumentName
	history.ModifyAt = uid
	history.MemberId = doc.MemberId
	history.ParentId = doc.ParentId
	history.Version = time.Now().Unix()
	history.Action = "restore"
	history.ActionName = "恢复文档"
	history.IsOpen = doc.IsOpen

	history.InsertOrUpdate()

	doc.DocumentName = m.DocumentName
	doc.Content = m.Content
	doc.Markdown = m.Markdown
	doc.Release = m.Content
	doc.Version = time.Now().Unix()
	doc.IsOpen = m.IsOpen

	err = DB.Save(doc).Error

	return err
}

func (m *DocumentHistory) InsertOrUpdate() (history *DocumentHistory, err error) {
	history = m

	if m.HistoryId > 0 {
		err = DB.Save(m).Error
	} else {
		err = DB.Create(m).Error
		if err == nil {
			if doc, e := NewDocument().Find(m.DocumentId); e == nil {
				if book, e := NewBook().Find(doc.BookId); e == nil && book.HistoryCount > 0 {
					//如果已存在的历史记录大于指定的记录，则清除旧记录
					var c int64
					DB.Table(m.TableName()).Where("document_id = ?", doc.DocumentId).Count(&c)
					if c > int64(book.HistoryCount) {

						deleteCount := c - int64(book.HistoryCount)
						logger.Info("需要删除的历史文档数量：", deleteCount)
						var lists []DocumentHistory

						if e := DB.Table(m.TableName()).Where("document_id = ?", doc.DocumentId).Order("history_id").Limit(int(deleteCount)).Select("history_id").Find(&lists).Error; e == nil {
							for _, d := range lists {
								DB.Delete(&d)
							}
						}
					} else {
						logger.Info(book.HistoryCount)
					}
				}
			}

		}
	}
	return
}

//分页查询指定文档的历史.
func (m *DocumentHistory) FindToPager(docId, pageIndex, pageSize int) (docs []*DocumentHistorySimpleResult, totalCount int, err error) {

	offset := (pageIndex - 1) * pageSize

	totalCount = 0

	sql := `SELECT history.*,m1.account,m2.account as modify_name
FROM md_document_history AS history
LEFT JOIN md_members AS m1 ON history.member_id = m1.member_id
LEFT JOIN md_members AS m2 ON history.modify_at = m2.member_id
WHERE history.document_id = ? ORDER BY history.history_id DESC limit ? offset ?;`

	err = DB.Raw(sql, docId, pageSize, offset).Scan(&docs).Error

	if err != nil {
		return
	}
	var count int64
	err = DB.Table(m.TableName()).Where("document_id = ?", docId).Count(&count).Error
	if err != nil {
		return
	}
	totalCount = int(count)

	return
}
