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
	"github.com/traPtitech/traQ/service/bot/handler/mock_handler"
)

func TestUserCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.UserCreated.String()}),
		State:           model.BotActive,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx := mock_handler.NewMockContext(ctrl)
		registerBot(t, handlerCtx, b)

		user := &model.User{
			ID:     uuid.NewV3(uuid.Nil, "u"),
			Name:   "new_user",
			Status: model.UserAccountStatusActive,
			Bot:    false,
		}
		et := time.Now()

		expectMulticast(handlerCtx, event.UserCreated, payload.MakeUserCreated(et, user), []*model.Bot{b})
		assert.NoError(t, UserCreated(handlerCtx, et, intevent.UserCreated, hub.Fields{
			"user_id": user.ID,
			"user":    user,
		}))
	})
}
