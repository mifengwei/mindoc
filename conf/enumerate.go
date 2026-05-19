// package conf 为配置相关.
package conf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 登录用户的Session名
const LoginSessionName = "LoginSessionName"

const CaptchaSessionName = "__captcha__"

const RegexpEmail = "^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"

// 允许用户名中出现点号
const RegexpAccount = `^[a-zA-Z0-9][a-zA-Z0-9\.-]{2,50}$`

// PageSize 默认分页条数.
const PageSize = 10

// 用户权限
const (
	// 超级管理员.
	MemberSuperRole SystemRole = iota
	//普通管理员.
	MemberAdminRole
	//普通用户.
	MemberGeneralRole
	//只读用户.
	MemberReaderRole
)

// 系统角色
type SystemRole int

const (
	// 创始人.
	BookFounder BookRole = iota
	//管理者
	BookAdmin
	//编辑者.
	BookEditor
	//观察者
	BookObserver
	//未指定关系
	BookRoleNoSpecific
)

// 项目角色
type BookRole int

const (
	LoggerOperate   = "operate"
	LoggerSystem    = "system"
	LoggerException = "exception"
	LoggerDocument  = "document"
)
const (
	//本地账户校验
	AuthMethodLocal = "local"
	//LDAP用户校验
	AuthMethodLDAP = "ldap"
)

var (
	VERSION    string
	BUILD_TIME string
	GO_VERSION string
)

var (
	ConfigurationFile = "./conf/app.conf"
	WorkingDirectory  = "./"
	LogFile           = "./runtime/logs"
	BaseUrl           = ""
	AutoLoadDelay     = 0
)

// app_key
func GetAppKey() string {
	return GetDefaultString("app_key", "mindoc")
}

func GetDatabasePrefix() string {
	return GetDefaultString("db_prefix", "md_")
}

// 获取默认头像
func GetDefaultAvatar() string {
	return URLForWithCdnImage(GetDefaultString("avatar", "/static/images/headimgurl.jpg"))
}

// 获取阅读令牌长度.
func GetTokenSize() int {
	return GetDefaultInt("token_size", 12)
}

// 获取默认文档封面.
func GetDefaultCover() string {

	return URLForWithCdnImage(GetDefaultString("cover", "/static/images/book.jpg"))
}

// 获取允许的上传文件的类型.
func GetUploadFileExt() []string {
	ext := GetDefaultString("upload_file_ext", "png|jpg|jpeg|gif|txt|doc|docx|pdf|mp4")

	temp := strings.Split(ext, "|")

	exts := make([]string, len(temp))

	i := 0
	for _, item := range temp {
		if item != "" {
			exts[i] = item
			i++
		}
	}
	return exts
}

// 获取上传文件允许的最大值
func GetUploadFileSize() int64 {
	size := GetDefaultString("upload_file_size", "0")

	if strings.HasSuffix(size, "TB") {
		if s, e := strconv.ParseInt(size[0:len(size)-2], 10, 64); e == nil {
			return s * 1024 * 1024 * 1024 * 1024
		}
	}
	if strings.HasSuffix(size, "GB") {
		if s, e := strconv.ParseInt(size[0:len(size)-2], 10, 64); e == nil {
			return s * 1024 * 1024 * 1024
		}
	}
	if strings.HasSuffix(size, "MB") {
		if s, e := strconv.ParseInt(size[0:len(size)-2], 10, 64); e == nil {
			return s * 1024 * 1024
		}
	}
	if strings.HasSuffix(size, "KB") {
		if s, e := strconv.ParseInt(size[0:len(size)-2], 10, 64); e == nil {
			return s * 1024
		}
	}
	if s, e := strconv.ParseInt(size, 10, 64); e == nil {
		return s
	}
	return 0
}

// 是否启用导出
func GetEnableExport() bool {
	return GetDefaultBool("enable_export", true)
}

// 是否启用iframe
func GetEnableIframe() bool {
	return GetDefaultBool("enable_iframe", false)
}

// 同一项目导出线程的并发数
func GetExportProcessNum() int {
	exportProcessNum := GetDefaultInt("export_process_num", 1)

	if exportProcessNum <= 0 || exportProcessNum > 4 {
		exportProcessNum = 1
	}
	return exportProcessNum
}

