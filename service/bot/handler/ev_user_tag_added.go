package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func UserTagAdded(ctx Context, _ string, fields hub.Fields) {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bots, err := ctx.R().GetBots(repository.BotsQuery{}.Active().Subscribe(event.TagAdded).BotUserID(userID))
	if err != nil {
		ctx.L().Error("failed to GetBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	t, err := ctx.R().GetTagByID(tagID)
	if err != nil {
		ctx.L().Error("failed to GetTagByID", zap.Error(err), zap.Stringer("id", tagID))
		return
	}

	if err := event.Unicast(
		ctx.D(),
		event.TagAdded,
		payload.MakeTagAdded(t),
		bots[0],
	); err != nil {
		ctx.L().Error("failed to unicast", zap.Error(err))
	}
}
