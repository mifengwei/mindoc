package models

import (
	"time"

	"github.com/mindoc-org/mindoc/conf"
)

type CommentVote struct {
	VoteId          int       `gorm:"column:vote_id;primaryKey;autoIncrement;unique" json:"vote_id"`
	CommentId       int       `gorm:"column:comment_id;type:int;index" json:"comment_id"`
	CommentMemberId int       `gorm:"column:comment_member_id;type:int;index;default:0" json:"comment_member_id"`
	VoteMemberId    int       `gorm:"column:vote_member_id;type:int;index" json:"vote_member_id"`
	VoteState       int       `gorm:"column:vote_state;type:int" json:"vote_state"`
	CreateTime      time.Time `gorm:"column:create_time;type:datetime;autoCreateTime" json:"create_time"`
}

// TableName 获取对应数据库表名.
func (m *CommentVote) TableName() string {
	return conf.GetDatabasePrefix() + "comment_votes"
}

// TableEngine 获取数据使用的引擎.
func (m *CommentVote) TableEngine() string {
	return "INNODB"
}

func NewCommentVote() *CommentVote {
	return &CommentVote{}
}
func (m *CommentVote) InsertOrUpdate() (*CommentVote, error) {
	if m.VoteId > 0 {
		err := DB.Save(m).Error
		return m, err
	} else {
		err := DB.Create(m).Error
		return m, err
	}
}
