package routers

import (
	"net/http"
	"net/url"
	"regexp"

	sessions "github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	redisSession "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

// HeaderMiddleware 添加响应头
func HeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("MinDoc-Version", conf.VERSION)
		c.Header("MinDoc-Site", "https://www.iminho.me")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}

// SessionValidationMiddleware 校验 sessionId 格式
func SessionValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessname := conf.GetDefaultString("sessionname", "mindoc_id")
		sessionId, _ := c.Cookie(sessname)
		if sessionId != "" {
			if ok, err := regexp.MatchString(`^[a-zA-Z0-9]{32,512}$`, sessionId); !ok || err != nil {
				// 格式不匹配时不阻断，仅清除 cookie
				c.SetCookie(sessname, "", -1, "/", "", false, true)
			}
		}
		c.Next()
	}
}

// AuthRequired 要求登录的中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		member := session.Get(conf.LoginSessionName)

		if member == nil {
			if c.GetHeader("X-Requested-With") != "" || c.GetHeader("Accept") == "application/json" {
				c.JSON(http.StatusForbidden, gin.H{
					"errcode": 403,
					"message":  "请登录后再操作",
				})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/login?url="+url.PathEscape(conf.BaseUrl+c.Request.URL.RequestURI()))
			c.Abort()
			return
		}
		c.Next()
	}
}

// MCPAuthMiddleware MCP 认证中间件
func MCPAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// MCPProxyHandler MCP 代理处理器
func MCPProxyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// setupSession 配置 session 中间件
func setupSession(r *gin.Engine) {
	provider := conf.GetDefaultString("sessionprovider", "cookie")
	config := conf.GetDefaultString("sessionproviderconfig", "")
	sessionName := conf.GetDefaultString("sessionname", "mindoc_id")
	maxLifetime := conf.GetDefaultInt("sessiongcmaxlifetime", 3600)
	appKey := conf.GetAppKey()

	opts := sessions.Options{
		Path:     "/",
		MaxAge:   maxLifetime,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	var store sessions.Store

	switch provider {
	case "redis":
		if config == "" {
			config = "127.0.0.1:6379"
		}
		rs, err := redisSession.NewStore(10, "tcp", config, "", []byte(appKey))
		if err != nil {
			logger.Error("session redis store 初始化失败: ", err, "，回退到 cookie store")
			cs := cookie.NewStore([]byte(appKey))
			cs.Options(opts)
			store = cs
		} else {
			store = rs
		}
	default:
		// cookie / file / memory 统一使用 cookie store
		cs := cookie.NewStore([]byte(appKey))
		cs.Options(opts)
		store = cs
	}

	r.Use(sessions.Sessions(sessionName, store))
}
