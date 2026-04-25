package routers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/i18n"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/controllers"
	"github.com/mindoc-org/mindoc/models"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

// SetupRouter 创建并配置 Gin 引擎
func SetupRouter() *gin.Engine {
	if conf.GetDefaultString("runmode", "prod") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(HeaderMiddleware())
	setupSession(r)

	// 模板 — 使用共享模板实例
	funcMap := template.FuncMap{
		"config":      models.GetOptionValue,
		"cdn":          cdnFunc,
		"cdnjs":        conf.URLForWithCdnJs,
		"cdncss":       conf.URLForWithCdnCss,
		"cdnimg":       conf.URLForWithCdnImage,
		"urlfor":       urlForFunc,
		"conf":         conf.CONF,
		"date_format":  func(t time.Time, format string) string { return t.Local().Format(format) },
		"date":         phpDateFunc,
		"str":          func(v interface{}) string { return fmt.Sprint(v) },
		"i18n":         i18n.Tr,
		"str2html":     func(s string) template.HTML { return template.HTML(s) },
		"html2str":     HTML2Str,
		"htmlquote":    Htmlquote,
		"htmlunquote":  Htmlunquote,
		"htfn":         Htmlfilter,
	}

	// 收集所有模板文件并创建带路径名的 define 块
	viewsPath := conf.WorkingDir("views")
	tmpl := template.New("").Funcs(funcMap)

	filepath.Walk(viewsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".tpl") {
			return nil
		}
		relPath, _ := filepath.Rel(viewsPath, path)
		tplName := filepath.ToSlash(relPath)

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// 如果模板内容没有 {{define}} 块，用路径名包装
		contentStr := string(content)
		if !strings.Contains(contentStr, "{{define") {
			contentStr = "{{define \"" + tplName + "\"}}" + contentStr + "{{end}}"
		}

		_, err = tmpl.Parse(contentStr)
		if err != nil {
			logger.Error("解析模板失败:", tplName, err)
		}
		return nil
	})

	r.SetHTMLTemplate(tmpl)

	// 模板名使用文件基础名（ParseFiles 的默认行为）
	// 在 bridge.go 中 renderTemplate 会转换 tplName

	// 静态文件
	r.Static("/static", filepath.Join(conf.WorkingDirectory, "static"))
	uploads := conf.WorkingDir("uploads")
	_ = os.MkdirAll(uploads, 0666)
	r.Static("/uploads", uploads)

	// favicon
	r.StaticFile("/favicon.ico", filepath.Join(conf.WorkingDirectory, "static", "favicon.ico"))

	// 最大上传大小
	if maxSize := conf.GetUploadFileSize(); maxSize > 0 {
		r.MaxMultipartMemory = maxSize
	}

	// 公开路由
	registerPublicRoutes(r)

	// 需要登录的路由
	registerAuthRoutes(r)

	// MCP 路由
	if conf.GetDefaultBool("enable_mcp_server", false) {
		mcpGroup := r.Group("/mcp")
		mcpGroup.Use(MCPAuthMiddleware())
		{
			mcpGroup.Any("/*path", MCPProxyHandler())
		}
	}

	// 错误处理
	r.NoRoute(func(c *gin.Context) {
		showErrorPage(c, http.StatusNotFound, "页面未找到或已删除")
	})
	r.NoMethod(func(c *gin.Context) {
		showErrorPage(c, http.StatusMethodNotAllowed, "请求方法不允许")
	})

	return r
}

func showErrorPage(c *gin.Context, code int, msg string) {
	data := gin.H{
		"ErrorCode":    code,
		"ErrorMessage": msg,
		"BaseUrl":      conf.BaseUrl,
		"SiteName":     "MinDoc",
	}
	c.HTML(code, "errors/error.tpl", data)
	c.Abort()
}

func createRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	viewsPath := conf.WorkingDir("views")

	// 遍历 views 目录注册所有模板
	// 使用基础模板 template.tpl 作为 layout
	baseTpl := filepath.Join(viewsPath, "template.tpl")

	// 获取所有模板目录
	dirs, err := os.ReadDir(viewsPath)
	if err != nil {
		logger.Error("读取模板目录失败 ->", err)
		return r
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		dirName := dir.Name()
		dirPath := filepath.Join(viewsPath, dirName)

		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".tpl") {
				continue
			}
			tplName := dirName + "/" + f.Name()
			tplPath := filepath.Join(dirPath, f.Name())
			logger.Debug("Registering template:", tplName, "->", tplPath)

			// 读取模板文件内容
			content, err := os.ReadFile(tplPath)
			if err != nil {
				continue
			}

			// 检查是否引用了 template.tpl
			contentStr := string(content)
			if strings.Contains(contentStr, "template.tpl") {
				r.AddFromFilesFuncs(tplName, template.FuncMap{
					"config":      models.GetOptionValue,
					"cdn":          cdnFunc,
					"cdnjs":        conf.URLForWithCdnJs,
					"cdncss":       conf.URLForWithCdnCss,
					"cdnimg":       conf.URLForWithCdnImage,
					"urlfor":       urlForFunc,
					"conf":         conf.CONF,
					"date_format":  func(t time.Time, format string) string { return t.Local().Format(format) },
					"date":         phpDateFunc,
					"str":          func(v interface{}) string { return fmt.Sprint(v) },
					"i18n":         i18n.Tr,
					"str2html":     func(s string) template.HTML { return template.HTML(s) },
					"html2str":     HTML2Str,
					"htmlquote":    Htmlquote,
					"htmlunquote":  Htmlunquote,
					"htfn":         Htmlfilter,
				}, baseTpl, tplPath)
			} else {
				r.AddFromFilesFuncs(tplName, template.FuncMap{
					"config":      models.GetOptionValue,
					"cdn":          cdnFunc,
					"cdnjs":        conf.URLForWithCdnJs,
					"cdncss":       conf.URLForWithCdnCss,
					"cdnimg":       conf.URLForWithCdnImage,
					"urlfor":       urlForFunc,
					"conf":         conf.CONF,
					"date_format":  func(t time.Time, format string) string { return t.Local().Format(format) },
					"date":         phpDateFunc,
					"str":          func(v interface{}) string { return fmt.Sprint(v) },
					"i18n":         i18n.Tr,
					"str2html":     func(s string) template.HTML { return template.HTML(s) },
					"html2str":     HTML2Str,
					"htmlquote":    Htmlquote,
					"htmlunquote":  Htmlunquote,
					"htfn":         Htmlfilter,
				}, tplPath)
			}
		}
	}

	return r
}

// setupTemplates 配置模板引擎
func setupTemplates(r *gin.Engine) {
	viewsPath := conf.WorkingDir("views")

	funcMap := template.FuncMap{
		"config":      models.GetOptionValue,
		"cdn":          cdnFunc,
		"cdnjs":        conf.URLForWithCdnJs,
		"cdncss":       conf.URLForWithCdnCss,
		"cdnimg":       conf.URLForWithCdnImage,
		"urlfor":       urlForFunc,
		"conf":         conf.CONF,
		"date_format":  func(t time.Time, format string) string { return t.Local().Format(format) },
		"date":         phpDateFunc,
		"str":          func(v interface{}) string { return fmt.Sprint(v) },
		"i18n":         i18n.Tr,
		"str2html":     func(s string) template.HTML { return template.HTML(s) },
		"html2str":     HTML2Str,
		"htmlquote":    Htmlquote,
		"htmlunquote":  Htmlunquote,
		"htfn":         Htmlfilter,
	}

	// 收集所有模板文件
	var tplFiles []string
	filepath.Walk(viewsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tpl") {
			tplFiles = append(tplFiles, path)
		}
		return nil
	})

	if len(tplFiles) == 0 {
		logger.Error("未找到任何模板文件 in", viewsPath)
		return
	}

	logger.Info("加载", len(tplFiles), "个模板文件")

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFiles(tplFiles...))
	r.SetHTMLTemplate(tmpl)
}

