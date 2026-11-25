package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, settings *service.SettingService, limiter *middleware.PanelRateLimiter) {
	v1.GET("/auth/oauth/wechat/payment/start", h.Auth.WeChatPaymentOAuthStart)
	v1.GET("/auth/oauth/wechat/payment/callback", h.Auth.WeChatPaymentOAuthCallback)
	public := v1.Group("/settings", limiter.PublicIP())
	public.GET("/public", h.Setting.GetPublicSettings)
	public.GET("/email-unsubscribe", h.Setting.UnsubscribeNotificationEmail)
	authenticated := v1.Group("", gin.HandlerFunc(jwtAuth), middleware.BackendModeUserGuard(settings), limiter.Global())
	authenticated.GET("/auth/me", h.Auth.GetCurrentUser)
}
