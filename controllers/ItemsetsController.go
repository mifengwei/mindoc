package controllers

import (
	"math"

	"gorm.io/gorm"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/models"
	"github.com/mindoc-org/mindoc/utils/pagination"
)

type ItemsetsController struct {
	BaseController
}

func (c *ItemsetsController) Prepare() {
	c.BaseController.Prepare()

	//如果没有开启你们访问则跳转到登录
	if !c.EnableAnonymous && c.Member == nil {
		c.Redirect(conf.URLFor("AccountController.Login"), 302)
		return
	}
}
func (c *ItemsetsController) Index() {
	c.Prepare()
	c.TplName = "items/index.tpl"
	pageSize := 16

	pageIndex, _ := c.GetInt("page", 0)

	items, totalCount, err := models.NewItemsets().FindToPager(pageIndex, pageSize)

	if err != nil && err != gorm.ErrRecordNotFound {
		c.ShowErrorPage(500, err.Error())
	}
	if err == gorm.ErrRecordNotFound || len(items) <= 0 {
		c.Data["Lists"] = items
		c.Data["PageHtml"] = ""
		return
	}

	if totalCount > 0 {
		pager := pagination.NewPagination(c.Gin.Request, totalCount, pageSize, c.BaseUrl())
		c.Data["PageHtml"] = pager.HtmlPages()
	} else {
		c.Data["PageHtml"] = ""
	}
	c.Data["TotalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	c.Data["Lists"] = items
}

func (c *ItemsetsController) List() {
	c.Prepare()
	c.TplName = "items/list.tpl"
	pageSize := 18
	itemKey := c.Gin.Param("key")
	pageIndex, _ := c.GetInt("page", 1)

	if itemKey == "" {
		c.Abort("404")
	}
	item, err := models.NewItemsets().FindFirst(itemKey)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Abort("404")
		} else {
			logger.Error(err)
			c.Abort("500")
		}
	}
	memberId := 0
	if c.Member != nil {
		memberId = c.Member.MemberId
	}
	searchResult, totalCount, err := models.NewItemsets().FindItemsetsByItemKey(itemKey, pageIndex, pageSize, memberId)

	if err != nil && err != gorm.ErrRecordNotFound {
		c.ShowErrorPage(500, "查询文档列表时出错")
	}
	if totalCount > 0 {
		pager := pagination.NewPagination(c.Gin.Request, totalCount, pageSize, c.BaseUrl())
		c.Data["PageHtml"] = pager.HtmlPages()
	} else {
		c.Data["PageHtml"] = ""
	}
	c.Data["TotalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	c.Data["Lists"] = searchResult

	c.Data["Model"] = item
}
