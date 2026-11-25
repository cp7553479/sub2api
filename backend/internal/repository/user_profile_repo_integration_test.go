//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

type UserProfileRepoSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   *userRepository
}

func TestUserProfileRepoSuite(t *testing.T) {
	suite.Run(t, new(UserProfileRepoSuite))
}

func (s *UserProfileRepoSuite) SetupTest() {
	s.ctx = context.Background()
	s.client = testEntClient(s.T())
	s.repo = newUserRepositoryWithSQL(s.client, integrationDB, testPlatformSchema)

	_, err := integrationDB.ExecContext(s.ctx, `
TRUNCATE TABLE
	identity_adoption_decisions,
	auth_identity_channels,
	auth_identities,
	pending_auth_sessions,
	user_provider_default_grants,
	user_avatars
RESTART IDENTITY`)
	s.Require().NoError(err)
}

func (s *UserProfileRepoSuite) mustCreateUser(label string) *dbent.User {
	s.T().Helper()

	user, err := s.client.User.Create().
		SetEmail(fmt.Sprintf("%s-%d@example.com", label, time.Now().UnixNano())).
		SetPasswordHash("test-password-hash").
		SetRole("user").
		SetStatus("active").
		Save(s.ctx)
	s.Require().NoError(err)
	return user
}

func (s *UserProfileRepoSuite) TestWithUserProfileTx_AllowsAvatarOnlyProfileUpdate() {
	user := s.mustCreateUser("avatar-only-update")

	model, err := s.repo.GetByID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().NotNil(model)

	err = s.repo.WithUserProfileTx(s.ctx, func(txCtx context.Context) error {
		_, err := s.repo.UpsertUserAvatar(txCtx, user.ID, service.UpsertUserAvatarInput{
			StorageProvider: "remote_url",
			URL:             "https://cdn.example.com/avatar.png",
		})
		if err != nil {
			return err
		}
		// 只改头像时用户行没有任何列需要写，掩码为空——与 UserService.updateProfile 一致。
		return s.repo.Update(txCtx, model, service.UserUpdateFields{})
	})
	s.Require().NoError(err)

	avatar, err := s.repo.GetUserAvatar(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().NotNil(avatar)
	s.Require().Equal("https://cdn.example.com/avatar.png", avatar.URL)
}

func (s *UserProfileRepoSuite) TestUserAvatarCRUDAndUserLookup() {
	user := s.mustCreateUser("avatar")

	inlineAvatar, err := s.repo.UpsertUserAvatar(s.ctx, user.ID, service.UpsertUserAvatarInput{
		StorageProvider: "inline",
		URL:             "data:image/png;base64,QUJD",
		ContentType:     "image/png",
		ByteSize:        3,
		SHA256:          "902fbdd2b1df0c4f70b4a5d23525e932",
	})
	s.Require().NoError(err)
	s.Require().Equal("inline", inlineAvatar.StorageProvider)
	s.Require().Equal("data:image/png;base64,QUJD", inlineAvatar.URL)

	loadedAvatar, err := s.repo.GetUserAvatar(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().NotNil(loadedAvatar)
	s.Require().Equal("image/png", loadedAvatar.ContentType)
	s.Require().Equal(3, loadedAvatar.ByteSize)

	_, err = s.repo.UpsertUserAvatar(s.ctx, user.ID, service.UpsertUserAvatarInput{
		StorageProvider: "remote_url",
		URL:             "https://cdn.example.com/avatar.png",
	})
	s.Require().NoError(err)

	loadedAvatar, err = s.repo.GetUserAvatar(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().NotNil(loadedAvatar)
	s.Require().Equal("remote_url", loadedAvatar.StorageProvider)
	s.Require().Equal("https://cdn.example.com/avatar.png", loadedAvatar.URL)
	s.Require().Zero(loadedAvatar.ByteSize)

	s.Require().NoError(s.repo.DeleteUserAvatar(s.ctx, user.ID))
	loadedAvatar, err = s.repo.GetUserAvatar(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Nil(loadedAvatar)
}

type sqlNullTime struct {
	Time  time.Time
	Valid bool
}

func (t *sqlNullTime) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		t.Valid = true
		return nil
	case nil:
		t.Time = time.Time{}
		t.Valid = false
		return nil
	default:
		return fmt.Errorf("unsupported scan type %T", value)
	}
}

func stringPtr(v string) *string {
	return &v
}
