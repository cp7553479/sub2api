package service

import (
	"context"
)

// ClientRequestMetadata records request attributes for audit and rate limiting.
type ClientRequestMetadata struct {
	IP        string
	UserAgent string
}

type clientRequestMetadataCtxKey struct{}

// WithClientRequestMetadata 将请求元数据注入 context（由 HTTP 入口中间件调用）。
func WithClientRequestMetadata(ctx context.Context, binding *ClientRequestMetadata) context.Context {
	if binding == nil {
		return ctx
	}
	return context.WithValue(ctx, clientRequestMetadataCtxKey{}, binding)
}

// ClientRequestMetadataFromContext 从 context 提取请求元数据；不存在时返回 nil。
func ClientRequestMetadataFromContext(ctx context.Context) *ClientRequestMetadata {
	if ctx == nil {
		return nil
	}
	binding, _ := ctx.Value(clientRequestMetadataCtxKey{}).(*ClientRequestMetadata)
	return binding
}
