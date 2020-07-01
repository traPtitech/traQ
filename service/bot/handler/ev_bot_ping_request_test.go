package handler

import (
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	intevent "github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/mock_event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"testing"
	"time"
)

func TestBotPingRequest(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypes{},
		State:           model.BotActive,
	}

	t.Run("activation succeeded", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, _, repo := setup(t, ctrl)
		d := mock_event.NewMockDispatcher(ctrl)
		handlerCtx.EXPECT().D().Return(d).AnyTimes()

		et := time.Now()

		buf, err := jsoniter.ConfigFastest.Marshal(payload.MakePing(et))
		require.NoError(t, err)

		repo.MockBotRepository.EXPECT().ChangeBotState(b.ID, model.BotActive).Return(nil).Times(1)
		d.EXPECT().Send(b, event.Ping, buf).Return(true).Times(1)

		assert.NoError(t, BotPingRequest(handlerCtx, et, intevent.BotPingRequest, hub.Fields{
			"bot": b,
		}))
	})

	t.Run("activation failed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		handlerCtx, _, repo := setup(t, ctrl)
		d := mock_event.NewMockDispatcher(ctrl)
		handlerCtx.EXPECT().D().Return(d).AnyTimes()

		et := time.Now()

		buf, err := jsoniter.ConfigFastest.Marshal(payload.MakePing(et))
		require.NoError(t, err)

		repo.MockBotRepository.EXPECT().ChangeBotState(b.ID, model.BotPaused).Return(nil).Times(1)
		d.EXPECT().Send(b, event.Ping, buf).Return(false).Times(1)

		assert.NoError(t, BotPingRequest(handlerCtx, et, intevent.BotPingRequest, hub.Fields{
			"bot": b,
		}))
	})
}
