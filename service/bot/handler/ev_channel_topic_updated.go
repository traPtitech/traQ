package handler

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"time"
)

func ChannelTopicUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	chID := fields["channel_id"].(uuid.UUID)
	topic := fields["topic"].(string)
	updaterID := fields["updater_id"].(uuid.UUID)

	bots, err := ctx.GetChannelBots(chID, event.ChannelTopicChanged)
	if err != nil {
		return fmt.Errorf("failed to GetChannelBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	ch, err := ctx.CM().GetChannel(chID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	chCreator, err := ctx.R().GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	user, err := ctx.R().GetUser(updaterID, false)
	if err != nil {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	if err := ctx.Multicast(
		event.ChannelTopicChanged,
		payload.MakeChannelTopicChanged(datetime, ch, ctx.CM().PublicChannelTree().GetChannelPath(ch.ID), chCreator, topic, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
