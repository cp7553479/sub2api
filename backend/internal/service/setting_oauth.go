package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// GetWeChatConnectOAuthConfig supplies WeChat MP credentials for payment authorization.
func (s *SettingService) GetWeChatConnectOAuthConfig(ctx context.Context) (WeChatConnectOAuthConfig, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyWeChatConnectMPAppID, SettingKeyWeChatConnectMPAppSecret, SettingKeyWeChatConnectMPEnabled})
	if err != nil {
		return WeChatConnectOAuthConfig{}, fmt.Errorf("get WeChat payment credentials: %w", err)
	}
	cfg := s.effectiveWeChatPaymentConfig(values)
	if !cfg.MPEnabled {
		return WeChatConnectOAuthConfig{}, infraerrors.NotFound("PAYMENT_OAUTH_DISABLED", "WeChat payment authorization is disabled")
	}
	if cfg.MPAppID == "" || cfg.MPAppSecret == "" {
		return WeChatConnectOAuthConfig{}, infraerrors.InternalServer("WECHAT_PAYMENT_MP_NOT_CONFIGURED", "WeChat payment credentials are required")
	}
	return cfg, nil
}

// Configuration supplies the MP base values; explicit stored settings override them.
func (s *SettingService) effectiveWeChatPaymentConfig(values map[string]string) WeChatConnectOAuthConfig {
	cfg := WeChatConnectOAuthConfig{}
	if s.cfg != nil {
		cfg.MPAppID = s.cfg.WeChat.MPAppID
		cfg.MPAppSecret = s.cfg.WeChat.MPAppSecret
		cfg.MPEnabled = s.cfg.WeChat.MPEnabled
	}
	if v, ok := values[SettingKeyWeChatConnectMPAppID]; ok {
		cfg.MPAppID = v
	}
	if v, ok := values[SettingKeyWeChatConnectMPAppSecret]; ok {
		cfg.MPAppSecret = v
	}
	if v, ok := values[SettingKeyWeChatConnectMPEnabled]; ok {
		cfg.MPEnabled = v == "true"
	}
	cfg.MPAppID = strings.TrimSpace(cfg.MPAppID)
	cfg.MPAppSecret = strings.TrimSpace(cfg.MPAppSecret)
	return cfg
}
