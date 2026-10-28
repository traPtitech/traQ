package gorm

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
)

func TestGormRepository_UpdateChannel(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common, false)

	cases := []repository.UpdateChannelArgs{
		{
			UpdaterID: user.GetID(),
			Topic:     optional.From("test"),
		},
		{
			UpdaterID: user.GetID(),
			Topic:     optional.From(""),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.From(true),
			ForcedNotification: optional.From(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.From(true),
			ForcedNotification: optional.From(false),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.From(false),
			ForcedNotification: optional.From(true),
		},
		{
			UpdaterID:          user.GetID(),
			Visibility:         optional.From(false),
			ForcedNotification: optional.From(false),
		},
	}

	for i, v := range cases {
		v := v
		i := i
		t.Run(fmt.Sprintf("Case%d", i), func(t *testing.T) {
			t.Parallel()
			ch := mustMakeChannel(t, repo, rand)
			changed, err := repo.UpdateChannel(context.TODO(), ch.ID, v)
			if assert.NoError(t, err) {
				ch, err := repo.GetChannel(context.TODO(), ch.ID)
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
		ch, err := repo.GetChannel(context.TODO(), channel.ID)
		if assert.NoError(err) {
			assert.Equal(channel.ID, ch.ID)
			assert.Equal(channel.Name, ch.Name)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		_, err := repo.GetChannel(context.TODO(), uuid.Nil)
		assert.Error(t, err)
	})
}

func TestGormRepository_ChangeChannelSubscription(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, _, err := repo.ChangeChannelSubscription(context.TODO(), uuid.Nil, repository.ChangeChannelSubscriptionArgs{})
		assert.EqualError(err, repository.ErrNilID.Error())
	})

	t.Run("Success (UUIDv4)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		ch := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, true)
		user2 := mustMakeUser(t, repo, rand, true)

		args := repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV7()): model.ChannelSubscribeLevelMarkAndNotify,
			},
		}
		_, _, err := repo.ChangeChannelSubscription(context.TODO(), ch.ID, args)
		if assert.NoError(err) {
			assert.Equal(2, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}

		args = repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV7()): model.ChannelSubscribeLevelNone,
			},
		}
		_, _, err = repo.ChangeChannelSubscription(context.TODO(), ch.ID, args)
		if assert.NoError(err) {
			assert.Equal(1, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}
	})

	t.Run("Success (UUIDv7)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		ch := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, false)
		user2 := mustMakeUser(t, repo, rand, false)

		args := repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelMarkAndNotify,
				uuid.Must(uuid.NewV7()): model.ChannelSubscribeLevelMarkAndNotify,
			},
		}
		_, _, err := repo.ChangeChannelSubscription(context.TODO(), ch.ID, args)
		if assert.NoError(err) {
			assert.Equal(2, count(t, getDB(repo).Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{ChannelID: ch.ID})))
		}

		args = repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
				user1.GetID():           model.ChannelSubscribeLevelMarkAndNotify,
				user2.GetID():           model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV4()): model.ChannelSubscribeLevelNone,
				uuid.Must(uuid.NewV7()): model.ChannelSubscribeLevelNone,
			},
		}
		_, _, err = repo.ChangeChannelSubscription(context.TODO(), ch.ID, args)
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

		_, err := repo.GetChannelStats(context.TODO(), uuid.Nil, false)
		assert.Error(t, err)
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetChannelStats(context.TODO(), uuid.Must(uuid.NewV4()), false)
		assert.Error(t, err)
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetChannelStats(context.TODO(), uuid.Must(uuid.NewV7()), false)
		assert.Error(t, err)
	})

	t.Run("success (include deleted messages) - UUIDv4", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, true)
		user2 := mustMakeUser(t, repo, rand, true)
		stamp1 := mustMakeStamp(t, repo, rand, user1.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user1.GetID())

		u1Messages := make([]*model.Message, 13)
		u2Messages := make([]*model.Message, 14)

		for i := range 13 {
			u1Messages[i] = mustMakeMessage(t, repo, user1.GetID(), channel.ID)
		}

		for i := range 14 {
			u2Messages[i] = mustMakeMessage(t, repo, user2.GetID(), channel.ID)
		}
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[12].ID))
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[13].ID))

		for i := range 7 {
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user2.GetID())
		}

		for i := range 12 {
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
		}

		stats, err := repo.GetChannelStats(context.TODO(), channel.ID, false)
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

	t.Run("success (include deleted messages) - UUIDv7", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, false)
		user2 := mustMakeUser(t, repo, rand, false)
		stamp1 := mustMakeStamp(t, repo, rand, user1.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user1.GetID())

		u1Messages := make([]*model.Message, 13)
		u2Messages := make([]*model.Message, 14)

		for i := range 13 {
			u1Messages[i] = mustMakeMessage(t, repo, user1.GetID(), channel.ID)
		}

		for i := range 14 {
			u2Messages[i] = mustMakeMessage(t, repo, user2.GetID(), channel.ID)
		}
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[12].ID))
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[13].ID))

		for i := range 7 {
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user2.GetID())
		}

		for i := range 12 {
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
		}

		stats, err := repo.GetChannelStats(context.TODO(), channel.ID, false)
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

	t.Run("success (exclude deleted messages) - UUIDv4", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, true)
		user2 := mustMakeUser(t, repo, rand, true)
		stamp1 := mustMakeStamp(t, repo, rand, user1.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user1.GetID())

		u1Messages := make([]*model.Message, 13)
		u2Messages := make([]*model.Message, 14)

		for i := range 13 {
			u1Messages[i] = mustMakeMessage(t, repo, user1.GetID(), channel.ID)
		}

		for i := range 14 {
			u2Messages[i] = mustMakeMessage(t, repo, user2.GetID(), channel.ID)
		}

		for i := range 7 {
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user2.GetID())
		}

		for i := range 14 {
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
		}

		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[12].ID))
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[13].ID))

		stats, err := repo.GetChannelStats(context.TODO(), channel.ID, true)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, stats.DateTime)

			assert.EqualValues(t, 25, stats.TotalMessageCount)

			if assert.Len(t, stats.Users, 2) {
				assert.EqualValues(t, user1.GetID(), stats.Users[0].ID)
				assert.EqualValues(t, 13, stats.Users[0].MessageCount)
				assert.EqualValues(t, user2.GetID(), stats.Users[1].ID)
				assert.EqualValues(t, 12, stats.Users[1].MessageCount)
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

	t.Run("success (exclude deleted messages) - UUIDv7", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		user1 := mustMakeUser(t, repo, rand, false)
		user2 := mustMakeUser(t, repo, rand, false)
		stamp1 := mustMakeStamp(t, repo, rand, user1.GetID())
		stamp2 := mustMakeStamp(t, repo, rand, user1.GetID())

		u1Messages := make([]*model.Message, 13)
		u2Messages := make([]*model.Message, 14)

		for i := range 13 {
			u1Messages[i] = mustMakeMessage(t, repo, user1.GetID(), channel.ID)
		}

		for i := range 14 {
			u2Messages[i] = mustMakeMessage(t, repo, user2.GetID(), channel.ID)
		}

		for i := range 7 {
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u1Messages[i].ID, stamp1.ID, user2.GetID())
		}

		for i := range 14 {
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
			mustAddMessageStamp(t, repo, u2Messages[i].ID, stamp2.ID, user1.GetID())
		}

		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[12].ID))
		require.NoError(t, repo.DeleteMessage(context.TODO(), u2Messages[13].ID))

		stats, err := repo.GetChannelStats(context.TODO(), channel.ID, true)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, stats.DateTime)

			assert.EqualValues(t, 25, stats.TotalMessageCount)

			if assert.Len(t, stats.Users, 2) {
				assert.EqualValues(t, user1.GetID(), stats.Users[0].ID)
				assert.EqualValues(t, 13, stats.Users[0].MessageCount)
				assert.EqualValues(t, user2.GetID(), stats.Users[1].ID)
				assert.EqualValues(t, 12, stats.Users[1].MessageCount)
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

func TestGormRepository_CreateThread(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		conflict string
	}{
		{name: "success"},
		{name: "duplicate public channel name", conflict: "channel"},
		{name: "duplicate thread name", conflict: "thread"},
		{name: "duplicate parent message rolls back channel", conflict: "message"},
		{name: "missing parent message rolls back channel", conflict: "missing"},
		{name: "concurrent parent message conflict", conflict: "concurrent"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
			ctx := context.Background()
			message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
			channelArgs := func(name string) model.Channel {
				return model.Channel{Name: name, ParentID: parent.ID, CreatorID: user.GetID(), UpdaterID: user.GetID(), IsVisible: true}
			}
			name := "thread"
			messageID := message.ID
			var existingChildren int64
			switch tt.conflict {
			case "channel":
				_, err := repo.CreateChannel(ctx, channelArgs(name), nil, model.ChannelTypePublic)
				require.NoError(t, err)
				existingChildren = 1
			case "thread", "message":
				_, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
				require.NoError(t, err)
				existingChildren = 1
				if tt.conflict == "thread" {
					messageID = mustMakeMessage(t, repo, user.GetID(), parent.ID).ID
				} else {
					name = "different-thread"
				}
			case "missing":
				messageID = uuid.Must(uuid.NewV7())
			}
			if tt.conflict == "concurrent" {
				start := make(chan struct{})
				results := make(chan error, 2)
				for i := range 2 {
					go func() {
						<-start
						_, err := repo.CreateThread(ctx, channelArgs(fmt.Sprintf("thread-%d", i)), model.Thread{MessageID: messageID})
						results <- err
					}()
				}
				close(start)
				var successes, conflicts int
				for range 2 {
					err := <-results
					if err == nil {
						successes++
					} else {
						require.ErrorIs(t, err, repository.ErrAlreadyExists)
						conflicts++
					}
				}
				assert.Equal(t, 1, successes)
				assert.Equal(t, 1, conflicts)
				existingChildren = 1
			} else {
				th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
				if tt.conflict == "" {
					require.NoError(t, err)
					require.NotNil(t, th)
					require.NotNil(t, th.Channel)
					ch := th.Channel
					assert.NotEqual(t, uuid.Nil, ch.ID)
					assert.Equal(t, model.ChannelTypeThread, ch.Type)
					assert.Equal(t, parent.ID, ch.ParentID)
					assert.Equal(t, user.GetID(), ch.CreatorID)
					assert.Equal(t, user.GetID(), ch.UpdaterID)
					assert.Equal(t, ch.ID, th.ChannelID)
					assert.Equal(t, messageID, th.MessageID)
					storedChannel, err := repo.GetChannel(ctx, ch.ID)
					require.NoError(t, err)
					assert.Equal(t, name, storedChannel.Name)
					assert.Equal(t, model.ChannelTypeThread, storedChannel.Type)
					var storedThread model.Thread
					require.NoError(t, getDB(repo).First(&storedThread, "channel_id = ?", ch.ID).Error)
					assert.Equal(t, messageID, storedThread.MessageID)
					threads, err := repo.GetThreadChannels(ctx, repository.GetThreadChannelQuery{
						ChannelID: parent.ID, Limit: 20, Sort: repository.ThreadChannelSortCreatedAtDesc,
					})
					require.NoError(t, err)
					require.Len(t, threads, 1)
					assert.Equal(t, ch.ID, threads[0].ChannelID)
					assert.Equal(t, messageID, threads[0].MessageID)
					require.NotNil(t, threads[0].Channel)
					assert.Equal(t, name, threads[0].Channel.Name)
					existingChildren = 1
				} else {
					if tt.conflict == "missing" {
						require.Error(t, err)
						assert.NotErrorIs(t, err, repository.ErrAlreadyExists)
					} else {
						require.ErrorIs(t, err, repository.ErrAlreadyExists)
					}
					assert.Nil(t, th)
				}
			}
			var children, threads int64
			require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
			assert.Equal(t, existingChildren, children)
			require.NoError(t, getDB(repo).Model(&model.Thread{}).
				Joins("JOIN channels ON channels.id = threads.channel_id").
				Where("channels.parent_id = ?", parent.ID).Count(&threads).Error)
			expectedThreads := existingChildren
			if tt.conflict == "channel" {
				expectedThreads = 0
			}
			assert.Equal(t, expectedThreads, threads)
		})
	}
}
