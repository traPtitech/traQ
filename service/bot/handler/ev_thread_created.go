package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func ThreadCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	th := fields["thread"].(*model.Channel)
	bots, err := ctx.GetBots(event.ThreadCreated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	user, err := ctx.R().GetUser(context.Background(), th.CreatorID, false)
	if err != nil {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	if err := ctx.Multicast(
		event.ThreadCreated,
		payload.MakeThreadCreated(datetime, th, ctx.CM().PublicChannelTree(context.Background()).GetChannelPath(th.ParentID), user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
