package gorm

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
)

func TestGormRepository_CreateChannel(t *testing.T) {
	t.Parallel()

	t.Run("duplicate public channel name", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := setup(t, common)
		parent := mustMakeChannel(t, repo, rand)
		_, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:     "conflict",
			ParentID: parent.ID,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		_, err = repo.CreateChannel(context.TODO(), model.Channel{
			Name:     "conflict",
			ParentID: parent.ID,
		}, nil, model.ChannelTypePublic)
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
	})

	t.Run("duplicate thread name", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user := setupWithUser(t, common, false)
		parent := mustMakeChannel(t, repo, rand)
		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		_, err := repo.CreateThread(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  parent.ID,
			CreatorID: user.GetID(),
			UpdaterID: user.GetID(),
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		_, err = repo.CreateChannel(context.TODO(), model.Channel{
			Name:     "conflict",
			ParentID: parent.ID,
		}, nil, model.ChannelTypePublic)
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
	})
}

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

func TestGormRepository_UpdateChannel_NameConflict(t *testing.T) {
	t.Parallel()

	t.Run("rename conflicts with public channel", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user := setupWithUser(t, common, false)
		parent := mustMakeChannel(t, repo, rand)
		_, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  parent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		ch, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "original",
			ParentID:  parent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		_, err = repo.UpdateChannel(context.TODO(), ch.ID, repository.UpdateChannelArgs{
			UpdaterID: user.GetID(),
			Name:      optional.From("conflict"),
		})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		stored, err := repo.GetChannel(context.TODO(), ch.ID)
		require.NoError(t, err)
		assert.Equal(t, "original", stored.Name)
		assert.Equal(t, parent.ID, stored.ParentID)
	})

	t.Run("rename conflicts with thread", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user := setupWithUser(t, common, false)
		parent := mustMakeChannel(t, repo, rand)
		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		_, err := repo.CreateThread(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  parent.ID,
			IsVisible: true,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		ch, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "original",
			ParentID:  parent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		_, err = repo.UpdateChannel(context.TODO(), ch.ID, repository.UpdateChannelArgs{
			UpdaterID: user.GetID(),
			Name:      optional.From("conflict"),
		})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		stored, err := repo.GetChannel(context.TODO(), ch.ID)
		require.NoError(t, err)
		assert.Equal(t, "original", stored.Name)
		assert.Equal(t, parent.ID, stored.ParentID)
	})

	t.Run("move conflicts with public channel", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user := setupWithUser(t, common, false)
		parent := mustMakeChannel(t, repo, rand)
		_, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  parent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		originalParent := mustMakeChannel(t, repo, rand)
		ch, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  originalParent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		_, err = repo.UpdateChannel(context.TODO(), ch.ID, repository.UpdateChannelArgs{
			UpdaterID: user.GetID(),
			Parent:    optional.From(parent.ID),
		})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		stored, err := repo.GetChannel(context.TODO(), ch.ID)
		require.NoError(t, err)
		assert.Equal(t, "conflict", stored.Name)
		assert.Equal(t, originalParent.ID, stored.ParentID)
	})

	t.Run("move conflicts with thread", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user := setupWithUser(t, common, false)
		parent := mustMakeChannel(t, repo, rand)
		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		_, err := repo.CreateThread(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  parent.ID,
			IsVisible: true,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		originalParent := mustMakeChannel(t, repo, rand)
		ch, err := repo.CreateChannel(context.TODO(), model.Channel{
			Name:      "conflict",
			ParentID:  originalParent.ID,
			IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		_, err = repo.UpdateChannel(context.TODO(), ch.ID, repository.UpdateChannelArgs{
			UpdaterID: user.GetID(),
			Parent:    optional.From(parent.ID),
		})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		stored, err := repo.GetChannel(context.TODO(), ch.ID)
		require.NoError(t, err)
		assert.Equal(t, "conflict", stored.Name)
		assert.Equal(t, originalParent.ID, stored.ParentID)
	})
}

func TestGormRepository_UpdateThread(t *testing.T) {
	t.Parallel()

	t.Run("success rename", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		// 並列テストで発行されるイベントと混ざらないようにHubを分離する。
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		parent := mustMakeChannel(t, &repo, rand)
		message := mustMakeMessage(t, &repo, user.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name:      "thread",
			ParentID:  parent.ID,
			CreatorID: user.GetID(),
			UpdaterID: user.GetID(),
			IsVisible: true,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		sub := repo.hub.Subscribe(8, event.ThreadUpdated, event.ChannelUpdated)
		updated, err := repo.UpdateThread(ctx, thread.ChannelID, repository.UpdateThreadArgs{
			UpdaterID: user.GetID(),
			Name:      optional.From("renamed"),
		})
		require.NoError(t, err)
		stored, err := repo.GetChannel(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, stored, updated)
		assert.Equal(t, "renamed", stored.Name)

		// Hub.Publishは同期的に配信するため、待機せずに確認できる。
		select {
		case ev := <-sub.Receiver:
			assert.Equal(t, event.ThreadUpdated, ev.Name)
			assert.Equal(t, hub.Fields{"channel_id": thread.ChannelID}, ev.Fields)
		default:
			t.Fatal("THREAD_UPDATED was not published")
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})

	t.Run("success archive", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		// 並列テストで発行されるイベントと混ざらないようにHubを分離する。
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		parent := mustMakeChannel(t, &repo, rand)
		message := mustMakeMessage(t, &repo, user.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name:      "thread",
			ParentID:  parent.ID,
			CreatorID: user.GetID(),
			UpdaterID: user.GetID(),
			IsVisible: true,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		sub := repo.hub.Subscribe(8, event.ThreadUpdated, event.ChannelUpdated)
		updated, err := repo.UpdateThread(ctx, thread.ChannelID, repository.UpdateThreadArgs{
			UpdaterID:  user.GetID(),
			Visibility: optional.From(false),
		})
		require.NoError(t, err)
		stored, err := repo.GetChannel(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, stored, updated)
		assert.False(t, stored.IsVisible)

		// Hub.Publishは同期的に配信するため、待機せずに確認できる。
		select {
		case ev := <-sub.Receiver:
			assert.Equal(t, event.ThreadUpdated, ev.Name)
			assert.Equal(t, hub.Fields{"channel_id": thread.ChannelID}, ev.Fields)
		default:
			t.Fatal("THREAD_UPDATED was not published")
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})

	t.Run("success unarchive", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		// 並列テストで発行されるイベントと混ざらないようにHubを分離する。
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		parent := mustMakeChannel(t, &repo, rand)
		message := mustMakeMessage(t, &repo, user.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name:      "thread",
			ParentID:  parent.ID,
			CreatorID: user.GetID(),
			UpdaterID: user.GetID(),
			IsVisible: false,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		sub := repo.hub.Subscribe(8, event.ThreadUpdated, event.ChannelUpdated)
		updated, err := repo.UpdateThread(ctx, thread.ChannelID, repository.UpdateThreadArgs{
			UpdaterID:  user.GetID(),
			Visibility: optional.From(true),
		})
		require.NoError(t, err)
		stored, err := repo.GetChannel(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, stored, updated)
		assert.True(t, stored.IsVisible)

		// Hub.Publishは同期的に配信するため、待機せずに確認できる。
		select {
		case ev := <-sub.Receiver:
			assert.Equal(t, event.ThreadUpdated, ev.Name)
			assert.Equal(t, hub.Fields{"channel_id": thread.ChannelID}, ev.Fields)
		default:
			t.Fatal("THREAD_UPDATED was not published")
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})

	t.Run("duplicate sibling channel name", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		// 並列テストで発行されるイベントと混ざらないようにHubを分離する。
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		parent := mustMakeChannel(t, &repo, rand)
		message := mustMakeMessage(t, &repo, user.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name:      "thread",
			ParentID:  parent.ID,
			CreatorID: user.GetID(),
			UpdaterID: user.GetID(),
			IsVisible: true,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)
		_, err = repo.CreateChannel(ctx, model.Channel{
			Name: "sibling", ParentID: parent.ID, IsVisible: true,
		}, nil, model.ChannelTypePublic)
		require.NoError(t, err)

		sub := repo.hub.Subscribe(8, event.ThreadUpdated, event.ChannelUpdated)
		_, err = repo.UpdateThread(ctx, thread.ChannelID, repository.UpdateThreadArgs{
			UpdaterID: user.GetID(),
			Name:      optional.From("sibling"),
		})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		stored, err := repo.GetChannel(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, "thread", stored.Name)
		assert.True(t, stored.IsVisible)
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})

	t.Run("channel not found", func(t *testing.T) {
		t.Parallel()
		base, _, _ := setup(t, common)
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()

		sub := repo.hub.Subscribe(8, event.ThreadUpdated, event.ChannelUpdated)
		_, err := repo.UpdateThread(context.Background(), mustGenerateUUID(false), repository.UpdateThreadArgs{})
		require.ErrorIs(t, err, repository.ErrNotFound)
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})
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

	t.Run("Replace preserves mark level", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		parent := mustMakeChannel(t, repo, rand)
		added := mustMakeUser(t, repo, rand, false).GetID()
		removed := mustMakeUser(t, repo, rand, false).GetID()
		marked := mustMakeUser(t, repo, rand, false).GetID()

		_, _, err := repo.ChangeChannelSubscription(ctx, parent.ID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{removed: model.ChannelSubscribeLevelMarkAndNotify, marked: model.ChannelSubscribeLevelMark},
		})
		require.NoError(t, err)

		on, off, err := repo.ChangeChannelSubscription(ctx, parent.ID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{added: model.ChannelSubscribeLevelMarkAndNotify}, KeepOffLevel: true, Replace: true,
		})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{added}, on)
		assert.Equal(t, []uuid.UUID{removed}, off)

		subs, err := repo.GetChannelSubscriptions(ctx, repository.ChannelSubscriptionQuery{}.SetChannel(parent.ID).SetLevel(model.ChannelSubscribeLevelMark))
		require.NoError(t, err)
		require.Len(t, subs, 1)
		assert.Equal(t, marked, subs[0].UserID)
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

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		name := "thread"
		messageID := message.ID

		th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
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

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 1, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 1, threadCount)
	})

	t.Run("duplicate public channel name", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		name := "thread"
		messageID := message.ID
		_, err := repo.CreateChannel(ctx, channelArgs(name), nil, model.ChannelTypePublic)
		require.NoError(t, err)

		th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		assert.Nil(t, th)

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 1, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 0, threadCount)
	})

	t.Run("duplicate thread name", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		name := "thread"
		messageID := message.ID
		_, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.NoError(t, err)
		messageID = mustMakeMessage(t, repo, user.GetID(), parent.ID).ID

		th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		assert.Nil(t, th)

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 1, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 1, threadCount)
	})

	t.Run("duplicate parent message rolls back channel", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		name := "thread"
		messageID := message.ID
		_, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.NoError(t, err)
		name = "different-thread"

		th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.ErrorIs(t, err, repository.ErrAlreadyExists)
		assert.Nil(t, th)

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 1, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 1, threadCount)
	})

	t.Run("missing parent message rolls back channel", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		name := "thread"
		messageID := uuid.Must(uuid.NewV7())

		th, err := repo.CreateThread(ctx, channelArgs(name), model.Thread{MessageID: messageID})
		require.Error(t, err)
		assert.NotErrorIs(t, err, repository.ErrAlreadyExists)

		assert.Nil(t, th)

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 0, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 0, threadCount)
	})

	t.Run("concurrent parent message conflict", func(t *testing.T) {
		t.Parallel()
		repo, _, _, user, parent := setupWithUserAndChannel(t, common, false)
		ctx := context.Background()

		message := mustMakeMessage(t, repo, user.GetID(), parent.ID)
		channelArgs := func(name string) model.Channel {
			return model.Channel{
				Name:      name,
				ParentID:  parent.ID,
				CreatorID: user.GetID(),
				UpdaterID: user.GetID(),
				IsVisible: true,
			}
		}
		messageID := message.ID

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

		var children, threadCount int64
		require.NoError(t, getDB(repo).Model(&model.Channel{}).Where("parent_id = ?", parent.ID).Count(&children).Error)
		assert.EqualValues(t, 1, children)
		require.NoError(t, getDB(repo).Model(&model.Thread{}).
			Joins("JOIN channels ON channels.id = threads.channel_id").
			Where("channels.parent_id = ?", parent.ID).Count(&threadCount).Error)

		assert.EqualValues(t, 1, threadCount)
	})
}

