package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type resolvedPlatformFixture struct {
	admin bool
	err   error
	token string
	calls int
}

func (r *resolvedPlatformFixture) ResolveUser(_ context.Context, token string) (*service.User, *service.SupabaseIdentity, error) {
	r.calls++
	r.token = token
	if r.err != nil {
		return nil, nil, r.err
	}
	role := service.RoleUser
	if r.admin {
		role = service.RoleAdmin
	}
	return &service.User{ID: 12, Concurrency: 3, Role: role}, &service.SupabaseIdentity{UserID: uuid.New(), SessionID: uuid.New(), Email: "platform@example.test", IsAdmin: r.admin}, nil
}

func TestPlatformAdminHTTPAndWebSocketUseSameResolver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, transport := range []string{"Bearer", "WebSocket", "WebSocketBearer"} {
		t.Run(transport, func(t *testing.T) {
			auth := &resolvedPlatformFixture{admin: true}
			router := gin.New()
			router.GET("/admin", platformAuth(auth, nil, true), func(c *gin.Context) {
				subject, ok := GetAuthSubjectFromContext(c)
				require.True(t, ok)
				require.Equal(t, int64(12), subject.UserID)
				role, ok := GetUserRoleFromContext(c)
				require.True(t, ok)
				require.Equal(t, service.RoleAdmin, role)
				require.Equal(t, "platform@example.test", c.GetString(ContextKeyAuthEmail))
				require.NotEmpty(t, c.GetString(ContextKeySessionID))
				c.Status(204)
			})
			request := func() *http.Request {
				req := httptest.NewRequest("GET", "/admin", nil)
				if transport == "Bearer" || transport == "WebSocketBearer" {
					req.Header.Set("Authorization", "Bearer platform-session")
					if transport == "WebSocketBearer" {
						req.Header.Set("Connection", "Upgrade")
						req.Header.Set("Upgrade", "websocket")
					}
				} else {
					req.Header.Set("Connection", "Upgrade")
					req.Header.Set("Upgrade", "websocket")
					req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt.platform-session")
				}
				return req
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, request())
			require.Equal(t, 204, res.Code)
			require.Equal(t, "platform-session", auth.token)
			auth.admin = false
			res = httptest.NewRecorder()
			router.ServeHTTP(res, request())
			require.Equal(t, 403, res.Code)
			auth.err = service.ErrPlatformSession
			res = httptest.NewRecorder()
			router.ServeHTTP(res, request())
			require.Equal(t, 401, res.Code)
			auth.err = infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "offline")
			res = httptest.NewRecorder()
			router.ServeHTTP(res, request())
			require.Equal(t, 503, res.Code)
			require.Equal(t, 4, auth.calls)
		})
	}
}

func TestPlatformAdminDoesNotAcceptLegacyAdministratorKeys(t *testing.T) {
	for _, credential := range []string{"legacy-secret", "adm_sc_old-key"} {
		auth := &resolvedPlatformFixture{admin: true}
		router := gin.New()
		router.GET("/admin", platformAuth(auth, nil, true), func(c *gin.Context) { t.Fatal("legacy key authorized") })
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("x-api-key", credential)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, 401, res.Code)
		require.Zero(t, auth.calls)
	}
}

