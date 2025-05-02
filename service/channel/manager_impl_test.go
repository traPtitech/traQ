package channel

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/set"
)

func initCM(t *testing.T, repo repository.ChannelRepository) *managerImpl {
	return &managerImpl{
		R:               repo,
		L:               zap.NewNop(),
		T:               makeTestChannelTree(t),
		MaxChannelDepth: 5,
	}
}

func TestInitChannelManager(t *testing.T) {
	t.Parallel()

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetPublicChannels().
			Return(nil, mockErr).
			Times(1)

		_, err := InitChannelManager(repo, zap.NewNop())
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("bad tree", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)

		repo.EXPECT().
			GetPublicChannels().
			Return([]*model.Channel{
				{ID: cABC, Name: "c", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
				{ID: cABCD, Name: "d", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
				{ID: cABCE, Name: "e", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: true},
			}, nil).
			Times(1)

		_, err := InitChannelManager(repo, zap.NewNop())
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)

		repo.EXPECT().
			GetPublicChannels().
			Return([]*model.Channel{}, nil).
			Times(1)

		m, err := InitChannelManager(repo, zap.NewNop())
		if assert.NoError(t, err) {
			assert.NotNil(t, m)
		}
	})
}

func TestManagerImpl_GetChannel(t *testing.T) {
	t.Parallel()

	must := func(c *model.Channel, _ error) *model.Channel { return c }

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		dm1 := &model.Channel{
			ID:        uuid.NewV3(uuid.Nil, "c 1-1"),
			Name:      "a",
			ParentID:  dmChannelRootUUID,
			IsForced:  false,
			IsPublic:  false,
			IsVisible: true,
		}

		repo.EXPECT().
			GetChannel(dm1.ID).
			Return(dm1, nil).
			Times(1)

		cases := []struct {
			ID  uuid.UUID
			Exp *model.Channel
		}{
			{ID: cABFA, Exp: must(cm.PublicChannelTree().GetModel(cABFA))},
			{ID: cABB, Exp: must(cm.PublicChannelTree().GetModel(cABB))},
			{ID: dm1.ID, Exp: dm1},
		}
		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				ch, err := cm.GetChannel(c.ID)
				if assert.NoError(t, err) {
					assert.EqualValues(t, c.Exp, ch)
				}
			})
		}
	})

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetChannel(cNotFound).
			Return(nil, repository.ErrNotFound).
			Times(1)

		_, err := cm.GetChannel(cNotFound)
		assert.EqualError(t, err, ErrChannelNotFound.Error())
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetChannel(cNotFound).
			Return(nil, mockErr).
			Times(1)

		_, err := cm.GetChannel(cNotFound)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_CreatePublicChannel(t *testing.T) {
	t.Parallel()

	t.Run("ErrInvalidChannelName", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		cases := []string{
			"ああああ",
		}
		for _, name := range cases {
			_, err := cm.CreatePublicChannel(name, uuid.Nil, uuid.Nil)
			assert.EqualError(t, err, ErrInvalidChannelName.Error())
		}
	})

	t.Run("ErrChannelNameConflicts", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		cases := []struct {
			Name   string
			Parent uuid.UUID
		}{
			{Name: "a", Parent: uuid.Nil},
			{Name: "b", Parent: cA},
			{Name: "d", Parent: cA},
			{Name: "c", Parent: cAB},
			{Name: "f", Parent: cAB},
			{Name: "b", Parent: cAB},
			{Name: "e", Parent: uuid.Nil},
		}
		for _, ch := range cases {
			_, err := cm.CreatePublicChannel(ch.Name, ch.Parent, uuid.Nil)
			assert.EqualError(t, err, ErrChannelNameConflicts.Error())
		}
	})

	t.Run("ErrInvalidParentChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		cases := []struct {
			Name   string
			Parent uuid.UUID
		}{
			{Name: "b", Parent: cNotFound},
		}
		for _, c := range cases {
			_, err := cm.CreatePublicChannel(c.Name, c.Parent, uuid.Nil)
			assert.EqualError(t, err, ErrInvalidParentChannel.Error())
		}
	})

	t.Run("ErrChannelArchived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		cases := []struct {
			Name   string
			Parent uuid.UUID
		}{
			{Name: "a", Parent: cABB},
			{Name: "a", Parent: cABBC},
		}
		for _, c := range cases {
			_, err := cm.CreatePublicChannel(c.Name, c.Parent, uuid.Nil)
			assert.EqualError(t, err, ErrChannelArchived.Error())
		}
	})

	t.Run("ErrTooDeepChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		cases := []struct {
			Name   string
			Parent uuid.UUID
		}{
			{Name: "a", Parent: cEFGHI},
		}
		for _, c := range cases {
			_, err := cm.CreatePublicChannel(c.Name, c.Parent, uuid.Nil)
			assert.EqualError(t, err, ErrTooDeepChannel.Error())
		}
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			CreateChannel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, mockErr).
			AnyTimes()

		_, err := cm.CreatePublicChannel("test", uuid.Nil, uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			Name    string
			Parent  uuid.UUID
			Creator uuid.UUID
		}{
			{Name: "test1", Parent: uuid.Nil, Creator: cA},
			{Name: "test1", Parent: cA, Creator: cAB},
			{Name: "test2", Parent: cA, Creator: cABC},
			{Name: "test2", Parent: cEFGJ, Creator: cABCD},
		}
		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				repo := mock_repository.NewMockChannelRepository(ctrl)
				cm := initCM(t, repo)

				cid := uuid.Must(uuid.NewV7())
				createdAt := time.Now()
				expected := &model.Channel{
					ID:         cid,
					Name:       c.Name,
					ParentID:   c.Parent,
					CreatorID:  c.Creator,
					UpdaterID:  c.Creator,
					IsPublic:   true,
					IsForced:   false,
					IsVisible:  true,
					CreatedAt:  createdAt,
					UpdatedAt:  createdAt,
					DeletedAt:  gorm.DeletedAt{},
					ChildrenID: make([]uuid.UUID, 0),
				}

				repo.EXPECT().
					CreateChannel(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(expected, nil).
					AnyTimes()
				if c.Parent != uuid.Nil {
					repo.EXPECT().
						RecordChannelEvent(c.Parent, model.ChannelEventChildCreated, gomock.Eq(model.ChannelEventDetail{
							"userId":    c.Creator,
							"channelId": cid,
						}), createdAt).
						Return(nil).
						Times(1)
				}

				ch, err := cm.CreatePublicChannel(c.Name, c.Parent, c.Creator)
				cm.P.Wait()
				if assert.NoError(t, err) {
					assert.Equal(t, expected, ch)
					assert.True(t, cm.PublicChannelTree().IsChannelPresent(cid))
				}
			})
		}
	})
}

