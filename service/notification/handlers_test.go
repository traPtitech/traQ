package notification

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/testutils"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
)

type messageViewerTestSession struct {
	ws.Session
	channelID uuid.UUID
}

func (s messageViewerTestSession) ViewState() (uuid.UUID, viewer.State) {
	return s.channelID, viewer.StateNone
}

func TestMessageChannelViewers(t *testing.T) {
	t.Parallel()

	channelID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())
	otherID := uuid.Must(uuid.NewV7())

	t.Run("public channel viewers only", func(t *testing.T) {
		t.Parallel()

		target := messageChannelViewers(&model.Channel{
			ID:       channelID,
			ParentID: parentID,
			Type:     model.ChannelTypePublic,
		})
		assert.True(t, target(messageViewerTestSession{channelID: channelID}))
		assert.False(t, target(messageViewerTestSession{channelID: parentID}))
		assert.False(t, target(messageViewerTestSession{channelID: otherID}))
		assert.False(t, target(messageViewerTestSession{channelID: uuid.Nil}))
	})

	t.Run("thread and parent channel viewers", func(t *testing.T) {
		t.Parallel()

		target := messageChannelViewers(&model.Channel{
			ID:       channelID,
			ParentID: parentID,
			Type:     model.ChannelTypeThread,
		})
		assert.True(t, target(messageViewerTestSession{channelID: channelID}))
		assert.True(t, target(messageViewerTestSession{channelID: parentID}))
		assert.False(t, target(messageViewerTestSession{channelID: otherID}))
		assert.False(t, target(messageViewerTestSession{channelID: uuid.Nil}))
	})

	t.Run("DM channel viewers only", func(t *testing.T) {
		t.Parallel()

		target := messageChannelViewers(&model.Channel{
			ID:       channelID,
			ParentID: parentID,
			Type:     model.ChannelTypeDM,
		})
		assert.True(t, target(messageViewerTestSession{channelID: channelID}))
		assert.False(t, target(messageViewerTestSession{channelID: parentID}))
		assert.False(t, target(messageViewerTestSession{channelID: otherID}))
		assert.False(t, target(messageViewerTestSession{channelID: uuid.Nil}))
	})
}

type subscriptionTestSession struct {
	ws.Session
	userID    uuid.UUID
	channelID uuid.UUID
}

func (s subscriptionTestSession) UserID() uuid.UUID { return s.userID }
func (s subscriptionTestSession) ViewState() (uuid.UUID, viewer.State) {
	return s.channelID, viewer.StateNone
}

func TestSubscriptionChangeTarget(t *testing.T) {
	t.Parallel()

	parent := uuid.Must(uuid.NewV4())
	thread := uuid.Must(uuid.NewV4())
	otherChannel := uuid.Must(uuid.NewV4())
	changedUser := uuid.Must(uuid.NewV4())
	otherUser := uuid.Must(uuid.NewV4())

	checkTarget := func(t *testing.T, includeThreads bool, userID, viewedChannel uuid.UUID, want bool) {
		t.Helper()

		cm := mock_channel.NewMockManager(gomock.NewController(t))

		if includeThreads {
			cm.EXPECT().GetThreadChannels(gomock.Any(), repository.GetThreadChannelQuery{
				ChannelID: parent, Archived: true, Limit: -1,
			}).Return([]*model.Thread{{ChannelID: thread}}, nil)
		}

		ns := &Service{cm: cm, logger: zap.NewNop()}
		target := subscriptionChangeTarget(ns, parent, []uuid.UUID{changedUser}, includeThreads)

		assert.Equal(t, want, target(subscriptionTestSession{userID: userID, channelID: viewedChannel}))
	}

	t.Run("Parent viewer receives parent setting change", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, true, otherUser, parent, true)
	})

	t.Run("Child thread viewer receives parent setting change", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, true, otherUser, thread, true)
	})

	t.Run("Unrelated channel viewer does not receive parent setting change", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, true, otherUser, otherChannel, false)
	})

	t.Run("Changed user receives event while viewing another channel", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, true, changedUser, otherChannel, true)
	})

	t.Run("Thread setting change does not include child thread viewers", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, false, otherUser, thread, false)
	})

	t.Run("Thread setting change includes its own viewers", func(t *testing.T) {
		t.Parallel()
		checkTarget(t, false, otherUser, parent, true)
	})

	t.Run("Thread lookup failure preserves original recipients", func(t *testing.T) {
		t.Parallel()
		cm := mock_channel.NewMockManager(gomock.NewController(t))

		cm.EXPECT().GetThreadChannels(gomock.Any(), gomock.Any()).Return(nil, errors.New("lookup failed"))

		ns := &Service{cm: cm, logger: zap.NewNop()}
		target := subscriptionChangeTarget(ns, parent, []uuid.UUID{changedUser}, true)

		assert.True(t, target(subscriptionTestSession{userID: otherUser, channelID: parent}))
		assert.True(t, target(subscriptionTestSession{userID: changedUser, channelID: otherChannel}))
		assert.False(t, target(subscriptionTestSession{userID: otherUser, channelID: otherChannel}))
	})
}

