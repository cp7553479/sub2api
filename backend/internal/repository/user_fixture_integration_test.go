//go:build integration

package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const testPlatformSchema = "sub2api_test_platform"

func integrationUserConfig() *config.Config {
	return &config.Config{Supabase: config.SupabaseConfig{BridgeSchema: testPlatformSchema}}
}

// seedBusinessUser supplies both state owners explicitly for business fixtures.
func seedBusinessUser(ctx context.Context, client *dbent.Client, sqlq sqlExecutor, user *service.User) error {
	id := uuid.New()
	created, err := client.User.Create().SetSupabaseUserID(id).
		SetEmail(user.Email).SetUsername(user.Username).SetNotes(user.Notes).
		SetRole(service.RoleUser).SetStatus(user.Status).SetBalance(user.Balance).
		SetConcurrency(user.Concurrency).SetRpmLimit(user.RPMLimit).
		SetRestrictPublicGroups(user.RestrictPublicGroups).
		SetNillableLastActiveAt(user.LastActiveAt).Save(ctx)
	if err != nil {
		return err
	}
	user.ID, user.SupabaseUserID = created.ID, id.String()
	user.CreatedAt, user.UpdatedAt = created.CreatedAt, created.UpdatedAt
	_, err = sqlq.ExecContext(ctx, `INSERT INTO sub2api_test_platform.platform_users
 (supabase_user_id,email,status,is_admin) VALUES ($1,$2,$3,$4)`, id, user.Email, user.Status, user.IsAdmin())
	if err != nil {
		return err
	}
	return newUserRepositoryWithSQL(client, sqlq, testPlatformSchema).syncUserAllowedGroupsWithClient(ctx, client, user.ID, user.AllowedGroups)
}
