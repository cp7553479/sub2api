package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ClientRequestMetadataContext records the trusted client IP and User-Agent for request audit.
// IP 取值与 API Key IP 限制共用转发 IP 开关：开启时旧版原始转发头逻辑
// 接管解析，关闭时使用 Gin 的 server.trusted_proxies 可信代理链。
func ClientRequestMetadataContext(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		forwardedIPSettings := cfg.ForwardedClientIPSettings()
		ip.SetForwardedIPSettings(c, forwardedIPSettings.TrustForwardedIP, forwardedIPSettings.Headers)
		userAgent := normalizePersistentText(c.Request.UserAgent(), maxPersistentUserAgentBytes)
		c.Request.Header.Set("User-Agent", userAgent)
		binding := &service.ClientRequestMetadata{
			IP:        ip.GetSecurityClientIP(c, forwardedIPSettings.TrustForwardedIP),
			UserAgent: userAgent,
		}
		c.Request = c.Request.WithContext(service.WithClientRequestMetadata(c.Request.Context(), binding))
		c.Next()
	}
}

// SecurityClientIP 返回当前请求用于安全敏感记录（审计日志等）的客户端 IP。
// 与会话绑定、API Key IP 限制共用同一套客户端 IP 来源。
func SecurityClientIP(c *gin.Context) string {
	if binding := service.ClientRequestMetadataFromContext(c.Request.Context()); binding != nil &&
		strings.TrimSpace(binding.IP) != "" {
		return binding.IP
	}
	return ip.GetTrustedClientIP(c)
}