func TestGormRepository_GetThreadSubscribers(t *testing.T) {
	t.Parallel()

	repo, _, _ := setup(t, common)
	ctx := context.Background()

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetThreadSubscribers(ctx, uuid.Nil)
		require.ErrorIs(t, err, repository.ErrNilID)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetThreadSubscribers(ctx, mustGenerateUUID(false))
		require.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("Public channel", func(t *testing.T) {
		t.Parallel()

		public := mustMakeChannel(t, repo, rand)

		_, err := repo.GetThreadSubscribers(ctx, public.ID)
		require.ErrorIs(t, err, repository.ErrNotFound)
	})

	// 各ケースで独立した親・スレッド・ユーザーを作成する。
	checkSubscriber := func(t *testing.T, threadLevel, defaultLevel, parentLevel int, forced, subscribed bool) {
		t.Helper()

		parent := mustMakeChannel(t, repo, rand)
		require.NoError(t, getDB(repo).Model(parent).Update("is_forced", forced).Error)
		user := mustMakeUser(t, repo, rand, false)
		id := user.GetID()
		message := mustMakeMessage(t, repo, id, parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name: "subscribers", ParentID: parent.ID,
			CreatorID: id, UpdaterID: id,
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)
		_, _, err = repo.ChangeChannelSubscription(ctx, parent.ID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{id: model.ChannelSubscribeLevel(parentLevel)},
		})
		require.NoError(t, err)
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, id, parent.ID, defaultLevel))
		require.NoError(t, repo.SetThreadSubscription(ctx, id, thread.ChannelID, threadLevel))

		actual, err := repo.GetThreadSubscribers(ctx, thread.ChannelID)
		require.NoError(t, err)
		if forced {
			// 強制通知では、他のテストのユーザーも対象になる。
			if subscribed {
				assert.Contains(t, actual, id)
			} else {
				assert.NotContains(t, actual, id)
			}
		} else if subscribed {
			assert.ElementsMatch(t, []uuid.UUID{id}, actual)
		} else {
			assert.Empty(t, actual)
		}
	}

	t.Run("Thread notify", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, 2, 0, 0, false, true)
	})

	t.Run("Thread none overrides parent notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, 0, 2, 2, false, false)
	})

	t.Run("Thread mark overrides parent notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, 1, 2, 2, false, false)
	})

	t.Run("Inherit default notify", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, 2, 0, false, true)
	})

	t.Run("Default none overrides parent notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, 0, 2, false, false)
	})

	t.Run("Default mark overrides parent notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, 1, 2, false, false)
	})

	t.Run("Inherit parent notify", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, -1, 2, false, true)
	})

	t.Run("Inherit parent mark", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, -1, 1, false, false)
	})

	t.Run("No subscription", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, -1, 0, false, false)
	})

	t.Run("Inherit forced notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, -1, 0, true, true)
	})

	t.Run("Thread none overrides forced notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, 0, -1, 0, true, false)
	})

	t.Run("Thread mark overrides forced notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, 1, -1, 0, true, false)
	})

	t.Run("Default none overrides forced notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, 0, 0, true, false)
	})

	t.Run("Default mark overrides forced notification", func(t *testing.T) {
		t.Parallel()
		checkSubscriber(t, -1, 1, 0, true, false)
	})
}

