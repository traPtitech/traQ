package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func BotJoined(ctx Context, _ string, fields hub.Fields) {
	botID := fields["bot_id"].(uuid.UUID)
	channelID := fields["channel_id"].(uuid.UUID)

	bots, err := ctx.R().GetBots(repository.BotsQuery{}.Active().BotID(botID))
	if err != nil {
		ctx.L().Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	ch, err := ctx.CM().GetChannel(channelID)
	if err != nil {
		ctx.L().Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", channelID))
		return
	}
	user, err := ctx.R().GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	err = event.Unicast(
		ctx.D(),
		event.Joined,
		payload.MakeJoined(ch, ctx.CM().PublicChannelTree().GetChannelPath(channelID), user),
		bots[0],
	)
	if err != nil {
		ctx.L().Error("failed to unicast", zap.Error(err))
	}
}