// 导出项目队列的并发数量
func GetExportLimitNum() int {
	exportLimitNum := GetDefaultInt("export_limit_num", 1)

	if exportLimitNum < 0 {
		exportLimitNum = 1
	}
	return exportLimitNum
}

// 等待导出队列的长度
func GetExportQueueLimitNum() int {
	exportQueueLimitNum := GetDefaultInt("export_queue_limit_num", 10)

	if exportQueueLimitNum <= 0 {
		exportQueueLimitNum = 100
	}
	return exportQueueLimitNum
}

// 默认导出项目的缓存目录
func GetExportOutputPath() string {
	exportOutputPath := filepath.Join(GetDefaultString("export_output_path", filepath.Join(WorkingDirectory, "cache")), "books")

	return exportOutputPath
}

// 判断是否是允许上传的文件类型.
func IsAllowUploadFileExt(ext string) bool {

	if strings.HasPrefix(ext, ".") {
		ext = string(ext[1:])
	}
	exts := GetUploadFileExt()

	for _, item := range exts {
		if item == "*" {
			return true
		}
		if strings.EqualFold(item, ext) {
			return true
		}
	}
	return false
}

// 读取配置文件值
func CONF(key string, value ...string) string {
	defaultValue := ""
	if len(value) > 0 {
		defaultValue = value[0]
	}
	return GetDefaultString(key, defaultValue)
}