func TestGormRepository_ChangeThreadSubscription(t *testing.T) {
	t.Parallel()

	repo, _, _ := setup(t, common)
	ctx := context.Background()

	newThread := func(t *testing.T) (*model.Channel, *model.Thread) {
		t.Helper()

		parent := mustMakeChannel(t, repo, rand)
		creator := mustMakeUser(t, repo, rand, false)
		message := mustMakeMessage(t, repo, creator.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{Name: "subscribers", ParentID: parent.ID,
			CreatorID: creator.GetID(), UpdaterID: creator.GetID()}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)

		return parent, thread
	}

	t.Run("Nil ID", func(t *testing.T) {
		t.Parallel()

		_, _, err := repo.ChangeThreadSubscription(ctx, uuid.Nil, repository.ChangeChannelSubscriptionArgs{})
		require.ErrorIs(t, err, repository.ErrNilID)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()

		_, _, err := repo.ChangeThreadSubscription(ctx, mustGenerateUUID(false), repository.ChangeChannelSubscriptionArgs{})
		require.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("Invalid level", func(t *testing.T) {
		t.Parallel()

		_, thread := newThread(t)

		_, _, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{mustGenerateUUID(false): model.ChannelSubscribeLevelMark},
		})
		require.Error(t, err)
	})

	t.Run("Patch preserves unspecified users and ignores nonexistent users", func(t *testing.T) {
		t.Parallel()

		parent, thread := newThread(t)
		added := mustMakeUser(t, repo, rand, true).GetID()
		removed := mustMakeUser(t, repo, rand, false).GetID()
		unchanged := mustMakeUser(t, repo, rand, false).GetID()

		_, _, err := repo.ChangeChannelSubscription(ctx, parent.ID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{removed: model.ChannelSubscribeLevelMarkAndNotify, unchanged: model.ChannelSubscribeLevelMarkAndNotify},
		})
		require.NoError(t, err)

		on, off, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{added: model.ChannelSubscribeLevelMarkAndNotify, removed: model.ChannelSubscribeLevelNone, mustGenerateUUID(false): model.ChannelSubscribeLevelMarkAndNotify},
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{added}, on)
		assert.ElementsMatch(t, []uuid.UUID{removed}, off)

		subscribers, err := repo.GetThreadSubscribers(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{added, unchanged}, subscribers)

		setting, err := repo.GetThreadSubscription(ctx, removed, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, model.ChannelSubscribeLevelNone, setting.Level)

		_, err = repo.GetThreadSubscription(ctx, unchanged, thread.ChannelID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	checkOff := func(t *testing.T, threadLevel, defaultLevel, parentLevel int, forced, removed bool) {
		t.Helper()

		parent, thread := newThread(t)
		id := mustMakeUser(t, repo, rand, false).GetID()

		_, _, err := repo.ChangeChannelSubscription(ctx, parent.ID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{id: model.ChannelSubscribeLevel(parentLevel)},
		})
		require.NoError(t, err)
		require.NoError(t, repo.SetThreadSubscription(ctx, id, thread.ChannelID, threadLevel))
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, id, parent.ID, defaultLevel))
		require.NoError(t, getDB(repo).Model(parent).Update("is_forced", forced).Error)

		on, off, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{id: model.ChannelSubscribeLevelNone},
		})
		require.NoError(t, err)
		assert.Empty(t, on)
		if removed {
			assert.Equal(t, []uuid.UUID{id}, off)
			threadLevel = 0
		} else {
			assert.Empty(t, off)
		}

		setting, err := repo.GetThreadSubscription(ctx, id, thread.ChannelID)
		if threadLevel == -1 {
			assert.ErrorIs(t, err, repository.ErrNotFound)
		} else {
			require.NoError(t, err)
			assert.EqualValues(t, threadLevel, setting.Level)
		}
	}

	t.Run("Off explicit notification", func(t *testing.T) {
		t.Parallel()
		checkOff(t, 2, -1, 0, false, true)
	})

	t.Run("Off default notification", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, 2, 0, false, true)
	})

	t.Run("Off parent notification", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, -1, 2, false, true)
	})

	t.Run("Off forced notification", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, -1, 0, true, true)
	})

	t.Run("Keep explicit mark", func(t *testing.T) {
		t.Parallel()
		checkOff(t, 1, 2, 2, false, false)
	})

	t.Run("Keep default mark inheritance", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, 1, 2, false, false)
	})

	t.Run("Keep parent mark inheritance", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, -1, 1, false, false)
	})

	t.Run("Keep default none under forced notification", func(t *testing.T) {
		t.Parallel()
		checkOff(t, -1, 0, 0, true, false)
	})

	t.Run("Put replaces only notified users", func(t *testing.T) {
		t.Parallel()

		parent, thread := newThread(t)
		added := mustMakeUser(t, repo, rand, false).GetID()
		removed := mustMakeUser(t, repo, rand, false).GetID()
		marked := mustMakeUser(t, repo, rand, false).GetID()
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, removed, parent.ID, 2))
		require.NoError(t, repo.SetThreadSubscription(ctx, marked, thread.ChannelID, 1))

		on, off, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{added: model.ChannelSubscribeLevelMarkAndNotify}, Replace: true,
		})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{added}, on)
		assert.Equal(t, []uuid.UUID{removed}, off)

		setting, err := repo.GetThreadSubscription(ctx, marked, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, model.ChannelSubscribeLevelMark, setting.Level)

		actual, err := repo.GetThreadSubscribers(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{added}, actual)

		// 空のPUTで全通知購読者を外しても、未読管理は維持する。
		on, off, err = repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{Replace: true})
		require.NoError(t, err)
		assert.Empty(t, on)
		assert.Equal(t, []uuid.UUID{added}, off)

		actual, err = repo.GetThreadSubscribers(ctx, thread.ChannelID)
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("On pins inherited notification and unchanged state returns no changes", func(t *testing.T) {
		t.Parallel()

		parent, thread := newThread(t)
		id := mustMakeUser(t, repo, rand, false).GetID()
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, id, parent.ID, 2))

		on, off, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{id: model.ChannelSubscribeLevelMarkAndNotify},
		})
		require.NoError(t, err)
		assert.Empty(t, on)
		assert.Empty(t, off)

		setting, err := repo.GetThreadSubscription(ctx, id, thread.ChannelID)
		require.NoError(t, err)
		assert.Equal(t, model.ChannelSubscribeLevelMarkAndNotify, setting.Level)
	})
}

