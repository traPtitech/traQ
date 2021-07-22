package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func ChannelCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	ch := fields["channel"].(*model.Channel)
	if ch.IsPublic {
		bots, err := ctx.GetBots(event.ChannelCreated)
		if err != nil {
			return fmt.Errorf("failed to GetBots: %w", err)
		}
		if len(bots) == 0 {
			return nil
		}

		user, err := ctx.R().GetUser(ch.CreatorID, false)
		if err != nil {
			return fmt.Errorf("failed to GetUser: %w", err)
		}

		if err := ctx.Multicast(
			event.ChannelCreated,
			payload.MakeChannelCreated(datetime, ch, ctx.CM().PublicChannelTree().GetChannelPath(ch.ID), user),
			bots,
		); err != nil {
			return fmt.Errorf("failed to multicast: %w", err)
		}
	}
	return nil
}
