package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"golang.org/x/sync/singleflight"
)

const (
	GrokDefaultBaseURLModeAPI     = "api"
	GrokDefaultBaseURLModeUSEast1 = "us-east-1"
	GrokDefaultBaseURLModeUSWest2 = "us-west-2"
	GrokDefaultBaseURLModeEUWest1 = "eu-west-1"
	GrokDefaultBaseURLModeCLI     = "cli"
)

func normalizeGrokDefaultBaseURLMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case GrokDefaultBaseURLModeAPI:
		return GrokDefaultBaseURLModeAPI
	case GrokDefaultBaseURLModeUSEast1:
		return GrokDefaultBaseURLModeUSEast1
	case GrokDefaultBaseURLModeUSWest2:
		return GrokDefaultBaseURLModeUSWest2
	case GrokDefaultBaseURLModeEUWest1:
		return GrokDefaultBaseURLModeEUWest1
	case GrokDefaultBaseURLModeCLI:
		return GrokDefaultBaseURLModeCLI
	default:
		return GrokDefaultBaseURLModeCLI
	}
}

func GrokBaseURLForMode(mode string) string {
	switch normalizeGrokDefaultBaseURLMode(mode) {
	case GrokDefaultBaseURLModeAPI:
		return xai.DefaultBaseURL
	case GrokDefaultBaseURLModeUSEast1:
		return xai.DefaultUSEast1BaseURL
	case GrokDefaultBaseURLModeUSWest2:
		return xai.DefaultUSWest2BaseURL
	case GrokDefaultBaseURLModeEUWest1:
		return xai.DefaultEUWest1BaseURL
	default:
		return xai.DefaultCLIBaseURL
	}
}

func (s *SettingService) GetGrokDefaultBaseURLMode(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return GrokDefaultBaseURLModeCLI
	}
	dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gatewayForwardingDBTimeout)
	defer cancel()
	raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyGrokDefaultBaseURLMode)
	if err != nil {
		return GrokDefaultBaseURLModeCLI
	}
	return normalizeGrokDefaultBaseURLMode(raw)
}

func (s *SettingService) GetGrokDefaultBaseURL(ctx context.Context) string {
	return GrokBaseURLForMode(s.GetGrokDefaultBaseURLMode(ctx))
}

func (s *SettingService) ResolveGrokBaseURL(ctx context.Context, account *Account) string {
	def := xai.DefaultCLIBaseURL
	if s != nil {
		def = s.GetGrokDefaultBaseURL(ctx)
	}
	if account == nil {
		return def
	}
	return account.GetGrokBaseURLOr(def)
}

var ErrSettingNotFound = infraerrors.NotFound("SETTING_NOT_FOUND", "setting not found")

type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
	GetAll(ctx context.Context) (map[string]string, error)
	Delete(ctx context.Context, key string) error
}

// WebSearchManagerBuilder creates a websearch.Manager from config (injected by infra layer).
// proxyURLs maps proxy ID to resolved URL for provider-level proxy support.
type WebSearchManagerBuilder func(cfg *WebSearchEmulationConfig, proxyURLs map[int64]string)