func TestGormRepository_SetThreadSubscriptionEvents(t *testing.T) {
	t.Parallel()
	checkEvent := func(t *testing.T, previous, next int, wantEvent bool) {
		t.Helper()
		base, _, _, user := setupWithUser(t, common, false)
		repo := *base.(*Repository)
		// 並列テストのイベントが混ざらないようにHubを分離する。
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		channel := mustMakeChannel(t, &repo, rand)
		require.NoError(t, repo.SetThreadSubscription(ctx, user.GetID(), channel.ID, previous))
		sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ThreadDefaultSubscribersChanged, event.ChannelSubscribersChanged)
		require.NoError(t, repo.SetThreadSubscription(ctx, user.GetID(), channel.ID, next))
		setting, err := repo.GetThreadSubscription(ctx, user.GetID(), channel.ID)
		if next == -1 {
			require.ErrorIs(t, err, repository.ErrNotFound)
		} else {
			require.NoError(t, err)
			assert.Equal(t, model.ChannelSubscribeLevel(next), setting.Level)
		}
		if wantEvent {
			select {
			case ev := <-sub.Receiver:
				assert.Equal(t, event.ThreadSubscribersChanged, ev.Name)
				assert.Equal(t, hub.Fields{"channel_id": channel.ID, "subscriber_ids": []uuid.UUID{user.GetID()}}, ev.Fields)
			default:
				t.Fatal("subscription event was not published")
			}
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	}

	t.Run("Inherit to Inherit does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, -1, false)
	})

	t.Run("Inherit to None publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 0, true)
	})

	t.Run("Inherit to Mark publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 1, true)
	})

	t.Run("Inherit to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 2, true)
	})

	t.Run("None to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, -1, true)
	})

	t.Run("None to None does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 0, false)
	})

	t.Run("None to Mark does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 1, false)
	})

	t.Run("None to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 2, true)
	})

	t.Run("Mark to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, -1, true)
	})

	t.Run("Mark to None does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 0, false)
	})

	t.Run("Mark to Mark does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 1, false)
	})

	t.Run("Mark to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 2, true)
	})

	t.Run("Notify to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, -1, true)
	})

	t.Run("Notify to None publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 0, true)
	})

	t.Run("Notify to Mark publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 1, true)
	})

	t.Run("Notify to Notify does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 2, false)
	})

	t.Run("Invalid writes do not publish events", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		channel := mustMakeChannel(t, &repo, rand)
		sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ThreadDefaultSubscribersChanged, event.ChannelSubscribersChanged)
		require.Error(t, repo.SetThreadSubscription(ctx, user.GetID(), channel.ID, -2))
		require.Error(t, repo.SetThreadSubscription(ctx, user.GetID(), channel.ID, 3))
		require.ErrorIs(t, repo.SetThreadSubscription(ctx, uuid.Nil, channel.ID, 2), repository.ErrNilID)
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})
}

