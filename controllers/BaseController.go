package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/i18n"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/models"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/utils"
)

// AbortPanic 用于 JsonResult 中止控制器方法执行
type AbortPanic struct{}

type BaseController struct {
	Gin                  *gin.Context
	Ctx                  *BeegoCtx
	Member               *models.Member
	Option               map[string]string
	EnableAnonymous      bool
	EnableDocumentHistory bool
	Lang                 string
	Data                 map[string]interface{}
	TplName              string
	ViewPath             string
	EnableXSRF           bool
	xsrfToken            string
	controllerName       string
	actionName           string
	CruSession           *BeegoSession
}

// BeegoCtx 提供与 Beego context.Context 兼容的接口
type BeegoCtx struct {
	Input          *BeegoInput
	Output         *BeegoOutput
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	gin            *gin.Context
}

func newBeegoCtx(c *gin.Context) *BeegoCtx {
	// 读取请求体（供 RequestBody 字段使用）
	var body []byte
	if c.Request != nil && c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
	}

	ctx := &BeegoCtx{
		Request:        c.Request,
		ResponseWriter: c.Writer,
		gin:            c,
	}
	ctx.Input = &BeegoInput{gin: c, RequestBody: body}
	ctx.Output = &BeegoOutput{gin: c, ctx: ctx}
	return ctx
}

// BeegoInput 模拟 Beego Input 接口
type BeegoInput struct {
	gin         *gin.Context
	RequestBody []byte
}

func (i *BeegoInput) IsPost() bool {
	return i.gin.Request.Method == "POST"
}

func (i *BeegoInput) IsGet() bool {
	return i.gin.Request.Method == "GET"
}

func (i *BeegoInput) IsAjax() bool {
	return i.gin.GetHeader("X-Requested-With") != ""
}

func (i *BeegoInput) Param(key string) string {
	if strings.HasPrefix(key, ":") {
		key = key[1:]
	}
	return i.gin.Param(key)
}

func (i *BeegoInput) Query(key string) string {
	return i.gin.Query(key)
}

func (i *BeegoInput) UserAgent() string {
	return i.gin.GetHeader("User-Agent")
}

