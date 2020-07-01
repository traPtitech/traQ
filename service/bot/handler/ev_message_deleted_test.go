package handler

import (
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"testing"
	"time"
)

func TestMessageDeleted(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:        uuid.NewV3(uuid.Nil, "b"),
		BotUserID: uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{
			event.MessageDeleted.String(),
			event.DirectMessageDeleted.String(),
		}),
		State: model.BotActive,
	}
	ch := &model.Channel{
		ID:       uuid.NewV3(uuid.Nil, "c"),
		Name:     "test",
		IsPublic: true,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, _ := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "u"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		registerChannel(cm, ch)
		et := time.Now()

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageDeleted).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		expectMulticast(handlerCtx, event.MessageDeleted, payload.MakeMessageDeleted(et, m), []*model.Bot{b})
		assert.NoError(t, MessageDeleted(handlerCtx, et, intevent.MessageDeleted, hub.Fields{
			"message": m,
		}))
	})

	t.Run("success (no sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, _ := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "bu"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		registerChannel(cm, ch)
		et := time.Now()

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageDeleted).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		assert.NoError(t, MessageDeleted(handlerCtx, et, intevent.MessageDeleted, hub.Fields{
			"message": m,
		}))
	})

	t.Run("success (dm)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)
		dmc, u := createDMChannel(handlerCtx, cm, repo, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    u.GetID(),
			ChannelID: dmc.ID,
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		et := time.Now()

		expectUnicast(handlerCtx, event.DirectMessageDeleted, payload.MakeDirectMessageDeleted(et, m), b)
		assert.NoError(t, MessageDeleted(handlerCtx, et, intevent.MessageDeleted, hub.Fields{
			"message_id": m.ID,
			"message":    m,
		}))
	})

	t.Run("success (dm, no sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)
		dmc, _ := createDMChannel(handlerCtx, cm, repo, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    b.BotUserID,
			ChannelID: dmc.ID,
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		et := time.Now()

		assert.NoError(t, MessageDeleted(handlerCtx, et, intevent.MessageDeleted, hub.Fields{
			"message_id": m.ID,
			"message":    m,
		}))
	})
}
