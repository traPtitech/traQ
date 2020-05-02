package repository

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"testing"
)

func TestGormRepository_UpdateChannel(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	cases := []UpdateChannelArgs{
		{
			UpdaterID: user.GetID(),
			Topic:     null.StringFrom("test"),
		},
		{
			UpdaterID: user.GetID(),
			Topic:     null.StringFrom(""),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         null.BoolFrom(true),
			ForcedNotification: null.BoolFrom(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         null.BoolFrom(true),
			ForcedNotification: null.BoolFrom(false),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         null.BoolFrom(false),
			ForcedNotification: null.BoolFrom(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         null.BoolFrom(false),
			ForcedNotification: null.BoolFrom(false),
		},
	}

	for i, v := range cases {
		v := v
		i := i
		t.Run(fmt.Sprintf("Case%d", i), func(t *testing.T) {
			t.Parallel()
			ch := mustMakeChannel(t, repo, random)
			if assert.NoError(t, repo.UpdateChannel(ch.ID, v)) {
				ch, err := repo.GetChannel(ch.ID)
				require.NoError(t, err)

				if v.Topic.Valid {
					assert.Equal(t, v.Topic.String, ch.Topic)
				}
				if v.ForcedNotification.Valid {
					assert.Equal(t, v.ForcedNotification.Bool, ch.IsForced)
				}
				if v.Visibility.Valid {
					assert.Equal(t, v.Visibility.Bool, ch.IsVisible)
				}
			}
		})
	}
}

func TestRepositoryImpl_GetChannelByMessageID(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()

		message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		ch, err := repo.GetChannelByMessageID(message.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, channel.ID, ch.ID)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetChannelByMessageID(uuid.Nil)
		assert.Error(t, err)
	})
}

func TestRepositoryImpl_GetChannel(t *testing.T) {
	t.Parallel()
	repo, _, _, channel := setupWithChannel(t, common)

	t.Run("Exists", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		ch, err := repo.GetChannel(channel.ID)
		if assert.NoError(err) {
			assert.Equal(channel.ID, ch.ID)
			assert.Equal(channel.Name, ch.Name)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		_, err := repo.GetChannel(uuid.Nil)
		assert.Error(t, err)
	})
}

func TestRepositoryImpl_GetChannelPath(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	ch1 := mustMakeChannelDetail(t, repo, uuid.Nil, random, uuid.Nil)
	ch2 := mustMakeChannelDetail(t, repo, uuid.Nil, random, ch1.ID)
	ch3 := mustMakeChannelDetail(t, repo, uuid.Nil, random, ch2.ID)

	t.Run("ch1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		path, err := repo.GetChannelPath(ch1.ID)
		if assert.NoError(err) {
			assert.Equal(ch1.Name, path)
		}
	})

	t.Run("ch2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		path, err := repo.GetChannelPath(ch2.ID)
		if assert.NoError(err) {
			assert.Equal(fmt.Sprintf("%s/%s", ch1.Name, ch2.Name), path)
		}
	})

	t.Run("ch3", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		path, err := repo.GetChannelPath(ch3.ID)
		if assert.NoError(err) {
			assert.Equal(fmt.Sprintf("%s/%s/%s", ch1.Name, ch2.Name, ch3.Name), path)
		}
	})

	t.Run("NotExists1", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetChannelPath(uuid.Nil)
		assert.Error(err)
	})

	t.Run("NotExists2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetChannelPath(uuid.Must(uuid.NewV4()))
		assert.Error(err)
	})
}

func TestRepositoryImpl_ChangeChannelName(t *testing.T) {
	t.Parallel()
	repo, _, _, parent := setupWithChannel(t, common)

	c2 := mustMakeChannelDetail(t, repo, uuid.Nil, "test2", parent.ID)
	c3 := mustMakeChannelDetail(t, repo, uuid.Nil, "test3", c2.ID)
	mustMakeChannelDetail(t, repo, uuid.Nil, "test4", c2.ID)

	t.Run("fail", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.Error(repo.UpdateChannel(parent.ID, UpdateChannelArgs{Name: null.StringFrom("")}))
		assert.Error(repo.UpdateChannel(parent.ID, UpdateChannelArgs{Name: null.StringFrom("あああ")}))
		assert.Error(repo.UpdateChannel(parent.ID, UpdateChannelArgs{Name: null.StringFrom("test2???")}))
	})

	t.Run("c2", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		if assert.NoError(repo.UpdateChannel(c2.ID, UpdateChannelArgs{Name: null.StringFrom("aiueo")})) {
			c, err := repo.GetChannel(c2.ID)
			require.NoError(t, err)
			assert.Equal("aiueo", c.Name)
		}
	})

	t.Run("c3", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.Error(repo.UpdateChannel(c3.ID, UpdateChannelArgs{Name: null.StringFrom("test4")}))
		if assert.NoError(repo.UpdateChannel(c3.ID, UpdateChannelArgs{Name: null.StringFrom("test2")})) {
			c, err := repo.GetChannel(c3.ID)
			require.NoError(t, err)
			assert.Equal("test2", c.Name)
		}
	})
}