// SettingService 系统设置服务
type SettingService struct {
	settingRepo                 SettingRepository
	proxyRepo                   ProxyRepository // for resolving websearch provider proxy URLs
	cfg                         *config.Config
	onUpdate                    func() // Callback when settings are updated (for cache invalidation)
	version                     string // Application version
	webSearchManagerBuilder     WebSearchManagerBuilder
	antigravityUAVersionCache   atomic.Value // *cachedAntigravityUserAgentVersion
	antigravityUAVersionSF      singleflight.Group
	openAICodexUACache          atomic.Value // *cachedOpenAICodexUserAgent
	openAICodexUASF             singleflight.Group
	openAICodexVersionCache     atomic.Value // *cachedOpenAICodexClientVersion
	openAICodexVersionSF        singleflight.Group
	claudeCodeVersionCache      atomic.Value // *cachedClaudeCodeClientVersion
	claudeCodeVersionSF         singleflight.Group
	codexRestrictionPolicyCache atomic.Value // *cachedCodexRestrictionPolicy
	codexRestrictionPolicySF    singleflight.Group

	cyberSessionBlockRuntimeMu    sync.Mutex
	cyberSessionBlockRuntimeCache atomic.Value // *cachedCyberSessionBlockRuntime
	cyberSessionBlockRuntimeSF    singleflight.Group

	// panelRateLimitCache 面板 API 限流配置进程内缓存（*cachedPanelRateLimitSettings）。
	// 面板每个认证请求都会读取，禁止在热路径上直接访问 DB。
	panelRateLimitCache atomic.Value
	panelRateLimitSF    singleflight.Group

	// registrationEmailBlacklistCache 精确邮箱黑名单进程内缓存（*registrationEmailBlacklistIndex，60s TTL）。
	// 公开的注册与发码接口每次请求都会读取，禁止在热路径上直接访问 DB。
	// registrationEmailBlacklistMu 串行化刷新时的「读 DB + 写缓存」与保存后的写缓存，
	// 避免进行中的刷新用旧名单覆盖刚保存的名单。

	// openAIQuotaAutoPauseSettingsCache holds the most recently observed quota auto-pause
	// settings. GetOpenAIQuotaAutoPauseSettings reads this atomic.Value on the request hot
	// path without ever blocking on the DB; when the cached entry expires, a background
	// goroutine refreshes it via openAIQuotaAutoPauseSettingsSF (stale-while-revalidate).
	// This per-service field also gives tests natural isolation — each SettingService
	// instance owns its own cache, no shared package-level state.
	openAIQuotaAutoPauseSettingsCache atomic.Value // *cachedOpenAIQuotaAutoPauseSettings
	openAIQuotaAutoPauseSettingsSF    singleflight.Group
	openAIAPIKeyHealthBreakerCache    atomic.Value // *cachedOpenAIAPIKeyHealthBreakerSettings

	channelMonitorRuntimeListenersMu sync.Mutex
	channelMonitorRuntimeListeners   []func()
}

// DefaultPlatformQuotaSetting 单 platform 三档限额（nil = 沿用上层；0 = 显式禁用；>0 = 上限）
type DefaultPlatformQuotaSetting struct {
	DailyLimitUSD   *float64 `json:"daily"`
	WeeklyLimitUSD  *float64 `json:"weekly"`
	MonthlyLimitUSD *float64 `json:"monthly"`
}

// HasAnyLimit 报告是否至少配置了一档限额（0 也算配置）。nil receiver 视为未配置。
func (q *DefaultPlatformQuotaSetting) HasAnyLimit() bool {
	return q != nil && (q.DailyLimitUSD != nil || q.WeeklyLimitUSD != nil || q.MonthlyLimitUSD != nil)
}

const (
	defaultLoginAgreementMode = "modal"
	defaultLoginAgreementDate = "2026-03-31"
)

// NewSettingService 创建系统设置服务实例
func NewSettingService(settingRepo SettingRepository, cfg *config.Config) *SettingService {
	return &SettingService{
		settingRepo: settingRepo,
		cfg:         cfg,
	}
}

// SetProxyRepository injects a proxy repo for resolving websearch provider proxy URLs.
func (s *SettingService) SetProxyRepository(repo ProxyRepository) {
	s.proxyRepo = repo
}

