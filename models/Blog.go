package models

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/cache"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/utils"
	"gorm.io/gorm"
)

// 博文表
type Blog struct {
	BlogId int `gorm:"primaryKey;autoIncrement;column:blog_id" json:"blog_id"`
	//文章标题
	BlogTitle string `gorm:"column:blog_title;size:500" json:"blog_title"`
	//文章标识
	BlogIdentify string `gorm:"column:blog_identify;size:100;uniqueIndex" json:"blog_identify"`
	//排序序号
	OrderIndex int `gorm:"column:order_index;type:int;default:0" json:"order_index"`
	//所属用户
	MemberId int `gorm:"column:member_id;type:int;default:0;index" json:"member_id"`
	//用户头像
	MemberAvatar string `gorm:"-" json:"member_avatar"`
	//文章类型:0 普通文章/1 链接文章
	BlogType int `gorm:"column:blog_type;type:int;default:0" json:"blog_type"`
	//链接到的项目中的文档ID
	DocumentId int `gorm:"column:document_id;type:int;default:0" json:"document_id"`
	//文章的标识
	DocumentIdentify string `gorm:"-" json:"document_identify"`
	//关联文档的项目标识
	BookIdentify string `gorm:"-" json:"book_identify"`
	//关联文档的项目ID
	BookId int `gorm:"-" json:"book_id"`
	//文章摘要
	BlogExcerpt string `gorm:"column:blog_excerpt;size:1500" json:"blog_excerpt"`
	//文章内容
	BlogContent string `gorm:"column:blog_content;type:text" json:"blog_content"`
	//发布后的文章内容
	BlogRelease string `gorm:"column:blog_release;type:text" json:"blog_release"`
	//文章当前的状态，枚举enum('publish','draft','password')值，publish为已 发表，draft为草稿，password 为私人内容(不会被公开) 。默认为publish。
	BlogStatus string `gorm:"column:blog_status;size:100;default:publish" json:"blog_status"`
	//文章密码，varchar(100)值。文章编辑才可为文章设定一个密码，凭这个密码才能对文章进行重新强加或修改。
	Password string `gorm:"column:password;size:100" json:"-"`
	//最后修改时间
	Modified time.Time `gorm:"column:modify_time;type:datetime;autoUpdateTime" json:"modify_time"`
	//修改人id
	ModifyAt       int    `gorm:"column:modify_at;type:int" json:"-"`
	ModifyRealName string `gorm:"-" json:"modify_real_name"`
	//创建时间
	Created    time.Time `gorm:"column:create_time;type:datetime;autoCreateTime" json:"create_time"`
	CreateName string    `gorm:"-" json:"create_name"`
	//版本号
	Version int64 `gorm:"type:bigint;column:version" json:"version"`
	//附件列表
	AttachList []*Attachment `gorm:"-" json:"attach_list"`
}

// TableName 获取对应数据库表名.
func (b *Blog) TableName() string {
	return conf.GetDatabasePrefix() + "blogs"
}

func NewBlog() *Blog {
	return &Blog{
		BlogStatus: "public",
	}
}

// 根据文章ID查询文章
func (b *Blog) Find(blogId int) (*Blog, error) {
	err := DB.Table(b.TableName()).Where("blog_id = ?", blogId).First(b).Error
	if err != nil {
		logger.Error("查询文章时失败 -> ", err)
		return nil, err
	}

	return b.Link()
}

// 从缓存中读取文章
func (b *Blog) FindFromCache(blogId int) (blog *Blog, err error) {
	key := fmt.Sprintf("blog-id-%d", blogId)
	var temp Blog
	err = cache.Get(key, &temp)
	if err == nil {
		b = &temp
		b.Link()
		logger.Debug("从缓存读取文章成功 ->", key)
		return b, nil
	} else {
		logger.Error("读取缓存失败 ->", err)
	}

	blog, err = b.Find(blogId)
	if err == nil {
		//默认一个小时
		if err := cache.Put(key, blog, time.Hour*1); err != nil {
			logger.Error("将文章存入缓存失败 ->", err)
		}
	}
	return
}

// 查找指定用户的指定文章
func (b *Blog) FindByIdAndMemberId(blogId, memberId int) (*Blog, error) {
	err := DB.Table(b.TableName()).Where("blog_id = ?", blogId).Where("member_id = ?", memberId).First(b).Error
	if err != nil {
		logger.Error("查询文章时失败 -> ", err)
		return nil, err
	}

	return b.Link()
}

