package gorm

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestRepositoryImpl_GetUsers(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, ex2)
	mustMakeUser(t, repo, "traq")

	u0, err := repo.GetUserByName("traq", false)
	require.NoError(err)
	g1 := mustMakeUserGroup(t, repo, rand, u0.GetID())
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
		u := mustMakeUser(t, repo, rand)
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

			q := repository.UsersQuery{}
			if v.bot == 0 {
				q.IsBot = optional.From(false)
			} else if v.bot == 1 {
				q.IsBot = optional.From(true)
			}
			if v.active == 0 {
				q.IsActive = optional.From(false)
			} else if v.active == 1 {
				q.IsActive = optional.From(true)
			}
			if v.g1 == 1 {
				q.IsGMemberOf = optional.From(g1.ID)
			}

			uids, err := repo.GetUserIDs(q)
			if assert.NoError(err) {
				assert.ElementsMatch(uids, ans)
			}
		})
	}

	t.Run("GetUsers", func(t *testing.T) {
		t.Parallel()

		users, err := repo.GetUsers(repository.UsersQuery{})
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
		assert := assert.New(t)

		assert.NoError(repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{}))
	})

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		assert.EqualError(repo.UpdateUser(uuid.Nil, repository.UpdateUserArgs{}), repository.ErrNilID.Error())
	})

	t.Run("Unknown User", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		assert.EqualError(repo.UpdateUser(uuid.Must(uuid.NewV7()), repository.UpdateUserArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("DisplayName", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, rand)

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newDN := random2.AlphaNumeric(32)

			if assert.NoError(repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{DisplayName: optional.From(newDN)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newDN, u.GetDisplayName())
			}
		})
	})

	t.Run("TwitterID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, rand)

		t.Run("Failed", func(t *testing.T) {
			assert := assert.New(t)

			err := repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{TwitterID: optional.From("ああああ")})
			if assert.IsType(&repository.ArgumentError{}, err) {
				assert.Equal("args.TwitterID", err.(*repository.ArgumentError).FieldName)
			}
		})

		t.Run("Success1", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := "aiueo"

			if assert.NoError(repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{TwitterID: optional.From(newTwitter)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newTwitter, u.GetTwitterID())
			}
		})

		t.Run("Success2", func(t *testing.T) {
			assert, require := assertAndRequire(t)
			newTwitter := ""

			if assert.NoError(repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{TwitterID: optional.From(newTwitter)})) {
				u, err := repo.GetUser(user.GetID(), true)
				require.NoError(err)
				assert.Equal(newTwitter, u.GetTwitterID())
			}
		})
	})

	t.Run("Role", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, repo, rand)

		t.Run("Success", func(t *testing.T) {
			assert, require := assertAndRequire(t)

			if assert.NoError(repo.UpdateUser(user.GetID(), repository.UpdateUserArgs{Role: optional.From("admin")})) {
				u, err := repo.GetUser(user.GetID(), false)
				require.NoError(err)
				assert.Equal("admin", u.GetRole())
			}
		})
	})
}

func TestGormRepository_GetUserStats(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserStats(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserStats(uuid.Must(uuid.NewV7()))
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user := mustMakeUser(t, repo, rand)
		stamp1 := mustMakeStamp(t, repo, rand, user.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user.GetID())

		messages := make([]*model.Message, 15)

		for i := 0; i < 15; i++ {
			messages[i] = mustMakeMessage(t, repo, user.GetID(), channel.ID)
		}
		require.NoError(t, repo.DeleteMessage(messages[14].ID))
		require.NoError(t, repo.DeleteMessage(messages[13].ID))

		for i := 0; i < 5; i++ {
			for j := 0; j < 3; j++ {
				mustAddMessageStamp(t, repo, messages[i].ID, stamp1.ID, user.GetID())
			}
		}

		for i := 0; i < 12; i++ {
			mustAddMessageStamp(t, repo, messages[i].ID, stamp2.ID, user.GetID())
		}

		stats, err := repo.GetUserStats(user.GetID())
		if assert.NoError(t, err) {
			assert.NotEmpty(t, stats.DateTime)

			assert.EqualValues(t, 15, stats.TotalMessageCount)

			if assert.Len(t, stats.Stamps, 2) {
				assert.EqualValues(t, stamp2.ID, stats.Stamps[0].ID)
				assert.EqualValues(t, 12, stats.Stamps[0].Count)
				assert.EqualValues(t, 12, stats.Stamps[0].Total)
				assert.EqualValues(t, stamp1.ID, stats.Stamps[1].ID)
				assert.EqualValues(t, 5, stats.Stamps[1].Count)
				assert.EqualValues(t, 15, stats.Stamps[1].Total)
			}
		}
	})

}