func TestManagerImpl_UpdateChannel(t *testing.T) {
	t.Parallel()

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetChannel(gomock.Any()).
			Return(nil, repository.ErrNotFound).
			AnyTimes()

		err := cm.UpdateChannel(cNotFound, repository.UpdateChannelArgs{Topic: optional.From("test")})
		assert.EqualError(t, err, ErrChannelNotFound.Error())
	})

	t.Run("ErrChannelNameConflicts", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Name: optional.From("e")})
		assert.EqualError(t, err, ErrChannelNameConflicts.Error())
	})

	t.Run("ErrInvalidChannelName", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Name: optional.From("あああ")})
		assert.EqualError(t, err, ErrInvalidChannelName.Error())
	})

	t.Run("ErrInvalidParentChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Parent: optional.From(cNotFound)})
		assert.EqualError(t, err, ErrInvalidParentChannel.Error())
	})

	t.Run("ErrInvalidParentChannel (archived)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Parent: optional.From(cABBC)})
		assert.EqualError(t, err, ErrInvalidParentChannel.Error())
	})

	t.Run("ErrTooDeepChannel (loop)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Parent: optional.From(cABCE)})
		assert.EqualError(t, err, ErrTooDeepChannel.Error())
	})

	t.Run("ErrTooDeepChannel (limit exceeded)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Parent: optional.From(cEF)})
		assert.EqualError(t, err, ErrTooDeepChannel.Error())
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			UpdateChannel(gomock.Any(), gomock.Any()).
			Return(nil, mockErr).
			AnyTimes()

		err := cm.UpdateChannel(cA, repository.UpdateChannelArgs{Topic: optional.From("test")})
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			ID   uuid.UUID
			Args repository.UpdateChannelArgs
		}{
			{
				ID: cA,
				Args: repository.UpdateChannelArgs{
					UpdaterID: uuid.Must(uuid.NewV7()),
					Topic:     optional.From(""),
				},
			},
			{
				ID: cA,
				Args: repository.UpdateChannelArgs{
					UpdaterID:  uuid.Must(uuid.NewV7()),
					Visibility: optional.From(true),
				},
			},
			{
				ID: cA,
				Args: repository.UpdateChannelArgs{
					UpdaterID:          uuid.Must(uuid.NewV7()),
					ForcedNotification: optional.From(true),
				},
			},
			{
				ID: cABBC,
				Args: repository.UpdateChannelArgs{
					UpdaterID: uuid.Must(uuid.NewV7()),
					Parent:    optional.From(pubChannelRootUUID),
				},
			},
			{
				ID: cABCE,
				Args: repository.UpdateChannelArgs{
					UpdaterID: uuid.Must(uuid.NewV7()),
					Parent:    optional.From(cABCD),
				},
			},
			{
				ID: cEFGHI,
				Args: repository.UpdateChannelArgs{
					UpdaterID:          uuid.Must(uuid.NewV7()),
					Name:               optional.From("test"),
					Topic:              optional.From("test"),
					Visibility:         optional.From(false),
					ForcedNotification: optional.From(true),
					Parent:             optional.From(cE),
				},
			},
		}
		for i, c := range cases {
			c := c
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()
				ctrl := gomock.NewController(t)
				repo := mock_repository.NewMockChannelRepository(ctrl)
				cm := initCM(t, repo)

				ch, err := cm.PublicChannelTree().GetModel(c.ID)
				require.NoError(t, err)
				args := c.Args
				newChan := *ch
				newChan.UpdaterID = args.UpdaterID
				newChan.UpdatedAt = time.Now()

				// トピックが同じだった場合、トピックの引数自体を無効化
				if ch.Topic == args.Topic.V {
					args.Topic = optional.New("", false)
				}
				if args.Topic.Valid {
					repo.EXPECT().
						RecordChannelEvent(c.ID, model.ChannelEventTopicChanged, model.ChannelEventDetail{
							"userId": args.UpdaterID,
							"before": ch.Topic,
							"after":  args.Topic.V,
						}, gomock.Any()).
						Return(nil).
						Times(1)
					newChan.Topic = args.Topic.V
				}
				if args.Visibility.Valid && ch.IsVisible != args.Visibility.V {
					repo.EXPECT().
						RecordChannelEvent(c.ID, model.ChannelEventVisibilityChanged, model.ChannelEventDetail{
							"userId":     args.UpdaterID,
							"visibility": args.Visibility.V,
						}, gomock.Any()).
						Return(nil).
						Times(1)
					newChan.IsVisible = args.Visibility.V
				}
				if args.ForcedNotification.Valid && ch.IsForced != args.ForcedNotification.V {
					repo.EXPECT().
						RecordChannelEvent(c.ID, model.ChannelEventForcedNotificationChanged, model.ChannelEventDetail{
							"userId": args.UpdaterID,
							"force":  args.ForcedNotification.V,
						}, gomock.Any()).
						Return(nil).
						Times(1)
					newChan.IsForced = args.ForcedNotification.V
				}
				if args.Name.Valid {
					repo.EXPECT().
						RecordChannelEvent(c.ID, model.ChannelEventNameChanged, model.ChannelEventDetail{
							"userId": args.UpdaterID,
							"before": ch.Name,
							"after":  args.Name.V,
						}, gomock.Any()).
						Return(nil).
						Times(1)
					newChan.Name = args.Name.V
				}
				if args.Parent.Valid {
					repo.EXPECT().
						RecordChannelEvent(c.ID, model.ChannelEventParentChanged, model.ChannelEventDetail{
							"userId": args.UpdaterID,
							"before": ch.ParentID,
							"after":  args.Parent.V,
						}, gomock.Any()).
						Return(nil).
						Times(1)
					newChan.ParentID = args.Parent.V
				}

				repo.EXPECT().
					UpdateChannel(c.ID, args).
					Return(&newChan, nil).
					Times(1)

				err = cm.UpdateChannel(c.ID, args)
				cm.P.Wait()
				if assert.NoError(t, err) {
					v, err := cm.GetChannel(c.ID)
					require.NoError(t, err)
					sort.Slice(v.ChildrenID, func(i, j int) bool {
						return strings.Compare(v.ChildrenID[i].String(), v.ChildrenID[j].String()) > 0
					})
					sort.Slice(newChan.ChildrenID, func(i, j int) bool {
						return strings.Compare(newChan.ChildrenID[i].String(), newChan.ChildrenID[j].String()) > 0
					})
					assert.EqualValues(t, &newChan, v)
				}
			})
		}
	})
}

