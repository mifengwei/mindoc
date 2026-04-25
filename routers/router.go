package routers

// 旧版 Beego 路由已被 Gin 路由替代。
// 所有路由注册现在在 router_gin.go 中完成。
// CorsTransport 和辅助函数保留在此文件中供 bridge.go 使用。

import (
	"net/http"
	"strings"
)

type CorsTransport struct {
	http.RoundTripper
}

func (t *CorsTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Header.Set("Access-Control-Allow-Origin", "*")
	resp.Header.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	hs := ""
	for name, values := range resp.Header {
		hs = hs + name + ", "
		_ = values
	}
	hs = strings.TrimRight(hs, " ")
	hs = strings.TrimRight(hs, ",")
	resp.Header.Set("Access-Control-Allow-Headers", hs)
	resp.Header.Del("Mindoc-Version")
	resp.Header.Del("Mindoc-Site")
	resp.Header.Del("Server")
	resp.Header.Del("X-Xss-Protection")
	return resp, nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