func (s *SettingService) LoadForwardedClientIPSettings(ctx context.Context) error {
	if s == nil || s.cfg == nil || s.settingRepo == nil {
		return nil
	}

	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyAPIKeyACLTrustForwardedIP,
		SettingKeyForwardedClientIPHeaders,
		settingKeyForwardedClientIPModeV2,
	})
	if err != nil {
		s.cfg.SetForwardedClientIPSettings(false, nil)
		return fmt.Errorf("get forwarded client ip settings: %w", err)
	}

	enabled := s.cfg.Security.TrustForwardedIPForAPIKeyACL
	headers := s.cfg.ForwardedClientIPSettings().Headers
	storedValue, hasStoredValue := values[SettingKeyAPIKeyACLTrustForwardedIP]
	if hasStoredValue {
		enabled = storedValue == "true"
	}

	var headersErr error
	if storedHeaders, ok := values[SettingKeyForwardedClientIPHeaders]; ok {
		headers, headersErr = parseForwardedClientIPHeadersSetting(storedHeaders)
		if headersErr != nil {
			enabled = false
			headers = []string{}
			headersErr = fmt.Errorf("load forwarded client ip headers: %w", headersErr)
		}
	}

	updates := make(map[string]string)
	if _, hasStoredHeaders := values[SettingKeyForwardedClientIPHeaders]; !hasStoredHeaders {
		headersJSON, marshalErr := json.Marshal(headers)
		if marshalErr != nil {
			headers = []string{}
			headersErr = errors.Join(headersErr, fmt.Errorf("marshal forwarded client ip headers: %w", marshalErr))
			headersJSON = []byte("[]")
		}
		updates[SettingKeyForwardedClientIPHeaders] = string(headersJSON)
	}
	if values[settingKeyForwardedClientIPModeV2] != "true" {
		updates[settingKeyForwardedClientIPModeV2] = "true"
		// Before this migration, new installations persisted false by default.
		// Restore compatibility only when no trusted-proxy policy was configured.
		if headersErr == nil && hasStoredValue && !enabled && !s.cfg.Server.TrustedProxiesConfigured {
			enabled = true
			updates[SettingKeyAPIKeyACLTrustForwardedIP] = "true"
		}
	}
	if len(updates) > 0 {
		if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
			s.cfg.SetForwardedClientIPSettings(enabled, headers)
			return errors.Join(headersErr, fmt.Errorf("migrate forwarded client ip setting: %w", err))
		}
	}

	s.cfg.SetForwardedClientIPSettings(enabled, headers)
	return headersErr
}

// GetAllSettings 获取所有系统设置
func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	return s.parseSettings(settings), nil
}

// SetOnUpdateCallback sets a callback function to be called when settings are updated
// This is used for cache invalidation (e.g., HTML cache in frontend server)
func (s *SettingService) SetOnUpdateCallback(callback func()) {
	s.onUpdate = callback
}

// SubscribeChannelMonitorRuntime registers a listener that is invoked after
// settings are successfully persisted (and process caches refreshed).
// Used by ChannelMonitorRunner / ChannelMonitorV2Aggregator for immediate
// mode flips without waiting for poll intervals.
func (s *SettingService) SubscribeChannelMonitorRuntime(listener func()) (unsubscribe func()) {
	if s == nil || listener == nil {
		return func() {}
	}
	s.channelMonitorRuntimeListenersMu.Lock()
	s.channelMonitorRuntimeListeners = append(s.channelMonitorRuntimeListeners, listener)
	idx := len(s.channelMonitorRuntimeListeners) - 1
	s.channelMonitorRuntimeListenersMu.Unlock()
	return func() {
		s.channelMonitorRuntimeListenersMu.Lock()
		defer s.channelMonitorRuntimeListenersMu.Unlock()
		if idx < 0 || idx >= len(s.channelMonitorRuntimeListeners) {
			return
		}
		s.channelMonitorRuntimeListeners[idx] = nil
	}
}

func (s *SettingService) notifyChannelMonitorRuntimeListeners() {
	if s == nil {
		return
	}
	s.channelMonitorRuntimeListenersMu.Lock()
	listeners := make([]func(), 0, len(s.channelMonitorRuntimeListeners))
	for _, l := range s.channelMonitorRuntimeListeners {
		if l != nil {
			listeners = append(listeners, l)
		}
	}
	s.channelMonitorRuntimeListenersMu.Unlock()
	for _, l := range listeners {
		func(fn func()) {
			defer func() {
				if recovered := recover(); recovered != nil {
					_ = recovered // keep settings path healthy
				}
			}()
			fn()
		}(l)
	}
}

// SetVersion sets the application version for injection into public settings
func (s *SettingService) SetVersion(version string) {
	s.version = version
}

// getStringOrDefault 获取字符串值或默认值
func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
	if value, ok := settings[key]; ok && value != "" {
		return value
	}
	return defaultValue
}