func TestManagerImpl_ChangeChannelSubscriptions(t *testing.T) {
	t.Parallel()

	uid1 := uuid.NewV3(uuid.Nil, "u1")
	uid2 := uuid.NewV3(uuid.Nil, "u2")

	t.Run("ErrInvalidChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.ChangeChannelSubscriptions(cNotFound, map[uuid.UUID]model.ChannelSubscribeLevel{}, false, uuid.Nil)
		assert.EqualError(t, err, ErrInvalidChannel.Error())
	})

	t.Run("ErrForcedNotification", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.ChangeChannelSubscriptions(cE, map[uuid.UUID]model.ChannelSubscribeLevel{}, false, uuid.Nil)
		assert.EqualError(t, err, ErrForcedNotification.Error())
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			ChangeChannelSubscription(gomock.Any(), gomock.Any()).
			Return(nil, nil, mockErr).
			AnyTimes()

		err := cm.ChangeChannelSubscriptions(cA, map[uuid.UUID]model.ChannelSubscribeLevel{}, false, uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			ID            uuid.UUID
			Subscriptions map[uuid.UUID]model.ChannelSubscribeLevel
		}{
			{ID: cAB, Subscriptions: map[uuid.UUID]model.ChannelSubscribeLevel{uid1: model.ChannelSubscribeLevelMarkAndNotify, uid2: model.ChannelSubscribeLevelNone}},
			{ID: cABC, Subscriptions: map[uuid.UUID]model.ChannelSubscribeLevel{uid1: model.ChannelSubscribeLevelMark, uid2: model.ChannelSubscribeLevelMarkAndNotify}},
		}
		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				repo := mock_repository.NewMockChannelRepository(ctrl)
				cm := initCM(t, repo)

				on := make([]uuid.UUID, 0)
				off := make([]uuid.UUID, 0)
				for u, level := range c.Subscriptions {
					switch level {
					case model.ChannelSubscribeLevelMarkAndNotify:
						on = append(on, u)
					case model.ChannelSubscribeLevelNone:
						off = append(off, u)
					}
				}

				repo.EXPECT().
					ChangeChannelSubscription(c.ID, gomock.Any()).
					Return(on, off, nil).
					AnyTimes()
				repo.EXPECT().
					RecordChannelEvent(c.ID, model.ChannelEventSubscribersChanged, model.ChannelEventDetail{
						"userId": uid1,
						"on":     on,
						"off":    off,
					}, gomock.Any()).
					Return(nil).
					Times(1)

				err := cm.ChangeChannelSubscriptions(c.ID, c.Subscriptions, false, uid1)
				cm.P.Wait()
				assert.NoError(t, err)
			})
		}
	})
}

