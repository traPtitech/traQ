package handler

import (
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/utils/message"
	"time"
)

func MessageUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	m := fields["message"].(*model.Message)
	parsed := fields["parse_result"].(*message.ParseResult)

	ch, err := ctx.CM().GetChannel(m.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	user, err := ctx.R().GetUser(m.UserID, false)
	if err != nil {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	if !ch.IsDMChannel() {
		// 購読BOT
		bots, err := ctx.GetChannelBots(m.ChannelID, event.MessageUpdated)
		if err != nil {
			return fmt.Errorf("failed to GetChannelBots: %w", err)
		}

		// ev_message_created.go で定義済み
		bots = filterBotUserIDNotEquals(bots, m.UserID)
		if len(bots) == 0 {
			return nil
		}

		if err := ctx.Multicast(
			event.MessageUpdated,
			payload.MakeMessageUpdated(datetime, m, user, parsed),
			bots,
		); err != nil {
			return fmt.Errorf("failed to multicast: %w", err)
		}
	}
	return nil
}