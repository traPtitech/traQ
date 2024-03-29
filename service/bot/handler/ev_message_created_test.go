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
	"github.com/traPtitech/traQ/utils/message"
)

func TestMessageCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:        uuid.NewV3(uuid.Nil, "b"),
		BotUserID: uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{
			event.MessageCreated.String(),
			event.DirectMessageCreated.String(),
		}),
		State: model.BotActive,
	}
	bu := &model.User{
		ID:     b.BotUserID,
		Name:   "bot",
		Status: model.UserAccountStatusActive,
		Bot:    true,
	}
	ch := &model.Channel{
		ID:       uuid.NewV3(uuid.Nil, "c"),
		Name:     "test",
		IsPublic: true,
	}

	t.Run("success (public message, sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "u"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		parsed := message.Parse(m.Text)
		mu := &model.User{
			ID:   m.UserID,
			Name: "testman",
		}
		registerUser(repo, mu)
		registerChannel(cm, ch)
		et := time.Now()

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageCreated).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		expectMulticast(handlerCtx, event.MessageCreated, payload.MakeMessageCreated(et, m, mu, parsed), []*model.Bot{b})
		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})

	t.Run("success (public message, no targets)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    uuid.NewV3(uuid.Nil, "bu"),
			ChannelID: uuid.NewV3(uuid.Nil, "c"),
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		mu := &model.User{
			ID:   m.UserID,
			Name: "BOT_TEST",
		}
		registerUser(repo, mu)
		registerChannel(cm, ch)

		handlerCtx.EXPECT().
			GetChannelBots(m.ChannelID, event.MessageCreated).
			Return([]*model.Bot{b}, nil).
			AnyTimes()

		assert.NoError(t, MessageCreated(handlerCtx, time.Now(), intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": message.Parse(m.Text),
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
		parsed := message.Parse(m.Text)
		et := time.Now()

		expectUnicast(handlerCtx, event.DirectMessageCreated, payload.MakeDirectMessageCreated(et, m, u, parsed), b)
		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})

	t.Run("success (dm, no sent)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, cm, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)
		registerUser(repo, bu)
		dmc, _ := createDMChannel(handlerCtx, cm, repo, b)

		m := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m"),
			UserID:    b.BotUserID,
			ChannelID: dmc.ID,
			Text:      "test message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		parsed := message.Parse(m.Text)
		et := time.Now()

		assert.NoError(t, MessageCreated(handlerCtx, et, intevent.MessageCreated, hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parsed,
		}))
	})
}
