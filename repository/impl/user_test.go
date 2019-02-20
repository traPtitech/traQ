package impl

import (
	"encoding/hex"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"strings"
	"testing"
)

func TestRepositoryImpl_GetUsers(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, ex2)

	for i := 0; i < 5; i++ {
		mustMakeUser(t, repo, random)
	}
	users, err := repo.GetUsers()
	if assert.NoError(err) {
		// traqユーザーがいるので
		assert.Len(users, 5+1)
	}
}

func TestRepositoryImpl_CreateUser(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	_, err := repo.CreateUser("あああ", "test@test.test", "test", role.User)
	assert.Error(err)
	_, err = repo.CreateUser("aaa", "アイウエオ", "test", role.User)
	assert.Error(err)

	s := utils.RandAlphabetAndNumberString(10)
	user, err := repo.CreateUser(s, "test@test.test", "test", role.User)
	if assert.NoError(err) {
		assert.NotEmpty(user.ID)
		assert.Equal(s, user.Name)
		assert.NotEmpty(user.Salt)
		assert.NotEmpty(user.Password)
		assert.Equal("test@test.test", user.Email)
		assert.Equal(role.User.ID(), user.Role)
	}

	_, err = repo.CreateUser(s, "test@test.test", "test", role.User)
	assert.Error(err)
}

func TestRepositoryImpl_GetUser(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	_, err := repo.GetUser(uuid.Nil)
	assert.Error(err)

	u, err := repo.GetUser(user.ID)
	if assert.NoError(err) {
		assert.Equal(user.ID, u.ID)
		assert.Equal(user.Name, u.Name)
	}
}

func TestRepositoryImpl_GetUserByName(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	_, err := repo.GetUserByName("")
	assert.Error(err)

	u, err := repo.GetUserByName(user.Name)
	if assert.NoError(err) {
		assert.Equal(user.ID, u.ID)
		assert.Equal(user.Name, u.Name)
	}
}

func TestRepositoryImpl_ChangeUserPassword(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newPass := "aiueo123"

	if assert.NoError(repo.ChangeUserPassword(user.ID, newPass)) {
		u, err := repo.GetUser(user.ID)
		require.NoError(err)

		salt, err := hex.DecodeString(u.Salt)
		require.NoError(err)
		assert.Equal(u.Password, hex.EncodeToString(utils.HashPassword(newPass, salt)))
	}
}

func TestRepositoryImpl_ChangeUserIcon(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newIcon := uuid.NewV4()
	if assert.NoError(repo.ChangeUserIcon(user.ID, newIcon)) {
		u, err := repo.GetUser(user.ID)
		require.NoError(err)
		assert.Equal(newIcon, u.Icon)
	}
}

func TestRepositoryImpl_ChangeUserDisplayName(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newDN := uuid.NewV4().String()

	if assert.NoError(repo.ChangeUserDisplayName(user.ID, newDN)) {
		u, err := repo.GetUser(user.ID)
		require.NoError(err)
		assert.Equal(newDN, u.DisplayName)
	}

	assert.Error(repo.ChangeUserDisplayName(user.ID, strings.Repeat("a", 100)))
}

func TestRepositoryImpl_ChangeUserTwitterID(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newTwitter := "aiueo"

	if assert.NoError(repo.ChangeUserTwitterID(user.ID, newTwitter)) {
		u, err := repo.GetUser(user.ID)
		require.NoError(err)
		assert.Equal(newTwitter, u.TwitterID)
	}

	assert.Error(repo.ChangeUserTwitterID(user.ID, "あああああ"))
}

func TestRepositoryImpl_ChangeUserAccountStatus(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserAccountStatus(uuid.Nil, model.UserAccountStatusSuspended), repository.ErrNilID.Error())
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserAccountStatus(uuid.NewV4(), model.UserAccountStatusSuspended), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		if assert.NoError(t, repo.ChangeUserAccountStatus(user.ID, model.UserAccountStatusSuspended)) {
			u, err := repo.GetUser(user.ID)
			require.NoError(t, err)
			assert.Equal(t, u.Status, model.UserAccountStatusSuspended)
		}
	})
}
