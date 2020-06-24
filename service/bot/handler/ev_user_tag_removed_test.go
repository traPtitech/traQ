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
	"testing"
	"time"
)

func TestUserTagRemoved(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.TagRemoved.String()}),
		State:           model.BotActive,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, _, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		tag := &model.Tag{
			ID:   uuid.NewV3(uuid.Nil, "t"),
			Name: "test",
		}
		registerTag(repo, tag)
		et := time.Now()

		expectUnicast(handlerCtx, event.TagRemoved, payload.MakeTagRemoved(et, tag), b)
		assert.NoError(t, UserTagRemoved(handlerCtx, et, intevent.UserTagRemoved, hub.Fields{
			"user_id": b.BotUserID,
			"tag_id":  tag.ID,
		}))
	})

	t.Run("not subscribe TAG_REMOVED", func(t *testing.T) {
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

		tag := &model.Tag{
			ID:   uuid.NewV3(uuid.Nil, "t"),
			Name: "test",
		}
		et := time.Now()

		assert.NoError(t, UserTagRemoved(handlerCtx, et, intevent.UserTagRemoved, hub.Fields{
			"user_id": b.BotUserID,
			"tag_id":  tag.ID,
		}))
	})
}
