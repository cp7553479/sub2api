package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type currentPlatformState struct {
	platformTestRepository
	identity      SupabaseIdentity
	activeSession bool
	err           error
	userReads     int
	keyReads      int
}

func (r *currentPlatformState) GetUser(context.Context, uuid.UUID) (*SupabaseIdentity, error) {
	r.userReads++
	if r.err != nil {
		return nil, r.err
	}
	result := r.identity
	return &result, nil
}
func (r *currentPlatformState) SessionActive(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.activeSession, nil
}
func (r *currentPlatformState) RequireBusinessUser(context.Context, int64) (*SupabaseIdentity, error) {
	r.keyReads++
	if r.err != nil {
		return nil, r.err
	}
	result := r.identity
	return &result, nil
}

type bindingRecorder struct {
	calls  int
	userID uuid.UUID
	err    error
}

func (r *bindingRecorder) GetOrCreateSupabaseUser(_ context.Context, id uuid.UUID) (*User, error) {
	r.calls++
	r.userID = id
	if r.err != nil {
		return nil, r.err
	}
	// A stored native admin role must never grant platform administration.
	return &User{ID: 42, SupabaseUserID: id.String(), Email: "stale-native@example.test", Role: RoleAdmin, Status: StatusActive}, nil
}

type platformAuthFixture struct {
	service        *SupabaseAuthService
	platform       *currentPlatformState
	bindings       *bindingRecorder
	key            *ecdsa.PrivateKey
	reads          atomic.Int64
	server         *httptest.Server
	responseStatus atomic.Int64
}

func newPlatformAuthFixture(t *testing.T) *platformAuthFixture {
	t.Helper()
	f := &platformAuthFixture{}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	f.key = key
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.reads.Add(1)
		if status := f.responseStatus.Load(); status != 0 {
			w.WriteHeader(int(status))
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "EC", "crv": "P-256", "kid": "platform-key", "alg": "ES256", "use": "sig",
			"x": base64.RawURLEncoding.EncodeToString(f.key.X.FillBytes(make([]byte, 32))),
			"y": base64.RawURLEncoding.EncodeToString(f.key.Y.FillBytes(make([]byte, 32))),
		}}}))
	}))
	t.Cleanup(f.server.Close)
	f.platform = &currentPlatformState{identity: SupabaseIdentity{UserID: uuid.New(), Email: "current@example.test", Status: StatusActive}, activeSession: true}
	f.bindings = &bindingRecorder{}
	cfg := &config.Config{Supabase: config.SupabaseConfig{Issuer: "https://identity.example.test/auth/v1", Audience: "authenticated", APIURL: "https://platform.example.test", JWKSURL: f.server.URL, BridgeSchema: "platform_bridge", JWKSCacheTTLSeconds: 300, JWKSRefreshIntervalSeconds: 10, HTTPTimeoutSeconds: 2}}
	f.service, err = NewSupabaseAuthService(cfg, f.platform, f.bindings)
	require.NoError(t, err)
	return f
}
func (f *platformAuthFixture) claims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss": f.service.cfg.Issuer, "aud": "authenticated", "sub": f.platform.identity.UserID.String(), "session_id": uuid.NewString(), "exp": time.Now().Add(time.Hour).Unix(),
		"role": "admin", "user_metadata": map[string]any{"is_admin": true, "role": "admin"},
	}
}
func (f *platformAuthFixture) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = "platform-key"
	value, err := token.SignedString(f.key)
	require.NoError(t, err)
	return value
}

func TestSupabaseTokenBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(jwt.MapClaims)
	}{
		{"issuer", func(c jwt.MapClaims) { c["iss"] = "https://other.example.test" }},
		{"audience", func(c jwt.MapClaims) { c["aud"] = "service_role" }},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }},
		{"missing expiry", func(c jwt.MapClaims) { delete(c, "exp") }},
		{"future", func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Hour).Unix() }},
		{"subject", func(c jwt.MapClaims) { c["sub"] = "not-a-uuid" }},
		{"empty subject", func(c jwt.MapClaims) { c["sub"] = uuid.Nil.String() }},
		{"session", func(c jwt.MapClaims) { c["session_id"] = "not-a-uuid" }},
		{"missing session", func(c jwt.MapClaims) { delete(c, "session_id") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newPlatformAuthFixture(t)
			claims := f.claims()
			test.change(claims)
			_, _, err := f.service.ResolveUser(context.Background(), f.sign(t, claims))
			require.ErrorIs(t, err, ErrPlatformToken)
			require.Zero(t, f.platform.userReads)
			require.Zero(t, f.bindings.calls)
		})
	}
	t.Run("HMAC", func(t *testing.T) {
		f := newPlatformAuthFixture(t)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, f.claims())
		token.Header["kid"] = "platform-key"
		encoded, err := token.SignedString([]byte("native-secret"))
		require.NoError(t, err)
		_, err = f.service.Authenticate(context.Background(), encoded)
		require.ErrorIs(t, err, ErrPlatformToken)
		require.Zero(t, f.reads.Load())
	})
	t.Run("signature", func(t *testing.T) {
		f := newPlatformAuthFixture(t)
		other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		token := jwt.NewWithClaims(jwt.SigningMethodES256, f.claims())
		token.Header["kid"] = "platform-key"
		encoded, err := token.SignedString(other)
		require.NoError(t, err)
		_, err = f.service.Authenticate(context.Background(), encoded)
		require.ErrorIs(t, err, ErrPlatformToken)
	})
}

func TestSupabaseUsesCurrentPlatformStateAndUUID(t *testing.T) {
	f := newPlatformAuthFixture(t)
	token := f.sign(t, f.claims())
	user, identity, err := f.service.ResolveUser(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, RoleUser, user.Role)
	require.False(t, identity.IsAdmin)
	require.Equal(t, f.platform.identity.UserID, f.bindings.userID)
	f.platform.identity.IsAdmin = true
	user, _, err = f.service.ResolveUser(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, RoleAdmin, user.Role)
	f.platform.identity.IsAdmin = false
	f.platform.identity.Email = "changed@example.test"
	user, _, err = f.service.ResolveUser(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, RoleUser, user.Role)
	require.Equal(t, "changed@example.test", user.Email)
	require.Equal(t, int64(1), f.reads.Load(), "public JWKS can be cached")
	require.Equal(t, 3, f.platform.userReads, "authorization state must be read on every request")
	f.platform.identity.Status = StatusDisabled
	_, _, err = f.service.ResolveUser(context.Background(), token)
	require.ErrorIs(t, err, ErrPlatformDisabled)
	require.Equal(t, 3, f.bindings.calls)
	f.platform.identity.Status = StatusActive
	f.platform.activeSession = false
	_, _, err = f.service.ResolveUser(context.Background(), token)
	require.ErrorIs(t, err, ErrPlatformSession)
	require.Equal(t, 3, f.bindings.calls)
	f.platform.err = ErrPlatformIdentity
	_, _, err = f.service.ResolveUser(context.Background(), token)
	require.ErrorIs(t, err, ErrPlatformIdentity)
	require.Equal(t, 3, f.bindings.calls)
}

func TestSupabaseJWKSUnavailableNeverAuthorizesOrMasqueradesAsInvalidToken(t *testing.T) {
	f := newPlatformAuthFixture(t)
	token := f.sign(t, f.claims())
	f.responseStatus.Store(503)
	for range 2 {
		_, err := f.service.Authenticate(context.Background(), token)
		require.Equal(t, 503, infraerrors.Code(err))
		require.Zero(t, f.platform.userReads)
	}
	require.Equal(t, int64(1), f.reads.Load(), "failed refresh is rate limited")
	f.responseStatus.Store(0)
	f.service.refreshedAt = time.Time{}
	_, err := f.service.Authenticate(context.Background(), token)
	require.NoError(t, err)
	f.service.expiresAt = time.Time{}
	f.service.refreshedAt = time.Time{}
	f.responseStatus.Store(503)
	_, err = f.service.Authenticate(context.Background(), token)
	require.Equal(t, 503, infraerrors.Code(err))
	require.Equal(t, 1, f.platform.userReads, "stale signing keys must not authorize")
}

