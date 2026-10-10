package handler

import (
	"errors"
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

func TestStampDeleted(t *testing.T) {
	t.Parallel()

	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.StampDeleted.String()}),
		State:           model.BotActive,
	}
	testErr := errors.New("test error")

	for _, tt := range []struct {
		name         string
		noBots       bool
		getBotsErr   error
		multicastErr error
		wantErr      string
	}{
		{name: "success"},
		{name: "no subscribers", noBots: true},
		{name: "get bots error", getBotsErr: testErr, wantErr: "failed to GetBots: test error"},
		{name: "multicast error", multicastErr: testErr, wantErr: "failed to multicast: test error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			handlerCtx := mock_handler.NewMockContext(ctrl)
			stampID := uuid.NewV3(uuid.Nil, "s")
			et := time.Now()
			bots := []*model.Bot{b}
			if tt.noBots {
				bots = nil
			}
			handlerCtx.EXPECT().GetBots(event.StampDeleted).Return(bots, tt.getBotsErr)
			if !tt.noBots && tt.getBotsErr == nil {
				handlerCtx.EXPECT().Multicast(event.StampDeleted, &payload.StampDeleted{
					Base: payload.Base{EventTime: et},
					ID:   stampID,
				}, bots).Return(tt.multicastErr)
			}

			err := StampDeleted(handlerCtx, et, intevent.StampDeleted, hub.Fields{"stamp_id": stampID})
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				assert.ErrorIs(t, err, testErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
