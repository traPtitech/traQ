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

func TestBotJoined(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypes{},
		State:           model.BotActive,
	}
	u := &model.User{
		ID:   uuid.NewV3(uuid.Nil, "u"),
		Name: "testman",
	}
	ch := &model.Channel{
		ID:        uuid.NewV3(uuid.Nil, "c"),
		Name:      "test",
		IsPublic:  true,
		CreatorID: u.ID,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)

		tree := mock_channel.NewMockTree(ctrl)
		cm.EXPECT().PublicChannelTree().Return(tree).AnyTimes()
		tree.EXPECT().GetChannelPath(ch.ID).Return("test").AnyTimes()

		registerBot(t, handlerCtx, b)
		registerChannel(cm, ch)
		registerUser(repo, u)

		et := time.Now()

		expectUnicast(handlerCtx, event.Joined, payload.MakeJoined(et, ch, ch.Name, u), b)
		assert.NoError(t, BotJoined(handlerCtx, et, intevent.BotJoined, hub.Fields{
			"bot_id":     b.ID,
			"channel_id": ch.ID,
		}))
	})
}
