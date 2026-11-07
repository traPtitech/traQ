package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/repository"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func ChannelTopicUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	p := event.MustPayload[event.ChannelTopicUpdatedPayload](fields)
	chID := p.ChannelID
	topic := p.Topic
	updaterID := p.UpdaterID

	bots, err := ctx.GetChannelBots(chID, botevent.ChannelTopicChanged)
	if err != nil {
		return fmt.Errorf("failed to GetChannelBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	ch, err := ctx.CM().GetChannel(context.Background(), chID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	chCreator, err := ctx.R().GetUser(context.Background(), ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	user, err := ctx.R().GetUser(context.Background(), updaterID, false)
	if err != nil {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	if err := ctx.Multicast(
		botevent.ChannelTopicChanged,
		payload.MakeChannelTopicChanged(datetime, ch, ctx.CM().PublicChannelTree(context.Background()).GetChannelPath(ch.ID), chCreator, topic, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
