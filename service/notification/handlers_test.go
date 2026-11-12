package notification

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
)

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