func (i *BeegoInput) Scheme() string {
	if i.gin.Request.TLS != nil || i.gin.GetHeader("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

func (i *BeegoInput) IP() string {
	return i.gin.ClientIP()
}

func (i *BeegoInput) Method() string {
	return i.gin.Request.Method
}

func (i *BeegoInput) Header(key string) string {
	return i.gin.GetHeader(key)
}

func (i *BeegoInput) Cookie(key string) string {
	val, _ := i.gin.Cookie(key)
	return val
}

func (i *BeegoInput) Session(key interface{}) interface{} {
	session := sessions.Default(i.gin)
	return session.Get(fmt.Sprintf("%v", key))
}

func (i *BeegoInput) Referer() string {
	return i.gin.GetHeader("Referer")
}

// BeegoOutput 模拟 Beego Output 接口
type BeegoOutput struct {
	gin *gin.Context
	ctx *BeegoCtx
}

func (o *BeegoOutput) JSON(data interface{}, hasIndent bool, encoding bool) error {
	o.gin.JSON(http.StatusOK, data)
	return nil
}

func (o *BeegoOutput) Download(file string, filenames ...string) {
	if len(filenames) > 0 {
		o.gin.FileAttachment(file, filenames[0])
	} else {
		o.gin.File(file)
	}
}

func (o *BeegoOutput) Body(data []byte) {
	_, _ = o.gin.Writer.Write(data)
}

// BeegoSession 模拟 Beego Session 接口
type BeegoSession struct {
	gin *gin.Context
}

func newBeegoSession(c *gin.Context) *BeegoSession {
	return &BeegoSession{gin: c}
}

func (s *BeegoSession) Get(ctx context.Context, key interface{}) interface{} {
	session := sessions.Default(s.gin)
	return session.Get(fmt.Sprintf("%v", key))
}

func (s *BeegoSession) Set(ctx context.Context, key interface{}, value interface{}) error {
	session := sessions.Default(s.gin)
	session.Set(fmt.Sprintf("%v", key), value)
	return session.Save()
}

func (s *BeegoSession) Delete(ctx context.Context, key interface{}) error {
	session := sessions.Default(s.gin)
	session.Delete(fmt.Sprintf("%v", key))
	return session.Save()
}

func (s *BeegoSession) SessionID(ctx context.Context) string {
	return ""
}

func (s *BeegoSession) Flush() error {
	return nil
}

type CookieRemember struct {
	MemberId int
	Account  string
	Time     time.Time
}

// Prepare 预处理
func (c *BaseController) Prepare() {
	// 初始化 Beego 兼容上下文
	if c.Gin != nil && c.Ctx == nil {
		c.Ctx = newBeegoCtx(c.Gin)
		c.CruSession = newBeegoSession(c.Gin)
	}

	c.Data = make(map[string]interface{})
	c.Data["SiteName"] = "MinDoc"
	c.Data["Member"] = models.NewMember()

	c.EnableAnonymous = false
	c.EnableDocumentHistory = false

	if member, ok := c.GetSession(conf.LoginSessionName).(models.Member); ok && member.MemberId > 0 {
		c.Member = &member
		c.Data["Member"] = c.Member
	} else {
		var remember CookieRemember
		if cookie, ok := c.GetSecureCookie(conf.GetAppKey(), "login"); ok {
			if err := utils.Decode(cookie, &remember); err == nil {
				if member, err := models.NewMember().Find(remember.MemberId); err == nil {
					c.Member = member
					c.Data["Member"] = member
					c.SetMember(*member)
				}
			}
		}
	}
	conf.BaseUrl = c.BaseUrl()
	c.Data["BaseUrl"] = c.BaseUrl()

	if options, err := models.NewOption().All(); err == nil {
		c.Option = make(map[string]string, len(options))
		for _, item := range options {
			c.Data[item.OptionName] = item.OptionValue
			c.Option[item.OptionName] = item.OptionValue
		}
		c.EnableAnonymous = strings.EqualFold(c.Option["ENABLE_ANONYMOUS"], "true")
		c.EnableDocumentHistory = strings.EqualFold(c.Option["ENABLE_DOCUMENT_HISTORY"], "true")
	}
	c.Data["HighlightStyle"] = conf.GetDefaultString("highlight_style", "github")

	if c.ViewPath == "" {
		c.ViewPath = conf.WorkingDir("views")
	}
	if b, err := os.ReadFile(filepath.Join(c.ViewPath, "widgets", "scripts.tpl")); err == nil {
		c.Data["Scripts"] = template.HTML(string(b))
	}

	c.SetLang()
}

func (c *BaseController) isUserLoggedIn() bool {
	return c.Member != nil && c.Member.MemberId > 0
}

// SetMember 获取或设置当前登录用户信息
func (c *BaseController) SetMember(member models.Member) {
	if member.MemberId <= 0 {
		c.DelSession(conf.LoginSessionName)
		c.DelSession("uid")
		c.DestroySession()
	} else {
		c.SetSession(conf.LoginSessionName, member)
		c.SetSession("uid", member.MemberId)
	}
}

// JsonResult 响应 json 结果
func (c *BaseController) JsonResult(errCode int, errMsg string, data ...interface{}) {
	jsonData := make(map[string]interface{}, 3)
	jsonData["errcode"] = errCode
	jsonData["message"] = errMsg
	if len(data) > 0 && data[0] != nil {
		jsonData["data"] = data[0]
	}

	returnJSON, err := json.Marshal(jsonData)
	if err != nil {
		logger.Error(err)
	}

	c.Gin.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Gin.Writer.Header().Set("Cache-Control", "no-cache, no-store")
	_, err = c.Gin.Writer.Write(returnJSON)
	if err != nil {
		logger.Error(err)
	}
	c.Gin.Abort()
	panic(AbortPanic{})
}

// CheckJsonError 如果错误不为空，则响应错误信息到浏览器
func (c *BaseController) CheckJsonError(code int, err error) {
	if err == nil {
		return
	}
	jsonData := make(map[string]interface{}, 3)
	jsonData["errcode"] = code
	jsonData["message"] = err.Error()

	returnJSON, err := json.Marshal(jsonData)
	if err != nil {
		logger.Error(err)
	}

	c.Gin.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Gin.Writer.Header().Set("Cache-Control", "no-cache, no-store")
	_, err = c.Gin.Writer.Write(returnJSON)
	if err != nil {
		logger.Error(err)
	}
	c.Gin.Abort()
}

// ExecuteViewPathTemplate 执行指定的模板并返回执行结果
func (c *BaseController) ExecuteViewPathTemplate(tplName string, data interface{}) (string, error) {
	var buf bytes.Buffer
	tplPath := filepath.Join(c.ViewPath, tplName)
	tmpl, err := template.New(tplName).Funcs(template.FuncMap{
		"config":     models.GetOptionValue,
		"cdn":        cdnFunc,
		"cdnjs":      conf.URLForWithCdnJs,
		"cdncss":     conf.URLForWithCdnCss,
		"cdnimg":     conf.URLForWithCdnImage,
		"urlfor":     urlForFunc,
		"conf":       conf.CONF,
		"date_format": func(t time.Time, format string) string { return t.Local().Format(format) },
		"i18n":       i18n.Tr,
		"str2html":   func(s string) template.HTML { return template.HTML(s) },
		"html2str":   HTML2Str,
		"htmlquote":  Htmlquote,
		"htmlunquote": Htmlunquote,
		"htfn":       Htmlfilter,
	}).ParseFiles(tplPath)
	if err != nil {
		return "", err
	}
	name := filepath.Base(tplPath)
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *BaseController) BaseUrl() string {
	baseUrl := conf.GetDefaultString("baseurl", "")
	if baseUrl != "" {
		baseUrl = strings.TrimSuffix(baseUrl, "/")
	} else {
		scheme := "http"
		if c.Gin.Request.TLS != nil || c.Gin.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		baseUrl = scheme + "://" + c.Gin.Request.Host
	}
	return baseUrl
}

// ShowErrorPage 显示错误信息页面
func (c *BaseController) ShowErrorPage(errCode int, errMsg string) {
	c.TplName = "errors/error.tpl"
	c.Data["ErrorMessage"] = errMsg
	c.Data["ErrorCode"] = errCode

	exeData := map[string]interface{}{
		"ErrorMessage": errMsg,
		"ErrorCode":    errCode,
		"BaseUrl":      conf.BaseUrl,
		"Lang":         c.Lang,
	}

	htmlContent, err := c.ExecuteViewPathTemplate("errors/error.tpl", exeData)
	if err != nil {
		c.Gin.String(http.StatusInternalServerError, "Internal Server Error")
		c.Gin.Abort()
		return
	}
	if errCode >= 200 && errCode <= 510 {
		c.Gin.String(errCode, htmlContent)
	} else {
		c.Gin.String(http.StatusInternalServerError, htmlContent)
	}
	c.Gin.Abort()
	panic(AbortPanic{})
}

func (c *BaseController) CheckErrorResult(code int, err error) {
	if err != nil {
		c.ShowErrorPage(code, err.Error())
	}
}

func (c *BaseController) SetLang() {
	hasCookie := false
	lang := c.GetString("lang")
	if len(lang) == 0 {
		lang, _ = c.Gin.Cookie("lang")
		hasCookie = true
	}
	if len(lang) == 0 || !i18n.IsExist(lang) {
		if v, ok := c.Data["language"]; ok {
			lang = v.(string)
		} else {
			lang = conf.GetDefaultString("default_lang", "zh-CN")
		}
	}
	if !hasCookie {
		c.Gin.SetCookie("lang", lang, 1<<31-1, "/", "", false, false)
	}
	c.Data["Lang"] = lang
	c.Lang = lang
}

// ========== 参数获取方法 ==========

// GetString 获取请求参数（同时支持 query/form/path 参数）
func (c *BaseController) GetString(key string, def ...string) string {
	// 路径参数（如 :key, :id）
	if strings.HasPrefix(key, ":") {
		if val := c.Gin.Param(key[1:]); val != "" {
			return val
		}
	}
	val := c.Gin.Query(key)
	if val == "" {
		val = c.Gin.PostForm(key)
	}
	if val == "" && len(def) > 0 {
		return def[0]
	}
	return val
}

// GetInt 获取整数参数
func (c *BaseController) GetInt(key string, def ...int) (int, error) {
	str := c.GetString(key)
	if str == "" {
		if len(def) > 0 {
			return def[0], nil
		}
		return 0, fmt.Errorf("param %s is empty", key)
	}
	var val int
	_, err := fmt.Sscanf(str, "%d", &val)
	if err != nil {
		if len(def) > 0 {
			return def[0], nil
		}
		return 0, err
	}
	return val, nil
}

// GetInt64 获取 int64 参数
func (c *BaseController) GetInt64(key string, def ...int64) (int64, error) {
	str := c.GetString(key)
	if str == "" {
		if len(def) > 0 {
			return def[0], nil
		}
		return 0, fmt.Errorf("param %s is empty", key)
	}
	var val int64
	_, err := fmt.Sscanf(str, "%d", &val)
	if err != nil {
		if len(def) > 0 {
			return def[0], nil
		}
		return 0, err
	}
	return val, nil
}

// GetBool 获取布尔参数
func (c *BaseController) GetBool(key string, def ...bool) (bool, error) {
	str := c.GetString(key)
	if str == "" {
		if len(def) > 0 {
			return def[0], nil
		}
		return false, fmt.Errorf("param %s is empty", key)
	}
	return strings.EqualFold(str, "true") || str == "1", nil
}

// ========== Session 方法 ==========

// GetSession 获取 session
func (c *BaseController) GetSession(key interface{}) interface{} {
	session := sessions.Default(c.Gin)
	return session.Get(fmt.Sprintf("%v", key))
}

// SetSession 设置 session
func (c *BaseController) SetSession(key interface{}, value interface{}) {
	session := sessions.Default(c.Gin)
	session.Set(fmt.Sprintf("%v", key), value)
	_ = session.Save()
}

// DelSession 删除 session
func (c *BaseController) DelSession(key interface{}) {
	session := sessions.Default(c.Gin)
	session.Delete(fmt.Sprintf("%v", key))
	_ = session.Save()
}

// DestroySession 销毁 session
func (c *BaseController) DestroySession() {
	session := sessions.Default(c.Gin)
	session.Clear()
	_ = session.Save()
}

// ========== Cookie 方法 ==========

// GetSecureCookie 获取安全 Cookie
func (c *BaseController) GetSecureCookie(secret, key string) (string, bool) {
	val, err := c.Gin.Cookie(key)
	if err != nil || val == "" {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return val, true
	}
	return string(decoded), true
}

// SetSecureCookie 设置安全 Cookie
func (c *BaseController) SetSecureCookie(secret, name, value string, others ...interface{}) {
	encoded := base64.StdEncoding.EncodeToString([]byte(value))
	maxAge := 0
	path := "/"
	if len(others) > 0 {
		if v, ok := others[0].(int); ok {
			maxAge = v
		}
	}
	if len(others) > 1 {
		if v, ok := others[1].(string); ok {
			path = v
		}
	}
	c.Gin.SetCookie(name, encoded, maxAge, path, "", false, true)
}

// ========== 流程控制方法 ==========

// StopRun 停止执行后续逻辑
func (c *BaseController) StopRun() {
	c.Gin.Abort()
}

// Abort 中止请求
func (c *BaseController) Abort(code string) {
	statusMap := map[string]int{
		"401": http.StatusUnauthorized,
		"403": http.StatusForbidden,
		"404": http.StatusNotFound,
		"500": http.StatusInternalServerError,
	}
	if status, ok := statusMap[code]; ok {
		c.Gin.AbortWithStatus(status)
	} else {
		c.Gin.AbortWithStatus(http.StatusInternalServerError)
	}
}

// CustomAbort 自定义中止请求
func (c *BaseController) CustomAbort(status int, body string) {
	c.Gin.String(status, body)
	c.Gin.Abort()
}

// Redirect 重定向
func (c *BaseController) Redirect(url string, code int) {
	c.Gin.Redirect(code, url)
	c.Gin.Abort()
	panic(AbortPanic{})
}

// IsAjax 判断是否是 AJAX 请求
func (c *BaseController) IsAjax() bool {
	return c.Gin.GetHeader("X-Requested-With") != ""
}

// GetControllerAndAction 获取控制器名和动作名
func (c *BaseController) GetControllerAndAction() (string, string) {
	return c.controllerName, c.actionName
}

// SetControllerAndAction 设置控制器名和动作名
func (c *BaseController) SetControllerAndAction(controller, action string) {
	c.controllerName = controller
	c.actionName = action
}

// ========== XSRF 方法 ==========

// XSRFToken 获取 XSRF Token
func (c *BaseController) XSRFToken() string {
	if c.xsrfToken != "" {
		return c.xsrfToken
	}
	session := sessions.Default(c.Gin)
	token := session.Get("_xsrf_token")
	if token == nil || token == "" {
		token = fmt.Sprintf("%x", time.Now().UnixNano())
		session.Set("_xsrf_token", token)
		if err := session.Save(); err != nil {
			logger.Errorf("[XSRFToken] Save failed: %v", err)
		}
	}
	c.xsrfToken = token.(string)
	return c.xsrfToken
}

// XSRFFormHTML 生成 XSRF 隐藏表单域
func (c *BaseController) XSRFFormHTML() template.HTML {
	token := c.XSRFToken()
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="_xsrf" value="%s" />`, token))
}

// ========== 文件上传方法 ==========

// GetFile 获取上传文件
func (c *BaseController) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	fileHeader, err := c.Gin.FormFile(key)
	if err != nil {
		return nil, nil, err
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, nil, err
	}
	return file, fileHeader, nil
}

// GetFiles 获取多个上传文件
func (c *BaseController) GetFiles(key string) ([]*multipart.FileHeader, error) {
	form, err := c.Gin.MultipartForm()
	if err != nil {
		return nil, err
	}
	return form.File[key], nil
}

// SaveToFile 保存上传文件
func (c *BaseController) SaveToFile(fromFile, toFile string) error {
	fileHeader, err := c.Gin.FormFile(fromFile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(toFile), 0755); err != nil {
		return err
	}

	dst, err := os.Create(toFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	return err
}

// ParseForm 解析表单
func (c *BaseController) ParseForm(obj interface{}) error {
	return c.Gin.ShouldBind(obj)
}

// ========== 模板渲染辅助 ==========

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

func urlForFunc(endpoint string, values ...interface{}) string {
	routeMap := map[string]string{
		"HomeController.Index":        "/",
		"AccountController.Login":     "/login",
		"AccountController.Logout":    "/logout",
		"ManagerController.Index":     "/manager",
		"BookController.Index":        "/book",
		"DocumentController.Index":    "/docs/:key",
		"DocumentController.Read":     "/docs/:key/:id",
		"SearchController.Index":      "/search",
		"BlogController.List":         "/blogs",
		"SettingController.Index":     "/setting",
	}
	path, ok := routeMap[endpoint]
	if !ok {
		return "/"
	}
	result := path
	baseUrl := conf.BaseUrl
	if baseUrl == "" {
		return result
	}
	if strings.HasPrefix(result, "/") && strings.HasSuffix(baseUrl, "/") {
		return baseUrl + result[1:]
	}
	return baseUrl + result
}

// HTML2Str 将 HTML 转换为纯文本
func HTML2Str(s string) string {
	return strings.TrimSpace(s)
}

// Htmlquote HTML 转义
func Htmlquote(s string) string {
	return template.HTMLEscapeString(s)
}

// Htmlunquote HTML 反转义
func Htmlunquote(s string) string {
	return s
}

// Htmlfilter 过滤 HTML 标签
func Htmlfilter(s string) string {
	return s
}

// LoggedIn 记录登录日志
func (c *BaseController) LoggedIn(isMobile bool) {
	if c.Member != nil {
		logs := models.NewLogger()
		logs.MemberId = c.Member.MemberId
		logs.Category = "operate"
		logs.Content = "登录系统"
		_ = logs.Add()
	}
}
