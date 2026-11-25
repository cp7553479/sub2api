package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	FrontendURL string `json:"frontend_url"`

	AuditLogRetentionDays   int                          `json:"audit_log_retention_days"` // 审计日志保留天数
	LoginAgreementEnabled   bool                         `json:"login_agreement_enabled"`
	LoginAgreementMode      string                       `json:"login_agreement_mode"`
	LoginAgreementUpdatedAt string                       `json:"login_agreement_updated_at"`
	LoginAgreementDocuments []dto.LoginAgreementDocument `json:"login_agreement_documents"`

	// 邮件服务设置
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from_email"`
	SMTPFromName string `json:"smtp_from_name"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`

	// API Key IP 访问控制设置
	APIKeyACLTrustForwardedIP *bool     `json:"api_key_acl_trust_forwarded_ip"`
	ForwardedClientIPHeaders  *[]string `json:"forwarded_client_ip_headers"`

	// WeChat MP payment authorization

	WeChatConnectMPAppID     string `json:"wechat_connect_mp_app_id"`
	WeChatConnectMPAppSecret string `json:"wechat_connect_mp_app_secret"`

	WeChatConnectMPEnabled bool `json:"wechat_connect_mp_enabled"`

	// OEM设置
	SiteName                    string                `json:"site_name"`
	SiteLogo                    string                `json:"site_logo"`
	SiteSubtitle                string                `json:"site_subtitle"`
	APIBaseURL                  string                `json:"api_base_url"`
	ContactInfo                 string                `json:"contact_info"`
	DocURL                      string                `json:"doc_url"`
	HomeContent                 string                `json:"home_content"`
	CompactHomeEnabled          bool                  `json:"compact_home_enabled"`
	HideCcsImportButton         bool                  `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled *bool                 `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     *string               `json:"purchase_subscription_url"`
	TableDefaultPageSize        int                   `json:"table_default_page_size"`
	TablePageSizeOptions        []int                 `json:"table_page_size_options"`
	CustomMenuItems             *[]dto.CustomMenuItem `json:"custom_menu_items"`
	CustomEndpoints             *[]dto.CustomEndpoint `json:"custom_endpoints"`

	// 默认配置

	AffiliateRebateRate          *float64 `json:"affiliate_rebate_rate"`
	AffiliateRebateFreezeHours   *int     `json:"affiliate_rebate_freeze_hours"`
	AffiliateRebateDurationDays  *int     `json:"affiliate_rebate_duration_days"`
	AffiliateRebatePerInviteeCap *float64 `json:"affiliate_rebate_per_invitee_cap"`
	AdminRechargeRebateEnabled   *bool    `json:"affiliate_admin_recharge_enabled"`

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         *bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled *bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          *string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    *int    `json:"ops_metrics_interval_seconds"`

	MinClaudeCodeVersion string `json:"min_claude_code_version"`
	MaxClaudeCodeVersion string `json:"max_claude_code_version"`

	// 分组隔离
	AllowUngroupedKeyScheduling bool `json:"allow_ungrouped_key_scheduling"`

	// Backend Mode
	BackendModeEnabled bool `json:"backend_mode_enabled"`

	// Gateway forwarding behavior
	OpenAITTFTMode                         *string `json:"openai_ttft_mode"`
	EnableFingerprintUnification           *bool   `json:"enable_fingerprint_unification"`
	EnableMetadataPassthrough              *bool   `json:"enable_metadata_passthrough"`
	EnableCCHSigning                       *bool   `json:"enable_cch_signing"`
	EnableClaudeOAuthSystemPromptInjection *bool   `json:"enable_claude_oauth_system_prompt_injection"`
	ClaudeOAuthSystemPrompt                *string `json:"claude_oauth_system_prompt"`
	ClaudeOAuthSystemPromptBlocks          *string `json:"claude_oauth_system_prompt_blocks"`
	EnableAnthropicCacheTTL1hInjection     *bool   `json:"enable_anthropic_cache_ttl_1h_injection"`
	RewriteMessageCacheControl             *bool   `json:"rewrite_message_cache_control"`
	EnableClientDatelineNormalization      *bool   `json:"enable_client_dateline_normalization"`
	AntigravityUserAgentVersion            *string `json:"antigravity_user_agent_version"`
	OpenAICodexUserAgent                   *string `json:"openai_codex_user_agent"`
	OpenAICodexClientVersion               *string `json:"openai_codex_client_version"`
	OpenAICodexVersionAutoSyncEnabled      *bool   `json:"openai_codex_version_auto_sync_enabled"`
	ClaudeCodeClientVersion                *string `json:"claude_code_client_version"`
	ClaudeCodeVersionAutoSyncEnabled       *bool   `json:"claude_code_version_auto_sync_enabled"`

	// codex_cli_only 加固（global-only）
	MinCodexVersion                      string `json:"min_codex_version"`
	MaxCodexVersion                      string `json:"max_codex_version"`
	CodexCLIOnlyBlacklist                string `json:"codex_cli_only_blacklist"`
	CodexCLIOnlyWhitelist                string `json:"codex_cli_only_whitelist"`
	CodexCLIOnlyAllowAppServerClients    *bool  `json:"codex_cli_only_allow_app_server_clients"`
	CodexCLIOnlyEngineFingerprintSignals string `json:"codex_cli_only_engine_fingerprint_signals"`

	// Payment visible method routing
	PaymentVisibleMethodAlipaySource  *string `json:"payment_visible_method_alipay_source"`
	PaymentVisibleMethodWxpaySource   *string `json:"payment_visible_method_wxpay_source"`
	PaymentVisibleMethodAlipayEnabled *bool   `json:"payment_visible_method_alipay_enabled"`
	PaymentVisibleMethodWxpayEnabled  *bool   `json:"payment_visible_method_wxpay_enabled"`

	// OpenAI account scheduling
	OpenAILowUpstreamRatePriorityEnabled               *bool    `json:"openai_low_upstream_rate_priority_enabled"`
	OpenAIOAuthSchedulingRateMultiplier                *float64 `json:"openai_oauth_scheduling_rate_multiplier"`
	OpenAIAdvancedSchedulerEnabled                     *bool    `json:"openai_advanced_scheduler_enabled"`
	OpenAIAdvancedSchedulerStickyWeightedEnabled       *bool    `json:"openai_advanced_scheduler_sticky_weighted_enabled"`
	OpenAIAdvancedSchedulerSubscriptionPriorityEnabled *bool    `json:"openai_advanced_scheduler_subscription_priority_enabled"`
	OpenAIAdvancedSchedulerLBTopK                      *string  `json:"openai_advanced_scheduler_lb_top_k"`
	OpenAIAdvancedSchedulerWeightPriority              *string  `json:"openai_advanced_scheduler_weight_priority"`
	OpenAIAdvancedSchedulerWeightLoad                  *string  `json:"openai_advanced_scheduler_weight_load"`
	OpenAIAdvancedSchedulerWeightQueue                 *string  `json:"openai_advanced_scheduler_weight_queue"`
	OpenAIAdvancedSchedulerWeightErrorRate             *string  `json:"openai_advanced_scheduler_weight_error_rate"`
	OpenAIAdvancedSchedulerWeightTTFT                  *string  `json:"openai_advanced_scheduler_weight_ttft"`
	OpenAIAdvancedSchedulerWeightReset                 *string  `json:"openai_advanced_scheduler_weight_reset"`
	OpenAIAdvancedSchedulerWeightQuotaHeadroom         *string  `json:"openai_advanced_scheduler_weight_quota_headroom"`
	OpenAIAdvancedSchedulerWeightUpstreamCost          *string  `json:"openai_advanced_scheduler_weight_upstream_cost"`
	OpenAIAdvancedSchedulerWeightPreviousResponse      *string  `json:"openai_advanced_scheduler_weight_previous_response"`
	OpenAIAdvancedSchedulerWeightSessionSticky         *string  `json:"openai_advanced_scheduler_weight_session_sticky"`

	// 余额不足提醒
	BalanceLowNotifyEnabled         *bool                   `json:"balance_low_notify_enabled"`
	BalanceLowNotifyThreshold       *float64                `json:"balance_low_notify_threshold"`
	BalanceLowNotifyRechargeURL     *string                 `json:"balance_low_notify_recharge_url"`
	SubscriptionExpiryNotifyEnabled *bool                   `json:"subscription_expiry_notify_enabled"`
	AccountQuotaNotifyEnabled       *bool                   `json:"account_quota_notify_enabled"`
	AccountQuotaNotifyEmails        *[]dto.NotifyEmailEntry `json:"account_quota_notify_emails"`

	// Payment configuration (integrated into settings, full replace)
	PaymentEnabled                   *bool    `json:"payment_enabled"`
	PaymentMinAmount                 *float64 `json:"payment_min_amount"`
	PaymentMaxAmount                 *float64 `json:"payment_max_amount"`
	PaymentDailyLimit                *float64 `json:"payment_daily_limit"`
	PaymentOrderTimeoutMin           *int     `json:"payment_order_timeout_minutes"`
	PaymentMaxPendingOrders          *int     `json:"payment_max_pending_orders"`
	PaymentEnabledTypes              []string `json:"payment_enabled_types"`
	PaymentBalanceDisabled           *bool    `json:"payment_balance_disabled"`
	PaymentBalanceRechargeMultiplier *float64 `json:"payment_balance_recharge_multiplier"`
	PaymentSubscriptionUSDToCNYRate  *float64 `json:"payment_subscription_usd_to_cny_rate"`
	PaymentRechargeFeeRate           *float64 `json:"payment_recharge_fee_rate"`
	// nil 表示不更新；空数组表示清空阶梯
	PaymentRechargeBonusTiers  *[]dto.RechargeBonusTier `json:"payment_recharge_bonus_tiers"`
	PaymentRechargeBonusMode   *string                  `json:"payment_recharge_bonus_mode"`
	PaymentRechargeBonusNotice *string                  `json:"payment_recharge_bonus_notice"`
	PaymentLoadBalanceStrat    *string                  `json:"payment_load_balance_strategy"`
	PaymentProductNamePrefix   *string                  `json:"payment_product_name_prefix"`
	PaymentProductNameSuffix   *string                  `json:"payment_product_name_suffix"`
	PaymentHelpImageURL        *string                  `json:"payment_help_image_url"`
	PaymentHelpText            *string                  `json:"payment_help_text"`

	// Cancel rate limit
	PaymentCancelRateLimitEnabled *bool   `json:"payment_cancel_rate_limit_enabled"`
	PaymentCancelRateLimitMax     *int    `json:"payment_cancel_rate_limit_max"`
	PaymentCancelRateLimitWindow  *int    `json:"payment_cancel_rate_limit_window"`
	PaymentCancelRateLimitUnit    *string `json:"payment_cancel_rate_limit_unit"`
	PaymentCancelRateLimitMode    *string `json:"payment_cancel_rate_limit_window_mode"`

	// Force Alipay mobile clients to use QR code payment instead of mobile redirect
	PaymentAlipayForceQRCode *bool `json:"payment_alipay_force_qrcode"`
	// Use Alipay face-to-face precreate and an app deep link on mobile clients.
	PaymentAlipayMobilePrecreateDeepLink *bool `json:"payment_alipay_mobile_precreate_deep_link"`

	// Channel Monitor feature switch
	ChannelMonitorEnabled                *bool   `json:"channel_monitor_enabled"`
	ChannelMonitorMode                   *string `json:"channel_monitor_mode"`
	ChannelMonitorDefaultIntervalSeconds *int    `json:"channel_monitor_default_interval_seconds"`
	ChannelMonitorHideThroughput         *bool   `json:"channel_monitor_hide_throughput"`
	ChannelMonitorShowQuota              *bool   `json:"channel_monitor_show_quota"`
	ChannelMonitorHideUserRanking        *bool   `json:"channel_monitor_hide_user_ranking"`

	// Grok model mapping policy
	GrokDefaultTextModel           *string `json:"grok_default_text_model"`
	GrokCrossClientModelMapEnabled *bool   `json:"grok_cross_client_model_map_enabled"`
	GrokDefaultBaseURLMode         *string `json:"grok_default_base_url_mode"`

	// Available Channels feature switch (user-facing)
	AvailableChannelsEnabled *bool `json:"available_channels_enabled"`

	// Subscription feature switch (user-facing subscription surface; see SettingKeySubscriptionEnabled)
	SubscriptionEnabled *bool `json:"subscription_enabled"`

	// Model Plaza feature switches + description
	ModelPlazaEnabled     *bool   `json:"model_plaza_enabled"`
	ModelPlazaRequireAuth *bool   `json:"model_plaza_require_auth"`
	ModelPlazaDescription *string `json:"model_plaza_description"`

	// Plugin management menu visibility switch; plugin runtime is unaffected.
	PluginManagementEnabled *bool `json:"plugin_management_enabled"`

	// Affiliate (邀请返利) feature switch
	AffiliateEnabled *bool `json:"affiliate_enabled"`

	// 风控中心功能开关
	RiskControlEnabled *bool `json:"risk_control_enabled"`

	// cyber 会话屏蔽开关 + TTL
	CyberSessionBlockEnabled    *bool   `json:"cyber_session_block_enabled"`
	CyberPolicyUserAllowlist    *string `json:"cyber_policy_user_allowlist"`
	CyberSessionBlockTTLSeconds *int    `json:"cyber_session_block_ttl_seconds"`

	// OpenAI fast/flex policy (optional, only updated when provided)
	OpenAIFastPolicySettings *dto.OpenAIFastPolicySettings `json:"openai_fast_policy_settings,omitempty"`

	// 系统全局 platform quota 默认值（整体替换语义：nil = 不修改，non-nil = 整体覆盖）。
	DefaultPlatformQuotas map[string]*service.DefaultPlatformQuotaSetting `json:"default_platform_quotas"`

	// 各平台账号自动停调阈值（整体替换语义：nil = 不修改，non-nil = 整体覆盖）。
	AccountSchedulingThresholds map[string]int `json:"account_scheduling_thresholds"`

	AllowUserViewErrorRequests *bool `json:"allow_user_view_error_requests"`
}

