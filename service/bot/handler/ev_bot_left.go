package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func BotLeft(ctx Context, _ string, fields hub.Fields) {
	botID := fields["bot_id"].(uuid.UUID)
	channelID := fields["channel_id"].(uuid.UUID)

	bot, err := ctx.GetBot(botID)
	if err != nil {
		ctx.L().Error("failed to GetBot", zap.Error(err))
		return
	}
	if bot == nil {
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
		event.Left,
		payload.MakeLeft(ch, ctx.CM().PublicChannelTree().GetChannelPath(channelID), user),
		bot,
	)
	if err != nil {
		ctx.L().Error("failed to unicast", zap.Error(err))
	}
}
