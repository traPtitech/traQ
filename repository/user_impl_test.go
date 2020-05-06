package repository

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
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
				q.IsBot = optional.BoolFrom(false)
			} else if v.bot == 1 {
				q.IsBot = optional.BoolFrom(true)
			}
			if v.active == 0 {
				q.IsActive = optional.BoolFrom(false)
			} else if v.active == 1 {
				q.IsActive = optional.BoolFrom(true)
			}
			if v.g1 == 1 {
				q.IsGMemberOf = optional.UUIDFrom(g1.ID)
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

func TestRepositoryImpl_GetUser(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common2)

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
	repo, assert, _, user := setupWithUser(t, common2)

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
	repo, _, _, user := setupWithUser(t, common2)

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

			err := repo.UpdateUser(user.GetID(), UpdateUserArgs{DisplayName: optional.StringFrom(strings.Repeat("a", 65))})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.DisplayName", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newDN := random2.AlphaNumeric(30)

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{DisplayName: optional.StringFrom(newDN)})) {
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

			err := repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: optional.StringFrom("ああああ")})
			if assert.IsType(&ArgumentError{}, err) {
				assert.Equal("args.TwitterID", err.(*ArgumentError).FieldName)
			}
		})

		t.Run("Success1", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := "aiueo"

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: optional.StringFrom(newTwitter)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newTwitter, u.GetTwitterID())
			}
		})

		t.Run("Success2", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := ""

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{TwitterID: optional.StringFrom(newTwitter)})) {
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

			if assert.NoError(repo.UpdateUser(user.GetID(), UpdateUserArgs{Role: optional.StringFrom("admin")})) {
				u, err := repo.GetUser(user.GetID(), false)
				require.NoError(err)
				assert.Equal("admin", u.GetRole())
			}
		})
	})
}
