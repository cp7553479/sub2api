package service

import (
 "context"
 "github.com/google/uuid"
)

// Explicit platform state fixture for tests of unrelated key business behavior.
type platformTestRepository struct{}
func (platformTestRepository) GetUser(context.Context,uuid.UUID)(*SupabaseIdentity,error){ panic("unexpected platform user lookup") }
func (platformTestRepository) SessionActive(context.Context,uuid.UUID,uuid.UUID)(bool,error){ panic("unexpected browser session lookup") }
func (platformTestRepository) RequireBusinessUser(context.Context,int64)(*SupabaseIdentity,error){ return &SupabaseIdentity{UserID:uuid.MustParse("68c2fd76-2524-45c3-b7d3-7e24a9144357"),Email:"user@example.test",Status:StatusActive},nil }
func (platformTestRepository) FirstAdminEmail(context.Context)(string,error){ return "admin@example.test",nil }
