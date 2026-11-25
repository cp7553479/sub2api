package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseSchemaIsolation(t *testing.T) {
	cfg := DatabaseConfig{Host: "localhost", Port: 5432, User: "runtime", DBName: "postgres", SSLMode: "disable", Schema: "business", ExtensionsSchema: "trusted_extensions"}
	require.NoError(t, cfg.ValidateSchema())
	for _, dsn := range []string{cfg.DSN(), cfg.DSNWithTimezone("UTC")} {
		require.Contains(t, dsn, `search_path='"business","trusted_extensions"'`)
		require.NotContains(t, dsn, "public")
	}
	for _, schema := range []string{"", "public", "auth", "pg_catalog", "business,public", "business'", strings.Repeat("a", 64)} {
		cfg.Schema = schema
		require.Error(t, cfg.ValidateSchema(), schema)
	}
}

func TestSupabaseConfigurationRejectsInvalidTrustInputs(t *testing.T) {
	valid := SupabaseConfig{Issuer: "https://identity.example/auth/v1", Audience: "authenticated", JWKSURL: "https://identity.example/auth/v1/.well-known/jwks.json", APIURL: "https://api.example", BridgeSchema: "identity_bridge", JWKSCacheTTLSeconds: 300, JWKSRefreshIntervalSeconds: 10, HTTPTimeoutSeconds: 5}
	require.NoError(t, valid.Validate())
	for name, mutate := range map[string]func(*SupabaseConfig){
		"missing issuer":           func(c *SupabaseConfig) { c.Issuer = "" },
		"untrusted scheme":         func(c *SupabaseConfig) { c.JWKSURL = "file:///tmp/keys.json" },
		"missing audience":         func(c *SupabaseConfig) { c.Audience = "" },
		"invalid bridge":           func(c *SupabaseConfig) { c.BridgeSchema = "public; DROP SCHEMA auth" },
		"unbounded cache":          func(c *SupabaseConfig) { c.JWKSCacheTTLSeconds = 3601 },
		"invalid refresh interval": func(c *SupabaseConfig) { c.JWKSRefreshIntervalSeconds = 301 },
		"unbounded HTTP timeout":   func(c *SupabaseConfig) { c.HTTPTimeoutSeconds = 31 },
	} {
		t.Run(name, func(t *testing.T) { cfg := valid; mutate(&cfg); require.Error(t, cfg.Validate()) })
	}
}