func TestGormRepository_SetThreadSubscriptionDefaultEvents(t *testing.T) {
	t.Parallel()
	checkEvent := func(t *testing.T, previous, next int, wantEvent bool) {
		t.Helper()
		base, _, _, user := setupWithUser(t, common, false)
		repo := *base.(*Repository)
		// 並列テストのイベントが混ざらないようにHubを分離する。
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		channel := mustMakeChannel(t, &repo, rand)
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, user.GetID(), channel.ID, previous))
		sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ThreadDefaultSubscribersChanged, event.ChannelSubscribersChanged)
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, user.GetID(), channel.ID, next))
		settings, err := repo.GetThreadSubscriptionDefaults(ctx, user.GetID())
		require.NoError(t, err)
		if next == -1 {
			assert.Empty(t, settings)
		} else {
			require.Len(t, settings, 1)
			assert.Equal(t, model.ChannelSubscribeLevel(next), settings[0].Level)
		}
		if wantEvent {
			select {
			case ev := <-sub.Receiver:
				assert.Equal(t, event.ThreadDefaultSubscribersChanged, ev.Name)
				assert.Equal(t, hub.Fields{"channel_id": channel.ID, "subscriber_ids": []uuid.UUID{user.GetID()}}, ev.Fields)
			default:
				t.Fatal("subscription event was not published")
			}
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	}

	t.Run("Inherit to Inherit does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, -1, false)
	})

	t.Run("Inherit to None publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 0, true)
	})

	t.Run("Inherit to Mark publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 1, true)
	})

	t.Run("Inherit to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 2, true)
	})

	t.Run("None to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, -1, true)
	})

	t.Run("None to None does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 0, false)
	})

	t.Run("None to Mark does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 1, false)
	})

	t.Run("None to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 2, true)
	})

	t.Run("Mark to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, -1, true)
	})

	t.Run("Mark to None does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 0, false)
	})

	t.Run("Mark to Mark does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 1, false)
	})

	t.Run("Mark to Notify publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 1, 2, true)
	})

	t.Run("Notify to Inherit publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, -1, true)
	})

	t.Run("Notify to None publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 0, true)
	})

	t.Run("Notify to Mark publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 1, true)
	})

	t.Run("Notify to Notify does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 2, false)
	})

	t.Run("Invalid writes do not publish events", func(t *testing.T) {
		t.Parallel()
		base, _, _, user := setupWithUser(t, common, false)
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		channel := mustMakeChannel(t, &repo, rand)
		sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ThreadDefaultSubscribersChanged, event.ChannelSubscribersChanged)
		require.Error(t, repo.SetThreadSubscriptionDefault(ctx, user.GetID(), channel.ID, -2))
		require.Error(t, repo.SetThreadSubscriptionDefault(ctx, user.GetID(), channel.ID, 3))
		require.ErrorIs(t, repo.SetThreadSubscriptionDefault(ctx, uuid.Nil, channel.ID, 2), repository.ErrNilID)
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	})
}

