package models

type CommentResult struct {
	Comment
	Author       string `json:"author"`
	ReplyAccount string `json:"reply_account"`
}

func (m *CommentResult) FindForDocumentToPager(doc_id, page_index, page_size int) (comments []*CommentResult, totalCount int, err error) {

	sql1 := `
SELECT
  comment.* ,
  parent.* ,
  mdmb.account AS author,
  p_member.account AS reply_account
FROM md_comments AS comment
  LEFT JOIN md_members AS mdmb ON comment.member_id = mdmb.member_id
  LEFT JOIN md_comments AS parent ON comment.parent_id = parent.comment_id
  LEFT JOIN md_members AS p_member ON p_member.member_id = parent.member_id

WHERE comment.document_id = ? ORDER BY comment.comment_id DESC LIMIT ? OFFSET ?`

	offset := (page_index - 1) * page_size

	err = DB.Raw(sql1, doc_id, page_size, offset).Scan(&comments).Error

	if err != nil {
		return
	}

	var count int64
	err = DB.Table(NewComment().TableName()).Where("document_id = ?", doc_id).Count(&count).Error

	if err == nil {
		totalCount = int(count)
	}

	return
}
