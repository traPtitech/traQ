package handler

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func ChannelCreated(ctx Context, _ string, fields hub.Fields) {
	ch := fields["channel"].(*model.Channel)
	if ch.IsPublic {
		bots, err := ctx.R().GetBots(repository.BotsQuery{}.Privileged().Active().Subscribe(event.ChannelCreated))
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

		if err := event.Multicast(
			ctx.D(),
			event.ChannelCreated,
			payload.MakeChannelCreated(ch, ctx.CM().PublicChannelTree().GetChannelPath(ch.ID), user),
			bots,
		); err != nil {
			ctx.L().Error("failed to multicast", zap.Error(err))
		}
	}
}