func TestGormRepository_ChangeThreadSubscriptionEvents(t *testing.T) {
	t.Parallel()
	checkEvent := func(t *testing.T, previous, next int, wantEvent bool) {
		t.Helper()
		base, _, _, user := setupWithUser(t, common, false)
		repo := *base.(*Repository)
		repo.hub = hub.New()
		defer repo.hub.Close()
		ctx := context.Background()
		parent := mustMakeChannel(t, &repo, rand)
		message := mustMakeMessage(t, &repo, user.GetID(), parent.ID)
		thread, err := repo.CreateThread(ctx, model.Channel{
			Name: "thread", ParentID: parent.ID, CreatorID: user.GetID(), UpdaterID: user.GetID(),
		}, model.Thread{MessageID: message.ID})
		require.NoError(t, err)
		require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, user.GetID(), parent.ID, 2))
		require.NoError(t, repo.SetThreadSubscription(ctx, user.GetID(), thread.ChannelID, previous))
		sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ChannelSubscribersChanged)
		_, _, err = repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
			Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{user.GetID(): model.ChannelSubscribeLevel(next)},
		})
		require.NoError(t, err)
		if wantEvent {
			select {
			case ev := <-sub.Receiver:
				assert.Equal(t, event.ThreadSubscribersChanged, ev.Name)
				assert.Equal(t, hub.Fields{"channel_id": thread.ChannelID, "subscriber_ids": []uuid.UUID{user.GetID()}}, ev.Fields)
			default:
				t.Fatal("subscription event was not published")
			}
		}
		select {
		case ev := <-sub.Receiver:
			t.Fatalf("unexpected event: %+v", ev)
		default:
		}
	}

	t.Run("Pin inherited notification publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, -1, 2, true)
	})

	t.Run("Unchanged notification does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 2, false)
	})

	t.Run("Disable notification publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 2, 0, true)
	})

	t.Run("Unchanged none does not publish event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 0, false)
	})

	t.Run("Enable notification publishes event", func(t *testing.T) {
		t.Parallel()
		checkEvent(t, 0, 2, true)
	})
}