func TestManagerImpl_ArchiveChannel(t *testing.T) {
	t.Parallel()

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetChannel(gomock.Any()).
			Return(nil, repository.ErrNotFound).
			AnyTimes()

		err := cm.ArchiveChannel(cNotFound, uuid.Nil)
		assert.EqualError(t, err, ErrChannelNotFound.Error())
	})

	t.Run("ErrInvalidChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		dm1 := &model.Channel{
			ID:        uuid.NewV3(uuid.Nil, "c 1-1"),
			Name:      "a",
			ParentID:  dmChannelRootUUID,
			IsForced:  false,
			IsPublic:  false,
			IsVisible: true,
		}

		repo.EXPECT().
			GetChannel(dm1.ID).
			Return(dm1, nil).
			AnyTimes()

		err := cm.ArchiveChannel(dm1.ID, uuid.Nil)
		assert.EqualError(t, err, ErrInvalidChannel.Error())
	})

	t.Run("noop (already archived)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.ArchiveChannel(cABBC, uuid.Nil)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		expects := []*model.Channel{
			{ID: cA, Name: "a", ParentID: uuid.Nil, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cAB, Name: "b", ParentID: cA, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cABC, Name: "c", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cABCD, Name: "d", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cABCE, Name: "e", ParentID: cABC, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cABF, Name: "f", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cABFA, Name: "a", ParentID: cABF, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
			{ID: cAD, Name: "d", ParentID: cA, Topic: "", IsForced: false, IsPublic: true, IsVisible: false},
		}
		repo.EXPECT().
			ArchiveChannels(gomock.Len(len(expects))).
			Return(expects, nil).
			Times(1)
		repo.EXPECT().
			RecordChannelEvent(gomock.Any(), model.ChannelEventVisibilityChanged, gomock.Any(), gomock.Any()).
			Return(nil).
			Times(len(expects))

		err := cm.ArchiveChannel(cA, uuid.Nil)
		cm.P.Wait()
		if assert.NoError(t, err) {
			for _, expect := range expects {
				assert.True(t, cm.PublicChannelTree().IsArchivedChannel(expect.ID))
			}
			assert.False(t, cm.PublicChannelTree().IsArchivedChannel(cE))
		}
	})
}

