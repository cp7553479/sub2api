package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrPlatformToken    = infraerrors.Unauthorized("INVALID_TOKEN", "Invalid Supabase access token")
	ErrPlatformIdentity = infraerrors.Unauthorized("USER_NOT_FOUND", "Platform identity does not exist")
	ErrPlatformSession  = infraerrors.Unauthorized("SESSION_REVOKED", "Platform session is revoked or expired")
	ErrPlatformDisabled = infraerrors.Forbidden("USER_DISABLED", "Platform account is disabled")
)

type SupabaseIdentity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Email     string
	Status    string
	IsAdmin   bool
}

// PlatformIdentityRepository reads current authorization state; results must not be cached.
type PlatformIdentityRepository interface {
	GetUser(context.Context, uuid.UUID) (*SupabaseIdentity, error)
	SessionActive(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	RequireBusinessUser(context.Context, int64) (*SupabaseIdentity, error)
	FirstAdminEmail(context.Context) (string, error)
}

type supabaseClaims struct {
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type SupabaseAuthService struct {
	cfg          config.SupabaseConfig
	platform     PlatformIdentityRepository
	bindings     UserBindingRepository
	client       *http.Client
	mu           sync.Mutex
	keys         map[string]*ecdsa.PublicKey
	expiresAt    time.Time
	refreshedAt  time.Time
	refreshError error
}

func NewSupabaseAuthService(cfg *config.Config, platform PlatformIdentityRepository, bindings UserBindingRepository) (*SupabaseAuthService, error) {
	if err := cfg.Supabase.Validate(); err != nil {
		return nil, err
	}
	return &SupabaseAuthService{
		cfg: cfg.Supabase, platform: platform, bindings: bindings,
		client: &http.Client{Timeout: time.Duration(cfg.Supabase.HTTPTimeoutSeconds) * time.Second},
	}, nil
}

func (s *SupabaseAuthService) Authenticate(ctx context.Context, accessToken string) (*SupabaseIdentity, error) {
	if len(accessToken) == 0 || len(accessToken) > 8192 {
		return nil, ErrPlatformToken
	}
	claims := &supabaseClaims{}
	var keyError error
	_, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, ErrPlatformToken
		}
		key, err := s.signingKey(ctx, kid)
		keyError = err
		return key, err
	}, jwt.WithValidMethods([]string{"ES256"}), jwt.WithExpirationRequired(), jwt.WithIssuer(s.cfg.Issuer), jwt.WithAudience(s.cfg.Audience))
	if keyError != nil {
		return nil, keyError
	}
	if err != nil {
		return nil, ErrPlatformToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil || userID == uuid.Nil {
		return nil, ErrPlatformToken
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil || sessionID == uuid.Nil {
		return nil, ErrPlatformToken
	}
	identity, err := s.platform.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if identity.Status != StatusActive {
		return nil, ErrPlatformDisabled
	}
	active, err := s.platform.SessionActive(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, ErrPlatformSession
	}
	identity.SessionID = sessionID
	return identity, nil
}

func (s *SupabaseAuthService) ResolveUser(ctx context.Context, accessToken string) (*User, *SupabaseIdentity, error) {
	identity, err := s.Authenticate(ctx, accessToken)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.bindings.GetOrCreateSupabaseUser(ctx, identity.UserID)
	if errors.Is(err, ErrUserNotFound) {
		return nil, nil, ErrPlatformIdentity
	}
	if err != nil {
		return nil, nil, err
	}
	if !user.IsActive() || user.DeletedAt != nil {
		return nil, nil, ErrPlatformDisabled
	}
	user.Role = RoleUser
	if identity.IsAdmin {
		user.Role = RoleAdmin
	}
	user.Email = identity.Email
	return user, identity, nil
}

// Only public signing keys are cached. Expired keys are never used after a failed refresh.
func (s *SupabaseAuthService) signingKey(ctx context.Context, kid string) (signingKey *ecdsa.PublicKey, resultErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	key, found := s.keys[kid]
	if found && now.Before(s.expiresAt) {
		return key, nil
	}
	if now.Sub(s.refreshedAt) < time.Duration(s.cfg.JWKSRefreshIntervalSeconds)*time.Second {
		if s.refreshError != nil {
			return nil, s.refreshError
		}
		return nil, ErrPlatformToken
	}
	s.refreshedAt = now
	defer func() {
		if infraerrors.Code(resultErr) >= 500 {
			s.refreshError = resultErr
		} else {
			s.refreshError = nil
		}
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.JWKSURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("JWKS_UNAVAILABLE", "Cannot load platform signing keys").WithCause(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, infraerrors.ServiceUnavailable("JWKS_UNAVAILABLE", "Cannot load platform signing keys")
	}
	var document struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			Crv string `json:"crv"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"keys"`
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "Invalid platform signing keys")
	}
	if err := json.Unmarshal(body, &document); err != nil || len(document.Keys) > 16 {
		return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "Invalid platform signing keys")
	}
	keys := make(map[string]*ecdsa.PublicKey)
	for _, key := range document.Keys {
		if key.Kty != "EC" || key.Crv != "P-256" || (key.Alg != "" && key.Alg != "ES256") || (key.Use != "" && key.Use != "sig") {
			continue
		}
		x, xErr := base64.RawURLEncoding.DecodeString(key.X)
		y, yErr := base64.RawURLEncoding.DecodeString(key.Y)
		if xErr != nil || yErr != nil || len(x) != 32 || len(y) != 32 || key.Kid == "" {
			return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "Invalid platform signing keys")
		}
		publicKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}
		if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
			return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "Invalid platform signing keys")
		}
		if _, exists := keys[key.Kid]; exists {
			return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "Duplicate platform signing key")
		}
		keys[key.Kid] = publicKey
	}
	if len(keys) == 0 {
		return nil, infraerrors.ServiceUnavailable("JWKS_INVALID", "No ES256 platform signing keys")
	}
	s.keys = keys
	s.expiresAt = now.Add(time.Duration(s.cfg.JWKSCacheTTLSeconds) * time.Second)
	key, found = keys[kid]
	if !found {
		return nil, ErrPlatformToken
	}
	return key, nil
}