// 根据文章标识查询文章
func (b *Blog) FindByIdentify(identify string) (*Blog, error) {
	err := DB.Table(b.TableName()).Where("blog_identify = ?", identify).First(b).Error
	if err != nil {
		logger.Error("查询文章时失败 -> ", err)
		return nil, err
	}
	return b, nil
}

// 获取指定文章的链接内容
func (b *Blog) Link() (*Blog, error) {
	//如果是链接文章，则需要从链接的项目中查找文章内容
	if b.BlogType == 1 && b.DocumentId > 0 {
		doc := NewDocument()
		if err := DB.Table(doc.TableName()).Where("document_id = ?", b.DocumentId).First(doc, "release", "markdown", "identify", "book_id").Error; err != nil {
			logger.Error("查询文章链接对象时出错 -> ", err)
		} else {
			b.DocumentIdentify = doc.Identify
			b.BlogRelease = doc.Release

			//目前仅支持markdown文档进行链接
			b.BlogContent = doc.Markdown
			book := NewBook()
			if err := DB.Table(book.TableName()).Where("book_id = ?", doc.BookId).First(book, "identify").Error; err != nil {
				logger.Error("查询关联文档的项目时出错 ->", err)
			} else {
				b.BookIdentify = book.Identify
				b.BookId = doc.BookId
			}
			//处理链接文档存在源文档修改时间的问题
			if content, err := goquery.NewDocumentFromReader(bytes.NewBufferString(b.BlogRelease)); err == nil {
				content.Find(".wiki-bottom").Remove()
				if html, err := content.Html(); err == nil {
					b.BlogRelease = html
				} else {
					logger.Error("处理文章失败 ->", err)
				}
			} else {
				logger.Error("处理文章失败 ->", err)
			}
		}
	}

	if b.ModifyAt > 0 {
		member := NewMember()
		if err := DB.Table(member.TableName()).Where("member_id = ?", b.ModifyAt).First(member, "real_name", "account").Error; err == nil {
			if member.RealName != "" {
				b.ModifyRealName = member.RealName
			} else {
				b.ModifyRealName = member.Account
			}
		}
	}
	if b.MemberId > 0 {
		member := NewMember()
		if err := DB.Table(member.TableName()).Where("member_id = ?", b.MemberId).First(member, "real_name", "account", "avatar").Error; err == nil {
			if member.RealName != "" {
				b.CreateName = member.RealName
			} else {
				b.CreateName = member.Account
			}
			b.MemberAvatar = member.Avatar
		}
	}

	return b, nil
}

// 判断指定的文章标识是否存在
func (b *Blog) IsExist(identify string) bool {
	var count int64
	DB.Table(b.TableName()).Where("blog_identify = ?", identify).Count(&count)
	return count > 0
}

// 保存文章
func (b *Blog) Save(cols ...string) error {
	if b.OrderIndex <= 0 {
		blog := NewBlog()
		if err := DB.Table(b.TableName()).Order("blog_id DESC").Limit(1).First(blog, "blog_id").Error; err == nil {
			b.OrderIndex = blog.BlogId + 1
		} else {
			var c int64
			DB.Table(b.TableName()).Count(&c)
			b.OrderIndex = int(c) + 1
		}
	}
	var err error

	b.Processor().Version = time.Now().Unix()

	if b.BlogId > 0 {
		b.Modified = time.Now()
		if len(cols) > 0 {
			err = DB.Select(cols).Save(b).Error
		} else {
			err = DB.Save(b).Error
		}
		key := fmt.Sprintf("blog-id-%d", b.BlogId)
		_ = cache.Delete(key)

	} else {

		b.Created = time.Now()
		err = DB.Create(b).Error
	}

	if err == nil && b.BlogId > 0 {
		// 刷新倒排索引
		go func(blogId int, blogTitle, blogRelease, blogContent string) {
			content := blogRelease
			if content == "" {
				content = blogContent
			}
			content = blogTitle + "\n" + content
			content = utils.StripTags(content)
			if err := BuildIndexForBlog(blogId, content); err != nil {
				logger.Error("构建Blog倒排索引失败 ->", blogId, err)
			}
		}(b.BlogId, b.BlogTitle, b.BlogRelease, b.BlogContent)
	}

	return err
}

