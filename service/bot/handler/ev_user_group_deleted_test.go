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

func TestUserGroupDeleted(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.UserGroupDeleted.String()}),
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
		group := model.UserGroup{
			ID:          uuid.NewV3(uuid.Nil, "g"),
			Name:        "new_group",
			Description: "new_group_description",
			Type:        "new_group_type",
		}
		group.Admins = append(group.Admins, &model.UserGroupAdmin{GroupID: group.ID, UserID: user.ID})
		group.Members = append(group.Members, &model.UserGroupMember{GroupID: group.ID, UserID: user.ID})
		et := time.Now()

		expectMulticast(handlerCtx, event.UserGroupDeleted, payload.MakeUserGroupDeleted(et, group), []*model.Bot{b})
		assert.NoError(t, UserGroupDeleted(handlerCtx, et, intevent.UserGroupDeleted, hub.Fields{
			"group_id": group.ID,
			"group":    group,
		}))
	})

	t.Run("not subscribe USER_GROUP_DELETED", func(t *testing.T) {
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

		user := &model.User{
			ID:     uuid.NewV3(uuid.Nil, "u"),
			Name:   "new_user",
			Status: model.UserAccountStatusActive,
			Bot:    false,
		}
		group := model.UserGroup{
			ID:          uuid.NewV3(uuid.Nil, "g"),
			Name:        "new_group",
			Description: "new_group_description",
			Type:        "new_group_type",
		}
		group.Admins = append(group.Admins, &model.UserGroupAdmin{GroupID: group.ID, UserID: user.ID})
		group.Members = append(group.Members, &model.UserGroupMember{GroupID: group.ID, UserID: user.ID})
		et := time.Now()

		expectMulticast(handlerCtx, event.UserGroupDeleted, payload.MakeUserGroupDeleted(et, group), []*model.Bot{b})
		assert.NoError(t, UserGroupDeleted(handlerCtx, et, intevent.UserGroupDeleted, hub.Fields{
			"group_id": group.ID,
			"group":    group,
		}))
	})
}