func TestPlatformAdminAuditUsesCurrentSubjectAndSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := &resolvedPlatformFixture{admin: true}
	repository := &auditCaptureRepository{}
	auditService := service.NewAuditLogService(repository, nil)
	auditService.Start()
	router := gin.New()
	var sessionID string
	router.POST("/api/v1/admin/accounts", platformAuth(auth, nil, true), gin.HandlerFunc(NewAuditLogMiddleware(auditService)), func(c *gin.Context) {
		sessionID = c.GetString(ContextKeySessionID)
		SetAuditExtra(c, map[string]any{"session_id": "untrusted", "result": "success"})
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", nil)
	req.Header.Set("Authorization", "Bearer platform-session-for-audit")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	auditService.Stop()
	require.Equal(t, http.StatusNoContent, res.Code)
	repository.mu.Lock()
	defer repository.mu.Unlock()
	require.Len(t, repository.logs, 1)
	entry := repository.logs[0]
	require.EqualValues(t, 12, *entry.ActorUserID)
	require.Equal(t, service.RoleAdmin, entry.ActorRole)
	require.Equal(t, "platform@example.test", entry.ActorEmail)
	require.Equal(t, sessionID, entry.Extra["session_id"])
	require.NotEqual(t, "untrusted", entry.Extra["session_id"])
	require.NotContains(t, entry.CredentialMasked, "platform-session-for-audit")
}

func TestPlatformBrowserAuthenticationFailsClosed(t *testing.T) {
	auth := &resolvedPlatformFixture{}
	router := gin.New()
	router.GET("/user", platformAuth(auth, nil, false), func(c *gin.Context) {
		role, _ := GetUserRoleFromContext(c)
		require.Equal(t, service.RoleUser, role)
		c.Status(204)
	})
	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer a b"} {
		req := httptest.NewRequest("GET", "/user", nil)
		req.Header.Set("Authorization", header)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, 401, res.Code)
	}
	require.Zero(t, auth.calls)
	for _, item := range []struct {
		err    error
		status int
	}{{service.ErrPlatformDisabled, 403}, {service.ErrPlatformIdentity, 401}, {infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "offline"), 503}} {
		auth.err = item.err
		req := httptest.NewRequest("GET", "/user", nil)
		req.Header.Set("Authorization", "Bearer signed-platform-token")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, item.status, res.Code)
	}
}

type keyPlatformFailure struct {
	platformTestRepository
	err error
}

func (p keyPlatformFailure) RequireBusinessUser(context.Context, int64) (*service.SupabaseIdentity, error) {
	return nil, p.err
}

type platformKeyLookup struct{ service.APIKeyRepository }

func (platformKeyLookup) GetByKeyForAuth(context.Context, string) (*service.APIKey, error) {
	return &service.APIKey{ID: 1, UserID: 12, Status: service.StatusActive, User: &service.User{ID: 12, Status: service.StatusActive}}, nil
}

func TestPlatformKeyErrorsPreserveNativeAndGoogleProtocolStatus(t *testing.T) {
	for _, google := range []bool{false, true} {
		for _, item := range []struct {
			err    error
			status int
		}{{service.ErrPlatformDisabled, 403}, {service.ErrPlatformIdentity, 401}, {infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "offline"), 503}} {
			cfg := &config.Config{RunMode: config.RunModeSimple}
			svc := service.NewAPIKeyService(platformKeyLookup{}, nil, nil, nil, nil, nil, cfg, keyPlatformFailure{err: item.err})
			router := gin.New()
			auth := gin.HandlerFunc(NewAPIKeyAuthMiddleware(svc, nil, cfg))
			if google {
				auth = APIKeyAuthGoogle(svc, cfg)
			}
			router.GET("/models", auth, func(c *gin.Context) { t.Fatal("unavailable platform user reached gateway") })
			req := httptest.NewRequest("GET", "/models", nil)
			req.Header.Set("Authorization", "Bearer sk-test")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			require.Equal(t, item.status, res.Code)
			if google {
				require.Contains(t, res.Body.String(), `"error"`)
				require.Contains(t, res.Body.String(), `"status"`)
			} else {
				require.Contains(t, res.Body.String(), `"code"`)
			}
		}
	}
}

func TestOptionalPlatformAuthenticationDoesNotDowngradeInvalidSessions(t *testing.T) {
	auth := &resolvedPlatformFixture{err: service.ErrPlatformSession}
	router := gin.New()
	router.GET("/public", gin.HandlerFunc(optionalPlatformAuth(auth, nil)), func(c *gin.Context) { c.Status(204) })
	request := func(header string) int {
		req := httptest.NewRequest("GET", "/public", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res.Code
	}
	require.Equal(t, 204, request(""))
	require.Zero(t, auth.calls)
	require.Equal(t, 401, request("Basic legacy"))
	require.Zero(t, auth.calls)
	require.Equal(t, 401, request("Bearer revoked-session"))
	require.Equal(t, 1, auth.calls)
}