// 过滤文章的危险标签，处理文章外链以及图片.
func (b *Blog) Processor() *Blog {

	b.BlogRelease = utils.SafetyProcessor(b.BlogRelease)

	//解析文档中非本站的链接，并设置为新窗口打开
	if content, err := goquery.NewDocumentFromReader(bytes.NewBufferString(b.BlogRelease)); err == nil {

		content.Find("a").Each(func(i int, contentSelection *goquery.Selection) {
			if src, ok := contentSelection.Attr("href"); ok {
				if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
					//logs.Info(src,conf.BaseUrl,strings.HasPrefix(src,conf.BaseUrl))
					if conf.BaseUrl != "" && !strings.HasPrefix(src, conf.BaseUrl) {
						contentSelection.SetAttr("target", "_blank")
						if html, err := content.Html(); err == nil {
							b.BlogRelease = html
						}
					}
				}

			}
		})
		//设置图片为CDN地址
		if cdnimg, _ := conf.GetString("cdnimg"); cdnimg != "" {
			content.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
				if src, ok := contentSelection.Attr("src"); ok && strings.HasPrefix(src, "/uploads/") {
					contentSelection.SetAttr("src", utils.JoinURI(cdnimg, src))
				}
			})
		}
	}

	return b
}

// 分页查询文章列表
func (b *Blog) FindToPager(pageIndex, pageSize int, memberId int, status string) (blogList []*Blog, totalCount int, err error) {

	offset := (pageIndex - 1) * pageSize

	query := DB.Table(b.TableName())

	if memberId > 0 {
		query = query.Where("member_id = ?", memberId)
	}
	if status != "" && status != "all" {
		query = query.Where("blog_status = ?", status)
	}

	if status == "" {
		query = query.Where("blog_status != ?", "private")
	}

	// Build count query separately since Find consumes the session
	countQuery := DB.Table(b.TableName())
	if memberId > 0 {
		countQuery = countQuery.Where("member_id = ?", memberId)
	}
	if status != "" && status != "all" {
		countQuery = countQuery.Where("blog_status = ?", status)
	}
	if status == "" {
		countQuery = countQuery.Where("blog_status != ?", "private")
	}
	var c int64
	if err := countQuery.Count(&c).Error; err != nil {
		logger.Error("获取文章数量时出错 ->", err)
		return nil, 0, err
	}
	totalCount = int(c)

	err = query.Order("order_index DESC, blog_id DESC").Offset(offset).Limit(pageSize).Find(&blogList).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			err = nil
		}
		logger.Error("获取文章列表时出错 ->", err)
		return
	}
	for _, blog := range blogList {
		if blog.BlogType == 1 {
			blog.Link()
		}
	}

	return
}

// 删除文章
func (b *Blog) Delete(blogId int) error {
	// 删除文章缓存
	key := fmt.Sprintf("blog-id-%d", blogId)
	_ = cache.Delete(key)

	// 删除博客的倒排索引
	index := NewContentReverseIndex()
	_ = index.DeleteByContentTypeAndContentId(2, blogId)

	err := DB.Table(b.TableName()).Where("blog_id = ?", blogId).Delete(nil).Error
	if err != nil {
		logger.Error("删除文章失败 ->", err)
	}
	return err
}

// 查询下一篇文章
func (b *Blog) QueryNext(blogId int) (*Blog, error) {
	blog := NewBlog()

	if err := DB.Table(b.TableName()).Where("blog_id = ?", blogId).First(blog, "order_index").Error; err != nil {
		logger.Error("查询文章时出错 ->", err)
		return b, err
	}

	err := DB.Table(b.TableName()).Where("order_index >= ?", blog.OrderIndex).Where("blog_id > ?", blogId).Order("order_index ASC, blog_id ASC").First(blog).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Error("查询文章时出错 ->", err)
	}
	return blog, err
}

// 查询下一篇文章
func (b *Blog) QueryPrevious(blogId int) (*Blog, error) {
	blog := NewBlog()

	if err := DB.Table(b.TableName()).Where("blog_id = ?", blogId).First(blog, "order_index").Error; err != nil {
		logger.Error("查询文章时出错 ->", err)
		return b, err
	}

	err := DB.Table(b.TableName()).Where("order_index <= ?", blog.OrderIndex).Where("blog_id < ?", blogId).Order("order_index DESC, blog_id DESC").First(blog).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		logger.Error("查询文章时出错 ->", err)
	}
	return blog, err
}

// 关联文章附件
func (b *Blog) LinkAttach() (err error) {

	var attachList []*Attachment
	//当不是关联文章时，用文章ID去查询附件
	if b.BlogType != 1 || b.DocumentId <= 0 {
		err = DB.Table(NewAttachment().TableName()).Where("document_id = ?", b.BlogId).Where("book_id = ?", 0).Find(&attachList).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Error("查询文章附件时出错 ->", err)
		}
	} else {
		err = DB.Table(NewAttachment().TableName()).Where("document_id = ?", b.DocumentId).Where("book_id = ?", b.BookId).Find(&attachList).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			logger.Error("查询文章附件时出错 ->", err)
		}
	}
	b.AttachList = attachList
	return
}
