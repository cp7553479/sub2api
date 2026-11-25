//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestSupabasePrivateSchemaMigrationsAndBindings(t *testing.T) {
	ctx := context.Background()
	name := "binding_" + uuid.New().String()[:8]
	quoted := pq.QuoteIdentifier(name)
	_, err := integrationDB.ExecContext(ctx, "CREATE ROLE "+quoted+" LOGIN PASSWORD 'isolated-test' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE")
		_, _ = integrationDB.ExecContext(ctx, "REVOKE USAGE ON SCHEMA extensions FROM "+quoted)
		_, _ = integrationDB.ExecContext(ctx, "DROP ROLE "+quoted)
	})
	_, err = integrationDB.ExecContext(ctx, "CREATE SCHEMA "+quoted+" AUTHORIZATION "+quoted)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, "GRANT USAGE ON SCHEMA extensions TO "+quoted)
	require.NoError(t, err)
	dsn, err := url.Parse(integrationDSN)
	require.NoError(t, err)
	dsn.User = url.UserPassword(name, "isolated-test")
	params := dsn.Query()
	params.Set("search_path", name+",extensions")
	dsn.RawQuery = params.Encode()
	db, err := sql.Open("postgres", dsn.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	require.NoError(t, ApplyMigrations(ctx, db))
	require.NoError(t, ApplyMigrations(ctx, db), "restart must use this schema's migration ledger")

	var currentSchema string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&currentSchema))
	require.Equal(t, name, currentSchema)
	var trigramIndexes int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname LIKE '%_trgm'").Scan(&trigramIndexes))
	require.Equal(t, 9, trigramIndexes)
	_, err = db.ExecContext(ctx, "CREATE TABLE public.sub2api_runtime_must_not_write (id bigint)")
	require.Error(t, err, "runtime must not create platform tables")
	_, err = db.ExecContext(ctx, "SELECT password_hash FROM public.users")
	require.Error(t, err, "runtime must not read other schema credentials")

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	_, err = db.ExecContext(ctx, `CREATE TABLE platform_users (supabase_user_id uuid PRIMARY KEY, email text NOT NULL, status text NOT NULL DEFAULT 'active', is_admin boolean NOT NULL DEFAULT false)`)
	require.NoError(t, err)
	repo := NewUserBindingRepository(client, db, &config.Config{Supabase: config.SupabaseConfig{BridgeSchema: name}})
	userID := uuid.New()
	email := name + "@example.test"
	const concurrent = 32
	bindConcurrently := func(id uuid.UUID, address string) []*service.User {
		t.Helper()
		_, err := db.ExecContext(ctx, `INSERT INTO platform_users (supabase_user_id,email) VALUES ($1,$2)`, id, address)
		require.NoError(t, err)
		users := make([]*service.User, concurrent)
		failures := make([]error, concurrent)
		var wg sync.WaitGroup
		for i := range concurrent {
			wg.Add(1)
			go func(i int) { defer wg.Done(); users[i], failures[i] = repo.GetOrCreateSupabaseUser(ctx, id) }(i)
		}
		wg.Wait()
		for i := range concurrent {
			require.NoError(t, failures[i])
			require.Equal(t, users[0].ID, users[i].ID)
			require.Equal(t, id.String(), users[i].SupabaseUserID)
		}
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE supabase_user_id=$1", id).Scan(&count))
		require.Equal(t, 1, count)
		return users
	}
	missingID := uuid.New()
	_, err = repo.GetOrCreateSupabaseUser(ctx, missingID)
	require.ErrorIs(t, err, service.ErrUserNotFound)
	var missingCount int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE supabase_user_id=$1", missingID).Scan(&missingCount))
	require.Zero(t, missingCount, "a missing platform identity cannot create a business mapping")
	disabledID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO platform_users (supabase_user_id,email,status) VALUES ($1,$2,'disabled')`, disabledID, "disabled-"+email)
	require.NoError(t, err)
	_, err = repo.GetOrCreateSupabaseUser(ctx, disabledID)
	require.ErrorIs(t, err, service.ErrUserNotFound)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE supabase_user_id=$1", disabledID).Scan(&missingCount))
	require.Zero(t, missingCount)
	users := bindConcurrently(userID, email)
	for range 7 {
		id := uuid.New()
		bindConcurrently(id, id.String()+"@example.test")
	}
	var hasNoPassword bool
	require.NoError(t, db.QueryRowContext(ctx, "SELECT password_hash IS NULL FROM users WHERE id = $1", users[0].ID).Scan(&hasNoPassword))
	require.True(t, hasNoPassword)
	_, err = db.ExecContext(ctx, `UPDATE platform_users SET email=$2 WHERE supabase_user_id=$1`, userID, "changed-"+email)
	require.NoError(t, err)
	renamed, err := repo.GetOrCreateSupabaseUser(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, users[0].ID, renamed.ID)
	require.Equal(t, "changed-"+email, renamed.Email)
	duplicateID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO platform_users (supabase_user_id,email) VALUES ($1,$2)`, duplicateID, renamed.Email)
	require.NoError(t, err)
	_, err = repo.GetOrCreateSupabaseUser(ctx, duplicateID)
	require.Error(t, err, "matching an active email cannot bind another UUID")

	_, err = db.ExecContext(ctx, "UPDATE users SET deleted_at = NOW(), status = 'disabled' WHERE id = $1", renamed.ID)
	require.NoError(t, err)
	_, err = repo.GetOrCreateSupabaseUser(ctx, userID)
	require.ErrorIs(t, err, service.ErrUserNotFound, "deleted UUID must not be resurrected")
	newID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO platform_users (supabase_user_id,email) VALUES ($1,$2)`, newID, renamed.Email)
	require.NoError(t, err)
	replacement, err := repo.GetOrCreateSupabaseUser(ctx, newID)
	require.NoError(t, err)
	require.NotEqual(t, renamed.ID, replacement.ID, "a new platform UUID gets independent history")
	require.Equal(t, newID.String(), replacement.SupabaseUserID)
}
