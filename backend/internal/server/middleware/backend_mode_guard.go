package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// BackendModeUserGuard blocks non-admin users from accessing user routes when backend mode is enabled.
// Must be placed AFTER JWT auth middleware so that the user role is available in context.
func BackendModeUserGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsBackendModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if role == "admin" {
			c.Next()
			return
		}
		response.Forbidden(c, "Backend mode is active. User self-service is disabled.")
		c.Abort()
	}
}
