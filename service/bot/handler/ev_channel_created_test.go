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

func TestChannelCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.ChannelCreated.String()}),
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
		tree.EXPECT().GetChannelPath(ch.ID).Return(ch.Name).AnyTimes()

		registerBot(t, handlerCtx, b)
		registerUser(repo, u)

		et := time.Now()

		expectMulticast(handlerCtx, event.ChannelCreated, payload.MakeChannelCreated(et, ch, ch.Name, u), []*model.Bot{b})
		assert.NoError(t, ChannelCreated(handlerCtx, et, intevent.ChannelCreated, hub.Fields{
			"channel": ch,
		}))
	})

	t.Run("success (private channel)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, _, _ := setup(t, ctrl)

		assert.NoError(t, ChannelCreated(handlerCtx, time.Now(), intevent.ChannelCreated, hub.Fields{
			"channel": &model.Channel{
				ID:        uuid.NewV3(uuid.Nil, "pc"),
				Name:      "private channel",
				IsPublic:  false,
				CreatorID: u.ID,
			},
		}))
	})

}