func TestGormRepository_ChangeThreadSubscriptionEventUsers(t *testing.T) {
	t.Parallel()
	base, _, _, creator := setupWithUser(t, common, false)
	repo := *base.(*Repository)
	repo.hub = hub.New()
	defer repo.hub.Close()
	ctx := context.Background()
	parent := mustMakeChannel(t, &repo, rand)
	message := mustMakeMessage(t, &repo, creator.GetID(), parent.ID)
	thread, err := repo.CreateThread(ctx, model.Channel{
		Name: "thread", ParentID: parent.ID, CreatorID: creator.GetID(), UpdaterID: creator.GetID(),
	}, model.Thread{MessageID: message.ID})
	require.NoError(t, err)
	inherited := mustMakeUser(t, &repo, rand, false).GetID()
	unchanged := mustMakeUser(t, &repo, rand, false).GetID()
	removed := mustMakeUser(t, &repo, rand, false).GetID()
	require.NoError(t, repo.SetThreadSubscriptionDefault(ctx, inherited, parent.ID, 2))
	require.NoError(t, repo.SetThreadSubscription(ctx, unchanged, thread.ChannelID, 2))
	require.NoError(t, repo.SetThreadSubscription(ctx, removed, thread.ChannelID, 2))
	sub := repo.hub.Subscribe(8, event.ThreadSubscribersChanged, event.ChannelSubscribersChanged)
	on, off, err := repo.ChangeThreadSubscription(ctx, thread.ChannelID, repository.ChangeChannelSubscriptionArgs{
		Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{inherited: 2, unchanged: 2, removed: 0},
	})
	require.NoError(t, err)
	assert.Empty(t, on)
	assert.ElementsMatch(t, []uuid.UUID{removed}, off)
	select {
	case ev := <-sub.Receiver:
		assert.Equal(t, event.ThreadSubscribersChanged, ev.Name)
		assert.Equal(t, thread.ChannelID, ev.Fields["channel_id"])
		assert.ElementsMatch(t, []uuid.UUID{inherited, removed}, ev.Fields["subscriber_ids"])
	default:
		t.Fatal("subscription event was not published")
	}
	select {
	case ev := <-sub.Receiver:
		t.Fatalf("unexpected event: %+v", ev)
	default:
	}
}
