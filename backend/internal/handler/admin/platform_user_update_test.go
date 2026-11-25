package admin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminBusinessUserUpdateRejectsPlatformIdentityFields(t *testing.T) {
	router, _ := setupAdminRouter()
	for _, payload := range []string{`{"email":"changed@example.test"}`, `{"password":"new-password"}`, `{"role":"admin"}`, `{"status":"active"}`} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/v1/admin/users/1", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(res, req)
		require.Equal(t, 400, res.Code, payload)
	}
}
