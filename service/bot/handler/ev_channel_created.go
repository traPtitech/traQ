package handler

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
	"time"
)

func ChannelCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) {
	ch := fields["channel"].(*model.Channel)
	if ch.IsPublic {
		bots, err := ctx.GetBots(event.ChannelCreated)
		if err != nil {
			ctx.L().Error("failed to GetBots", zap.Error(err))
			return
		}
		if len(bots) == 0 {
			return
		}

		user, err := ctx.R().GetUser(ch.CreatorID, false)
		if err != nil {
			ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
			return
		}

		if err := ctx.Multicast(
			event.ChannelCreated,
			payload.MakeChannelCreated(datetime, ch, ctx.CM().PublicChannelTree().GetChannelPath(ch.ID), user),
			bots,
		); err != nil {
			ctx.L().Error("failed to multicast", zap.Error(err))
		}
	}
}
