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

func TestUserGroupAdminRemoved(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.UserGroupAdminRemoved.String()}),
		State:           model.BotActive,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx := mock_handler.NewMockContext(ctrl)
		registerBot(t, handlerCtx, b)

		userID := uuid.NewV3(uuid.Nil, "u")
		groupID := uuid.NewV3(uuid.Nil, "g")
		et := time.Now()

		expectMulticast(handlerCtx, event.UserGroupAdminRemoved, payload.MakeUserGroupAdminRemoved(et, groupID, userID), []*model.Bot{b})
		assert.NoError(t, UserGroupAdminRemoved(handlerCtx, et, intevent.UserGroupAdminRemoved, hub.Fields{
			"group_id": groupID,
			"user_id":  userID,
		}))
	})

	t.Run("not subscribe USER_GROUP_ADMIN_REMOVED", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx := mock_handler.NewMockContext(ctrl)
		b2 := &model.Bot{
			ID:              uuid.NewV3(uuid.Nil, "b2"),
			BotUserID:       uuid.NewV3(uuid.Nil, "bu2"),
			SubscribeEvents: model.BotEventTypesFromArray([]string{event.MessageCreated.String()}),
			State:           model.BotActive,
		}
		registerBot(t, handlerCtx, b)
		registerBot(t, handlerCtx, b2)

		userID := uuid.NewV3(uuid.Nil, "u")
		groupID := uuid.NewV3(uuid.Nil, "g")
		et := time.Now()

		expectMulticast(handlerCtx, event.UserGroupAdminRemoved, payload.MakeUserGroupAdminRemoved(et, groupID, userID), []*model.Bot{b})
		assert.NoError(t, UserGroupAdminRemoved(handlerCtx, et, intevent.UserGroupAdminRemoved, hub.Fields{
			"group_id": groupID,
			"user_id":  userID,
		}))
	})
}
