package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNativeIdentityIssuingAndMutationRoutesAreAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	handlers := &handler.Handlers{Auth: &handler.AuthHandler{}, User: &handler.UserHandler{}, Setting: &handler.SettingHandler{}, Admin: &handler.AdminHandlers{}}
	limiter := middleware.NewPanelRateLimiter(nil, nil)
	reject := func(c *gin.Context) { c.AbortWithStatus(401) }
	RegisterAuthRoutes(v1, handlers, middleware.JWTAuthMiddleware(reject), nil, limiter)
	RegisterUserRoutes(v1, handlers, middleware.JWTAuthMiddleware(reject), middleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() }), nil, limiter)
	RegisterAdminRoutes(v1, handlers, middleware.AdminAuthMiddleware(reject), middleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() }), nil, limiter)
	for _, item := range []struct{ method, path string }{
		{"POST", "/auth/login"}, {"POST", "/auth/register"}, {"POST", "/auth/refresh"}, {"POST", "/auth/logout"},
		{"POST", "/auth/forgot-password"}, {"POST", "/auth/reset-password"}, {"POST", "/auth/revoke-all-sessions"},
		{"POST", "/auth/passkey/login/begin"}, {"POST", "/auth/passkey/login/finish"}, {"POST", "/auth/login/2fa"},
		{"GET", "/auth/oauth/oidc/start"}, {"GET", "/auth/oauth/wechat/start"}, {"GET", "/auth/oauth/github/start"},
		{"GET", "/auth/oauth/google/start"}, {"GET", "/auth/oauth/linuxdo/start"}, {"GET", "/auth/oauth/dingtalk/start"},
		{"POST", "/auth/oauth/pending/exchange"}, {"POST", "/auth/oauth/pending/create-account"}, {"POST", "/auth/oauth/bind-token"},
		{"PUT", "/user/password"}, {"POST", "/user/account-bindings/email"}, {"POST", "/user/auth-identities/bind/start"},
		{"GET", "/user/totp/status"}, {"POST", "/user/totp/step-up"}, {"POST", "/user/passkeys/register/begin"},
		{"GET", "/user/mcp/config"}, {"GET", "/mcp"}, {"POST", "/mcp/oauth/register"}, {"POST", "/mcp/oauth/token"},
		{"POST", "/admin/users"}, {"DELETE", "/admin/users/12"}, {"POST", "/admin/users/12/auth-identities"}, {"POST", "/admin/users/batch-disable"},
		{"GET", "/admin/settings/admin-api-key"}, {"POST", "/admin/settings/admin-api-key/regenerate"},
	} {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(item.method, "/api/v1"+item.path, nil))
		require.Equal(t, 404, res.Code, item.method+" "+item.path)
	}
	for _, path := range []string{"/api/v1/auth/me", "/api/v1/user/profile", "/api/v1/admin/users", "/api/v1/admin/accounts"} {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 401, res.Code, path)
	}
}