type notificationTestRepository struct {
	*mock_repository.MockChannelRepository
	*mock_repository.MockUserRepository
	*mock_repository.MockMessageRepository
	testutils.EmptyTestRepository
}

type captureFCM struct {
	fcm.Client
	targets set.UUID
}

func (c *captureFCM) Send(targets set.UUID, payload *fcm.Payload, withUnreadCount bool) {
	c.targets = targets
}

func TestMessageCreatedThread(t *testing.T) {
	t.Parallel()
	author := uuid.NewV3(uuid.Nil, "author")
	subscriber := uuid.NewV3(uuid.Nil, "subscriber")

	checkNotification := func(t *testing.T, userID uuid.UUID, notify, viewing, notifyExpected, markExpected bool) {
		t.Helper()
		ctrl := gomock.NewController(t)
		repo := &notificationTestRepository{
			MockChannelRepository: mock_repository.NewMockChannelRepository(ctrl),
			MockUserRepository:    mock_repository.NewMockUserRepository(ctrl),
			MockMessageRepository: mock_repository.NewMockMessageRepository(ctrl),
		}
		cm := mock_channel.NewMockManager(ctrl)
		tree := mock_channel.NewMockTree(ctrl)
		parent := uuid.NewV3(uuid.Nil, "parent")
		thread := uuid.NewV3(uuid.Nil, "thread")
		mid := uuid.NewV3(uuid.Nil, "mid")
		m := &model.Message{ID: mid, ChannelID: thread, UserID: author}
		cm.EXPECT().PublicChannelTree(gomock.Any()).Return(tree)
		tree.EXPECT().IsForceChannel(thread).Return(false)
		tree.EXPECT().GetChannelPath(parent).Return("general")
		repo.MockChannelRepository.EXPECT().GetChannel(gomock.Any(), thread).Return(&model.Channel{
			ID: thread, ParentID: parent, Name: "discussion", Type: model.ChannelTypeThread,
		}, nil)
		repo.MockUserRepository.EXPECT().GetUser(gomock.Any(), author, false).Return(&model.User{ID: author, Name: "author", Status: model.UserAccountStatusActive}, nil)
		q := repository.UsersQuery{}.Active().NotBot()
		notifyUsers := []uuid.UUID{}
		if notify {
			notifyUsers = append(notifyUsers, userID)
		}
		repo.MockUserRepository.EXPECT().GetUserIDs(gomock.Any(), q.ThreadSubscriberAtNotifyLevelOf(thread)).Return(notifyUsers, nil)
		repo.MockUserRepository.EXPECT().GetUserIDs(gomock.Any(), q.ThreadSubscriberAtMarkLevelOf(thread)).Return([]uuid.UUID{userID}, nil)
		unreads := map[uuid.UUID]bool{}
		if markExpected {
			unreads[userID] = false
		}
		repo.MockMessageRepository.EXPECT().SetMessageUnreads(gomock.Any(), unreads, mid).Return(nil)
		h := hub.New()
		defer h.Close()
		vm := viewer.NewManager(h)
		if viewing {
			vm.SetViewer("viewing", "viewing", userID, thread, viewer.StateMonitoring)
		}
		push := &captureFCM{}
		ns := &Service{repo: repo, cm: cm, logger: zap.NewNop(), fcm: push, vm: vm, ws: ws.NewStreamer(h, vm, nil, zap.NewNop())}
		messageCreatedHandler(ns, hub.Message{Name: event.MessageCreated, Fields: hub.Fields{
			"message": m, "parse_result": &message.ParseResult{},
		}})
		if notifyExpected {
			assert.Equal(t, set.UUIDSetFromArray([]uuid.UUID{userID}), push.targets)
		} else {
			assert.Empty(t, push.targets)
		}
	}

	t.Run("Notify subscriber", func(t *testing.T) {
		t.Parallel()
		checkNotification(t, subscriber, true, false, true, true)
	})

	t.Run("Mark only subscriber", func(t *testing.T) {
		t.Parallel()
		checkNotification(t, subscriber, false, false, false, true)
	})

	t.Run("Exclude author", func(t *testing.T) {
		t.Parallel()
		checkNotification(t, author, true, false, false, false)
	})

	t.Run("Exclude viewer", func(t *testing.T) {
		t.Parallel()
		checkNotification(t, subscriber, true, true, false, false)
	})
}
