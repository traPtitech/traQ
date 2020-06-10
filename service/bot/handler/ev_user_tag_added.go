package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func UserTagAdded(ctx Context, _ string, fields hub.Fields) {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bot, err := ctx.GetBotByBotUserID(userID)
	if err != nil {
		ctx.L().Error("failed to GetBotByBotUserID", zap.Error(err))
		return
	}
	if bot == nil || !bot.SubscribeEvents.Contains(event.TagAdded) {
		return
	}

	t, err := ctx.R().GetTagByID(tagID)
	if err != nil {
		ctx.L().Error("failed to GetTagByID", zap.Error(err), zap.Stringer("id", tagID))
		return
	}

	if err := ctx.Unicast(
		event.TagAdded,
		payload.MakeTagAdded(t),
		bot,
	); err != nil {
		ctx.L().Error("failed to unicast", zap.Error(err))
	}
}
