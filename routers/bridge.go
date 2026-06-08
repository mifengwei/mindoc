package routers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mindoc-org/mindoc/controllers"
	"github.com/mindoc-org/mindoc/pkg/logger"
)

// wrapAny 创建一个 Gin HandlerFunc，将请求桥接到控制器
func wrapAny(controllerInterface interface{}, methodName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(controllers.AbortPanic); ok {
					return
				}
				panic(r)
			}
		}()

		ctrlType := reflect.TypeOf(controllerInterface)
		ctrlValue := reflect.New(ctrlType.Elem())

		base := ctrlValue.Elem().FieldByName("BaseController")
		if base.IsValid() {
			if f := base.FieldByName("Gin"); f.IsValid() && f.CanSet() {
				f.Set(reflect.ValueOf(c))
			}
		}

		if m := ctrlValue.MethodByName("Prepare"); m.IsValid() {
			m.Call(nil)
		}

		if c.IsAborted() {
			return
		}

		// 将 ControllerName 和 ActionName 注入 Data，模板中使用 {{.ControllerName}} / {{.ActionName}}
		controllerName := ctrlType.Elem().Name() // e.g. "BookController"
		if base.IsValid() {
			if dataField := base.FieldByName("Data"); dataField.IsValid() && !dataField.IsNil() {
				dataField.SetMapIndex(reflect.ValueOf("ControllerName"), reflect.ValueOf(controllerName))
				dataField.SetMapIndex(reflect.ValueOf("ActionName"), reflect.ValueOf(methodName))
			}
			if f := base.FieldByName("controllerName"); f.IsValid() && f.CanSet() {
				f.SetString(controllerName)
			}
			if f := base.FieldByName("actionName"); f.IsValid() && f.CanSet() {
				f.SetString(methodName)
			}
		}

		if m := ctrlValue.MethodByName(methodName); m.IsValid() {
			m.Call(nil)
		} else {
			c.String(http.StatusNotFound, "method %s not found", methodName)
			return
		}

		if c.IsAborted() || c.Writer.Written() {
			return
		}

		// 模板渲染
		renderTemplate(c, ctrlValue)
	}
}

func renderTemplate(c *gin.Context, ctrlValue reflect.Value) {
	base := ctrlValue.Elem().FieldByName("BaseController")
	if !base.IsValid() {
		return
	}

	tplName := base.FieldByName("TplName").String()
	if tplName == "" {
		return
	}

	dataField := base.FieldByName("Data")
	if !dataField.IsValid() || dataField.IsNil() {
		return
	}

	dataMap := make(gin.H)
	iter := dataField.MapRange()
	for iter.Next() {
		dataMap[iter.Key().String()] = iter.Value().Interface()
	}

	c.HTML(http.StatusOK, tplName, dataMap)
}

func corsAnywhereHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, _ := url.PathUnescape(c.Query("url"))
		if len(u) > 0 && strings.HasPrefix(u, "http") {
			target, _ := url.Parse(u)
			if target.Path == c.Request.URL.Path {
				c.String(http.StatusOK, "")
				return
			}

			logger.Error("target: ", target)
			reverseProxy := httputil.NewSingleHostReverseProxy(target)
			reverseProxy.Director = func(req *http.Request) {
				for name, values := range c.Request.Header {
					for _, value := range values {
						req.Header.Set(name, value)
					}
				}
				req.Header.Add("X-Forwarded-Host", req.Host)
				req.Header.Add("X-Origin-Host", target.Host)
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host

				proxyPath := target.Path
				if strings.HasSuffix(proxyPath, "/") && len(proxyPath) > 1 {
					proxyPath = proxyPath[:len(proxyPath)-1]
				}
				req.URL.Path = proxyPath
			}
			reverseProxy.Transport = &CorsTransport{http.DefaultTransport}
			reverseProxy.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		} else {
			c.String(http.StatusBadRequest, "400 Bad Request")
		}
	}
}
