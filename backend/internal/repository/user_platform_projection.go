package repository

import (
	"context"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// platformExpression is shared by authorization projections, filters and sorting.
// Local status is only the business moderation gate; local role and email never
// determine the current platform identity.
func (r *userRepository) platformExpression(b *entsql.Builder, subject, businessStatus, field string) {
	switch field {
	case dbuser.FieldRole:
		b.WriteString("CASE WHEN EXISTS (SELECT 1 FROM ").WriteString(r.platformUsers).
			WriteString(" p WHERE p.supabase_user_id=").Ident(subject).
			WriteString(" AND p.is_admin) THEN ").Arg(service.RoleAdmin).
			WriteString(" ELSE ").Arg(service.RoleUser).WriteString(" END")
	case dbuser.FieldStatus:
		b.WriteString("CASE WHEN ").Ident(businessStatus).WriteString(" = ").Arg(service.StatusActive).
			WriteString(" AND EXISTS (SELECT 1 FROM ").WriteString(r.platformUsers).
			WriteString(" p WHERE p.supabase_user_id=").Ident(subject).
			WriteString(" AND p.status=").Arg(service.StatusActive).
			WriteString(") THEN ").Arg(service.StatusActive).
			WriteString(" ELSE ").Arg(service.StatusDisabled).WriteString(" END")
	case dbuser.FieldEmail:
		b.WriteString("COALESCE((SELECT p.email FROM ").WriteString(r.platformUsers).
			WriteString(" p WHERE p.supabase_user_id=").Ident(subject).WriteString("), '')")
	}
}

func (r *userRepository) platformFilter(field, value string) predicate.User {
	return func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			r.platformExpression(b, s.C(dbuser.FieldSupabaseUserID), s.C(dbuser.FieldStatus), field)
			b.WriteString(" = ").Arg(value)
		}))
	}
}

func (r *userRepository) platformEmailSearch(value string) predicate.User {
	pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(value)) + "%"
	return func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("LOWER(")
			r.platformExpression(b, s.C(dbuser.FieldSupabaseUserID), s.C(dbuser.FieldStatus), dbuser.FieldEmail)
			b.WriteString(") LIKE ").Arg(pattern)
		}))
	}
}

func (r *userRepository) projectPlatformUsers(ctx context.Context, users []*service.User) error {
	ids := make([]int64, 0, len(users))
	byID := make(map[int64]*service.User, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
		byID[user.ID] = user
		user.Role, user.Status, user.Email = service.RoleUser, service.StatusDisabled, ""
	}
	rows, err := txAwareSQLExecutor(ctx, r.sql, r.client).QueryContext(ctx, `
SELECT u.id, COALESCE(p.email, ''), COALESCE(p.is_admin, false),
       u.status = $2 AND COALESCE(p.status = $2, false)
FROM users u LEFT JOIN `+r.platformUsers+` p ON p.supabase_user_id=u.supabase_user_id
WHERE u.id = ANY($1)`, pq.Array(ids), service.StatusActive)
	if err != nil {
		return infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform authorization state").WithCause(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var email string
		var admin, active bool
		if err := rows.Scan(&id, &email, &admin, &active); err != nil {
			return infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform authorization state").WithCause(err)
		}
		user := byID[id]
		user.Email = email
		if admin {
			user.Role = service.RoleAdmin
		}
		if active {
			user.Status = service.StatusActive
		}
	}
	if err := rows.Err(); err != nil {
		return infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform authorization state").WithCause(err)
	}
	return nil
}
