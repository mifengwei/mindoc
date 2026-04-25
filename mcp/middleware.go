package mcp

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mindoc-org/mindoc/conf"
)

// AuthMiddleware 返回一个 Gin 中间件，用于验证 MCP 请求中的认证令牌
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		presetMcpApiKey := conf.GetDefaultString("mcp_api_key", "")
		mcpApiKeyParamValue := c.Query("api_key")
		if presetMcpApiKey != "" && presetMcpApiKey != mcpApiKeyParamValue {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid mcp authorization key"})
			return
		}

		// Add mcp_api_key to request context
		ctx := context.WithValue(c.Request.Context(), "mcp_api_key", mcpApiKeyParamValue)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AuthMiddlewareBeego 兼容旧的 Beego 调用方式
func AuthMiddlewareBeego(w http.ResponseWriter, r *http.Request) {
	presetMcpApiKey := conf.GetDefaultString("mcp_api_key", "")
	mcpApiKeyParamValue := r.URL.Query().Get("api_key")
	if presetMcpApiKey != "" && presetMcpApiKey != mcpApiKeyParamValue {
		http.Error(w, "Missing or invalid mcp authorization key", http.StatusUnauthorized)
		return
	}
	ctx := context.WithValue(r.Context(), "mcp_api_key", mcpApiKeyParamValue)
	_ = ctx
}
