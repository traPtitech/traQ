package repository

import (
	"encoding/hex"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
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

	_, err := repo.CreateUser("あああ", "test", role.User)
	assert.Error(err)

	s := utils.RandAlphabetAndNumberString(10)
	user, err := repo.CreateUser(s, "test", role.User)
	if assert.NoError(err) {
		assert.NotEmpty(user.ID)
		assert.Equal(s, user.Name)
		assert.NotEmpty(user.Salt)
		assert.NotEmpty(user.Password)
		assert.Equal(role.User.ID(), user.Role)
	}

	_, err = repo.CreateUser(s, "test", role.User)
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

func TestRepositoryImpl_UpdateUser(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("No Args", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.NoError(repo.UpdateUser(user.ID, UpdateUserArgs{}))
	})

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		assert.EqualError(repo.UpdateUser(uuid.Nil, UpdateUserArgs{}), ErrNilID.Error())
	})

	t.Run("Unknown User", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		assert.EqualError(repo.UpdateUser(uuid.Must(uuid.NewV4()), UpdateUserArgs{}), ErrNotFound.Error())
	})

	t.Run("DisplayName", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, random)

		t.Run("Failed", func(t *testing.T) {
			assert, _ := assertAndRequire(t)

			err := repo.UpdateUser(user.ID, UpdateUserArgs{DisplayName: null.StringFrom(strings.Repeat("a", 65))})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.DisplayName", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newDN := utils.RandAlphabetAndNumberString(30)

			if assert.NoError(repo.UpdateUser(user.ID, UpdateUserArgs{DisplayName: null.StringFrom(newDN)})) {
				u, err := repo.GetUser(user.ID)
				require.NoError(err)
				assert.Equal(newDN, u.DisplayName)
			}
		})
	})

	t.Run("TwitterID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, random)

		t.Run("Failed", func(t *testing.T) {
			assert, _ := assertAndRequire(t)

			err := repo.UpdateUser(user.ID, UpdateUserArgs{TwitterID: null.StringFrom("ああああ")})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.TwitterID", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success1", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := "aiueo"

			if assert.NoError(repo.UpdateUser(user.ID, UpdateUserArgs{TwitterID: null.StringFrom(newTwitter)})) {
				u, err := repo.GetUser(user.ID)
				require.NoError(err)
				assert.Equal(newTwitter, u.TwitterID)
			}
		})

		t.Run("Success2", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := ""

			if assert.NoError(repo.UpdateUser(user.ID, UpdateUserArgs{TwitterID: null.StringFrom(newTwitter)})) {
				u, err := repo.GetUser(user.ID)
				require.NoError(err)
				assert.Equal(newTwitter, u.TwitterID)
			}
		})
	})

	t.Run("Role", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, random)

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)

			if assert.NoError(repo.UpdateUser(user.ID, UpdateUserArgs{Role: null.StringFrom("admin")})) {
				u, err := repo.GetUser(user.ID)
				require.NoError(err)
				assert.Equal("admin", u.Role)
			}
		})
	})
}

func TestRepositoryImpl_ChangeUserPassword(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		newPass := "aiueo123456"
		if assert.NoError(repo.ChangeUserPassword(user.ID, newPass)) {
			u, err := repo.GetUser(user.ID)
			require.NoError(err)

			salt, err := hex.DecodeString(u.Salt)
			require.NoError(err)
			assert.Equal(u.Password, hex.EncodeToString(utils.HashPassword(newPass, salt)))
		}
	})

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserPassword(uuid.Nil, ""), ErrNilID.Error())
	})

	t.Run("invalid password", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.ChangeUserPassword(user.ID, "a"))
	})
}

func TestRepositoryImpl_ChangeUserIcon(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newIcon := uuid.Must(uuid.NewV4())
	if assert.NoError(repo.ChangeUserIcon(user.ID, newIcon)) {
		u, err := repo.GetUser(user.ID)
		require.NoError(err)
		assert.Equal(newIcon, u.Icon)
	}
}

func TestRepositoryImpl_ChangeUserAccountStatus(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserAccountStatus(uuid.Nil, model.UserAccountStatusDeactivated), ErrNilID.Error())
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserAccountStatus(uuid.Must(uuid.NewV4()), model.UserAccountStatusDeactivated), ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		if assert.NoError(t, repo.ChangeUserAccountStatus(user.ID, model.UserAccountStatusDeactivated)) {
			u, err := repo.GetUser(user.ID)
			require.NoError(t, err)
			assert.Equal(t, u.Status, model.UserAccountStatusDeactivated)
		}
	})
}
