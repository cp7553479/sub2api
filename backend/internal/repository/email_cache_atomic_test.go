package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newMiniredisEmailCache(t *testing.T) (service.EmailCache, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewEmailCache(rdb), mr, rdb
}

func TestEmailCache_ConcurrentNotifyAttemptsAreAtomic(t *testing.T) {
	cache, _, _ := newMiniredisEmailCache(t)
	ctx := context.Background()
	email := "user@example.com"
	require.NoError(t, cache.SetNotifyVerifyCode(ctx, email, &service.VerificationCodeData{Code: "123456"}, time.Minute))
	const workers = 50
	counts := make(chan int, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := cache.IncrNotifyVerifyCodeAttempts(ctx, email)
			if err != nil {
				t.Errorf("increment: %v", err)
				return
			}
			counts <- n
		}()
	}
	wg.Wait()
	close(counts)
	seen := map[int]bool{}
	for n := range counts {
		require.False(t, seen[n])
		seen[n] = true
	}
	require.Len(t, seen, workers)
	data, err := cache.GetNotifyVerifyCode(ctx, email)
	require.NoError(t, err)
	require.Equal(t, workers, data.Attempts)
}

func TestEmailCache_AttemptsResetOnNewCodeAndTTLFollowsCode(t *testing.T) {
	cache, mr, _ := newMiniredisEmailCache(t)
	ctx := context.Background()
	email := "User@Example.com"

	require.NoError(t, cache.SetNotifyVerifyCode(ctx, email, &service.VerificationCodeData{Code: "1"}, time.Minute))
	n, err := cache.IncrNotifyVerifyCodeAttempts(ctx, email)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Greater(t, mr.TTL(notifyVerifyKey(email)+attemptsKeySuffix), time.Duration(0))

	require.NoError(t, cache.SetNotifyVerifyCode(ctx, email, &service.VerificationCodeData{Code: "2"}, time.Minute))
	data, err := cache.GetNotifyVerifyCode(ctx, email)
	require.NoError(t, err)
	require.Equal(t, 0, data.Attempts)

	require.NoError(t, cache.DeleteNotifyVerifyCode(ctx, email))
	_, err = cache.IncrNotifyVerifyCodeAttempts(ctx, email)
	require.Error(t, err)
	require.False(t, mr.Exists(notifyVerifyKey(email)+attemptsKeySuffix))
}
