//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryPlatformProjectionAndPagination(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUserRepository(client, integrationDB, integrationUserConfig())
	prefix := "projection-" + uuid.NewString()
	seed := func(name string) *service.User {
		u := &service.User{Email: prefix + name + "@example.test", Username: prefix + name, Role: service.RoleUser, Status: service.StatusActive, Concurrency: 1}
		require.NoError(t, seedBusinessUser(ctx, client, integrationDB, u))
		return u
	}
	a, b, banned, missing := seed("a"), seed("b"), seed("ban"), seed("missing")
	_, err := integrationDB.ExecContext(ctx, `UPDATE sub2api_test_platform.platform_users SET is_admin=true,email=$2 WHERE supabase_user_id=$1`, a.SupabaseUserID, prefix+"current@example.test")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET role='admin',status='disabled' WHERE id=$1`, banned.ID)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `DELETE FROM sub2api_test_platform.platform_users WHERE supabase_user_id=$1`, missing.SupabaseUserID)
	require.NoError(t, err)
	get := func(u *service.User, role, status string) *service.User {
		t.Helper()
		current, err := repo.GetByID(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, role, current.Role)
		require.Equal(t, status, current.Status)
		return current
	}
	require.Equal(t, prefix+"current@example.test", get(a, service.RoleAdmin, service.StatusActive).Email)
	get(banned, service.RoleUser, service.StatusDisabled)
	require.Empty(t, get(missing, service.RoleUser, service.StatusDisabled).Email)
	currentEmailMatches, currentEmailTotal, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 10}, service.UserListFilters{Search: prefix + "current@"})
	require.NoError(t, err)
	require.EqualValues(t, 1, currentEmailTotal.Total)
	require.Equal(t, a.ID, currentEmailMatches[0].ID)
	_, oldEmailTotal, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 10}, service.UserListFilters{Search: a.Email})
	require.NoError(t, err)
	require.Zero(t, oldEmailTotal.Total, "search must not use the stale native email cache")
	for _, field := range []string{"role", "status"} {
		ordered, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: field, SortOrder: "asc"}, service.UserListFilters{Search: prefix})
		require.NoError(t, err)
		require.Len(t, ordered, 4)
		if field == "role" {
			require.Equal(t, a.ID, ordered[0].ID, "sorting must use the current platform admin flag")
		} else {
			require.Equal(t, service.StatusActive, ordered[0].Status)
			require.Equal(t, service.StatusActive, ordered[1].Status)
			require.Equal(t, service.StatusDisabled, ordered[2].Status)
			require.Equal(t, service.StatusDisabled, ordered[3].Status)
		}
	}
	list := func(filters service.UserListFilters, page, size int) ([]service.User, int64) {
		t.Helper()
		filters.Search = prefix
		users, result, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: page, PageSize: size, SortBy: "email", SortOrder: "asc"}, filters)
		require.NoError(t, err)
		return users, result.Total
	}
	users, total := list(service.UserListFilters{Role: service.RoleAdmin, Status: service.StatusActive}, 1, 1)
	require.EqualValues(t, 1, total)
	require.Len(t, users, 1)
	require.Equal(t, a.ID, users[0].ID)
	users, total = list(service.UserListFilters{Status: service.StatusDisabled}, 2, 1)
	require.EqualValues(t, 2, total)
	require.Len(t, users, 1, "current status filters must run before pagination")
	require.Equal(t, banned.ID, users[0].ID, "sorting must use the current email, including a missing identity's empty email")
	users, total = list(service.UserListFilters{Status: service.StatusActive}, 2, 1)
	require.EqualValues(t, 2, total)
	require.Len(t, users, 1)
	_, err = integrationDB.ExecContext(ctx, `UPDATE sub2api_test_platform.platform_users SET is_admin=false,status='disabled' WHERE supabase_user_id=$1`, a.SupabaseUserID)
	require.NoError(t, err)
	get(a, service.RoleUser, service.StatusDisabled)
	_, total = list(service.UserListFilters{Role: service.RoleAdmin}, 1, 1)
	require.Zero(t, total)
	_, total = list(service.UserListFilters{Status: service.StatusActive}, 1, 1)
	require.EqualValues(t, 1, total)
	_, err = integrationDB.ExecContext(ctx, `UPDATE sub2api_test_platform.platform_users SET is_admin=true WHERE supabase_user_id=$1`, b.SupabaseUserID)
	require.NoError(t, err)
	get(b, service.RoleAdmin, service.StatusActive)
	var storedRole string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT role FROM users WHERE id=$1", b.ID).Scan(&storedRole))
	require.Equal(t, service.RoleUser, storedRole, "current platform role must never be synchronized to a second owner")
}
