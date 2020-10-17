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
	"github.com/traPtitech/traQ/service/bot/handler/mock_handler"
	"github.com/traPtitech/traQ/service/message"
	"testing"
	"time"
)

func TestMessageStampsUpdated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.BotMessageStampsUpdated.String()}),
		State:           model.BotActive,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx := mock_handler.NewMockContext(ctrl)

		registerBot(t, handlerCtx, b)

		m := &messageImpl{
			ID:     uuid.NewV3(uuid.Nil, "m"),
			UID:    uuid.NewV3(uuid.Nil, "bu"),
			Stamps: []model.MessageStamp{},
		}
		et := time.Now()

		expectUnicast(handlerCtx, event.BotMessageStampsUpdated, payload.MakeBotMessageStampsUpdated(et, m.ID, m.Stamps), b)
		assert.NoError(t, MessageStampsUpdated(handlerCtx, et, intevent.MessageStampsUpdated, hub.Fields{
			"message":    m,
			"message_id": m.ID,
		}))
	})

	t.Run("not subscribe BotMessageStampsUpdated", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx := mock_handler.NewMockContext(ctrl)

		b := &model.Bot{
			ID:              uuid.NewV3(uuid.Nil, "b"),
			BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
			SubscribeEvents: model.BotEventTypesFromArray([]string{event.MessageCreated.String()}),
			State:           model.BotActive,
		}
		registerBot(t, handlerCtx, b)

		m := &messageImpl{
			ID:     uuid.NewV3(uuid.Nil, "m"),
			UID:    b.BotUserID,
			Stamps: []model.MessageStamp{},
		}
		et := time.Now()

		assert.NoError(t, MessageStampsUpdated(handlerCtx, et, intevent.MessageStampsUpdated, hub.Fields{
			"message":    m,
			"message_id": m.ID,
		}))
	})
}

type messageImpl struct {
	message.Message
	ID     uuid.UUID
	UID    uuid.UUID
	Stamps []model.MessageStamp
}

func (m *messageImpl) GetID() uuid.UUID {
	return m.ID
}

func (m *messageImpl) GetUserID() uuid.UUID {
	return m.UID
}

func (m *messageImpl) GetStamps() []model.MessageStamp {
	return m.Stamps
}
