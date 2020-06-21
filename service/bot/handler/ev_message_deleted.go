package handler

import (
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"time"
)

func MessageDeleted(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	m := fields["message"].(*model.Message)

	ch, err := ctx.CM().GetChannel(m.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	if ch.IsDMChannel() {

	} else {
		bots, err := ctx.GetChannelBots(m.ChannelID, event.MessageDeleted)
		if err != nil {
			return fmt.Errorf("failed to GetChannelBots: %w", err)
		}

		if err := ctx.Multicast(
			event.MessageDeleted,
			payload.MakeMessageDeleted(datetime, m),
			bots,
		); err != nil {
			return fmt.Errorf("failed to multicast: %w", err)
		}
	}
	return nil
}
