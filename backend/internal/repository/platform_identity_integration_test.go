//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPlatformCleanupStartupRequiresRestrictedDefiner(t *testing.T) {
	ctx := context.Background()
	_, err := integrationDB.ExecContext(ctx, `
CREATE ROLE sub2api_cleanup NOLOGIN NOSUPERUSER NOINHERIT NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE cleanup_runtime_test LOGIN PASSWORD 'isolated-test' NOSUPERUSER NOINHERIT NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE TABLE public.app_users(id uuid);
REVOKE USAGE ON SCHEMA public FROM PUBLIC;
CREATE FUNCTION sub2api_test_platform.disable_deleted_user() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp
AS $$ BEGIN RETURN OLD; END $$;
ALTER FUNCTION sub2api_test_platform.disable_deleted_user() OWNER TO sub2api_cleanup;
CREATE TRIGGER sub2api_on_platform_user_deleted AFTER DELETE ON public.app_users
FOR EACH ROW EXECUTE FUNCTION sub2api_test_platform.disable_deleted_user();`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `GRANT USAGE ON SCHEMA public TO PUBLIC;
 DROP TABLE public.app_users;
DROP FUNCTION sub2api_test_platform.disable_deleted_user();
REVOKE sub2api_cleanup FROM cleanup_runtime_test;
DROP ROLE cleanup_runtime_test; DROP ROLE sub2api_cleanup;`)
	})
	dsn, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	dsn.User = url.UserPassword("cleanup_runtime_test", "isolated-test")
	runtimeDB, err := sql.Open("postgres", dsn.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtimeDB.Close() })
	cfg := &config.Config{Database: config.DatabaseConfig{Schema: "public"}, Supabase: config.SupabaseConfig{
		Issuer: "https://auth.example.test/auth/v1", Audience: "authenticated",
		APIURL: "https://api.example.test", JWKSURL: "https://auth.example.test/auth/v1/.well-known/jwks.json",
		BridgeSchema: testPlatformSchema, JWKSCacheTTLSeconds: 300, JWKSRefreshIntervalSeconds: 10, HTTPTimeoutSeconds: 5,
	}}
	_, err = NewPlatformIdentityRepository(runtimeDB, cfg)
	require.NoError(t, err, "a low-privilege definer must satisfy normal startup")
	_, err = NewPlatformIdentityRepository(integrationDB, cfg)
	require.Error(t, err, "a superuser runtime can assume the cleanup role and must be rejected")
	for _, scenario := range []struct{ name, weaken, restore string }{
		{"invoker", "ALTER FUNCTION sub2api_test_platform.disable_deleted_user() SECURITY INVOKER", "ALTER FUNCTION sub2api_test_platform.disable_deleted_user() SECURITY DEFINER"},
		{"login role", "ALTER ROLE sub2api_cleanup LOGIN", "ALTER ROLE sub2api_cleanup NOLOGIN"},
		{"inheriting role", "ALTER ROLE sub2api_cleanup INHERIT", "ALTER ROLE sub2api_cleanup NOINHERIT"},
		{"unsafe search path", "ALTER FUNCTION sub2api_test_platform.disable_deleted_user() SET search_path=public,pg_catalog", "ALTER FUNCTION sub2api_test_platform.disable_deleted_user() SET search_path=pg_catalog,public,pg_temp"},
		{"runtime membership", "GRANT sub2api_cleanup TO cleanup_runtime_test", "REVOKE sub2api_cleanup FROM cleanup_runtime_test"},
		{"disabled trigger", "ALTER TABLE public.app_users DISABLE TRIGGER sub2api_on_platform_user_deleted", "ALTER TABLE public.app_users ENABLE TRIGGER sub2api_on_platform_user_deleted"},
		{"wrong trigger event", "DROP TRIGGER sub2api_on_platform_user_deleted ON public.app_users; CREATE TRIGGER sub2api_on_platform_user_deleted AFTER INSERT ON public.app_users FOR EACH ROW EXECUTE FUNCTION sub2api_test_platform.disable_deleted_user()", "DROP TRIGGER sub2api_on_platform_user_deleted ON public.app_users; CREATE TRIGGER sub2api_on_platform_user_deleted AFTER DELETE ON public.app_users FOR EACH ROW EXECUTE FUNCTION sub2api_test_platform.disable_deleted_user()"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := integrationDB.ExecContext(ctx, scenario.weaken)
			require.NoError(t, err)
			_, err = NewPlatformIdentityRepository(runtimeDB, cfg)
			require.Error(t, err)
			_, err = integrationDB.ExecContext(ctx, scenario.restore)
			require.NoError(t, err)
			_, err = NewPlatformIdentityRepository(runtimeDB, cfg)
			require.NoError(t, err)
		})
	}
}