func TestManagerImpl_UnarchiveChannel(t *testing.T) {
	t.Parallel()

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetChannel(gomock.Any()).
			Return(nil, repository.ErrNotFound).
			AnyTimes()

		err := cm.UnarchiveChannel(cNotFound, uuid.Nil)
		assert.EqualError(t, err, ErrChannelNotFound.Error())
	})

	t.Run("noop (already not archived)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UnarchiveChannel(cA, uuid.Nil)
		assert.NoError(t, err)
	})

	t.Run("ErrInvalidParentChannel", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		err := cm.UnarchiveChannel(cABBC, uuid.Nil)
		assert.EqualError(t, err, ErrInvalidParentChannel.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			UpdateChannel(cABB, repository.UpdateChannelArgs{Visibility: optional.From(true)}).
			Return(&model.Channel{ID: cABB, Name: "b", ParentID: cAB, Topic: "", IsForced: false, IsPublic: true, IsVisible: true}, nil).
			Times(1)
		repo.EXPECT().
			RecordChannelEvent(gomock.Any(), model.ChannelEventVisibilityChanged, gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		err := cm.UnarchiveChannel(cABB, uuid.Nil)
		cm.P.Wait()
		if assert.NoError(t, err) {
			assert.False(t, cm.PublicChannelTree().IsArchivedChannel(cABB))
			assert.True(t, cm.PublicChannelTree().IsArchivedChannel(cABBC))
		}
	})
}

