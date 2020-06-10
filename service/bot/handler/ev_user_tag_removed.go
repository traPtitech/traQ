package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func UserTagRemoved(ctx Context, _ string, fields hub.Fields) {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bot, err := ctx.GetBotByBotUserID(userID)
	if err != nil {
		ctx.L().Error("failed to GetBotByBotUserID", zap.Error(err))
		return
	}
	if bot == nil || !bot.SubscribeEvents.Contains(event.TagRemoved) {
		return
	}

	t, err := ctx.R().GetTagByID(tagID)
	if err != nil {
		ctx.L().Error("failed to GetTagByID", zap.Error(err), zap.Stringer("id", tagID))
		return
	}

	if err := event.Unicast(
		ctx.D(),
		event.TagRemoved,
		payload.MakeTagRemoved(t),
		bot,
	); err != nil {
		ctx.L().Error("failed to unicast", zap.Error(err))
	}
}
