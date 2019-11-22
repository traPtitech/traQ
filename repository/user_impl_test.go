package repository

import (
	"encoding/hex"
	"fmt"
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
	repo, assert, require := setup(t, ex2)

	u0, err := repo.GetUserByName("traq")
	require.NoError(err)
	g1 := mustMakeUserGroup(t, repo, random, u0.ID)
	c1 := mustMakeChannel(t, repo, random)
	us := make([]uuid.UUID, 0)

	type u struct {
		Bot    bool
		Active bool
		c1     bool
		g1     bool
	}
	ut := []u{
		{false, false, true, false},
		{true, false, true, false},
		{false, true, true, true},
		{true, true, true, true},
		{false, false, false, true},
		{true, false, false, true},
		{false, true, false, false},
		{true, true, false, false},
	}
	for _, v := range ut {
		u := mustMakeUser(t, repo, random)
		us = append(us, u.ID)

		if v.Bot {
			getDB(repo).Model(&model.User{ID: u.ID}).Update("bot", true)
		}
		if !v.Active {
			getDB(repo).Model(&model.User{ID: u.ID}).Update("status", model.UserAccountStatusDeactivated)
		}
		if v.c1 {
			mustChangeChannelSubscription(t, repo, c1.ID, u.ID, true)
		}
		if v.g1 {
			mustAddUserToGroup(t, repo, u.ID, g1.ID)
		}
	}

	ut = append(ut, u{false, true, false, false}) // traQユーザー
	us = append(us, u0.ID)

	tt := []struct {
		bot    int
		active int
		c1     int
		g1     int
	}{
		{-1, -1, -1, -1},
		{0, -1, -1, -1},
		{1, -1, -1, -1},
		{-1, 0, -1, -1},
		{-1, 1, -1, -1},
		{0, 1, 1, -1},
		{0, 1, -1, 1},
		{0, 1, -1, -1},
		{0, 1, 1, 1},
		{0, 1, -1, 1},
		{0, 1, 1, -1},
		{0, 1, 1, 1},
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
				if (v.c1 == 0 && u.c1) || (v.c1 == 1 && !u.c1) {
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
			if v.c1 == 1 {
				q.IsSubscriberOf = uuid.NullUUID{
					UUID:  c1.ID,
					Valid: true,
				}
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

		users, err := repo.GetUsers()
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
		assert.NotEmpty(user.ID)
		assert.Equal(s, user.Name)
		assert.NotEmpty(user.Salt)
		assert.NotEmpty(user.Password)
		assert.Equal(role.User, user.Role)
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
