package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type platformIdentityRepository struct {
	db            *sql.DB
	users         string
	sessions      string
	businessUsers string
}

func NewPlatformIdentityRepository(db *sql.DB, cfg *config.Config) (service.PlatformIdentityRepository, error) {
	if err := cfg.Supabase.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Supabase.HTTPTimeoutSeconds)*time.Second)
	defer cancel()
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS (
 SELECT 1 FROM pg_catalog.pg_trigger t
 JOIN pg_catalog.pg_proc p ON p.oid=t.tgfoid
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 JOIN pg_catalog.pg_roles owner ON owner.oid=p.proowner
 JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid
 JOIN pg_catalog.pg_namespace tn ON tn.oid=c.relnamespace
 WHERE tn.nspname='public' AND c.relname='app_users' AND t.tgname='sub2api_on_platform_user_deleted'
 AND t.tgenabled IN ('O','A') AND NOT t.tgisinternal AND t.tgtype=9
 AND n.nspname=$1 AND p.proname='disable_deleted_user' AND p.prosecdef
 AND owner.rolname='sub2api_cleanup'
 AND NOT (owner.rolcanlogin OR owner.rolsuper OR owner.rolinherit OR owner.rolcreatedb
          OR owner.rolcreaterole OR owner.rolbypassrls OR owner.rolreplication)
 AND NOT pg_catalog.pg_has_role(current_user,owner.oid,'MEMBER')
 AND (SELECT count(*)=1 AND bool_and(replace(setting,' ','')=$2)
      FROM unnest(p.proconfig) setting WHERE split_part(setting,'=',1)='search_path'))`,
		cfg.Supabase.BridgeSchema, "search_path=pg_catalog,"+cfg.Database.Schema+",pg_temp").Scan(&ready)
	if err != nil {
		return nil, fmt.Errorf("check platform identity cleanup: %w", err)
	}
	if !ready {
		return nil, fmt.Errorf("platform identity cleanup does not satisfy the required trigger, role and search_path contract")
	}
	prefix := pq.QuoteIdentifier(cfg.Supabase.BridgeSchema) + "."
	return &platformIdentityRepository{db: db, users: prefix + "platform_users", sessions: prefix + "platform_sessions", businessUsers: pq.QuoteIdentifier(cfg.Database.Schema) + ".users"}, nil
}

func (r *platformIdentityRepository) GetUser(ctx context.Context, userID uuid.UUID) (*service.SupabaseIdentity, error) {
	return r.readUser(ctx, `SELECT supabase_user_id,email,status,is_admin FROM `+r.users+` WHERE supabase_user_id=$1`, userID)
}

func (r *platformIdentityRepository) readUser(ctx context.Context, query string, id any) (*service.SupabaseIdentity, error) {
	identity := &service.SupabaseIdentity{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&identity.UserID, &identity.Email, &identity.Status, &identity.IsAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPlatformIdentity
	}
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform authorization state").WithCause(err)
	}
	return identity, nil
}

func (r *platformIdentityRepository) SessionActive(ctx context.Context, sessionID, userID uuid.UUID) (bool, error) {
	var active bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM `+r.sessions+` WHERE id=$1 AND user_id=$2 AND (not_after IS NULL OR not_after>now()))`, sessionID, userID).Scan(&active)
	if err != nil {
		return false, infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform session state").WithCause(err)
	}
	return active, nil
}

func (r *platformIdentityRepository) RequireBusinessUser(ctx context.Context, userID int64) (*service.SupabaseIdentity, error) {
	identity, err := r.readUser(ctx, `SELECT p.supabase_user_id,p.email,p.status,p.is_admin FROM `+r.users+` p JOIN `+r.businessUsers+` u ON u.supabase_user_id=p.supabase_user_id WHERE u.id=$1 AND u.deleted_at IS NULL AND u.status='active'`, userID)
	if err != nil {
		return nil, err
	}
	if identity.Status != service.StatusActive {
		return nil, service.ErrPlatformDisabled
	}
	return identity, nil
}

func (r *platformIdentityRepository) FirstAdminEmail(ctx context.Context) (string, error) {
	var email string
	err := r.db.QueryRowContext(ctx, `SELECT email FROM `+r.users+` WHERE status='active' AND is_admin ORDER BY supabase_user_id LIMIT 1`).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", service.ErrPlatformIdentity
	}
	if err != nil {
		return "", infraerrors.ServiceUnavailable("PLATFORM_UNAVAILABLE", "Cannot read platform administrator").WithCause(err)
	}
	return email, nil
}