// 路由名称到路径的映射
var routeMap = map[string]string{
	"AccountController.Auth2AutoAccount": "/auth2/account/auto/:app",
	"AccountController.Auth2BindAccount": "/auth2/account/bind/:app",
	"AccountController.Auth2Callback":    "/auth2/callback/:app",
	"AccountController.Auth2Redirect":    "/auth2/redirect/:app",
	"AccountController.Captcha":          "/captcha",
	"AccountController.FindPassword":     "/find_password",
	"AccountController.Login":            "/login",
	"AccountController.Logout":           "/logout",
	"AccountController.Register":         "/register",
	"AccountController.ValidEmail":       "/valid_email",
	"BlogController.Index":               "/blog/:id",
	"BlogController.List":                "/blogs",
	"BlogController.Download":            "/blog/attach/:id/:attach_id",
	"BlogController.ManageDelete":        "/manage/blogs/delete",
	"BlogController.ManageEdit":          "/manage/blogs/edit",
	"BlogController.ManageList":          "/manage/blogs",
	"BlogController.ManageSetting":       "/manage/blogs/setting",
	"BlogController.RemoveAttachment":    "/manage/blogs/attach/:id",
	"BlogController.Upload":              "/manage/blogs/upload",
	"BookController.Copy":                "/book/users/copy",
	"BookController.Create":              "/book/create",
	"BookController.CreateToken":         "/manager/books/token",
	"BookController.Dashboard":           "/book/:key/dashboard",
	"BookController.Delete":              "/book/setting/delete",
	"BookController.Import":              "/book/users/import",
	"BookController.Index":               "/book",
	"BookController.ItemsetsSearch":      "/book/itemsets/search",
	"BookController.PrivatelyOwned":      "/book/setting/open",
	"BookController.Release":             "/book/:key/release",
	"BookController.SaveBook":            "/book/setting/save",
	"BookController.SaveSort":            "/book/:key/sort",
	"BookController.Setting":             "/book/:key/setting",
	"BookController.Team":                "/book/:key/teams",
	"BookController.TeamAdd":             "/book/team/add",
	"BookController.TeamDelete":          "/book/team/delete",
	"BookController.TeamSearch":          "/book/team/search",
	"BookController.Transfer":            "/book/setting/transfer",
	"BookController.UpdateBookOrder":     "/book/updatebookorder",
	"BookController.UploadCover":         "/book/setting/upload",
	"BookController.Users":               "/book/:key/users",
	"BookMemberController.AddMember":     "/book/users/create",
	"BookMemberController.ChangeRole":    "/book/users/change",
	"BookMemberController.RemoveMember":  "/book/users/delete",
	"CommentController.Create":           "/comment/create",
	"CommentController.Delete":           "/comment/delete",
	"CommentController.Index":            "/comment/index",
	"CommentController.Lists":            "/comment/lists",
	"DocumentController.CheckPassword":   "/docs/:key/check-password",
	"DocumentController.Compare":         "/api/:key/compare/:id",
	"DocumentController.Content":         "/api/:key/content/:id",
	"DocumentController.Create":          "/api/:key/create",
	"DocumentController.Delete":          "/api/:key/delete",
	"DocumentController.DeleteHistory":   "/history/delete",
	"DocumentController.DownloadAttachment": "/attach_files/:key/:attach_id",
	"DocumentController.Edit":            "/api/:key/edit/:id",
	"DocumentController.Export":          "/export/:key",
	"DocumentController.History":         "/history/get",
	"DocumentController.Index":           "/docs/:key",
	"DocumentController.QrCode":          "/qrcode/:key",
	"DocumentController.Read":            "/docs/:key/:id",
	"DocumentController.RemoveAttachment": "/api/attach/remove/",
	"DocumentController.RestoreHistory":  "/history/restore",
	"DocumentController.Search":          "/docs/:key/search",
	"DocumentController.Upload":          "/api/upload",
	"HomeController.Index":               "/",
	"ItemsetsController.Index":           "/items",
	"ItemsetsController.List":            "/items/:key",
	"LabelController.Index":             "/tag/:key",
	"LabelController.List":              "/tags",
	"ManagerController.AttachClean":     "/manager/attach/clean",
	"ManagerController.AttachDelete":    "/manager/attach/delete",
	"ManagerController.AttachDetailed":  "/manager/attach/detailed/:id",
	"ManagerController.AttachList":      "/manager/attach/list",
	"ManagerController.Books":           "/manager/books",
	"ManagerController.ChangeMemberRole": "/manager/member/change-member-role",
	"ManagerController.Comments":        "/manager/comments",
	"ManagerController.Config":          "/manager/config",
	"ManagerController.CreateMember":    "/manager/member/create",
	"ManagerController.CreateToken":     "/manager/books/token",
	"ManagerController.DeleteBook":      "/manager/books/delete",
	"ManagerController.DeleteMember":    "/manager/member/delete",
	"ManagerController.EditBook":        "/manager/books/edit/:key",
	"ManagerController.EditMember":      "/manager/users/edit/:id",
	"ManagerController.Index":           "/manager",
	"ManagerController.Itemsets":        "/manager/itemsets",
	"ManagerController.ItemsetsDelete":  "/manager/itemsets/delete",
	"ManagerController.ItemsetsEdit":    "/manager/itemsets/edit",
	"ManagerController.LabelDelete":     "/manager/label/delete/:id",
	"ManagerController.LabelList":       "/manager/label/list",
	"ManagerController.PrivatelyOwned":  "/manager/books/open",
	"ManagerController.Setting":         "/manager/setting",
	"ManagerController.Team":            "/manager/team",
	"ManagerController.TeamBookAdd":     "/manager/team/book/add",
	"ManagerController.TeamBookDelete":  "/manager/team/book/delete",
	"ManagerController.TeamBookList":    "/manager/team/book/list/:id",
	"ManagerController.TeamChangeMemberRole": "/manager/team/member/change_role",
	"ManagerController.TeamCreate":      "/manager/team/create",
	"ManagerController.TeamDelete":      "/manager/team/delete",
	"ManagerController.TeamEdit":        "/manager/team/edit",
	"ManagerController.TeamMemberAdd":   "/manager/team/member/add",
	"ManagerController.TeamMemberDelete": "/manager/team/member/delete",
	"ManagerController.TeamMemberList":  "/manager/team/member/list/:id",
	"ManagerController.TeamSearchBook":  "/manager/team/book/search",
	"ManagerController.TeamSearchMember": "/manager/team/member/search",
	"ManagerController.Transfer":        "/manager/books/transfer",
	"ManagerController.UpdateMemberStatus": "/manager/member/update-member-status",
	"ManagerController.Users":           "/manager/users",
	"SearchController.Index":            "/search",
	"SearchController.IndexV2":          "/search-v2",
	"SearchController.SearchV2":         "/api/search-v2",
	"SearchController.User":             "/api/search/user/:key",
	"SettingController.Index":           "/setting",
	"SettingController.Password":        "/setting/password",
	"SettingController.Upload":          "/setting/upload",
	"TemplateController.Add":            "/api/template/add",
	"TemplateController.Delete":         "/api/template/remove",
	"TemplateController.Get":            "/api/template/get",
	"TemplateController.List":           "/api/template/list",
}