// UpdateSettings 更新系统设置
// PUT /api/v1/admin/settings
// settingKeyJSONAliases covers the request fields whose JSON name differs from
// the setting key they persist to. Every other field of UpdateSettingsRequest
// is named after its setting key.
var settingKeyJSONAliases = map[string]string{
	"smtp_from_email": service.SettingKeySMTPFrom,
}

// settingKeyByJSONName maps the value-typed top-level JSON fields of
// UpdateSettingsRequest to the setting key each one writes. Resolved once from
// the struct tags so new fields are covered without touching this file.
//
// Pointer-typed fields are deliberately excluded: they already carry their own
// "omitted = keep the stored value" merge in UpdateSettings, and some of them
// rely on being rewritten on every save to re-normalize fail-closed security
// state (see TestUpdateSettingsMalformedForwardedClientIPHeadersRemainFailClosedWhenOmitted).
// Only the value-typed fields are indistinguishable from a deliberate clear.
var settingKeyByJSONName = buildSettingKeyByJSONName()

func buildSettingKeyByJSONName() map[string]string {
	t := reflect.TypeOf(UpdateSettingsRequest{})
	out := make(map[string]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type.Kind() == reflect.Pointer {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		if alias, ok := settingKeyJSONAliases[name]; ok {
			out[name] = alias
			continue
		}
		out[name] = name
	}
	return out
}

// omittedSettingKeys reports the setting keys this payload never mentioned.
// Saving settings is a whole-document PUT, so without this a client that sends
// only the one field it cares about resets every other field to a zero value.
func omittedSettingKeys(sentFields map[string]json.RawMessage) service.OmittedSettingKeys {
	omitted := make(service.OmittedSettingKeys, len(settingKeyByJSONName))
	for jsonName, settingKey := range settingKeyByJSONName {
		if _, sent := sentFields[jsonName]; !sent {
			omitted[settingKey] = struct{}{}
		}
	}
	return omitted
}

func settingsAuditRequest(req UpdateSettingsRequest) UpdateSettingsRequest {

	return req
}

func (h *SettingHandler) UpdateSettings(c *gin.Context) {
	var sentFields map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&sentFields, binding.JSON); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var req UpdateSettingsRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	auditReq := settingsAuditRequest(req)
	omitted := omittedSettingKeys(sentFields)

	previousSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 两个安全开关的请求字段为指针：省略字段=保持现值，避免旧客户端/脚本
	// 用不含新字段的全量 payload 保存设置时把安全开关静默重置。

	forwardedClientIPHeaders := append([]string(nil), previousSettings.ForwardedClientIPHeaders...)
	if req.ForwardedClientIPHeaders != nil {
		forwardedClientIPHeaders = append([]string(nil), (*req.ForwardedClientIPHeaders)...)
	}

	// 验证参数

	affiliateRebateRate := previousSettings.AffiliateRebateRate
	if req.AffiliateRebateRate != nil {
		affiliateRebateRate = *req.AffiliateRebateRate
	}
	if affiliateRebateRate < service.AffiliateRebateRateMin {
		affiliateRebateRate = service.AffiliateRebateRateMin
	}
	if affiliateRebateRate > service.AffiliateRebateRateMax {
		affiliateRebateRate = service.AffiliateRebateRateMax
	}
	affiliateRebateFreezeHours := previousSettings.AffiliateRebateFreezeHours
	if req.AffiliateRebateFreezeHours != nil {
		affiliateRebateFreezeHours = *req.AffiliateRebateFreezeHours
	}
	if affiliateRebateFreezeHours < 0 {
		affiliateRebateFreezeHours = service.AffiliateRebateFreezeHoursDefault
	}
	if affiliateRebateFreezeHours > service.AffiliateRebateFreezeHoursMax {
		affiliateRebateFreezeHours = service.AffiliateRebateFreezeHoursMax
	}
	affiliateRebateDurationDays := previousSettings.AffiliateRebateDurationDays
	if req.AffiliateRebateDurationDays != nil {
		affiliateRebateDurationDays = *req.AffiliateRebateDurationDays
	}
	if affiliateRebateDurationDays < 0 {
		affiliateRebateDurationDays = service.AffiliateRebateDurationDaysDefault
	}
	if affiliateRebateDurationDays > service.AffiliateRebateDurationDaysMax {
		affiliateRebateDurationDays = service.AffiliateRebateDurationDaysMax
	}
	affiliateRebatePerInviteeCap := previousSettings.AffiliateRebatePerInviteeCap
	if req.AffiliateRebatePerInviteeCap != nil {
		affiliateRebatePerInviteeCap = *req.AffiliateRebatePerInviteeCap
	}
	if affiliateRebatePerInviteeCap < 0 {
		affiliateRebatePerInviteeCap = service.AffiliateRebatePerInviteeCapDefault
	}
	adminRechargeRebateEnabled := previousSettings.AdminRechargeRebateEnabled
	if req.AdminRechargeRebateEnabled != nil {
		adminRechargeRebateEnabled = *req.AdminRechargeRebateEnabled
	}
	// 通用表格配置：兼容旧客户端未传字段时保留当前值。
	if req.TableDefaultPageSize <= 0 {
		req.TableDefaultPageSize = previousSettings.TableDefaultPageSize
	}
	if req.TablePageSizeOptions == nil {
		req.TablePageSizeOptions = previousSettings.TablePageSizeOptions
	}
	req.SMTPHost = strings.TrimSpace(req.SMTPHost)
	req.SMTPUsername = strings.TrimSpace(req.SMTPUsername)
	req.SMTPPassword = strings.TrimSpace(req.SMTPPassword)
	req.SMTPFrom = strings.TrimSpace(req.SMTPFrom)
	req.SMTPFromName = strings.TrimSpace(req.SMTPFromName)

	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}

	// SMTP 配置保护：如果请求中 smtp_host 为空但数据库中已有配置，则保留已有 SMTP 配置
	// 防止前端加载设置失败时空表单覆盖已保存的 SMTP 配置
	if req.SMTPHost == "" && previousSettings.SMTPHost != "" {
		req.SMTPHost = previousSettings.SMTPHost
		req.SMTPPort = previousSettings.SMTPPort
		req.SMTPUsername = previousSettings.SMTPUsername
		req.SMTPFrom = previousSettings.SMTPFrom
		req.SMTPFromName = previousSettings.SMTPFromName
		req.SMTPUseTLS = previousSettings.SMTPUseTLS
	}

	loginAgreementMode := strings.ToLower(strings.TrimSpace(req.LoginAgreementMode))
	if loginAgreementMode == "" {
		loginAgreementMode = strings.ToLower(strings.TrimSpace(previousSettings.LoginAgreementMode))
	}
	switch loginAgreementMode {
	case "", "modal":
		loginAgreementMode = "modal"
	case "checkbox":
	default:
		response.BadRequest(c, "Login agreement mode must be modal or checkbox")
		return
	}
	loginAgreementUpdatedAt := strings.TrimSpace(req.LoginAgreementUpdatedAt)
	if loginAgreementUpdatedAt == "" {
		loginAgreementUpdatedAt = strings.TrimSpace(previousSettings.LoginAgreementUpdatedAt)
	}
	loginAgreementDocuments := loginAgreementDocumentsToService(req.LoginAgreementDocuments)
	if len(loginAgreementDocuments) == 0 {
		loginAgreementDocuments = previousSettings.LoginAgreementDocuments
	}
	for _, doc := range loginAgreementDocuments {
		if strings.TrimSpace(doc.Title) == "" {
			response.BadRequest(c, "Login agreement document title is required")
			return
		}
		if len(doc.Title) > 80 {
			response.BadRequest(c, "Login agreement document title is too long (max 80 characters)")
			return
		}
		if len(doc.ContentMD) > 200*1024 {
			response.BadRequest(c, "Login agreement document content is too large (max 200KB)")
			return
		}
	}
	if req.LoginAgreementEnabled && len(loginAgreementDocuments) == 0 {
		response.BadRequest(c, "Login agreement documents are required when enabled")
		return
	}

	// “购买订阅”页面配置验证
	purchaseEnabled := previousSettings.PurchaseSubscriptionEnabled
	if req.PurchaseSubscriptionEnabled != nil {
		purchaseEnabled = *req.PurchaseSubscriptionEnabled
	}
	purchaseURL := previousSettings.PurchaseSubscriptionURL
	if req.PurchaseSubscriptionURL != nil {
		purchaseURL = strings.TrimSpace(*req.PurchaseSubscriptionURL)
	}

	// - 启用时要求 URL 合法且非空
	// - 禁用时允许为空；若提供了 URL 也做基本校验，避免误配置
	if purchaseEnabled {
		if purchaseURL == "" {
			response.BadRequest(c, "Purchase Subscription URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	} else if purchaseURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	}

	// Frontend URL 验证

	// 自定义菜单项验证
	const (
		maxCustomMenuItems    = 20
		maxMenuItemLabelLen   = 50
		maxMenuItemURLLen     = 2048
		maxMenuItemIconSVGLen = 10 * 1024 // 10KB
		maxMenuItemIDLen      = 32
	)

	customMenuJSON := previousSettings.CustomMenuItems
	if req.CustomMenuItems != nil {
		items := *req.CustomMenuItems
		if len(items) > maxCustomMenuItems {
			response.BadRequest(c, "Too many custom menu items (max 20)")
			return
		}
		for i, item := range items {
			if strings.TrimSpace(item.Label) == "" {
				response.BadRequest(c, "Custom menu item label is required")
				return
			}
			if len(item.Label) > maxMenuItemLabelLen {
				response.BadRequest(c, "Custom menu item label is too long (max 50 characters)")
				return
			}
			urlTrimmed := strings.TrimSpace(item.URL)
			if strings.HasPrefix(urlTrimmed, "md:") {
				// Markdown page mode: URL = "md:<slug>"
				slug := strings.TrimPrefix(urlTrimmed, "md:")
				if slug == "" {
					response.BadRequest(c, "Custom menu item markdown slug cannot be empty (use md:slug format)")
					return
				}
			} else {
				if urlTrimmed == "" {
					response.BadRequest(c, "Custom menu item URL is required (use md:slug for markdown pages)")
					return
				}
				if len(item.URL) > maxMenuItemURLLen {
					response.BadRequest(c, "Custom menu item URL is too long (max 2048 characters)")
					return
				}
				if err := config.ValidateAbsoluteHTTPURL(urlTrimmed); err != nil {
					response.BadRequest(c, "Custom menu item URL must be an absolute http(s) URL or md:<slug>")
					return
				}
			}
			if item.Visibility != "user" && item.Visibility != "admin" {
				response.BadRequest(c, "Custom menu item visibility must be 'user' or 'admin'")
				return
			}
			if len(item.IconSVG) > maxMenuItemIconSVGLen {
				response.BadRequest(c, "Custom menu item icon SVG is too large (max 10KB)")
				return
			}
			// Auto-generate ID if missing
			if strings.TrimSpace(item.ID) == "" {
				id, err := generateMenuItemID()
				if err != nil {
					response.Error(c, http.StatusInternalServerError, "Failed to generate menu item ID")
					return
				}
				items[i].ID = id
			} else if len(item.ID) > maxMenuItemIDLen {
				response.BadRequest(c, "Custom menu item ID is too long (max 32 characters)")
				return
			} else if !menuItemIDPattern.MatchString(item.ID) {
				response.BadRequest(c, "Custom menu item ID contains invalid characters (only a-z, A-Z, 0-9, - and _ are allowed)")
				return
			}
		}
		// ID uniqueness check
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			if _, exists := seen[item.ID]; exists {
				response.BadRequest(c, "Duplicate custom menu item ID: "+item.ID)
				return
			}
			seen[item.ID] = struct{}{}
		}
		menuBytes, err := json.Marshal(items)
		if err != nil {
			response.BadRequest(c, "Failed to serialize custom menu items")
			return
		}
		customMenuJSON = string(menuBytes)
	}

	// 自定义端点验证
	const (
		maxCustomEndpoints        = 10
		maxEndpointNameLen        = 50
		maxEndpointURLLen         = 2048
		maxEndpointDescriptionLen = 200
	)

	customEndpointsJSON := previousSettings.CustomEndpoints
	if req.CustomEndpoints != nil {
		endpoints := *req.CustomEndpoints
		if len(endpoints) > maxCustomEndpoints {
			response.BadRequest(c, "Too many custom endpoints (max 10)")
			return
		}
		for _, ep := range endpoints {
			if strings.TrimSpace(ep.Name) == "" {
				response.BadRequest(c, "Custom endpoint name is required")
				return
			}
			if len(ep.Name) > maxEndpointNameLen {
				response.BadRequest(c, "Custom endpoint name is too long (max 50 characters)")
				return
			}
			if strings.TrimSpace(ep.Endpoint) == "" {
				response.BadRequest(c, "Custom endpoint URL is required")
				return
			}
			if len(ep.Endpoint) > maxEndpointURLLen {
				response.BadRequest(c, "Custom endpoint URL is too long (max 2048 characters)")
				return
			}
			if err := config.ValidateAbsoluteHTTPURL(strings.TrimSpace(ep.Endpoint)); err != nil {
				response.BadRequest(c, "Custom endpoint URL must be an absolute http(s) URL")
				return
			}
			if len(ep.Description) > maxEndpointDescriptionLen {
				response.BadRequest(c, "Custom endpoint description is too long (max 200 characters)")
				return
			}
		}
		endpointBytes, err := json.Marshal(endpoints)
		if err != nil {
			response.BadRequest(c, "Failed to serialize custom endpoints")
			return
		}
		customEndpointsJSON = string(endpointBytes)
	}

	// Ops metrics collector interval validation (seconds).
	if req.OpsMetricsIntervalSeconds != nil {
		v := *req.OpsMetricsIntervalSeconds
		if v < 60 {
			v = 60
		}
		if v > 3600 {
			v = 3600
		}
		req.OpsMetricsIntervalSeconds = &v
	}

	// 验证最低版本号格式（空字符串=禁用，或合法 semver）
	if req.MinClaudeCodeVersion != "" {
		if !semverPattern.MatchString(req.MinClaudeCodeVersion) {
			response.Error(c, http.StatusBadRequest, "min_claude_code_version must be empty or a valid semver (e.g. 2.1.63)")
			return
		}
	}

	// 验证最高版本号格式（空字符串=禁用，或合法 semver）
	if req.MaxClaudeCodeVersion != "" {
		if !semverPattern.MatchString(req.MaxClaudeCodeVersion) {
			response.Error(c, http.StatusBadRequest, "max_claude_code_version must be empty or a valid semver (e.g. 3.0.0)")
			return
		}
	}
	if req.AntigravityUserAgentVersion != nil {
		normalized := strings.TrimSpace(*req.AntigravityUserAgentVersion)
		req.AntigravityUserAgentVersion = &normalized
		if normalized != "" && !semverPattern.MatchString(normalized) {
			response.Error(c, http.StatusBadRequest, "antigravity_user_agent_version must be empty or a valid semver (e.g. 1.23.2)")
			return
		}
	}
	if req.OpenAICodexUserAgent != nil {
		normalized := strings.TrimSpace(*req.OpenAICodexUserAgent)
		req.OpenAICodexUserAgent = &normalized
		// 仅做长度上限保护，不限制具体格式（运维需要可自由调整 codex 版本号）
		if len(normalized) > 512 {
			response.Error(c, http.StatusBadRequest, "openai_codex_user_agent must be at most 512 characters")
			return
		}
	}
	if req.OpenAICodexClientVersion != nil {
		// 该值会被拼进出站 User-Agent 与 version 头，必须是合法版本号；空串表示跟随自动同步。
		normalized := strings.TrimSpace(*req.OpenAICodexClientVersion)
		if normalized != "" && service.NormalizeCodexClientVersion(normalized) == "" {
			response.Error(c, http.StatusBadRequest, "openai_codex_client_version must be empty or a valid version (e.g. 0.146.0)")
			return
		}
		req.OpenAICodexClientVersion = &normalized
	}
	if req.ClaudeCodeClientVersion != nil {
		// 该值会被拼进出站 User-Agent 与 billing attribution，必须是合法版本号；空串表示跟随自动同步。
		normalized := strings.TrimSpace(*req.ClaudeCodeClientVersion)
		if normalized != "" && service.NormalizeClaudeCodeClientVersion(normalized) == "" {
			response.Error(c, http.StatusBadRequest, "claude_code_client_version must be empty or a valid version (e.g. 2.1.258)")
			return
		}
		req.ClaudeCodeClientVersion = &normalized
	}

	// codex_cli_only 加固：最低/最高 Codex 版本（空=禁用，或合法 semver；max>=min）
	if req.MinCodexVersion != "" && !semverPattern.MatchString(req.MinCodexVersion) {
		response.Error(c, http.StatusBadRequest, "min_codex_version must be empty or a valid semver (e.g. 0.141.0)")
		return
	}
	if req.MaxCodexVersion != "" && !semverPattern.MatchString(req.MaxCodexVersion) {
		response.Error(c, http.StatusBadRequest, "max_codex_version must be empty or a valid semver (e.g. 0.200.0)")
		return
	}
	if req.MinCodexVersion != "" && req.MaxCodexVersion != "" && service.CompareVersions(req.MaxCodexVersion, req.MinCodexVersion) < 0 {
		response.Error(c, http.StatusBadRequest, "max_codex_version must be greater than or equal to min_codex_version")
		return
	}
	// codex_cli_only 黑/白名单：非空须为合法 []AllowedClientEntry JSON。
	// 黑名单 OR 宽 deny（允许 originator-only）；白名单双因子 AND，额外要求每条可命中（非空 originator + ua_contains）。
	if err := service.ValidateCodexClientEntriesJSON(req.CodexCLIOnlyBlacklist); err != nil {
		response.Error(c, http.StatusBadRequest, "codex_cli_only_blacklist "+err.Error())
		return
	}
	if err := service.ValidateCodexWhitelistEntriesJSON(req.CodexCLIOnlyWhitelist); err != nil {
		response.Error(c, http.StatusBadRequest, "codex_cli_only_whitelist "+err.Error())
		return
	}
	if err := service.ValidateEngineFingerprintSignalsJSON(req.CodexCLIOnlyEngineFingerprintSignals); err != nil {
		response.Error(c, http.StatusBadRequest, "codex_cli_only_engine_fingerprint_signals "+err.Error())
		return
	}

	// 交叉验证：如果同时设置了最低和最高版本号，最高版本号必须 >= 最低版本号
	if req.MinClaudeCodeVersion != "" && req.MaxClaudeCodeVersion != "" {
		if service.CompareVersions(req.MaxClaudeCodeVersion, req.MinClaudeCodeVersion) < 0 {
			response.Error(c, http.StatusBadRequest, "max_claude_code_version must be greater than or equal to min_claude_code_version")
			return
		}
	}

	if req.CyberPolicyUserAllowlist != nil {
		if _, err := service.ParseCyberPolicyUserAllowlist(*req.CyberPolicyUserAllowlist); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	// cyber 会话屏蔽 TTL 校验：提供时必须 > 0
	if req.CyberSessionBlockTTLSeconds != nil && *req.CyberSessionBlockTTLSeconds <= 0 {
		response.BadRequest(c, "cyber_session_block_ttl_seconds must be > 0")
		return
	}

	settings := &service.SystemSettings{
		FrontendURL: req.FrontendURL,
		// 系统全局 platform quota 默认值（整体替换语义）
		DefaultPlatformQuotas:       req.DefaultPlatformQuotas,
		AccountSchedulingThresholds: req.AccountSchedulingThresholds,

		AuditLogRetentionDays:   req.AuditLogRetentionDays,
		LoginAgreementEnabled:   req.LoginAgreementEnabled,
		LoginAgreementMode:      loginAgreementMode,
		LoginAgreementUpdatedAt: loginAgreementUpdatedAt,
		LoginAgreementDocuments: loginAgreementDocuments,
		SMTPHost:                req.SMTPHost,
		SMTPPort:                req.SMTPPort,
		SMTPUsername:            req.SMTPUsername,
		SMTPPassword:            req.SMTPPassword,
		SMTPFrom:                req.SMTPFrom,
		SMTPFromName:            req.SMTPFromName,
		SMTPUseTLS:              req.SMTPUseTLS,

		APIKeyACLTrustForwardedIP: func() bool {
			if req.APIKeyACLTrustForwardedIP != nil {
				return *req.APIKeyACLTrustForwardedIP
			}
			return previousSettings.APIKeyACLTrustForwardedIP
		}(),
		ForwardedClientIPHeaders: forwardedClientIPHeaders,

		WeChatConnectMPAppID:     req.WeChatConnectMPAppID,
		WeChatConnectMPAppSecret: req.WeChatConnectMPAppSecret,

		WeChatConnectMPEnabled: req.WeChatConnectMPEnabled,

		SiteName:                    req.SiteName,
		SiteLogo:                    req.SiteLogo,
		SiteSubtitle:                req.SiteSubtitle,
		APIBaseURL:                  req.APIBaseURL,
		ContactInfo:                 req.ContactInfo,
		DocURL:                      req.DocURL,
		HomeContent:                 req.HomeContent,
		CompactHomeEnabled:          req.CompactHomeEnabled,
		HideCcsImportButton:         req.HideCcsImportButton,
		PurchaseSubscriptionEnabled: purchaseEnabled,
		PurchaseSubscriptionURL:     purchaseURL,
		TableDefaultPageSize:        req.TableDefaultPageSize,
		TablePageSizeOptions:        req.TablePageSizeOptions,
		CustomMenuItems:             customMenuJSON,
		CustomEndpoints:             customEndpointsJSON,

		AffiliateRebateRate:          affiliateRebateRate,
		AffiliateRebateFreezeHours:   affiliateRebateFreezeHours,
		AffiliateRebateDurationDays:  affiliateRebateDurationDays,
		AffiliateRebatePerInviteeCap: affiliateRebatePerInviteeCap,
		AdminRechargeRebateEnabled:   adminRechargeRebateEnabled,

		EnableModelFallback:         req.EnableModelFallback,
		FallbackModelAnthropic:      req.FallbackModelAnthropic,
		FallbackModelOpenAI:         req.FallbackModelOpenAI,
		FallbackModelGemini:         req.FallbackModelGemini,
		FallbackModelAntigravity:    req.FallbackModelAntigravity,
		EnableIdentityPatch:         req.EnableIdentityPatch,
		IdentityPatchPrompt:         req.IdentityPatchPrompt,
		MinClaudeCodeVersion:        req.MinClaudeCodeVersion,
		MaxClaudeCodeVersion:        req.MaxClaudeCodeVersion,
		AllowUngroupedKeyScheduling: req.AllowUngroupedKeyScheduling,
		BackendModeEnabled:          req.BackendModeEnabled,
		AllowUserViewErrorRequests: func() bool {
			if req.AllowUserViewErrorRequests != nil {
				return *req.AllowUserViewErrorRequests
			}
			return previousSettings.AllowUserViewErrorRequests
		}(),
		OpsMonitoringEnabled: func() bool {
			if req.OpsMonitoringEnabled != nil {
				return *req.OpsMonitoringEnabled
			}
			return previousSettings.OpsMonitoringEnabled
		}(),
		OpsRealtimeMonitoringEnabled: func() bool {
			if req.OpsRealtimeMonitoringEnabled != nil {
				return *req.OpsRealtimeMonitoringEnabled
			}
			return previousSettings.OpsRealtimeMonitoringEnabled
		}(),
		OpsQueryModeDefault: func() string {
			if req.OpsQueryModeDefault != nil {
				return *req.OpsQueryModeDefault
			}
			return previousSettings.OpsQueryModeDefault
		}(),
		OpsMetricsIntervalSeconds: func() int {
			if req.OpsMetricsIntervalSeconds != nil {
				return *req.OpsMetricsIntervalSeconds
			}
			return previousSettings.OpsMetricsIntervalSeconds
		}(),
		EnableFingerprintUnification: func() bool {
			if req.EnableFingerprintUnification != nil {
				return *req.EnableFingerprintUnification
			}
			return previousSettings.EnableFingerprintUnification
		}(),
		OpenAITTFTMode: func() string {
			if req.OpenAITTFTMode != nil {
				return *req.OpenAITTFTMode
			}
			return previousSettings.OpenAITTFTMode
		}(),
		EnableMetadataPassthrough: func() bool {
			if req.EnableMetadataPassthrough != nil {
				return *req.EnableMetadataPassthrough
			}
			return previousSettings.EnableMetadataPassthrough
		}(),
		EnableCCHSigning: func() bool {
			if req.EnableCCHSigning != nil {
				return *req.EnableCCHSigning
			}
			return previousSettings.EnableCCHSigning
		}(),
		EnableClaudeOAuthSystemPromptInjection: func() bool {
			if req.EnableClaudeOAuthSystemPromptInjection != nil {
				return *req.EnableClaudeOAuthSystemPromptInjection
			}
			return previousSettings.EnableClaudeOAuthSystemPromptInjection
		}(),
		ClaudeOAuthSystemPrompt: func() string {
			if req.ClaudeOAuthSystemPrompt != nil {
				return *req.ClaudeOAuthSystemPrompt
			}
			return previousSettings.ClaudeOAuthSystemPrompt
		}(),
		ClaudeOAuthSystemPromptBlocks: func() string {
			if req.ClaudeOAuthSystemPromptBlocks != nil {
				return *req.ClaudeOAuthSystemPromptBlocks
			}
			return previousSettings.ClaudeOAuthSystemPromptBlocks
		}(),
		EnableAnthropicCacheTTL1hInjection: func() bool {
			if req.EnableAnthropicCacheTTL1hInjection != nil {
				return *req.EnableAnthropicCacheTTL1hInjection
			}
			return previousSettings.EnableAnthropicCacheTTL1hInjection
		}(),
		RewriteMessageCacheControl: func() bool {
			if req.RewriteMessageCacheControl != nil {
				return *req.RewriteMessageCacheControl
			}
			return previousSettings.RewriteMessageCacheControl
		}(),
		EnableClientDatelineNormalization: func() bool {
			if req.EnableClientDatelineNormalization != nil {
				return *req.EnableClientDatelineNormalization
			}
			return previousSettings.EnableClientDatelineNormalization
		}(),
		AntigravityUserAgentVersion: func() string {
			if req.AntigravityUserAgentVersion != nil {
				return *req.AntigravityUserAgentVersion
			}
			return previousSettings.AntigravityUserAgentVersion
		}(),
		OpenAICodexUserAgent: func() string {
			if req.OpenAICodexUserAgent != nil {
				return *req.OpenAICodexUserAgent
			}
			return previousSettings.OpenAICodexUserAgent
		}(),
		OpenAICodexClientVersion: func() string {
			if req.OpenAICodexClientVersion != nil {
				return *req.OpenAICodexClientVersion
			}
			return previousSettings.OpenAICodexClientVersion
		}(),
		// 同步值由自动同步任务独占写入，面板保存时原样带回，避免被清空。
		OpenAICodexClientVersionSynced: previousSettings.OpenAICodexClientVersionSynced,
		OpenAICodexVersionAutoSyncEnabled: func() bool {
			if req.OpenAICodexVersionAutoSyncEnabled != nil {
				return *req.OpenAICodexVersionAutoSyncEnabled
			}
			return previousSettings.OpenAICodexVersionAutoSyncEnabled
		}(),
		ClaudeCodeClientVersion: func() string {
			if req.ClaudeCodeClientVersion != nil {
				return *req.ClaudeCodeClientVersion
			}
			return previousSettings.ClaudeCodeClientVersion
		}(),
		// 同步值由自动同步任务独占写入，面板保存时原样带回，避免被清空。
		ClaudeCodeClientVersionSynced: previousSettings.ClaudeCodeClientVersionSynced,
		ClaudeCodeVersionAutoSyncEnabled: func() bool {
			if req.ClaudeCodeVersionAutoSyncEnabled != nil {
				return *req.ClaudeCodeVersionAutoSyncEnabled
			}
			return previousSettings.ClaudeCodeVersionAutoSyncEnabled
		}(),
		MinCodexVersion:       strings.TrimSpace(req.MinCodexVersion),
		MaxCodexVersion:       strings.TrimSpace(req.MaxCodexVersion),
		CodexCLIOnlyBlacklist: strings.TrimSpace(req.CodexCLIOnlyBlacklist),
		CodexCLIOnlyWhitelist: strings.TrimSpace(req.CodexCLIOnlyWhitelist),
		CodexCLIOnlyAllowAppServerClients: func() bool {
			if req.CodexCLIOnlyAllowAppServerClients != nil {
				return *req.CodexCLIOnlyAllowAppServerClients
			}
			return previousSettings.CodexCLIOnlyAllowAppServerClients
		}(),
		CodexCLIOnlyEngineFingerprintSignals: strings.TrimSpace(req.CodexCLIOnlyEngineFingerprintSignals),
		PaymentVisibleMethodAlipaySource: func() string {
			if req.PaymentVisibleMethodAlipaySource != nil {
				return strings.TrimSpace(*req.PaymentVisibleMethodAlipaySource)
			}
			return previousSettings.PaymentVisibleMethodAlipaySource
		}(),
		PaymentVisibleMethodWxpaySource: func() string {
			if req.PaymentVisibleMethodWxpaySource != nil {
				return strings.TrimSpace(*req.PaymentVisibleMethodWxpaySource)
			}
			return previousSettings.PaymentVisibleMethodWxpaySource
		}(),
		PaymentVisibleMethodAlipayEnabled: func() bool {
			if req.PaymentVisibleMethodAlipayEnabled != nil {
				return *req.PaymentVisibleMethodAlipayEnabled
			}
			return previousSettings.PaymentVisibleMethodAlipayEnabled
		}(),
		PaymentVisibleMethodWxpayEnabled: func() bool {
			if req.PaymentVisibleMethodWxpayEnabled != nil {
				return *req.PaymentVisibleMethodWxpayEnabled
			}
			return previousSettings.PaymentVisibleMethodWxpayEnabled
		}(),
		OpenAILowUpstreamRatePriorityEnabled: func() bool {
			if req.OpenAILowUpstreamRatePriorityEnabled != nil {
				return *req.OpenAILowUpstreamRatePriorityEnabled
			}
			return previousSettings.OpenAILowUpstreamRatePriorityEnabled
		}(),
		OpenAIOAuthSchedulingRateMultiplier: func() *float64 {
			// Omitted fields preserve the override; explicit null clears it.
			if _, sent := sentFields[service.SettingKeyOpenAIOAuthSchedulingRateMultiplier]; sent {
				return req.OpenAIOAuthSchedulingRateMultiplier
			}
			return previousSettings.OpenAIOAuthSchedulingRateMultiplier
		}(),
		OpenAIAdvancedSchedulerEnabled: func() bool {
			if req.OpenAIAdvancedSchedulerEnabled != nil {
				return *req.OpenAIAdvancedSchedulerEnabled
			}
			return previousSettings.OpenAIAdvancedSchedulerEnabled
		}(),
		OpenAIAdvancedSchedulerStickyWeightedEnabled: func() bool {
			if req.OpenAIAdvancedSchedulerStickyWeightedEnabled != nil {
				return *req.OpenAIAdvancedSchedulerStickyWeightedEnabled
			}
			return previousSettings.OpenAIAdvancedSchedulerStickyWeightedEnabled
		}(),
		OpenAIAdvancedSchedulerSubscriptionPriorityEnabled: func() bool {
			if req.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled != nil {
				return *req.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled
			}
			return previousSettings.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled
		}(),
		OpenAIAdvancedSchedulerLBTopK:                 stringSetting(req.OpenAIAdvancedSchedulerLBTopK, previousSettings.OpenAIAdvancedSchedulerLBTopK),
		OpenAIAdvancedSchedulerWeightPriority:         stringSetting(req.OpenAIAdvancedSchedulerWeightPriority, previousSettings.OpenAIAdvancedSchedulerWeightPriority),
		OpenAIAdvancedSchedulerWeightLoad:             stringSetting(req.OpenAIAdvancedSchedulerWeightLoad, previousSettings.OpenAIAdvancedSchedulerWeightLoad),
		OpenAIAdvancedSchedulerWeightQueue:            stringSetting(req.OpenAIAdvancedSchedulerWeightQueue, previousSettings.OpenAIAdvancedSchedulerWeightQueue),
		OpenAIAdvancedSchedulerWeightErrorRate:        stringSetting(req.OpenAIAdvancedSchedulerWeightErrorRate, previousSettings.OpenAIAdvancedSchedulerWeightErrorRate),
		OpenAIAdvancedSchedulerWeightTTFT:             stringSetting(req.OpenAIAdvancedSchedulerWeightTTFT, previousSettings.OpenAIAdvancedSchedulerWeightTTFT),
		OpenAIAdvancedSchedulerWeightReset:            stringSetting(req.OpenAIAdvancedSchedulerWeightReset, previousSettings.OpenAIAdvancedSchedulerWeightReset),
		OpenAIAdvancedSchedulerWeightQuotaHeadroom:    stringSetting(req.OpenAIAdvancedSchedulerWeightQuotaHeadroom, previousSettings.OpenAIAdvancedSchedulerWeightQuotaHeadroom),
		OpenAIAdvancedSchedulerWeightUpstreamCost:     stringSetting(req.OpenAIAdvancedSchedulerWeightUpstreamCost, previousSettings.OpenAIAdvancedSchedulerWeightUpstreamCost),
		OpenAIAdvancedSchedulerWeightPreviousResponse: stringSetting(req.OpenAIAdvancedSchedulerWeightPreviousResponse, previousSettings.OpenAIAdvancedSchedulerWeightPreviousResponse),
		OpenAIAdvancedSchedulerWeightSessionSticky:    stringSetting(req.OpenAIAdvancedSchedulerWeightSessionSticky, previousSettings.OpenAIAdvancedSchedulerWeightSessionSticky),
		BalanceLowNotifyEnabled: func() bool {
			if req.BalanceLowNotifyEnabled != nil {
				return *req.BalanceLowNotifyEnabled
			}
			return previousSettings.BalanceLowNotifyEnabled
		}(),
		BalanceLowNotifyThreshold: func() float64 {
			if req.BalanceLowNotifyThreshold != nil {
				return *req.BalanceLowNotifyThreshold
			}
			return previousSettings.BalanceLowNotifyThreshold
		}(),
		BalanceLowNotifyRechargeURL: func() string {
			if req.BalanceLowNotifyRechargeURL != nil {
				return *req.BalanceLowNotifyRechargeURL
			}
			return previousSettings.BalanceLowNotifyRechargeURL
		}(),
		SubscriptionExpiryNotifyEnabled: func() bool {
			if req.SubscriptionExpiryNotifyEnabled != nil {
				return *req.SubscriptionExpiryNotifyEnabled
			}
			return previousSettings.SubscriptionExpiryNotifyEnabled
		}(),
		AccountQuotaNotifyEnabled: func() bool {
			if req.AccountQuotaNotifyEnabled != nil {
				return *req.AccountQuotaNotifyEnabled
			}
			return previousSettings.AccountQuotaNotifyEnabled
		}(),
		AccountQuotaNotifyEmails: func() []service.NotifyEmailEntry {
			if req.AccountQuotaNotifyEmails != nil {
				return dto.NotifyEmailEntriesToService(*req.AccountQuotaNotifyEmails)
			}
			return previousSettings.AccountQuotaNotifyEmails
		}(),
		ChannelMonitorEnabled: func() bool {
			if req.ChannelMonitorEnabled != nil {
				return *req.ChannelMonitorEnabled
			}
			return previousSettings.ChannelMonitorEnabled
		}(),
		ChannelMonitorMode: func() string {
			if req.ChannelMonitorMode != nil {
				return *req.ChannelMonitorMode
			}
			return previousSettings.ChannelMonitorMode
		}(),
		ChannelMonitorDefaultIntervalSeconds: func() int {
			if req.ChannelMonitorDefaultIntervalSeconds != nil {
				return *req.ChannelMonitorDefaultIntervalSeconds
			}
			return previousSettings.ChannelMonitorDefaultIntervalSeconds
		}(),
		ChannelMonitorHideThroughput: func() bool {
			if req.ChannelMonitorHideThroughput != nil {
				return *req.ChannelMonitorHideThroughput
			}
			return previousSettings.ChannelMonitorHideThroughput
		}(),
		ChannelMonitorShowQuota: func() bool {
			if req.ChannelMonitorShowQuota != nil {
				return *req.ChannelMonitorShowQuota
			}
			return previousSettings.ChannelMonitorShowQuota
		}(),
		ChannelMonitorHideUserRanking: func() bool {
			if req.ChannelMonitorHideUserRanking != nil {
				return *req.ChannelMonitorHideUserRanking
			}
			return previousSettings.ChannelMonitorHideUserRanking
		}(),
		GrokDefaultTextModel: func() string {
			if req.GrokDefaultTextModel != nil {
				return *req.GrokDefaultTextModel
			}
			return previousSettings.GrokDefaultTextModel
		}(),
		GrokCrossClientModelMapEnabled: func() bool {
			if req.GrokCrossClientModelMapEnabled != nil {
				return *req.GrokCrossClientModelMapEnabled
			}
			return previousSettings.GrokCrossClientModelMapEnabled
		}(),
		GrokDefaultBaseURLMode: func() string {
			if req.GrokDefaultBaseURLMode != nil {
				return strings.TrimSpace(*req.GrokDefaultBaseURLMode)
			}
			return previousSettings.GrokDefaultBaseURLMode
		}(),
		AvailableChannelsEnabled: func() bool {
			if req.AvailableChannelsEnabled != nil {
				return *req.AvailableChannelsEnabled
			}
			return previousSettings.AvailableChannelsEnabled
		}(),
		SubscriptionEnabled: func() bool {
			if req.SubscriptionEnabled != nil {
				return *req.SubscriptionEnabled
			}
			return previousSettings.SubscriptionEnabled
		}(),
		ModelPlazaEnabled: func() bool {
			if req.ModelPlazaEnabled != nil {
				return *req.ModelPlazaEnabled
			}
			return previousSettings.ModelPlazaEnabled
		}(),
		ModelPlazaRequireAuth: func() bool {
			if req.ModelPlazaRequireAuth != nil {
				return *req.ModelPlazaRequireAuth
			}
			return previousSettings.ModelPlazaRequireAuth
		}(),
		ModelPlazaDescription: func() string {
			if req.ModelPlazaDescription != nil {
				return *req.ModelPlazaDescription
			}
			return previousSettings.ModelPlazaDescription
		}(),
		PluginManagementEnabled: func() bool {
			if req.PluginManagementEnabled != nil {
				return *req.PluginManagementEnabled
			}
			return previousSettings.PluginManagementEnabled
		}(),
		AffiliateEnabled: func() bool {
			if req.AffiliateEnabled != nil {
				return *req.AffiliateEnabled
			}
			return previousSettings.AffiliateEnabled
		}(),
		RiskControlEnabled: func() bool {
			if req.RiskControlEnabled != nil {
				return *req.RiskControlEnabled
			}
			return previousSettings.RiskControlEnabled
		}(),
		CyberPolicyUserAllowlist: func() string {
			if req.CyberPolicyUserAllowlist != nil {
				return *req.CyberPolicyUserAllowlist
			}
			return previousSettings.CyberPolicyUserAllowlist
		}(),
		CyberSessionBlockEnabled: func() bool {
			if req.CyberSessionBlockEnabled != nil {
				return *req.CyberSessionBlockEnabled
			}
			return previousSettings.CyberSessionBlockEnabled
		}(),
		CyberSessionBlockTTLSeconds: func() int {
			if req.CyberSessionBlockTTLSeconds != nil {
				return *req.CyberSessionBlockTTLSeconds
			}
			return previousSettings.CyberSessionBlockTTLSeconds
		}(),
	}

	// req.AuthSourceXxxPlatformQuotas 为 nil 表示本次请求未包含该 source 的 quota 配置（保留 previousAuthSourceDefaults 中的值）；
	// non-nil（含 empty map）表示整体覆盖：empty map = 清空该 source 的所有 quota 配置。

	if err := h.settingService.UpdateSettingsOmitting(c.Request.Context(), settings, omitted); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.opsService != nil {
		h.opsService.SetMonitoringEnabled(settings.OpsMonitoringEnabled)
	}

	// Update OpenAI fast policy (stored under dedicated key, only when provided).
	if req.OpenAIFastPolicySettings != nil {
		if err := h.settingService.SetOpenAIFastPolicySettings(c.Request.Context(), openaiFastPolicySettingsFromDTO(req.OpenAIFastPolicySettings)); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	// Update payment configuration (integrated into system settings).
	// Skip if no payment fields were provided (prevents accidental wipe).
	if h.paymentConfigService != nil && hasPaymentFields(req) {
		paymentReq := service.UpdatePaymentConfigRequest{
			Enabled:                       req.PaymentEnabled,
			MinAmount:                     req.PaymentMinAmount,
			MaxAmount:                     req.PaymentMaxAmount,
			DailyLimit:                    req.PaymentDailyLimit,
			OrderTimeoutMin:               req.PaymentOrderTimeoutMin,
			MaxPendingOrders:              req.PaymentMaxPendingOrders,
			EnabledTypes:                  req.PaymentEnabledTypes,
			BalanceDisabled:               req.PaymentBalanceDisabled,
			BalanceRechargeMultiplier:     req.PaymentBalanceRechargeMultiplier,
			SubscriptionUSDToCNYRate:      req.PaymentSubscriptionUSDToCNYRate,
			RechargeFeeRate:               req.PaymentRechargeFeeRate,
			RechargeBonusTiers:            rechargeBonusTiersFromDTO(req.PaymentRechargeBonusTiers),
			RechargeBonusMode:             req.PaymentRechargeBonusMode,
			RechargeBonusNotice:           req.PaymentRechargeBonusNotice,
			LoadBalanceStrategy:           req.PaymentLoadBalanceStrat,
			ProductNamePrefix:             req.PaymentProductNamePrefix,
			ProductNameSuffix:             req.PaymentProductNameSuffix,
			HelpImageURL:                  req.PaymentHelpImageURL,
			HelpText:                      req.PaymentHelpText,
			CancelRateLimitEnabled:        req.PaymentCancelRateLimitEnabled,
			CancelRateLimitMax:            req.PaymentCancelRateLimitMax,
			CancelRateLimitWindow:         req.PaymentCancelRateLimitWindow,
			CancelRateLimitUnit:           req.PaymentCancelRateLimitUnit,
			CancelRateLimitMode:           req.PaymentCancelRateLimitMode,
			AlipayForceQRCode:             req.PaymentAlipayForceQRCode,
			AlipayMobilePrecreateDeepLink: req.PaymentAlipayMobilePrecreateDeepLink,
		}
		if err := h.paymentConfigService.UpdatePaymentConfig(c.Request.Context(), paymentReq); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		// Refresh in-memory provider registry so config changes take effect immediately
		if h.paymentService != nil {
			h.paymentService.RefreshProviders(c.Request.Context())
		}
	}

	h.auditSettingsUpdate(c, previousSettings, settings, auditReq)

	// 重新获取设置返回
	updatedSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Reload payment config for response
	var updatedPaymentCfg *service.PaymentConfig
	if h.paymentConfigService != nil {
		updatedPaymentCfg, _ = h.paymentConfigService.GetPaymentConfig(c.Request.Context())
	}
	if updatedPaymentCfg == nil {
		updatedPaymentCfg = &service.PaymentConfig{}
	}

	payload := dto.SystemSettings{
		FrontendURL: updatedSettings.FrontendURL,

		AuditLogRetentionDays:   updatedSettings.AuditLogRetentionDays,
		LoginAgreementEnabled:   updatedSettings.LoginAgreementEnabled,
		LoginAgreementMode:      updatedSettings.LoginAgreementMode,
		LoginAgreementUpdatedAt: updatedSettings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments: loginAgreementDocumentsToDTO(updatedSettings.LoginAgreementDocuments),
		SMTPHost:                updatedSettings.SMTPHost,
		SMTPPort:                updatedSettings.SMTPPort,
		SMTPUsername:            updatedSettings.SMTPUsername,
		SMTPPasswordConfigured:  updatedSettings.SMTPPasswordConfigured,
		SMTPFrom:                updatedSettings.SMTPFrom,
		SMTPFromName:            updatedSettings.SMTPFromName,
		SMTPUseTLS:              updatedSettings.SMTPUseTLS,

		APIKeyACLTrustForwardedIP: updatedSettings.APIKeyACLTrustForwardedIP,
		ForwardedClientIPHeaders:  updatedSettings.ForwardedClientIPHeaders,

		WeChatConnectMPAppID:               updatedSettings.WeChatConnectMPAppID,
		WeChatConnectMPAppSecretConfigured: updatedSettings.WeChatConnectMPAppSecretConfigured,

		WeChatConnectMPEnabled: updatedSettings.WeChatConnectMPEnabled,

		SiteName:                    updatedSettings.SiteName,
		SiteLogo:                    updatedSettings.SiteLogo,
		SiteSubtitle:                updatedSettings.SiteSubtitle,
		APIBaseURL:                  updatedSettings.APIBaseURL,
		ContactInfo:                 updatedSettings.ContactInfo,
		DocURL:                      updatedSettings.DocURL,
		HomeContent:                 updatedSettings.HomeContent,
		CompactHomeEnabled:          updatedSettings.CompactHomeEnabled,
		HideCcsImportButton:         updatedSettings.HideCcsImportButton,
		PurchaseSubscriptionEnabled: updatedSettings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:     updatedSettings.PurchaseSubscriptionURL,
		TableDefaultPageSize:        updatedSettings.TableDefaultPageSize,
		TablePageSizeOptions:        updatedSettings.TablePageSizeOptions,
		CustomMenuItems:             dto.ParseCustomMenuItems(updatedSettings.CustomMenuItems),
		CustomEndpoints:             dto.ParseCustomEndpoints(updatedSettings.CustomEndpoints),

		AffiliateRebateRate:          updatedSettings.AffiliateRebateRate,
		AffiliateRebateFreezeHours:   updatedSettings.AffiliateRebateFreezeHours,
		AffiliateRebateDurationDays:  updatedSettings.AffiliateRebateDurationDays,
		AffiliateRebatePerInviteeCap: updatedSettings.AffiliateRebatePerInviteeCap,
		AdminRechargeRebateEnabled:   updatedSettings.AdminRechargeRebateEnabled,

		EnableModelFallback:                                    updatedSettings.EnableModelFallback,
		FallbackModelAnthropic:                                 updatedSettings.FallbackModelAnthropic,
		FallbackModelOpenAI:                                    updatedSettings.FallbackModelOpenAI,
		FallbackModelGemini:                                    updatedSettings.FallbackModelGemini,
		FallbackModelAntigravity:                               updatedSettings.FallbackModelAntigravity,
		EnableIdentityPatch:                                    updatedSettings.EnableIdentityPatch,
		IdentityPatchPrompt:                                    updatedSettings.IdentityPatchPrompt,
		OpsMonitoringEnabled:                                   updatedSettings.OpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:                           updatedSettings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                                    updatedSettings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:                              updatedSettings.OpsMetricsIntervalSeconds,
		MinClaudeCodeVersion:                                   updatedSettings.MinClaudeCodeVersion,
		MaxClaudeCodeVersion:                                   updatedSettings.MaxClaudeCodeVersion,
		AllowUngroupedKeyScheduling:                            updatedSettings.AllowUngroupedKeyScheduling,
		BackendModeEnabled:                                     updatedSettings.BackendModeEnabled,
		EnableFingerprintUnification:                           updatedSettings.EnableFingerprintUnification,
		EnableMetadataPassthrough:                              updatedSettings.EnableMetadataPassthrough,
		EnableCCHSigning:                                       updatedSettings.EnableCCHSigning,
		EnableClaudeOAuthSystemPromptInjection:                 updatedSettings.EnableClaudeOAuthSystemPromptInjection,
		ClaudeOAuthSystemPrompt:                                updatedSettings.ClaudeOAuthSystemPrompt,
		ClaudeOAuthSystemPromptBlocks:                          updatedSettings.ClaudeOAuthSystemPromptBlocks,
		EnableAnthropicCacheTTL1hInjection:                     updatedSettings.EnableAnthropicCacheTTL1hInjection,
		RewriteMessageCacheControl:                             updatedSettings.RewriteMessageCacheControl,
		EnableClientDatelineNormalization:                      updatedSettings.EnableClientDatelineNormalization,
		AntigravityUserAgentVersion:                            updatedSettings.AntigravityUserAgentVersion,
		OpenAICodexUserAgent:                                   updatedSettings.OpenAICodexUserAgent,
		OpenAICodexClientVersion:                               updatedSettings.OpenAICodexClientVersion,
		OpenAICodexClientVersionSynced:                         updatedSettings.OpenAICodexClientVersionSynced,
		OpenAICodexVersionAutoSyncEnabled:                      updatedSettings.OpenAICodexVersionAutoSyncEnabled,
		ClaudeCodeClientVersion:                                updatedSettings.ClaudeCodeClientVersion,
		ClaudeCodeClientVersionSynced:                          updatedSettings.ClaudeCodeClientVersionSynced,
		ClaudeCodeVersionAutoSyncEnabled:                       updatedSettings.ClaudeCodeVersionAutoSyncEnabled,
		MinCodexVersion:                                        updatedSettings.MinCodexVersion,
		MaxCodexVersion:                                        updatedSettings.MaxCodexVersion,
		CodexCLIOnlyBlacklist:                                  updatedSettings.CodexCLIOnlyBlacklist,
		CodexCLIOnlyWhitelist:                                  updatedSettings.CodexCLIOnlyWhitelist,
		CodexCLIOnlyAllowAppServerClients:                      updatedSettings.CodexCLIOnlyAllowAppServerClients,
		CodexCLIOnlyEngineFingerprintSignals:                   updatedSettings.CodexCLIOnlyEngineFingerprintSignals,
		PaymentVisibleMethodAlipaySource:                       updatedSettings.PaymentVisibleMethodAlipaySource,
		PaymentVisibleMethodWxpaySource:                        updatedSettings.PaymentVisibleMethodWxpaySource,
		PaymentVisibleMethodAlipayEnabled:                      updatedSettings.PaymentVisibleMethodAlipayEnabled,
		PaymentVisibleMethodWxpayEnabled:                       updatedSettings.PaymentVisibleMethodWxpayEnabled,
		OpenAILowUpstreamRatePriorityEnabled:                   updatedSettings.OpenAILowUpstreamRatePriorityEnabled,
		OpenAIOAuthSchedulingRateMultiplier:                    updatedSettings.OpenAIOAuthSchedulingRateMultiplier,
		OpenAIAdvancedSchedulerEnabled:                         updatedSettings.OpenAIAdvancedSchedulerEnabled,
		OpenAIAdvancedSchedulerStickyWeightedEnabled:           updatedSettings.OpenAIAdvancedSchedulerStickyWeightedEnabled,
		OpenAIAdvancedSchedulerSubscriptionPriorityEnabled:     updatedSettings.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled,
		OpenAIAdvancedSchedulerLBTopK:                          updatedSettings.OpenAIAdvancedSchedulerLBTopK,
		OpenAIAdvancedSchedulerWeightPriority:                  updatedSettings.OpenAIAdvancedSchedulerWeightPriority,
		OpenAIAdvancedSchedulerWeightLoad:                      updatedSettings.OpenAIAdvancedSchedulerWeightLoad,
		OpenAIAdvancedSchedulerWeightQueue:                     updatedSettings.OpenAIAdvancedSchedulerWeightQueue,
		OpenAIAdvancedSchedulerWeightErrorRate:                 updatedSettings.OpenAIAdvancedSchedulerWeightErrorRate,
		OpenAIAdvancedSchedulerWeightTTFT:                      updatedSettings.OpenAIAdvancedSchedulerWeightTTFT,
		OpenAIAdvancedSchedulerWeightReset:                     updatedSettings.OpenAIAdvancedSchedulerWeightReset,
		OpenAIAdvancedSchedulerWeightQuotaHeadroom:             updatedSettings.OpenAIAdvancedSchedulerWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerWeightUpstreamCost:              updatedSettings.OpenAIAdvancedSchedulerWeightUpstreamCost,
		OpenAIAdvancedSchedulerWeightPreviousResponse:          updatedSettings.OpenAIAdvancedSchedulerWeightPreviousResponse,
		OpenAIAdvancedSchedulerWeightSessionSticky:             updatedSettings.OpenAIAdvancedSchedulerWeightSessionSticky,
		OpenAIAdvancedSchedulerEffectiveLBTopK:                 updatedSettings.OpenAIAdvancedSchedulerEffectiveLBTopK,
		OpenAIAdvancedSchedulerEffectiveWeightPriority:         updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightPriority,
		OpenAIAdvancedSchedulerEffectiveWeightLoad:             updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightLoad,
		OpenAIAdvancedSchedulerEffectiveWeightQueue:            updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightQueue,
		OpenAIAdvancedSchedulerEffectiveWeightErrorRate:        updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightErrorRate,
		OpenAIAdvancedSchedulerEffectiveWeightTTFT:             updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightTTFT,
		OpenAIAdvancedSchedulerEffectiveWeightReset:            updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightReset,
		OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom:    updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerEffectiveWeightUpstreamCost:     updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightUpstreamCost,
		OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse: updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse,
		OpenAIAdvancedSchedulerEffectiveWeightSessionSticky:    updatedSettings.OpenAIAdvancedSchedulerEffectiveWeightSessionSticky,
		BalanceLowNotifyEnabled:                                updatedSettings.BalanceLowNotifyEnabled,
		BalanceLowNotifyThreshold:                              updatedSettings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:                            updatedSettings.BalanceLowNotifyRechargeURL,
		SubscriptionExpiryNotifyEnabled:                        updatedSettings.SubscriptionExpiryNotifyEnabled,
		AccountQuotaNotifyEnabled:                              updatedSettings.AccountQuotaNotifyEnabled,
		AccountQuotaNotifyEmails:                               dto.NotifyEmailEntriesFromService(updatedSettings.AccountQuotaNotifyEmails),
		PaymentEnabled:                                         updatedPaymentCfg.Enabled,
		PaymentMinAmount:                                       updatedPaymentCfg.MinAmount,
		PaymentMaxAmount:                                       updatedPaymentCfg.MaxAmount,
		PaymentDailyLimit:                                      updatedPaymentCfg.DailyLimit,
		PaymentOrderTimeoutMin:                                 updatedPaymentCfg.OrderTimeoutMin,
		PaymentMaxPendingOrders:                                updatedPaymentCfg.MaxPendingOrders,
		PaymentEnabledTypes:                                    updatedPaymentCfg.EnabledTypes,
		PaymentBalanceDisabled:                                 updatedPaymentCfg.BalanceDisabled,
		PaymentBalanceRechargeMultiplier:                       updatedPaymentCfg.BalanceRechargeMultiplier,
		PaymentSubscriptionUSDToCNYRate:                        updatedPaymentCfg.SubscriptionUSDToCNYRate,
		PaymentRechargeFeeRate:                                 updatedPaymentCfg.RechargeFeeRate,
		PaymentRechargeBonusTiers:                              rechargeBonusTiersToDTO(updatedPaymentCfg.RechargeBonusTiers),
		PaymentRechargeBonusMode:                               rechargeBonusModeToDTO(updatedPaymentCfg.RechargeBonusMode),
		PaymentRechargeBonusNotice:                             updatedPaymentCfg.RechargeBonusNotice,
		PaymentLoadBalanceStrat:                                updatedPaymentCfg.LoadBalanceStrategy,
		PaymentProductNamePrefix:                               updatedPaymentCfg.ProductNamePrefix,
		PaymentProductNameSuffix:                               updatedPaymentCfg.ProductNameSuffix,
		PaymentHelpImageURL:                                    updatedPaymentCfg.HelpImageURL,
		PaymentHelpText:                                        updatedPaymentCfg.HelpText,
		PaymentCancelRateLimitEnabled:                          updatedPaymentCfg.CancelRateLimitEnabled,
		PaymentCancelRateLimitMax:                              updatedPaymentCfg.CancelRateLimitMax,
		PaymentCancelRateLimitWindow:                           updatedPaymentCfg.CancelRateLimitWindow,
		PaymentCancelRateLimitUnit:                             updatedPaymentCfg.CancelRateLimitUnit,
		PaymentCancelRateLimitMode:                             updatedPaymentCfg.CancelRateLimitMode,
		PaymentAlipayForceQRCode:                               updatedPaymentCfg.AlipayForceQRCode,
		PaymentAlipayMobilePrecreateDeepLink:                   updatedPaymentCfg.AlipayMobilePrecreateDeepLink,

		ChannelMonitorEnabled:                updatedSettings.ChannelMonitorEnabled,
		ChannelMonitorMode:                   updatedSettings.ChannelMonitorMode,
		ChannelMonitorDefaultIntervalSeconds: updatedSettings.ChannelMonitorDefaultIntervalSeconds,
		ChannelMonitorHideThroughput:         updatedSettings.ChannelMonitorHideThroughput,
		ChannelMonitorShowQuota:              updatedSettings.ChannelMonitorShowQuota,
		ChannelMonitorHideUserRanking:        updatedSettings.ChannelMonitorHideUserRanking,

		GrokDefaultTextModel:           updatedSettings.GrokDefaultTextModel,
		GrokCrossClientModelMapEnabled: updatedSettings.GrokCrossClientModelMapEnabled,
		GrokDefaultBaseURLMode:         updatedSettings.GrokDefaultBaseURLMode,

		AvailableChannelsEnabled: updatedSettings.AvailableChannelsEnabled,
		SubscriptionEnabled:      updatedSettings.SubscriptionEnabled,

		ModelPlazaEnabled:       updatedSettings.ModelPlazaEnabled,
		ModelPlazaRequireAuth:   updatedSettings.ModelPlazaRequireAuth,
		ModelPlazaDescription:   updatedSettings.ModelPlazaDescription,
		PluginManagementEnabled: updatedSettings.PluginManagementEnabled,

		AffiliateEnabled: updatedSettings.AffiliateEnabled,

		RiskControlEnabled:          updatedSettings.RiskControlEnabled,
		CyberSessionBlockEnabled:    updatedSettings.CyberSessionBlockEnabled,
		CyberPolicyUserAllowlist:    updatedSettings.CyberPolicyUserAllowlist,
		CyberSessionBlockTTLSeconds: updatedSettings.CyberSessionBlockTTLSeconds,
		AccountSchedulingThresholds: updatedSettings.AccountSchedulingThresholds,
		AllowUserViewErrorRequests:  updatedSettings.AllowUserViewErrorRequests,
	}
	if fastPolicy, err := h.settingService.GetOpenAIFastPolicySettings(c.Request.Context()); err != nil {
		slog.Error("openai_fast_policy_settings_get_failed", "error", err)
	} else if fastPolicy != nil {
		payload.OpenAIFastPolicySettings = openaiFastPolicySettingsToDTO(fastPolicy)
	}

	// Default platform quotas（JSON map）—— 与 GetSettings 一致，避免保存后响应缺失该字段
	if platformQuotas, err := h.settingService.GetDefaultPlatformQuotas(c.Request.Context()); err != nil {
		slog.Error("default_platform_quotas_get_failed", "error", err)
	} else {
		payload.DefaultPlatformQuotas = platformQuotas
	}
	response.Success(c, payload)
}

// hasPaymentFields returns true if any payment-related field was explicitly provided.

func hasPaymentFields(req UpdateSettingsRequest) bool {
	return req.PaymentEnabled != nil || req.PaymentMinAmount != nil ||
		req.PaymentMaxAmount != nil || req.PaymentDailyLimit != nil ||
		req.PaymentOrderTimeoutMin != nil || req.PaymentMaxPendingOrders != nil ||
		req.PaymentEnabledTypes != nil || req.PaymentBalanceDisabled != nil ||
		req.PaymentBalanceRechargeMultiplier != nil || req.PaymentSubscriptionUSDToCNYRate != nil ||
		req.PaymentRechargeFeeRate != nil ||
		req.PaymentRechargeBonusTiers != nil || req.PaymentRechargeBonusMode != nil || req.PaymentRechargeBonusNotice != nil ||
		req.PaymentLoadBalanceStrat != nil || req.PaymentProductNamePrefix != nil ||
		req.PaymentProductNameSuffix != nil || req.PaymentHelpImageURL != nil ||
		req.PaymentHelpText != nil || req.PaymentCancelRateLimitEnabled != nil ||
		req.PaymentCancelRateLimitMax != nil || req.PaymentCancelRateLimitWindow != nil ||
		req.PaymentCancelRateLimitUnit != nil || req.PaymentCancelRateLimitMode != nil ||
		req.PaymentAlipayForceQRCode != nil || req.PaymentAlipayMobilePrecreateDeepLink != nil
}
