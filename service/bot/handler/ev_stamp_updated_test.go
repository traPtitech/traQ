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
)

func TestStampUpdated(t *testing.T) {
	t.Parallel()

	user := &model.User{
		ID:   uuid.NewV3(uuid.Nil, "u"),
		Name: "user",
	}
	b := &model.Bot{
		ID:              uuid.NewV3(uuid.Nil, "b"),
		BotUserID:       uuid.NewV3(uuid.Nil, "bu"),
		SubscribeEvents: model.BotEventTypesFromArray([]string{event.StampUpdated.String()}),
		State:           model.BotActive,
	}
	testErr := errors.New("test error")

	for _, tt := range []struct {
		name         string
		systemStamp  bool
		noBots       bool
		getBotsErr   error
		getUserErr   error
		multicastErr error
		wantErr      string
	}{
		{name: "success"},
		{name: "system stamp", systemStamp: true},
		{name: "no subscribers", noBots: true},
		{name: "get bots error", getBotsErr: testErr, wantErr: "failed to GetBots: test error"},
		{name: "get user error", getUserErr: testErr, wantErr: "failed to GetUser: test error"},
		{name: "multicast error", multicastErr: testErr, wantErr: "failed to multicast: test error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			handlerCtx, _, repo := setup(t, ctrl)
			stamp := &model.Stamp{
				ID:        uuid.NewV3(uuid.Nil, "s"),
				Name:      "updated_stamp",
				CreatorID: user.ID,
				FileID:    uuid.NewV3(uuid.Nil, "updated_file"),
			}
			if tt.systemStamp {
				stamp.CreatorID = uuid.Nil
			}
			et := time.Now()
			bots := []*model.Bot{b}
			if tt.noBots {
				bots = nil
			}
			handlerCtx.EXPECT().GetBots(event.StampUpdated).Return(bots, tt.getBotsErr)

			if !tt.noBots && tt.getBotsErr == nil {
				var creator payload.User
				if !tt.systemStamp {
					repo.MockUserRepository.EXPECT().GetUser(gomock.Any(), user.ID, false).Return(user, tt.getUserErr)
					creator = payload.MakeUser(user)
				}
				if tt.getUserErr == nil {
					handlerCtx.EXPECT().Multicast(event.StampUpdated, &payload.StampUpdated{
						Base:    payload.Base{EventTime: et},
						ID:      stamp.ID,
						Name:    stamp.Name,
						FileID:  stamp.FileID,
						Creator: creator,
					}, bots).Return(tt.multicastErr)
				}
			}

			err := StampUpdated(handlerCtx, et, intevent.StampUpdated, hub.Fields{"stamp": stamp})
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				assert.ErrorIs(t, err, testErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
