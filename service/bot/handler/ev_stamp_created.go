package handler

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func StampCreated(ctx Context, _ string, fields hub.Fields) {
	stamp := fields["stamp"].(*model.Stamp)

	bots, err := ctx.GetBots(event.StampCreated)
	if err != nil {
		ctx.L().Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	var user model.UserInfo
	if !stamp.IsSystemStamp() {
		user, err = ctx.R().GetUser(stamp.CreatorID, false)
		if err != nil {
			ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", stamp.CreatorID))
			return
		}
	}

	if err := event.Multicast(
		ctx.D(),
		event.StampCreated,
		payload.MakeStampCreated(stamp, user),
		bots,
	); err != nil {
		ctx.L().Error("failed to multicast", zap.Error(err))
	}
}
