//go:build integration

package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EmailCacheSuite struct {
	IntegrationRedisSuite
	cache service.EmailCache
}

func (s *EmailCacheSuite) SetupTest() {
	s.IntegrationRedisSuite.SetupTest()
	s.cache = NewEmailCache(s.rdb)
}

func (s *EmailCacheSuite) TestGetNotifyVerifyCode_Missing() {
	_, err := s.cache.GetNotifyVerifyCode(s.ctx, "nonexistent@example.com")
	require.True(s.T(), errors.Is(err, redis.Nil), "expected redis.Nil for missing verification code")
}

func (s *EmailCacheSuite) TestSetAndGetNotifyVerifyCode() {
	email := "a@example.com"
	emailTTL := 2 * time.Minute
	data := &service.VerificationCodeData{Code: "123456", Attempts: 1, CreatedAt: time.Now()}

	require.NoError(s.T(), s.cache.SetNotifyVerifyCode(s.ctx, email, data, emailTTL), "SetNotifyVerifyCode")

	got, err := s.cache.GetNotifyVerifyCode(s.ctx, email)
	require.NoError(s.T(), err, "GetNotifyVerifyCode")
	require.Equal(s.T(), "123456", got.Code)
	require.Equal(s.T(), 1, got.Attempts)
}

func (s *EmailCacheSuite) TestNotifyVerifyCode_TTL() {
	email := "ttl@example.com"
	emailTTL := 2 * time.Minute
	data := &service.VerificationCodeData{Code: "654321", Attempts: 0, CreatedAt: time.Now()}

	require.NoError(s.T(), s.cache.SetNotifyVerifyCode(s.ctx, email, data, emailTTL), "SetNotifyVerifyCode")

	emailKey := notifyVerifyKeyPrefix + email
	ttl, err := s.rdb.TTL(s.ctx, emailKey).Result()
	require.NoError(s.T(), err, "TTL emailKey")
	s.AssertTTLWithin(ttl, 1*time.Second, emailTTL)
}

func (s *EmailCacheSuite) TestDeleteNotifyVerifyCode() {
	email := "delete@example.com"
	data := &service.VerificationCodeData{Code: "999999", Attempts: 0, CreatedAt: time.Now()}

	require.NoError(s.T(), s.cache.SetNotifyVerifyCode(s.ctx, email, data, 2*time.Minute), "SetNotifyVerifyCode")

	// Verify it exists
	_, err := s.cache.GetNotifyVerifyCode(s.ctx, email)
	require.NoError(s.T(), err, "GetNotifyVerifyCode before delete")

	// Delete
	require.NoError(s.T(), s.cache.DeleteNotifyVerifyCode(s.ctx, email), "DeleteNotifyVerifyCode")

	// Verify it's gone
	_, err = s.cache.GetNotifyVerifyCode(s.ctx, email)
	require.True(s.T(), errors.Is(err, redis.Nil), "expected redis.Nil after delete")
}

func (s *EmailCacheSuite) TestDeleteNotifyVerifyCode_NonExistent() {
	// Deleting a non-existent key should not error
	require.NoError(s.T(), s.cache.DeleteNotifyVerifyCode(s.ctx, "nonexistent@example.com"), "DeleteNotifyVerifyCode non-existent")
}

func (s *EmailCacheSuite) TestGetNotifyVerifyCode_JSONCorruption() {
	emailKey := notifyVerifyKeyPrefix + "corrupted@example.com"

	require.NoError(s.T(), s.rdb.Set(s.ctx, emailKey, "not-json", 1*time.Minute).Err(), "Set invalid JSON")

	_, err := s.cache.GetNotifyVerifyCode(s.ctx, "corrupted@example.com")
	require.Error(s.T(), err, "expected error for corrupted JSON")
	require.False(s.T(), errors.Is(err, redis.Nil), "expected decoding error, not redis.Nil")
}

func TestEmailCacheSuite(t *testing.T) {
	suite.Run(t, new(EmailCacheSuite))
}
