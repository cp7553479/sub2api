package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func NewAdminAuthMiddleware(auth *service.SupabaseAuthService, userService *service.UserService) AdminAuthMiddleware {
	return AdminAuthMiddleware(platformAuth(auth, userService, true))
}

func isWebSocketUpgradeRequest(c *gin.Context) bool {
	return strings.EqualFold(c.GetHeader("Upgrade"), "websocket") && strings.Contains(strings.ToLower(c.GetHeader("Connection")), "upgrade")
}

func extractJWTFromWebSocketSubprotocol(c *gin.Context) string {
	for _, part := range strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",") {
		if token, ok := strings.CutPrefix(strings.TrimSpace(part), "jwt."); ok {
			return token
		}
	}
	return ""
}
