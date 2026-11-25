package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBusinessProfileRejectsPlatformIdentityFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &UserHandler{}
	router.PUT("/user", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
		c.Next()
	}, handler.UpdateProfile)
	for _, payload := range []string{`{"email":"changed@example.test"}`, `{"password":"new-password"}`, `{"role":"admin"}`, `{"status":"active"}`} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/user", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(res, req)
		require.Equal(t, 400, res.Code, payload)
	}
}
