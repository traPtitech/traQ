package handler

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
	"time"
)

func UserCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) {
	user := fields["user"].(model.UserInfo)

	bots, err := ctx.GetBots(event.UserCreated)
	if err != nil {
		ctx.L().Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	if err := ctx.Multicast(
		event.UserCreated,
		payload.MakeUserCreated(datetime, user),
		bots,
	); err != nil {
		ctx.L().Error("failed to multicast", zap.Error(err))
	}
}
