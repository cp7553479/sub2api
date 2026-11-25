package routes

import (
 "context"
 "github.com/google/uuid"
 "github.com/Wei-Shaw/sub2api/internal/service"
)

// Explicit platform state fixture for tests of unrelated key business behavior.
type platformTestRepository struct{}
func (platformTestRepository) GetUser(context.Context,uuid.UUID)(*service.SupabaseIdentity,error){ panic("unexpected platform user lookup") }
func (platformTestRepository) SessionActive(context.Context,uuid.UUID,uuid.UUID)(bool,error){ panic("unexpected browser session lookup") }
func (platformTestRepository) RequireBusinessUser(context.Context,int64)(*service.SupabaseIdentity,error){ return &service.SupabaseIdentity{UserID:uuid.MustParse("68c2fd76-2524-45c3-b7d3-7e24a9144357"),Email:"user@example.test",Status:service.StatusActive},nil }
func (platformTestRepository) FirstAdminEmail(context.Context)(string,error){ return "admin@example.test",nil }
