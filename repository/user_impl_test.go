package repository

import (
	"encoding/hex"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"strings"
	"testing"
)

func TestRepositoryImpl_GetUsers(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, ex2)

	u0, err := repo.GetUserByName("traq", false)
	require.NoError(err)
	g1 := mustMakeUserGroup(t, repo, random, u0.GetID())
	us := make([]uuid.UUID, 0)

	type u struct {
		Bot    bool
		Active bool
		g1     bool
	}
	ut := []u{
		{false, false, false},
		{true, false, false},
		{false, true, true},
		{true, true, true},
		{false, false, true},
		{true, false, true},
		{false, true, false},
		{true, true, false},
	}
	for _, v := range ut {
		u := mustMakeUser(t, repo, random)
		us = append(us, u.GetID())

		if v.Bot {
			getDB(repo).Model(&model.User{ID: u.GetID()}).Update("bot", true)
		}
		if !v.Active {
			getDB(repo).Model(&model.User{ID: u.GetID()}).Update("status", model.UserAccountStatusDeactivated)
		}
		if v.g1 {
			mustAddUserToGroup(t, repo, u.GetID(), g1.ID)
		}
	}

	ut = append(ut, u{false, true, false}) // traQユーザー
	us = append(us, u0.GetID())

	tt := []struct {
		bot    int
		active int
		g1     int
	}{
		{-1, -1, -1},
		{0, -1, -1},
		{1, -1, -1},
		{-1, 0, -1},
		{-1, 1, -1},
		{0, 1, -1},
		{0, 1, 1},
	}
	for i, v := range tt {
		v := v
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			t.Parallel()

			ans := make([]uuid.UUID, 0)
			for k, u := range ut {
				if (v.bot == 0 && u.Bot) || (v.bot == 1 && !u.Bot) {
					continue
				}
				if (v.active == 0 && u.Active) || (v.active == 1 && !u.Active) {
					continue
				}
				if (v.g1 == 0 && u.g1) || (v.g1 == 1 && !u.g1) {
					continue
				}
				ans = append(ans, us[k])
			}

			q := UsersQuery{}
			if v.bot == 0 {
				q.IsBot = null.BoolFrom(false)
			} else if v.bot == 1 {
				q.IsBot = null.BoolFrom(true)
			}
			if v.active == 0 {
				q.IsActive = null.BoolFrom(false)
			} else if v.active == 1 {
				q.IsActive = null.BoolFrom(true)
			}
			if v.g1 == 1 {
				q.IsGMemberOf = uuid.NullUUID{
					UUID:  g1.ID,
					Valid: true,
				}
			}

			uids, err := repo.GetUserIDs(q)
			if assert.NoError(err) {
				assert.ElementsMatch(uids, ans)
			}
		})
	}

	t.Run("GetUsers", func(t *testing.T) {
		t.Parallel()

		users, err := repo.GetUsers(UsersQuery{})
		if assert.NoError(err) {
			assert.Len(users, len(us))
		}
	})
}

func TestRepositoryImpl_CreateUser(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	_, err := repo.CreateUser("あああ", "test", role.User)
	assert.Error(err)

	s := utils.RandAlphabetAndNumberString(10)
	user, err := repo.CreateUser(s, "test", role.User)
	if assert.NoError(err) {
		assert.NotEmpty(user.GetID())
		assert.Equal(s, user.GetName())
		assert.Equal(role.User, user.GetRole())
	}

	_, err = repo.CreateUser(s, "test", role.User)
	assert.Error(err)
}

func TestRepositoryImpl_GetUser(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	_, err := repo.GetUser(uuid.Nil, false)
	assert.Error(err)

	u, err := repo.GetUser(user.GetID(), false)
	if assert.NoError(err) {
		assert.Equal(user.GetID(), u.GetID())
		assert.Equal(user.GetName(), u.GetName())
	}
}

func TestRepositoryImpl_GetUserByName(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	_, err := repo.GetUserByName("", false)
	assert.Error(err)

	u, err := repo.GetUserByName(user.GetName(), false)
	if assert.NoError(err) {
		assert.Equal(user.GetID(), u.GetID())
		assert.Equal(user.GetName(), u.GetName())
	}
}

func TestRepositoryImpl_UpdateUser(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("No Args", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{}))
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

			err := repo.UpdateUser(user.GetID(), UpdateUserArgs{DisplayName: null.StringFrom(strings.Repeat("a", 65))})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.DisplayName", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newDN := utils.RandAlphabetAndNumberString(30)

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{DisplayName: null.StringFrom(newDN)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newDN, u.GetDisplayName())
			}
		})
	})

	t.Run("TwitterID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, random)

		t.Run("Failed", func(t *testing.T) {
			assert, _ := assertAndRequire(t)

			err := repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: null.StringFrom("ああああ")})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.TwitterID", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success1", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := "aiueo"

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: null.StringFrom(newTwitter)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newTwitter, u.GetTwitterID())
			}
		})

		t.Run("Success2", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := ""

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: null.StringFrom(newTwitter)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newTwitter, u.GetTwitterID())
			}
		})
	})

	t.Run("Role", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, random)

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{Role: null.StringFrom("admin")})) {
				u, err := repo.GetUser(user.GetID(), false)
				require.NoError(err)
				assert.Equal("admin", u.GetRole())
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
		if assert.NoError(repo.ChangeUserPassword(user.GetID(), newPass)) {
			u, err := repo.GetUser(user.GetID(), false)
			require.NoError(err)

			um := u.(*model.User)
			salt, err := hex.DecodeString(um.Salt)
			require.NoError(err)
			assert.Equal(um.Password, hex.EncodeToString(utils.HashPassword(newPass, salt)))
		}
	})

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.ChangeUserPassword(uuid.Nil, ""), ErrNilID.Error())
	})
}

func TestRepositoryImpl_ChangeUserIcon(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	newIcon := uuid.Must(uuid.NewV4())
	if assert.NoError(repo.ChangeUserIcon(user.GetID(), newIcon)) {
		u, err := repo.GetUser(user.GetID(), false)
		require.NoError(err)
		assert.Equal(newIcon, u.GetIconFileID())
	}
}