func TestRepositoryImpl_ChangeChannelParent(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	chName := utils.RandAlphabetAndNumberString(20)
	c2 := mustMakeChannelDetail(t, repo, uuid.Nil, chName, uuid.Nil)
	c3 := mustMakeChannelDetail(t, repo, uuid.Nil, random, c2.ID)
	c4 := mustMakeChannelDetail(t, repo, uuid.Nil, chName, c3.ID)

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.UpdateChannel(c4.ID, UpdateChannelArgs{Parent: uuid.NullUUID{Valid: true, UUID: uuid.Nil}}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		if assert.NoError(t, repo.UpdateChannel(c3.ID, UpdateChannelArgs{Parent: uuid.NullUUID{Valid: true, UUID: uuid.Nil}})) {
			c, err := repo.GetChannel(c3.ID)
			require.NoError(t, err)
			assert.Equal(t, uuid.Nil, c.ParentID)
		}
	})
}

func TestRepositoryImpl_GetChildrenChannelIDs(t *testing.T) {
	t.Parallel()
	repo, _, _, c1 := setupWithChannel(t, common)

	c2 := mustMakeChannelDetail(t, repo, uuid.Nil, random, c1.ID)
	c3 := mustMakeChannelDetail(t, repo, uuid.Nil, random, c2.ID)
	c4 := mustMakeChannelDetail(t, repo, uuid.Nil, random, c2.ID)

	cases := []struct {
		name   string
		ch     uuid.UUID
		expect []uuid.UUID
	}{
		{"c1", c1.ID, []uuid.UUID{c2.ID}},
		{"c2", c2.ID, []uuid.UUID{c3.ID, c4.ID}},
		{"c3", c3.ID, []uuid.UUID{}},
		{"c4", c4.ID, []uuid.UUID{}},
	}

	for _, v := range cases {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			ids, err := repo.GetChildrenChannelIDs(v.ch)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, ids, v.expect)
			}
		})
	}
}

func TestGormRepository_ChangeChannelSubscription(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		assert.EqualError(repo.ChangeChannelSubscription(uuid.Nil, ChangeChannelSubscriptionArgs{}), ErrNilID.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		ch := mustMakeChannel(t, repo, random)
		user1 := mustMakeUser(t, repo, random)
		user2 := mustMakeUser(t, repo, random)

		args := ChangeChannelSubscriptionArgs{
			UpdaterID: uuid.Nil,
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelMarkAndNotify,
			},
		}
		if assert.NoError(repo.ChangeChannelSubscription(ch.ID, args)) {
			assert.Equal(2, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}

		args = ChangeChannelSubscriptionArgs{
			UpdaterID: uuid.Nil,
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelNone,
			},
		}
		if assert.NoError(repo.ChangeChannelSubscription(ch.ID, args)) {
			assert.Equal(1, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}
	})
}

func TestRepositoryImpl_CreatePublicChannel(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	name := utils.RandAlphabetAndNumberString(20)
	c, err := repo.CreatePublicChannel(name, uuid.Nil, user.GetID())
	if assert.NoError(err) {
		assert.NotEmpty(c.ID)
		assert.Equal(name, c.Name)
		assert.Equal(user.GetID(), c.CreatorID)
		assert.EqualValues(uuid.Nil, c.ParentID)
		assert.True(c.IsPublic)
		assert.True(c.IsVisible)
		assert.False(c.IsForced)
		assert.Equal(user.GetID(), c.UpdaterID)
		assert.Empty(c.Topic)
		assert.NotZero(c.CreatedAt)
		assert.NotZero(c.UpdatedAt)
		assert.Nil(c.DeletedAt)
	}

	_, err = repo.CreatePublicChannel(name, uuid.Nil, user.GetID())
	assert.Equal(ErrAlreadyExists, err)

	_, err = repo.CreatePublicChannel("ああああ", uuid.Nil, user.GetID())
	assert.Error(err)

	c2, err := repo.CreatePublicChannel("Parent2", c.ID, user.GetID())
	assert.NoError(err)
	c3, err := repo.CreatePublicChannel("Parent3", c2.ID, user.GetID())
	assert.NoError(err)
	c4, err := repo.CreatePublicChannel("Parent4", c3.ID, user.GetID())
	assert.NoError(err)
	_, err = repo.CreatePublicChannel("Parent4", c3.ID, user.GetID())
	assert.Equal(ErrAlreadyExists, err)
	c5, err := repo.CreatePublicChannel("Parent5", c4.ID, user.GetID())
	assert.NoError(err)
	_, err = repo.CreatePublicChannel("Parent6", c5.ID, user.GetID())
	assert.Equal(ErrChannelDepthLimitation, err)
}

func TestGormRepository_GetChannelStats(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetChannelStats(uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetChannelStats(uuid.Must(uuid.NewV4()))
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < 14; i++ {
			mustMakeMessage(t, repo, user.GetID(), channel.ID)
		}
		require.NoError(t, repo.DeleteMessage(mustMakeMessage(t, repo, user.GetID(), channel.ID).ID))

		stats, err := repo.GetChannelStats(channel.ID)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, stats.DateTime)
			assert.EqualValues(t, 15, stats.TotalMessageCount)
		}
	})
}
