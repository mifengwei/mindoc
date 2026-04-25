package models

import (
	"errors"
	"strings"
	"time"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/utils"
	"github.com/mindoc-org/mindoc/utils/cryptil"
	"gorm.io/gorm"
)

//项目空间
type Itemsets struct {
	ItemId      int       `gorm:"column:item_id;primaryKey;autoIncrement;uniqueIndex" json:"item_id"`
	ItemName    string    `gorm:"column:item_name;size:500;description:项目空间名称" json:"item_name"`
	ItemKey     string    `gorm:"column:item_key;size:100;uniqueIndex;description:项目空间标识" json:"item_key"`
	Description string    `gorm:"column:description;type:text;description:描述" json:"description"`
	MemberId    int       `gorm:"column:member_id;size:100;description:所属用户" json:"member_id"`
	CreateTime  time.Time `gorm:"column:create_time;type:datetime;autoCreateTime;description:创建时间" json:"create_time"`
	ModifyTime  time.Time `gorm:"column:modify_time;type:datetime;autoUpdateTime;description:修改时间" json:"modify_time"`
	ModifyAt    int       `gorm:"column:modify_at;type:int;description:修改人id" json:"modify_at"`

	BookNumber       int    `gorm:"-" json:"book_number"`
	CreateTimeString string `gorm:"-" json:"create_time_string"`
	CreateName       string `gorm:"-" json:"create_name"`
}

// TableName 获取对应数据库表名.
func (item *Itemsets) TableName() string {
	return conf.GetDatabasePrefix() + "itemsets"
}

func NewItemsets() *Itemsets {
	return &Itemsets{}
}

func (item *Itemsets) First(itemId int) (*Itemsets, error) {
	if itemId <= 0 {
		return nil, ErrInvalidParameter
	}
	err := DB.Table(item.TableName()).Where("item_id = ?", itemId).First(item).Error
	if err != nil {
		logger.Error("查询项目空间失败 -> item_id=", itemId, err)
	} else {
		item.Include()
	}
	return item, err
}

func (item *Itemsets) FindFirst(itemKey string) (*Itemsets, error) {
	err := DB.Table(item.TableName()).Where("item_key = ?", itemKey).First(item).Error
	if err != nil {
		logger.Error("查询项目空间失败 -> itemKey=", itemKey, err)
	} else {
		item.Include()
	}
	return item, err
}

func (item *Itemsets) Exist(itemId int) bool {
	var dummy Itemsets
	return DB.Table(item.TableName()).Where("item_id = ?", itemId).First(&dummy).Error == nil
}

//保存
func (item *Itemsets) Save() (err error) {

	item.ItemName = strings.TrimSpace(utils.StripTags(item.ItemName))
	item.Description = strings.TrimSpace(utils.StripTags(item.Description))
	item.ItemKey = strings.TrimSpace(item.ItemKey)

	if item.ItemName == "" {
		return errors.New("项目空间名称不能为空")
	}
	if item.ItemKey == "" {
		item.ItemKey = cryptil.NewRandChars(16)
	}

	var dummy Itemsets
	if DB.Table(item.TableName()).Where("item_id != ? AND item_key = ?", item.ItemId, item.ItemKey).First(&dummy).Error == nil {
		return errors.New("项目空间标识已存在")
	}
	if item.ItemId > 0 {
		err = DB.Save(item).Error
	} else {
		err = DB.Create(item).Error
	}
	return
}

//删除.
func (item *Itemsets) Delete(itemId int) (err error) {
	if itemId <= 0 {
		return ErrInvalidParameter
	}
	if itemId == 1 {
		return errors.New("默认项目空间不能删除")
	}
	if !item.Exist(itemId) {
		return errors.New("项目空间不存在")
	}
	tx := DB.Begin()
	if err != nil {
		logger.Error("开启事物失败 ->", err)
		return err
	}
	err = tx.Table(item.TableName()).Where("item_id = ?", itemId).Delete(&Itemsets{}).Error
	if err != nil {
		logger.Error("删除项目空间失败 -> item_id=", itemId, err)
		tx.Rollback()
	}
	err = tx.Exec("update md_books set item_id=1 where item_id=?;", itemId).Error
	if err != nil {
		logger.Error("删除项目空间失败 -> item_id=", itemId, err)
		tx.Rollback()
	}

	return tx.Commit().Error
}