func TestManagerImpl_GetDMChannel(t *testing.T) {
	t.Parallel()

	t.Run("ErrChannelNotFound", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		_, err := cm.GetDMChannel(uuid.Nil, uuid.Nil)
		assert.EqualError(t, err, ErrChannelNotFound.Error())
		_, err = cm.GetDMChannel(uuid.Must(uuid.NewV4()), uuid.Nil)
		assert.EqualError(t, err, ErrChannelNotFound.Error())
		_, err = cm.GetDMChannel(uuid.Nil, uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, ErrChannelNotFound.Error())
	})

	t.Run("repository error1", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetDirectMessageChannel(gomock.Any(), gomock.Any()).
			Return(nil, mockErr).
			AnyTimes()

		_, err := cm.GetDMChannel(uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("repository error2", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetDirectMessageChannel(gomock.Any(), gomock.Any()).
			Return(nil, repository.ErrNotFound).
			Times(1)
		repo.EXPECT().
			CreateChannel(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, mockErr).
			Times(1)

		_, err := cm.GetDMChannel(uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})

	t.Run("success (found)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		uid1 := uuid.NewV3(uuid.Nil, "u1")
		uid2 := uuid.NewV3(uuid.Nil, "u2")
		dm1 := &model.Channel{
			ID:        uuid.NewV3(uuid.Nil, "c 1-1"),
			Name:      "a",
			ParentID:  dmChannelRootUUID,
			IsForced:  false,
			IsPublic:  false,
			IsVisible: true,
		}
		repo.EXPECT().
			GetDirectMessageChannel(uid1, uid2).
			Return(dm1, nil).
			Times(1)

		ch, err := cm.GetDMChannel(uid1, uid2)
		if assert.NoError(t, err) {
			assert.EqualValues(t, dm1, ch)
		}
	})

	t.Run("succes (create)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		uid1 := uuid.NewV3(uuid.Nil, "u1")
		uid2 := uuid.NewV3(uuid.Nil, "u2")
		repo.EXPECT().
			GetDirectMessageChannel(uid1, uid2).
			Return(nil, repository.ErrNotFound).
			Times(1)

		repo.EXPECT().
			CreateChannel(gomock.Any(), set.UUIDSetFromArray([]uuid.UUID{uid1, uid2}), true).
			Return(&model.Channel{
				ID:        uuid.NewV3(uuid.Nil, "c 1-1"),
				Name:      "dm_" + random.AlphaNumeric(17),
				ParentID:  dmChannelRootUUID,
				IsForced:  false,
				IsPublic:  false,
				IsVisible: true,
			}, nil).
			Times(1)

		_, err := cm.GetDMChannel(uid1, uid2)
		assert.NoError(t, err)
	})
}

