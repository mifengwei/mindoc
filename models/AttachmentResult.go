package models

import (
	"strings"

	"github.com/mindoc-org/mindoc/utils/filetil"
)

type AttachmentResult struct {
	Attachment
	IsExist       bool
	BookName      string
	DocumentName  string
	FileShortSize string
	Account       string
	LocalHttpPath string
}

func NewAttachmentResult() *AttachmentResult {
	return &AttachmentResult{IsExist: false}
}

func (m *AttachmentResult) Find(id int) (*AttachmentResult, error) {

	attach := NewAttachment()

	err := DB.Table(NewAttachment().TableName()).Where("attachment_id = ?", id).First(attach).Error

	if err != nil {
		return m, err
	}

	m.Attachment = *attach

	if attach.BookId == 0 && attach.DocumentId > 0 {
		blog := NewBlog()
		if err := DB.Table(NewBlog().TableName()).Where("blog_id = ?", attach.DocumentId).Select("blog_title").First(blog).Error; err == nil {
			m.BookName = blog.BlogTitle
		} else {
			m.BookName = "[文章不存在]"
		}
	} else {
		book := NewBook()

		if e := DB.Table(NewBook().TableName()).Where("book_id = ?", attach.BookId).Select("book_name").First(book).Error; e == nil {
			m.BookName = book.BookName
		} else {
			m.BookName = "[不存在]"
		}
		doc := NewDocument()

		if e := DB.Table(NewDocument().TableName()).Where("document_id = ?", attach.DocumentId).Select("document_name").First(doc).Error; e == nil {
			m.DocumentName = doc.DocumentName
		} else {
			m.DocumentName = "[不存在]"
		}
	}
	if attach.CreateAt > 0 {
		member := NewMember()
		if e := DB.Table(NewMember().TableName()).Where("member_id = ?", attach.CreateAt).Select("account").First(member).Error; e == nil {
			m.Account = member.Account
		}
	}

	m.FileShortSize = filetil.FormatBytes(int64(attach.FileSize))
	m.LocalHttpPath = strings.Replace(m.FilePath, "\\", "/", -1)

	return m, nil
}
