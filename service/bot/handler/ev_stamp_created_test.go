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

func TestStampCreated(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.StampCreated.String()}),
		State:           model.BotActive,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, _, repo := setup(t, ctrl)
		registerBot(t, handlerCtx, b)

		user := &model.User{
			ID:   uuid.NewV3(uuid.Nil, "u"),
			Name: "user",
		}
		registerUser(repo, user)

		stamp := &model.Stamp{
			ID:        uuid.NewV3(uuid.Nil, "s"),
			Name:      "test",
			CreatorID: user.ID,
			FileID:    uuid.NewV3(uuid.Nil, "f"),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		et := time.Now()

		expectMulticast(handlerCtx, event.StampCreated, payload.MakeStampCreated(et, stamp, user), []*model.Bot{b})
		assert.NoError(t, StampCreated(handlerCtx, et, intevent.StampCreated, hub.Fields{
			"stamp": stamp,
		}))
	})
}