func (item *Itemsets) Include() (*Itemsets, error) {

	item.CreateTimeString = item.CreateTime.Format("2006-01-02 15:04:05")

	if item.MemberId > 0 {
		if m, err := NewMember().Find(item.MemberId, "account", "real_name"); err == nil {
			if m.RealName != "" {
				item.CreateName = m.RealName
			} else {
				item.CreateName = m.Account
			}
		}
	}

	var count int64
	err := DB.Table(NewBook().TableName()).Where("item_id = ?", item.ItemId).Count(&count).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return item, err
	}
	item.BookNumber = int(count)

	return item, nil
}

//分页查询.
func (item *Itemsets) FindToPager(pageIndex, pageSize int) (list []*Itemsets, totalCount int, err error) {

	offset := (pageIndex - 1) * pageSize

	err = DB.Table(item.TableName()).Order("item_id DESC").Offset(offset).Limit(pageSize).Find(&list).Error

	if err != nil {
		return
	}

	var c int64
	err = DB.Table(item.TableName()).Count(&c).Error
	if err != nil {
		return
	}
	totalCount = int(c)

	for _, item := range list {
		item.Include()
	}
	return
}

//根据项目空间名称查询.
func (item *Itemsets) FindItemsetsByName(name string, limit int) (*SelectMemberResult, error) {
	result := SelectMemberResult{}

	var itemsets []*Itemsets
	var err error
	if name == "" {
		err = DB.Table(item.TableName()).Limit(limit).Find(&itemsets).Error

	} else {
		err = DB.Table(item.TableName()).Where("item_name LIKE ?", "%"+name+"%").Limit(limit).Find(&itemsets).Error
	}
	if err != nil {
		logger.Error("查询项目空间失败 ->", err)
		return &result, err
	}

	items := make([]KeyValueItem, 0)

	for _, m := range itemsets {
		item := KeyValueItem{}
		item.Id = m.ItemId
		item.Text = m.ItemName
		items = append(items, item)
	}
	result.Result = items

	return &result, err
}

//根据项目空间标识查询项目空间的项目列表.
func (item *Itemsets) FindItemsetsByItemKey(key string, pageIndex, pageSize, memberId int) (books []*BookResult, totalCount int, err error) {

	err = DB.Table(item.TableName()).Where("item_key = ?", key).First(item).Error

	if err != nil {
		logger.Error("查询项目空间时出错 ->", key, err)
		return nil, 0, err
	}
	offset := (pageIndex - 1) * pageSize
	//如果是登录用户
	if memberId > 0 {
		sql1 := `SELECT COUNT(*)
FROM md_books AS book
  LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) as role_id
             from (select book_id,role_id
                   from md_team_relationship as mtr
                     left join md_team_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team on team.book_id = book.book_id
WHERE book.item_id = ? AND (book.privately_owned = 0 or rel.role_id >= 0 or team.role_id >= 0)`

		err = DB.Raw(sql1, memberId, memberId, item.ItemId).Scan(&totalCount).Error
		if err != nil {
			logger.Error("查询项目空间时出错 ->", key, err)
			return
		}
		sql2 := `SELECT book.*,rel1.*,mdmb.account AS create_name FROM md_books AS book
				LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
				left join (select book_id,min(role_id) as role_id from (select book_id,role_id
                   	from md_team_relationship as mtr
						left join md_team_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team
						on team.book_id = book.book_id
				LEFT JOIN md_relationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
				LEFT JOIN md_members AS mdmb ON rel1.member_id = mdmb.member_id
				WHERE book.item_id = ? AND (book.privately_owned = 0 or rel.role_id >= 0 or team.role_id >= 0)
				ORDER BY order_index desc,book.book_id DESC limit ? offset ?`

		err = DB.Raw(sql2, memberId, memberId, item.ItemId, pageSize, offset).Scan(&books).Error

		return

	} else {
		var count int64
		err1 := DB.Table(NewBook().TableName()).Where("privately_owned = ? AND item_id = ?", 0, item.ItemId).Count(&count).Error
		if err1 != nil {
			err = err1
			return
		}
		totalCount = int(count)

		sql := `SELECT book.*,rel.*,mdmb.account AS create_name FROM md_books AS book
				LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
				LEFT JOIN md_members AS mdmb ON rel.member_id = mdmb.member_id
				WHERE book.item_id = ? AND book.privately_owned = 0 ORDER BY order_index desc,book.book_id DESC limit ? offset ?`

		err = DB.Raw(sql, item.ItemId, pageSize, offset).Scan(&books).Error

		return

	}
}
