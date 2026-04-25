package models

import "github.com/mindoc-org/mindoc/conf"

type Dashboard struct {
	BookNumber       int64 `json:"book_number"`
	DocumentNumber   int64 `json:"document_number"`
	MemberNumber     int64 `json:"member_number"`
	CommentNumber    int64 `json:"comment_number"`
	AttachmentNumber int64 `json:"attachment_number"`
}

func NewDashboard() *Dashboard {
	return &Dashboard{}
}

func (m *Dashboard) Query() *Dashboard {
	DB.Table(conf.GetDatabasePrefix() + "books").Count(&m.BookNumber)
	DB.Table(conf.GetDatabasePrefix() + "documents").Count(&m.DocumentNumber)
	DB.Table(conf.GetDatabasePrefix() + "members").Count(&m.MemberNumber)

	//comment_number
	m.CommentNumber = 0

	DB.Table(conf.GetDatabasePrefix() + "attachment").Count(&m.AttachmentNumber)

	return m
}