func urlForPath(endpoint string, values ...interface{}) string {
	path, ok := routeMap[endpoint]
	if !ok {
		return "/"
	}
	result := path
	var queryParams []string
	for i := 0; i+1 < len(values); i += 2 {
		if key, ok := values[i].(string); ok {
			if strings.HasPrefix(key, ":") {
				result = strings.Replace(result, key, fmt.Sprint(values[i+1]), 1)
			} else {
				queryParams = append(queryParams, url.QueryEscape(key)+"="+url.QueryEscape(fmt.Sprint(values[i+1])))
			}
		}
	}
	if len(queryParams) > 0 {
		result += "?" + strings.Join(queryParams, "&")
	}
	return result
}

// 重写生成URL的方法，加上完整的域名
func URLFor(endpoint string, values ...interface{}) string {
	baseUrl := GetDefaultString("baseurl", "")
	pathUrl := urlForPath(endpoint, values...)

	if baseUrl == "" {
		baseUrl = BaseUrl
	}
	if strings.HasPrefix(pathUrl, "http://") {
		return pathUrl
	}
	if strings.HasPrefix(pathUrl, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + pathUrl[1:]
	}
	if !strings.HasPrefix(pathUrl, "/") && !strings.HasSuffix(baseUrl, "/") {
		return baseUrl + "/" + pathUrl
	}
	return baseUrl + pathUrl
}

func URLForNotHost(endpoint string, values ...interface{}) string {
	baseUrl := GetDefaultString("baseurl", "")
	pathUrl := urlForPath(endpoint, values...)

	if baseUrl == "" {
		baseUrl = "/"
	}
	if strings.HasPrefix(pathUrl, "http://") {
		return pathUrl
	}
	if strings.HasPrefix(pathUrl, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + pathUrl[1:]
	}
	if !strings.HasPrefix(pathUrl, "/") && !strings.HasSuffix(baseUrl, "/") {
		return baseUrl + "/" + pathUrl
	}
	return baseUrl + pathUrl
}

func URLForWithCdnImage(p string) string {
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	cdn := GetDefaultString("cdnimg", "")
	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := GetDefaultString("baseurl", "/")

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

func URLForWithCdnCss(p string, v ...string) string {
	cdn := GetDefaultString("cdncss", "")
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	filePath := WorkingDir(p)

	if f, err := os.Stat(filePath); err == nil && !strings.Contains(p, "?") && len(v) > 0 && v[0] == "version" {
		p = p + fmt.Sprintf("?v=%s", f.ModTime().Format("20060102150405"))
	}
	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := GetDefaultString("baseurl", "/")

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

func URLForWithCdnJs(p string, v ...string) string {
	cdn := GetDefaultString("cdnjs", "")
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}

	filePath := WorkingDir(p)

	if f, err := os.Stat(filePath); err == nil && !strings.Contains(p, "?") && len(v) > 0 && v[0] == "version" {
		p = p + fmt.Sprintf("?v=%s", f.ModTime().Format("20060102150405"))
	}

	//如果没有设置cdn，则使用baseURL拼接
	if cdn == "" {
		baseUrl := GetDefaultString("baseurl", "/")

		if strings.HasPrefix(p, "/") && strings.HasSuffix(baseUrl, "/") {
			return baseUrl + p[1:]
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(baseUrl, "/") {
			return baseUrl + "/" + p
		}
		return baseUrl + p
	}
	if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
		return cdn + string(p[1:])
	}
	if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
		return cdn + "/" + p
	}
	return cdn + p
}

func WorkingDir(elem ...string) string {

	elems := append([]string{WorkingDirectory}, elem...)

	return filepath.Join(elems...)
}

// resolveBaseDir 按优先级返回程序的工作根目录：
// 1. 可执行文件所在目录（存在 conf/app.conf 则认为有效）
// 2. 当前工作目录
func resolveBaseDir() string {
	// 优先：可执行文件所在目录
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		// 处理 go run 时的临时路径（路径含 go-build）
		if !strings.Contains(exeDir, "go-build") {
			if _, err := os.Stat(filepath.Join(exeDir, "conf", "app.conf")); err == nil {
				return exeDir
			}
		}
	}
	// 兜底：当前工作目录
	if cwd, err := filepath.Abs("."); err == nil {
		return cwd
	}
	return "."
}

func init() {
	base := resolveBaseDir()
	ConfigurationFile = filepath.Join(base, "conf", "app.conf")
	WorkingDirectory = base
	LogFile = filepath.Join(base, "runtime", "logs")
}
