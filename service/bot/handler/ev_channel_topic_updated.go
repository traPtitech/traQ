package handler

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"go.uber.org/zap"
)

func ChannelTopicUpdated(ctx Context, _ string, fields hub.Fields) {
	chID := fields["channel_id"].(uuid.UUID)
	topic := fields["topic"].(string)
	updaterID := fields["updater_id"].(uuid.UUID)

	bots, err := ctx.GetChannelBots(chID, event.ChannelTopicChanged)
	if err != nil {
		ctx.L().Error("failed to GetChannelBots", zap.Error(err))
		return
	}
	if len(bots) == 0 {
		return
	}

	ch, err := ctx.CM().GetChannel(chID)
	if err != nil {
		ctx.L().Error("failed to GetChannel", zap.Error(err), zap.Stringer("id", chID))
		return
	}

	chCreator, err := ctx.R().GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", ch.CreatorID))
		return
	}

	user, err := ctx.R().GetUser(updaterID, false)
	if err != nil {
		ctx.L().Error("failed to GetUser", zap.Error(err), zap.Stringer("id", updaterID))
		return
	}

	if err := ctx.Multicast(
		event.ChannelTopicChanged,
		payload.MakeChannelTopicChanged(ch, ctx.CM().PublicChannelTree().GetChannelPath(ch.ID), chCreator, topic, user),
		bots,
	); err != nil {
		ctx.L().Error("failed to multicast", zap.Error(err))
	}
}