func registerTemplateFunctions(r *gin.Engine) {
	// 模板函数已通过 createRenderer 注册
}

// phpDateFunc 兼容 PHP 风格的日期格式
func phpDateFunc(t time.Time, format string) string {
	goFormat := format
	goFormat = strings.ReplaceAll(goFormat, "Y", "2006")
	goFormat = strings.ReplaceAll(goFormat, "m", "01")
	goFormat = strings.ReplaceAll(goFormat, "d", "02")
	goFormat = strings.ReplaceAll(goFormat, "H", "15")
	goFormat = strings.ReplaceAll(goFormat, "i", "04")
	goFormat = strings.ReplaceAll(goFormat, "s", "05")
	return t.Local().Format(goFormat)
}

func cdnFunc(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	cdn := conf.GetDefaultString("cdn", "")
	if cdn == "" {
		baseUrl := conf.GetDefaultString("baseurl", "")
		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + p[1:]
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

// urlForFunc 路由名称到 URL 的映射
var routeNameMap = map[string]string{
	"AccountController.Auth2AutoAccount": "/auth2/account/auto/:app",
	"AccountController.Auth2BindAccount": "/auth2/account/bind/:app",
	"AccountController.Auth2Redirect": "/auth2/redirect/:app",
	"AccountController.Captcha": "/captcha",
	"AccountController.FindPassword": "/find_password",
	"AccountController.Login": "/login",
	"AccountController.Logout": "/logout",
	"AccountController.Register": "/register",
	"AccountController.ValidEmail": "/valid_email",
	"BlogController.Index": "/blog-:id([0-9]+).html",
	"BlogController.List": "/blogs",
	"BlogController.ManageDelete": "/manage/blogs/delete",
	"BlogController.ManageEdit": "/manage/blogs/edit",
	"BlogController.ManageList": "/manage/blogs",
	"BlogController.ManageSetting": "/manage/blogs/setting",
	"BlogController.RemoveAttachment": "/manage/blogs/attach/:id",
	"BlogController.Upload": "/manage/blogs/upload",
	"BookController.Copy": "/book/users/copy",
	"BookController.Create": "/book/create",
	"BookController.CreateToken": "/manager/books/token",
	"BookController.Dashboard": "/book/:key/dashboard",
	"BookController.Delete": "/book/setting/delete",
	"BookController.Import": "/book/users/import",
	"BookController.Index": "/book",
	"BookController.ItemsetsSearch": "/book/itemsets/search",
	"BookController.PrivatelyOwned": "/book/setting/open",
	"BookController.Release": "/book/:key/release",
	"BookController.SaveBook": "/book/setting/save",
	"BookController.SaveSort": "/book/:key/sort",
	"BookController.Setting": "/book/:key/setting",
	"BookController.Team": "/book/:key/teams",
	"BookController.TeamAdd": "/book/team/add",
	"BookController.TeamDelete": "/book/team/delete",
	"BookController.TeamSearch": "/book/team/search",
	"BookController.Transfer": "/book/setting/transfer",
	"BookController.UpdateBookOrder": "/book/updatebookorder",
	"BookController.UploadCover": "/book/setting/upload",
	"BookController.Users": "/book/:key/users",
	"BookMemberController.AddMember": "/book/users/create",
	"BookMemberController.ChangeRole": "/book/users/change",
	"BookMemberController.RemoveMember": "/book/users/delete",
	"CommentController.Create": "/comment/create",
	"CommentController.Index": "/comment/index",
	"DocumentController.CheckPassword": "/docs/:key/check-password",
	"DocumentController.Compare": "/api/:key/compare/:id",
	"DocumentController.Content": "/api/:key/content/:id",
	"DocumentController.Create": "/api/:key/create",
	"DocumentController.Delete": "/api/:key/delete",
	"DocumentController.DeleteHistory": "/history/delete",
	"DocumentController.DownloadAttachment": "/attach_files/:key/:attach_id",
	"DocumentController.Edit": "/api/:key/edit",
	"DocumentController.Export": "/export/:key",
	"DocumentController.History": "/history/get",
	"DocumentController.Index": "/docs/:key",
	"DocumentController.QrCode": "/qrcode/:key",
	"DocumentController.Read": "/docs/:key/:id",
	"DocumentController.RemoveAttachment": "/api/attach/remove/",
	"DocumentController.RestoreHistory": "/history/restore",
	"DocumentController.Search": "/docs/:key/search",
	"DocumentController.Upload": "/api/upload",
	"HomeController.Index": "/",
	"ItemsetsController.Index": "/items",
	"ItemsetsController.List": "/items/:key",
	"LabelController.Index": "/tag/:key",
	"ManagerController.AttachClean": "/manager/attach/clean",
	"ManagerController.AttachDelete": "/manager/attach/delete",
	"ManagerController.AttachDetailed": "/manager/attach/detailed/:id",
	"ManagerController.AttachList": "/manager/attach/list",
	"ManagerController.Books": "/manager/books",
	"ManagerController.ChangeMemberRole": "/manager/member/change-member-role",
	"ManagerController.Comments": "/manager/comments",
	"ManagerController.Config": "/manager/setting",
	"ManagerController.CreateMember": "/manager/member/create",
	"ManagerController.CreateToken": "/manager/books/token",
	"ManagerController.DeleteBook": "/manager/books/delete",
	"ManagerController.DeleteMember": "/manager/member/delete",
	"ManagerController.EditBook": "/manager/books/edit/:key",
	"ManagerController.EditMember": "/manager/users/edit/:id",
	"ManagerController.Index": "/manager",
	"ManagerController.Itemsets": "/manager/itemsets",
	"ManagerController.ItemsetsDelete": "/manager/itemsets/delete",
	"ManagerController.ItemsetsEdit": "/manager/itemsets/edit",
	"ManagerController.LabelDelete": "/manager/label/delete/:id",
	"ManagerController.LabelList": "/manager/label/list",
	"ManagerController.PrivatelyOwned": "/manager/books/open",
	"ManagerController.Setting": "/manager/setting",
	"ManagerController.Team": "/manager/team",
	"ManagerController.TeamBookAdd": "/manager/team/book/add",
	"ManagerController.TeamBookDelete": "/manager/team/book/delete",
	"ManagerController.TeamBookList": "/manager/team/book/list/:id",
	"ManagerController.TeamChangeMemberRole": "/manager/team/member/change_role",
	"ManagerController.TeamCreate": "/manager/team/create",
	"ManagerController.TeamDelete": "/manager/team/delete",
	"ManagerController.TeamEdit": "/manager/team/edit",
	"ManagerController.TeamMemberAdd": "/manager/team/member/add",
	"ManagerController.TeamMemberDelete": "/manager/team/member/delete",
	"ManagerController.TeamMemberList": "/manager/team/member/list/:id",
	"ManagerController.TeamSearchBook": "/manager/team/book/search",
	"ManagerController.TeamSearchMember": "/manager/team/member/search",
	"ManagerController.Transfer": "/manager/books/transfer",
	"ManagerController.UpdateMemberStatus": "/manager/member/update-member-status",
	"ManagerController.Users": "/manager/users",
	"SearchController.Index": "/search",
	"SearchController.IndexV2": "/search-v2",
	"SearchController.SearchV2": "/api/search-v2",
	"SearchController.User": "/api/search/user/:key",
	"SettingController.Index": "/setting",
	"SettingController.Password": "/setting/password",
	"SettingController.Upload": "/setting/upload",
	"TemplateController.Add": "/api/template/add",
	"TemplateController.Delete": "/api/template/remove",
	"TemplateController.Get": "/api/template/get",
	"TemplateController.List": "/api/template/list",
}

func urlForFunc(endpoint string, values ...interface{}) string {
	path, ok := routeNameMap[endpoint]
	if !ok {
		return "/"
	}
	result := path
	for i := 0; i < len(values)-1; i += 2 {
		if key, ok := values[i].(string); ok {
			placeholder := key
			if !strings.HasPrefix(placeholder, ":") {
				placeholder = ":" + placeholder
			}
			result = strings.Replace(result, placeholder, fmt.Sprint(values[i+1]), 1)
		}
	}
	baseUrl := conf.BaseUrl
	if baseUrl == "" {
		return result
	}
	if strings.HasPrefix(result, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + result[1:]
	}
	return baseUrl + result
}

// registerPublicRoutes 注册不需要登录的公开路由
func registerPublicRoutes(r *gin.Engine) {
	// 首页
	r.GET("/", wrapAny(&controllers.HomeController{}, "Index"))

	// 账户相关
	r.GET("/login", wrapAny(&controllers.AccountController{}, "Login"))
	r.POST("/login", wrapAny(&controllers.AccountController{}, "Login"))
	r.GET("/auth2/redirect/:app", wrapAny(&controllers.AccountController{}, "Auth2Redirect"))
	r.GET("/auth2/callback/:app", wrapAny(&controllers.AccountController{}, "Auth2Callback"))
	r.GET("/auth2/account/bind/:app", wrapAny(&controllers.AccountController{}, "Auth2BindAccount"))
	r.GET("/auth2/account/auto/:app", wrapAny(&controllers.AccountController{}, "Auth2AutoAccount"))
	r.Any("/logout", wrapAny(&controllers.AccountController{}, "Logout"))
	r.GET("/register", wrapAny(&controllers.AccountController{}, "Register"))
	r.POST("/register", wrapAny(&controllers.AccountController{}, "Register"))
	r.GET("/find_password", wrapAny(&controllers.AccountController{}, "FindPassword"))
	r.POST("/find_password", wrapAny(&controllers.AccountController{}, "FindPassword"))
	r.POST("/valid_email", wrapAny(&controllers.AccountController{}, "ValidEmail"))
	r.Any("/captcha", wrapAny(&controllers.AccountController{}, "Captcha"))

	// 文档阅读（公开部分）
	r.GET("/docs/:key", wrapAny(&controllers.DocumentController{}, "Index"))
	r.POST("/docs/:key/check-password", wrapAny(&controllers.DocumentController{}, "CheckPassword"))
	r.GET("/docs/:key/:id", wrapAny(&controllers.DocumentController{}, "Read"))
	r.POST("/docs/:key/search", wrapAny(&controllers.DocumentController{}, "Search"))
	r.Any("/export/:key", wrapAny(&controllers.DocumentController{}, "Export"))
	r.GET("/qrcode/:key", wrapAny(&controllers.DocumentController{}, "QrCode"))
	r.GET("/attach_files/:key/:attach_id", wrapAny(&controllers.DocumentController{}, "DownloadAttachment"))

	// 博客
	r.GET("/blogs", wrapAny(&controllers.BlogController{}, "List"))
	r.GET("/blog-:id([0-9]+).html", wrapAny(&controllers.BlogController{}, "Index"))
		r.POST("/blog-:id([0-9]+).html", wrapAny(&controllers.BlogController{}, "Index"))
		r.GET("/blog/attach/:id/:attach_id", wrapAny(&controllers.BlogController{}, "Download"))

	// 搜索
	r.GET("/search", wrapAny(&controllers.SearchController{}, "Index"))
	r.GET("/search-v2", wrapAny(&controllers.SearchController{}, "IndexV2"))
		r.POST("/search-v2", wrapAny(&controllers.SearchController{}, "IndexV2"))

	// 标签
	r.GET("/tag/:key", wrapAny(&controllers.LabelController{}, "Index"))
	r.GET("/tags", wrapAny(&controllers.LabelController{}, "List"))

	// 项目集
	r.GET("/items", wrapAny(&controllers.ItemsetsController{}, "Index"))
	r.GET("/items/:key", wrapAny(&controllers.ItemsetsController{}, "List"))

	// 评论
	r.POST("/comment/create", wrapAny(&controllers.CommentController{}, "Create"))
	r.POST("/comment/delete", wrapAny(&controllers.CommentController{}, "Delete"))
	r.GET("/comment/lists", wrapAny(&controllers.CommentController{}, "Lists"))
	r.Any("/comment/index", wrapAny(&controllers.CommentController{}, "Index"))

	// 历史
	r.GET("/history/get", wrapAny(&controllers.DocumentController{}, "History"))
	r.Any("/history/delete", wrapAny(&controllers.DocumentController{}, "DeleteHistory"))
	r.Any("/history/restore", wrapAny(&controllers.DocumentController{}, "RestoreHistory"))

	// CORS 代理
	r.Any("/cors-anywhere", corsAnywhereHandler())

	// API 公开搜索
	r.GET("/api/search-v2", wrapAny(&controllers.SearchController{}, "SearchV2"))
	r.GET("/api/search/user/:key", wrapAny(&controllers.SearchController{}, "User"))
}

// registerAuthRoutes 注册需要登录的路由
func registerAuthRoutes(r *gin.Engine) {
	auth := r.Group("")
	auth.Use(AuthRequired())
	{
		// 管理后台
		auth.GET("/manager", wrapAny(&controllers.ManagerController{}, "Index"))
		auth.GET("/manager/users", wrapAny(&controllers.ManagerController{}, "Users"))
		auth.GET("/manager/users/edit/:id", wrapAny(&controllers.ManagerController{}, "EditMember"))
		auth.POST("/manager/users/edit/:id", wrapAny(&controllers.ManagerController{}, "EditMember"))
		auth.POST("/manager/member/create", wrapAny(&controllers.ManagerController{}, "CreateMember"))
		auth.POST("/manager/member/delete", wrapAny(&controllers.ManagerController{}, "DeleteMember"))
		auth.POST("/manager/member/update-member-status", wrapAny(&controllers.ManagerController{}, "UpdateMemberStatus"))
		auth.POST("/manager/member/change-member-role", wrapAny(&controllers.ManagerController{}, "ChangeMemberRole"))
		auth.GET("/manager/books", wrapAny(&controllers.ManagerController{}, "Books"))
		auth.GET("/manager/books/edit/:key", wrapAny(&controllers.ManagerController{}, "EditBook"))
		auth.POST("/manager/books/edit/:key", wrapAny(&controllers.ManagerController{}, "EditBook"))
		auth.Any("/manager/books/delete", wrapAny(&controllers.ManagerController{}, "DeleteBook"))
		auth.GET("/manager/comments", wrapAny(&controllers.ManagerController{}, "Comments"))
		auth.GET("/manager/setting", wrapAny(&controllers.ManagerController{}, "Setting"))
			auth.POST("/manager/setting", wrapAny(&controllers.ManagerController{}, "Setting"))
		auth.GET("/manager/config", wrapAny(&controllers.ManagerController{}, "Config"))
		auth.POST("/manager/config", wrapAny(&controllers.ManagerController{}, "Config"))
		auth.POST("/manager/books/token", wrapAny(&controllers.ManagerController{}, "CreateToken"))
		auth.POST("/manager/books/transfer", wrapAny(&controllers.ManagerController{}, "Transfer"))
		auth.POST("/manager/books/open", wrapAny(&controllers.ManagerController{}, "PrivatelyOwned"))
		auth.GET("/manager/attach/list", wrapAny(&controllers.ManagerController{}, "AttachList"))
		auth.POST("/manager/attach/clean", wrapAny(&controllers.ManagerController{}, "AttachClean"))
		auth.GET("/manager/attach/detailed/:id", wrapAny(&controllers.ManagerController{}, "AttachDetailed"))
		auth.POST("/manager/attach/delete", wrapAny(&controllers.ManagerController{}, "AttachDelete"))
		auth.GET("/manager/label/list", wrapAny(&controllers.ManagerController{}, "LabelList"))
		auth.POST("/manager/label/delete/:id", wrapAny(&controllers.ManagerController{}, "LabelDelete"))
		auth.GET("/manager/team", wrapAny(&controllers.ManagerController{}, "Team"))
		auth.POST("/manager/team/create", wrapAny(&controllers.ManagerController{}, "TeamCreate"))
		auth.POST("/manager/team/edit", wrapAny(&controllers.ManagerController{}, "TeamEdit"))
		auth.POST("/manager/team/delete", wrapAny(&controllers.ManagerController{}, "TeamDelete"))
		auth.GET("/manager/team/member/list/:id", wrapAny(&controllers.ManagerController{}, "TeamMemberList"))
		auth.POST("/manager/team/member/add", wrapAny(&controllers.ManagerController{}, "TeamMemberAdd"))
		auth.POST("/manager/team/member/delete", wrapAny(&controllers.ManagerController{}, "TeamMemberDelete"))
		auth.POST("/manager/team/member/change_role", wrapAny(&controllers.ManagerController{}, "TeamChangeMemberRole"))
		auth.GET("/manager/team/member/search", wrapAny(&controllers.ManagerController{}, "TeamSearchMember"))
		auth.GET("/manager/team/book/list/:id", wrapAny(&controllers.ManagerController{}, "TeamBookList"))
		auth.POST("/manager/team/book/add", wrapAny(&controllers.ManagerController{}, "TeamBookAdd"))
		auth.POST("/manager/team/book/delete", wrapAny(&controllers.ManagerController{}, "TeamBookDelete"))
		auth.GET("/manager/team/book/search", wrapAny(&controllers.ManagerController{}, "TeamSearchBook"))
		auth.GET("/manager/itemsets", wrapAny(&controllers.ManagerController{}, "Itemsets"))
		auth.POST("/manager/itemsets/edit", wrapAny(&controllers.ManagerController{}, "ItemsetsEdit"))
		auth.POST("/manager/itemsets/delete", wrapAny(&controllers.ManagerController{}, "ItemsetsDelete"))

		// 个人设置
		auth.GET("/setting", wrapAny(&controllers.SettingController{}, "Index"))
		auth.POST("/setting", wrapAny(&controllers.SettingController{}, "Index"))
		auth.GET("/setting/password", wrapAny(&controllers.SettingController{}, "Password"))
		auth.POST("/setting/password", wrapAny(&controllers.SettingController{}, "Password"))
		auth.GET("/setting/upload", wrapAny(&controllers.SettingController{}, "Upload"))

		// 书籍管理
		auth.GET("/book", wrapAny(&controllers.BookController{}, "Index"))
		auth.GET("/book/:key/dashboard", wrapAny(&controllers.BookController{}, "Dashboard"))
		auth.GET("/book/:key/setting", wrapAny(&controllers.BookController{}, "Setting"))
		auth.GET("/book/:key/users", wrapAny(&controllers.BookController{}, "Users"))
		auth.POST("/book/:key/release", wrapAny(&controllers.BookController{}, "Release"))
		auth.POST("/book/:key/sort", wrapAny(&controllers.BookController{}, "SaveSort"))
		auth.GET("/book/:key/teams", wrapAny(&controllers.BookController{}, "Team"))
		auth.POST("/book/updatebookorder", wrapAny(&controllers.BookController{}, "UpdateBookOrder"))
		auth.GET("/book/create", wrapAny(&controllers.BookController{}, "Create"))
		auth.POST("/book/create", wrapAny(&controllers.BookController{}, "Create"))
		auth.GET("/book/itemsets/search", wrapAny(&controllers.BookController{}, "ItemsetsSearch"))

		// 书籍成员
		auth.POST("/book/users/create", wrapAny(&controllers.BookMemberController{}, "AddMember"))
		auth.POST("/book/users/change", wrapAny(&controllers.BookMemberController{}, "ChangeRole"))
		auth.POST("/book/users/delete", wrapAny(&controllers.BookMemberController{}, "RemoveMember"))
		auth.POST("/book/users/import", wrapAny(&controllers.BookController{}, "Import"))
		auth.POST("/book/users/copy", wrapAny(&controllers.BookController{}, "Copy"))

		// 书籍设置
		auth.POST("/book/setting/save", wrapAny(&controllers.BookController{}, "SaveBook"))
		auth.POST("/book/setting/open", wrapAny(&controllers.BookController{}, "PrivatelyOwned"))
		auth.POST("/book/setting/transfer", wrapAny(&controllers.BookController{}, "Transfer"))
		auth.POST("/book/setting/upload", wrapAny(&controllers.BookController{}, "UploadCover"))
		auth.POST("/book/setting/delete", wrapAny(&controllers.BookController{}, "Delete"))

		// 书籍团队
		auth.POST("/book/team/add", wrapAny(&controllers.BookController{}, "TeamAdd"))
		auth.POST("/book/team/delete", wrapAny(&controllers.BookController{}, "TeamDelete"))
		auth.GET("/book/team/search", wrapAny(&controllers.BookController{}, "TeamSearch"))

		// 博客管理
		auth.GET("/manage/blogs", wrapAny(&controllers.BlogController{}, "ManageList"))
		auth.GET("/manage/blogs/setting", wrapAny(&controllers.BlogController{}, "ManageSetting"))
		auth.POST("/manage/blogs/setting", wrapAny(&controllers.BlogController{}, "ManageSetting"))
		auth.GET("/manage/blogs/setting/:id", wrapAny(&controllers.BlogController{}, "ManageSetting"))
		auth.GET("/manage/blogs/edit", wrapAny(&controllers.BlogController{}, "ManageEdit"))
		auth.POST("/manage/blogs/edit", wrapAny(&controllers.BlogController{}, "ManageEdit"))
		auth.GET("/manage/blogs/edit/:id", wrapAny(&controllers.BlogController{}, "ManageEdit"))
		auth.POST("/manage/blogs/edit/:id", wrapAny(&controllers.BlogController{}, "ManageEdit"))
		auth.POST("/manage/blogs/delete", wrapAny(&controllers.BlogController{}, "ManageDelete"))
		auth.POST("/manage/blogs/upload", wrapAny(&controllers.BlogController{}, "Upload"))
		auth.POST("/manage/blogs/attach/:id", wrapAny(&controllers.BlogController{}, "RemoveAttachment"))

		// 模板 API
		auth.GET("/api/template/get", wrapAny(&controllers.TemplateController{}, "Get"))
		auth.POST("/api/template/list", wrapAny(&controllers.TemplateController{}, "List"))
		auth.POST("/api/template/add", wrapAny(&controllers.TemplateController{}, "Add"))
		auth.POST("/api/template/remove", wrapAny(&controllers.TemplateController{}, "Delete"))

		// 文档 API
		auth.POST("/api/attach/remove/", wrapAny(&controllers.DocumentController{}, "RemoveAttachment"))
		auth.GET("/api/:key/edit", wrapAny(&controllers.DocumentController{}, "Edit"))
		auth.POST("/api/:key/edit", wrapAny(&controllers.DocumentController{}, "Edit"))
		auth.GET("/api/:key/edit/:id", wrapAny(&controllers.DocumentController{}, "Edit"))
		auth.POST("/api/:key/edit/:id", wrapAny(&controllers.DocumentController{}, "Edit"))
		auth.POST("/api/upload", wrapAny(&controllers.DocumentController{}, "Upload"))
		auth.POST("/api/:key/create", wrapAny(&controllers.DocumentController{}, "Create"))
		auth.POST("/api/:key/delete", wrapAny(&controllers.DocumentController{}, "Delete"))
		auth.GET("/api/:key/content", wrapAny(&controllers.DocumentController{}, "Content"))
		auth.POST("/api/:key/content", wrapAny(&controllers.DocumentController{}, "Content"))
		auth.GET("/api/:key/content/:id", wrapAny(&controllers.DocumentController{}, "Content"))
		auth.GET("/api/:key/compare/:id", wrapAny(&controllers.DocumentController{}, "Compare"))
	}

	_ = fmt.Sprintf("registered %d auth routes", 0)
}
