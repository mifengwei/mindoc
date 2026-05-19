package models

import (
	"errors"
	"time"

	"github.com/mindoc-org/mindoc/conf"
)

// Comment struct
type Comment struct {
	CommentId int `gorm:"primaryKey;autoIncrement;column:comment_id" json:"comment_id"`
	Floor     int `gorm:"column:floor;default:0" json:"floor"`
	BookId    int `gorm:"column:book_id;type:int" json:"book_id"`
	// DocumentId 评论所属的文档.
	DocumentId int `gorm:"column:document_id;type:int" json:"document_id"`
	// Author 评论作者.
	Author string `gorm:"column:author;size:100" json:"author"`
	//MemberId 评论用户ID.
	MemberId int `gorm:"column:member_id;type:int" json:"member_id"`
	// IPAddress 评论者的IP地址
	IPAddress string `gorm:"column:ip_address;size:100" json:"ip_address"`
	// 评论日期.
	CommentDate time.Time `gorm:"column:comment_date;autoCreateTime" json:"comment_date"`
	//Content 评论内容.
	Content string `gorm:"column:content;size:2000" json:"content"`
	// Approved 评论状态：0 待审核/1 已审核/2 垃圾评论/ 3 已删除
	Approved int `gorm:"column:approved;type:int" json:"approved"`
	// UserAgent 评论者浏览器内容
	UserAgent string `gorm:"column:user_agent;size:500" json:"user_agent"`
	// Parent 评论所属父级
	ParentId     int    `gorm:"column:parent_id;type:int;default:0" json:"parent_id"`
	AgreeCount   int    `gorm:"column:agree_count;type:int;default:0" json:"agree_count"`
	AgainstCount int    `gorm:"column:against_count;type:int;default:0" json:"against_count"`
	Index        int    `gorm:"-" json:"index"`
	ShowDel      int    `gorm:"-" json:"show_del"`
	Avatar       string `gorm:"-" json:"avatar"`
}

// TableName 获取对应数据库表名.
func (m *Comment) TableName() string {
	return conf.GetDatabasePrefix() + "comments"
}

func NewComment() *Comment {
	return &Comment{}
}

// 是否有权限删除
func (m *Comment) CanDelete(user_memberid int, user_bookrole conf.BookRole) bool {
	return user_memberid == m.MemberId || user_bookrole == conf.BookFounder || user_bookrole == conf.BookAdmin
}

// 根据文档id查询文档评论
func (m *Comment) QueryCommentByDocumentId(doc_id, page, pagesize int, member *Member) (comments []*Comment, count int64, ret_page int) {
	doc, err := NewDocument().Find(doc_id)
	if err != nil {
		return
	}

	var c int64
	DB.Table(m.TableName()).Where("document_id = ?", doc_id).Count(&c)
	count = c
	if -1 == page { // 请求最后一页
		var total int = int(count)
		if total%pagesize == 0 {
			page = total / pagesize
		} else {
			page = total/pagesize + 1
		}
	}
	offset := (page - 1) * pagesize
	ret_page = page
	DB.Table(m.TableName()).Where("document_id = ?", doc_id).Order("comment_date ASC").Offset(offset).Limit(pagesize).Find(&comments)

	// 需要判断未登录的情况
	var bookRole conf.BookRole
	if member != nil {
		bookRole, _ = NewRelationship().FindForRoleId(doc.BookId, member.MemberId)
	}
	for i := 0; i < len(comments); i++ {
		comments[i].Index = (i + 1) + (page-1)*pagesize
		if member != nil && comments[i].CanDelete(member.MemberId, bookRole) {
			comments[i].ShowDel = 1
			comments[i].Avatar = member.Avatar
		}
	}
	return
}

func (m *Comment) Update(cols ...string) error {
	if len(cols) > 0 {
		return DB.Model(m).Select(cols[0], strsToInterfaces(cols[1:])...).Updates(m).Error
	}
	return DB.Save(m).Error
}

// Insert 添加一条评论.
func (m *Comment) Insert() error {
	if m.DocumentId <= 0 {
		return errors.New("评论文档不存在")
	}
	if m.Content == "" {
		return ErrCommentContentNotEmpty
	}

	if m.CommentId > 0 {
		comment := NewComment()
		//如果父评论不存在
		if err := DB.First(comment, m.CommentId).Error; err != nil {
			return err
		}
	}

	document := NewDocument()
	//如果评论的文档不存在
	if _, err := document.Find(m.DocumentId); err != nil {
		return err
	}
	book, err := NewBook().Find(document.BookId)
	//如果评论的项目不存在
	if err != nil {
		return err
	}
	//如果已关闭评论
	if book.CommentStatus == "closed" {
		return ErrCommentClosed
	}
	if book.CommentStatus == "registered_only" && m.MemberId <= 0 {
		return ErrPermissionDenied
	}
	//如果仅参与者评论
	if book.CommentStatus == "group_only" {
		if m.MemberId <= 0 {
			return ErrPermissionDenied
		}
		rel := NewRelationship()
		if _, err := rel.FindForRoleId(book.BookId, m.MemberId); err != nil {
			return ErrPermissionDenied
		}
	}

	if m.MemberId > 0 {
		member := NewMember()
		//如果用户不存在
		if _, err := member.Find(m.MemberId); err != nil {
			return ErrMemberNoExist
		}
		//如果用户被禁用
		if member.Status == 1 {
			return ErrMemberDisabled
		}
	} else if m.Author == "" {
		m.Author = "[匿名用户]"
	}
	m.BookId = book.BookId
	err = DB.Create(m).Error

	return err
}

// 删除一条评论
func (m *Comment) Delete() error {
	return DB.Delete(m).Error
}

func (m *Comment) Find(id int, cols ...string) (*Comment, error) {
	query := DB.Table(m.TableName()).Where("comment_id = ?", id)
	if len(cols) > 0 {
		query = query.Select(cols[0], strsToInterfaces(cols[1:])...)
	}
	if err := query.First(m).Error; err != nil {
		return m, err
	}
	return m, nil
}
