package middleware

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type platformAuthenticator interface {
	ResolveUser(context.Context, string) (*service.User, *service.SupabaseIdentity, error)
}

func NewJWTAuthMiddleware(auth *service.SupabaseAuthService, userService *service.UserService) JWTAuthMiddleware {
	return JWTAuthMiddleware(platformAuth(auth, userService, false))
}

func platformAuth(auth platformAuthenticator, activity userActivityToucher, admin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if admin && isWebSocketUpgradeRequest(c) {
			if websocketToken := extractJWTFromWebSocketSubprotocol(c); websocketToken != "" {
				token = websocketToken
			}
		}
		if token == "" {
			AbortWithError(c, 401, "UNAUTHORIZED", "Supabase access token is required")
			return
		}
		user, identity, err := auth.ResolveUser(c.Request.Context(), token)
		if err != nil {
			response.ErrorFrom(c, err)
			c.Abort()
			return
		}
		if admin && !identity.IsAdmin {
			AbortWithError(c, 403, "FORBIDDEN", "Admin access required")
			return
		}
		c.Set(string(ContextKeyUser), AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Set(ContextKeyAuthEmail, identity.Email)
		c.Set(ContextKeySessionID, identity.SessionID.String())
		c.Set("platform_user", user)
		c.Set("auth_method", service.AuditAuthMethodJWT)
		if activity != nil {
			activity.TouchLastActiveForUser(c.Request.Context(), user)
		}
		c.Next()
	}
}

type userActivityToucher interface {
	TouchLastActiveForUser(context.Context, *service.User)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func GetAuthenticatedUser(c *gin.Context) (*service.User, bool) {
	value, exists := c.Get("platform_user")
	user, ok := value.(*service.User)
	return user, exists && ok
}