func TestSupabaseIdentityDeletedBetweenVerificationAndBindingIsUnauthorized(t *testing.T) {
	f := newPlatformAuthFixture(t)
	f.bindings.err = ErrUserNotFound
	_, _, err := f.service.ResolveUser(context.Background(), f.sign(t, f.claims()))
	require.ErrorIs(t, err, ErrPlatformIdentity)
	require.Equal(t, http.StatusUnauthorized, infraerrors.Code(err))
	f.bindings.err = infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "offline")
	_, _, err = f.service.ResolveUser(context.Background(), f.sign(t, f.claims()))
	require.Equal(t, http.StatusServiceUnavailable, infraerrors.Code(err))
}

type cachedPlatformKey struct {
	*authInvalidationCacheStub
	entry *APIKeyAuthCacheEntry
	reads int
}

func (c *cachedPlatformKey) GetAuthCache(context.Context, string) (*APIKeyAuthCacheEntry, error) {
	c.reads++
	return c.entry, nil
}

func TestPlatformDisableIsCheckedAfterL1AndL2KeyCacheHits(t *testing.T) {
	for _, layer := range []string{"L1", "L2"} {
		t.Run(layer, func(t *testing.T) {
			ctx := context.Background()
			platform := &currentPlatformState{identity: SupabaseIdentity{UserID: uuid.New(), Status: StatusActive, IsAdmin: true}}
			cfg := &config.Config{APIKeyAuth: config.APIKeyAuthCacheConfig{L2TTLSeconds: 60}}
			cache := &cachedPlatformKey{authInvalidationCacheStub: &authInvalidationCacheStub{}}
			if layer == "L1" {
				cfg.APIKeyAuth.L1Size = 128
				cfg.APIKeyAuth.L1TTLSeconds = 60
			}
			svc := NewAPIKeyService(nil, nil, nil, nil, nil, cache, cfg, platform)
			entry := &APIKeyAuthCacheEntry{Snapshot: svc.snapshotFromAPIKey(ctx, &APIKey{ID: 9, UserID: 42, Status: StatusActive, User: &User{ID: 42, Role: RoleAdmin, Status: StatusActive}})}
			cache.entry = entry
			if layer == "L1" {
				svc.setAuthCacheL1(svc.authCacheKey("sk-test"), entry)
				svc.authCacheL1.Wait()
				t.Cleanup(svc.authCacheL1.Close)
			}
			key, err := svc.GetByKey(ctx, "sk-test")
			require.NoError(t, err)
			require.Equal(t, RoleAdmin, key.User.Role)
			platform.identity.IsAdmin = false
			key, err = svc.GetByKey(ctx, "sk-test")
			require.NoError(t, err)
			require.Equal(t, RoleUser, key.User.Role)
			platform.err = ErrPlatformDisabled
			key, err = svc.GetByKey(ctx, "sk-test")
			require.Nil(t, key)
			require.ErrorIs(t, err, ErrPlatformDisabled)
			platform.err = ErrPlatformIdentity
			key, err = svc.GetByKey(ctx, "sk-test")
			require.Nil(t, key)
			require.ErrorIs(t, err, ErrPlatformIdentity)
			platform.err = infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "offline")
			key, err = svc.GetByKey(ctx, "sk-test")
			require.Nil(t, key)
			require.Equal(t, 503, infraerrors.Code(err))
			require.Equal(t, 5, platform.keyReads)
			require.False(t, errors.Is(err, ErrPlatformToken))
		})
	}
}