func TestManagerImpl_GetDMChannelMembers(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		id := uuid.Must(uuid.NewV4())
		expected := []uuid.UUID{cA, cE}
		repo.EXPECT().
			GetPrivateChannelMemberIDs(id).
			Return(expected, nil).
			Times(1)

		repo.EXPECT().
			GetPrivateChannelMemberIDs(cNotFound).
			Return([]uuid.UUID{}, nil).
			Times(1)

		if ids, err := cm.GetDMChannelMembers(id); assert.NoError(t, err) {
			assert.ElementsMatch(t, expected, ids)
		}

		if ids, err := cm.GetDMChannelMembers(cNotFound); assert.NoError(t, err) {
			assert.Empty(t, ids)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetPrivateChannelMemberIDs(gomock.Any()).
			Return(nil, mockErr).
			Times(1)

		_, err := cm.GetDMChannelMembers(uuid.Nil)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_GetDMChannelMapping(t *testing.T) {
	t.Parallel()

	uid1 := uuid.NewV3(uuid.Nil, "u1")
	uid2 := uuid.NewV3(uuid.Nil, "u2")
	uid3 := uuid.NewV3(uuid.Nil, "u3")
	uid4 := uuid.NewV3(uuid.Nil, "u4")
	cid1 := uuid.NewV3(uuid.Nil, "c 1-1")
	cid2 := uuid.NewV3(uuid.Nil, "c 1-2")
	cid3 := uuid.NewV3(uuid.Nil, "c 1-3")

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetDirectMessageChannelMapping(uid1).
			Return([]*model.DMChannelMapping{
				{ChannelID: cid1, User1: uid1, User2: uid1},
				{ChannelID: cid2, User1: uid1, User2: uid2},
				{ChannelID: cid3, User1: uid1, User2: uid3},
			}, nil).
			Times(1)

		repo.EXPECT().
			GetDirectMessageChannelMapping(uid2).
			Return([]*model.DMChannelMapping{
				{ChannelID: cid2, User1: uid1, User2: uid2},
			}, nil).
			Times(1)

		repo.EXPECT().
			GetDirectMessageChannelMapping(uid4).
			Return([]*model.DMChannelMapping{}, nil).
			Times(1)

		if m, err := cm.GetDMChannelMapping(uid1); assert.NoError(t, err) {
			assert.EqualValues(t, m, map[uuid.UUID]uuid.UUID{
				cid1: uid1,
				cid2: uid2,
				cid3: uid3,
			})
		}

		if m, err := cm.GetDMChannelMapping(uid2); assert.NoError(t, err) {
			assert.EqualValues(t, m, map[uuid.UUID]uuid.UUID{
				cid2: uid1,
			})
		}

		if m, err := cm.GetDMChannelMapping(uid4); assert.NoError(t, err) {
			assert.EqualValues(t, m, map[uuid.UUID]uuid.UUID{})
		}
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetDirectMessageChannelMapping(uid1).
			Return(nil, mockErr).
			Times(1)

		_, err := cm.GetDMChannelMapping(uid1)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_IsChannelAccessibleToUser(t *testing.T) {
	t.Parallel()

	uid1 := uuid.NewV3(uuid.Nil, "u1")
	uid2 := uuid.NewV3(uuid.Nil, "u2")
	uid3 := uuid.NewV3(uuid.Nil, "u3")
	cid1 := uuid.NewV3(uuid.Nil, "c 1-1")
	cid2 := uuid.NewV3(uuid.Nil, "c 1-2")
	cid3 := uuid.NewV3(uuid.Nil, "c 1-3")

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		repo.EXPECT().
			GetPrivateChannelMemberIDs(cNotFound).
			Return([]uuid.UUID{}, nil).
			AnyTimes()

		repo.EXPECT().
			GetPrivateChannelMemberIDs(cid1).
			Return([]uuid.UUID{uid1}, nil).
			AnyTimes()

		repo.EXPECT().
			GetPrivateChannelMemberIDs(cid2).
			Return([]uuid.UUID{uid1, uid2}, nil).
			AnyTimes()

		repo.EXPECT().
			GetPrivateChannelMemberIDs(cid3).
			Return([]uuid.UUID{uid1, uid3}, nil).
			AnyTimes()

		cases := []struct {
			User    uuid.UUID
			Channel uuid.UUID
			OK      bool
		}{
			{User: uid1, Channel: cA, OK: true},
			{User: uid1, Channel: cid1, OK: true},
			{User: uid1, Channel: cid2, OK: true},
			{User: uid2, Channel: cABB, OK: true},
			{User: uid2, Channel: cid2, OK: true},
			{User: uid2, Channel: cid3, OK: false},
			{User: uid3, Channel: cid1, OK: false},
			{User: uid1, Channel: cNotFound, OK: false},
		}
		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				ok, err := cm.IsChannelAccessibleToUser(c.User, c.Channel)
				if assert.NoError(t, err) {
					assert.Equal(t, c.OK, ok)
				}
			})
		}
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		repo := mock_repository.NewMockChannelRepository(ctrl)
		cm := initCM(t, repo)

		mockErr := errors.New("mock error")
		repo.EXPECT().
			GetPrivateChannelMemberIDs(gomock.Any()).
			Return(nil, mockErr).
			Times(1)

		_, err := cm.IsChannelAccessibleToUser(uid1, cid1)
		if assert.Error(t, err) {
			assert.Equal(t, mockErr, errors.Unwrap(err))
		}
	})
}

func TestManagerImpl_IsPublicChannel(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := mock_repository.NewMockChannelRepository(ctrl)
	cm := initCM(t, repo)

	assert.True(t, cm.IsPublicChannel(cA))
	assert.False(t, cm.IsPublicChannel(cNotFound))
}
