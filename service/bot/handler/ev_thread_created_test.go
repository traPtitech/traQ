package handler

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"

	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
)

func TestThreadCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.ThreadCreated.String()}),
		State:           model.BotActive,
	}
	u := &model.User{
		ID:   uuid.NewV3(uuid.Nil, "u"),
		Name: "testman",
	}
	ch := &model.Channel{
		ID:        uuid.NewV3(uuid.Nil, "c"),
		Name:      "test",
		Type:      model.ChannelTypePublic,
		CreatorID: u.ID,
	}
	th := &model.Channel{
		ID:        uuid.NewV3(uuid.Nil, "t"),
		ParentID:  ch.ID,
		Name:      "testThread",
		Type:      model.ChannelTypeThread,
		CreatorID: u.ID,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		tree := mock_channel.NewMockTree(ctrl)
		cm.EXPECT().PublicChannelTree(gomock.Any()).Return(tree).AnyTimes()
		tree.EXPECT().GetChannelPath(ch.ID).Return(ch.Name).AnyTimes()

		registerBot(t, handlerCtx, b)
		registerUser(repo, u)

		et := time.Now()

		expectMulticast(handlerCtx, event.ThreadCreated, payload.MakeThreadCreated(et, th, ch.Name, u), []*model.Bot{b})
		assert.NoError(t, ThreadCreated(handlerCtx, et, intevent.ThreadCreated, hub.Fields{
			"thread": th,
		}))
	})

}
