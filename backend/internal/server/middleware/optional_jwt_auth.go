package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func NewOptionalJWTAuthMiddleware(auth *service.SupabaseAuthService, userService *service.UserService) OptionalJWTAuthMiddleware {
	return optionalPlatformAuth(auth, userService)
}

func optionalPlatformAuth(auth platformAuthenticator, activity userActivityToucher) OptionalJWTAuthMiddleware {
	strict := platformAuth(auth, activity, false)
	return OptionalJWTAuthMiddleware(func(c *gin.Context) {
		if strings.TrimSpace(c.GetHeader("Authorization")) == "" {
			c.Next()
			return
		}
		strict(c)
	})
}
