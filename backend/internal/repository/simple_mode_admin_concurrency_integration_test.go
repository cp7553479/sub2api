//go:build integration

package repository

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/setting"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSimpleModeConcurrencyUsesCurrentPlatformRole(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	_, err := client.Setting.Delete().Where(setting.KeyEQ(simpleModeAdminConcurrencyUpgradeKey)).Exec(ctx)
	require.NoError(t, err)
	admin := mustCreateUser(t, client, &service.User{Email: "simple-current-admin@example.test", Role: service.RoleAdmin})
	legacy := mustCreateUser(t, client, &service.User{Email: "simple-legacy-admin@example.test"})
	disabled := mustCreateUser(t, client, &service.User{Email: "simple-disabled-admin@example.test", Role: service.RoleAdmin})
	require.NoError(t, client.User.UpdateOneID(legacy.ID).SetRole(service.RoleAdmin).Exec(ctx))
	_, err = client.Driver().(sqlExecutor).ExecContext(ctx, `UPDATE sub2api_test_platform.platform_users SET status='disabled' WHERE supabase_user_id=$1`, disabled.SupabaseUserID)
	require.NoError(t, err)
	require.NoError(t, ensureSimpleModeAdminConcurrency(ctx, client, integrationUserConfig()))
	for _, item := range []struct {
		id   int64
		want int
	}{{admin.ID, simpleModeTargetAdminConcurrency}, {legacy.ID, simpleModeLegacyAdminConcurrency}, {disabled.ID, simpleModeLegacyAdminConcurrency}} {
		user, err := client.User.Get(ctx, item.id)
		require.NoError(t, err)
		require.Equal(t, item.want, user.Concurrency)
	}
}
