package repository

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"testing"
)

func TestGormRepository_UpdateChannel(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	cases := []UpdateChannelArgs{
		{
			UpdaterID: user.GetID(),
			Topic:     optional.StringFrom("test"),
		},
		{
			UpdaterID: user.GetID(),
			Topic:     optional.StringFrom(""),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.BoolFrom(true),
			ForcedNotification: optional.BoolFrom(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.BoolFrom(true),
			ForcedNotification: optional.BoolFrom(false),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.BoolFrom(false),
			ForcedNotification: optional.BoolFrom(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.BoolFrom(false),
			ForcedNotification: optional.BoolFrom(false),
		},
	}

	for i, v := range cases {
		v := v
		i := i
		t.Run(fmt.Sprintf("Case%d", i), func(t *testing.T) {
			t.Parallel()
			ch := mustMakeChannel(t, repo, rand)
			changed, err := repo.UpdateChannel(ch.ID, v)
			if assert.NoError(t, err) {
				ch, err := repo.GetChannel(ch.ID)
				require.NoError(t, err)
				assert.EqualValues(t, ch, changed)
			}
		})
	}
}

func TestRepositoryImpl_GetChannel(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)
	channel := mustMakeChannel(t, repo, rand)

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

func TestGormRepository_ChangeChannelSubscription(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, _, err := repo.ChangeChannelSubscription(uuid.Nil, ChangeChannelSubscriptionArgs{})
		assert.EqualError(err, ErrNilID.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		ch := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand)
		user2 := mustMakeUser(t, repo, rand)

		args := ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelMarkAndNotify,
			},
		}
		_, _, err := repo.ChangeChannelSubscription(ch.ID, args)
		if assert.NoError(err) {
			assert.Equal(2, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}

		args = ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelNone,
			},
		}
		_, _, err = repo.ChangeChannelSubscription(ch.ID, args)
		if assert.NoError(err) {
			assert.Equal(1, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}
	})
}

func TestGormRepository_GetChannelStats(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

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

		channel := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand)
		user2 := mustMakeUser(t, repo, rand)
		stamp1 := mustMakeStamp(t, repo, rand, user1.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user1.GetID())

		var u1Message, u2Message []*model.Message
		u1Message = make([]*model.Message, 13)
		u2Message = make([]*model.Message, 14)

		for i := 0; i < 13; i++ {
			u1Message[i] = mustMakeMessage(t, repo, user1.GetID(), channel.ID)
		}

		for i := 0; i < 14; i++ {
			u2Message[i] = mustMakeMessage(t, repo, user2.GetID(), channel.ID)
		}
		require.NoError(t, repo.DeleteMessage(u2Message[12].ID))
		require.NoError(t, repo.DeleteMessage(u2Message[13].ID))

		for i := 0; i < 7; i++ {
			mustAddMessageStamp(t, repo, u1Message[i].ID, stamp1.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u1Message[i].ID, stamp1.ID, user2.GetID())
		}

		for i := 0; i < 12; i++ {
			mustAddMessageStamp(t, repo, u2Message[i].ID, stamp2.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u2Message[i].ID, stamp2.ID, user1.GetID())
		}

		stats, err := repo.GetChannelStats(channel.ID)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, stats.DateTime)

			assert.EqualValues(t, 27, stats.TotalMessageCount)

			if assert.Len(t, stats.Users, 2) {
				assert.EqualValues(t, user2.GetID(), stats.Users[0].ID)
				assert.EqualValues(t, 14, stats.Users[0].MessageCount)
				assert.EqualValues(t, user1.GetID(), stats.Users[1].ID)
				assert.EqualValues(t, 13, stats.Users[1].MessageCount)
			}

			if assert.Len(t, stats.Stamps, 2) {
				assert.EqualValues(t, stamp1.ID, stats.Stamps[0].ID)
				assert.EqualValues(t, 14, stats.Stamps[0].Count)
				assert.EqualValues(t, 14, stats.Stamps[0].Total)
				assert.EqualValues(t, stamp2.ID, stats.Stamps[1].ID)
				assert.EqualValues(t, 12, stats.Stamps[1].Count)
				assert.EqualValues(t, 24, stats.Stamps[1].Total)
			}
		}
	})

}
