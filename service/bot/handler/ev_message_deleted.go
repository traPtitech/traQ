package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func MessageDeleted(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	m := event.MustPayload[event.MessageDeletedPayload](fields).Message

	ch, err := ctx.CM().GetChannel(context.Background(), m.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}

	if ch.IsDMChannel() {
		ids, err := ctx.CM().GetDMChannelMembers(context.Background(), ch.ID)
		if err != nil {
			return fmt.Errorf("failed to GetDMChannelMembers: %w", err)
		}

		var id uuid.UUID
		for _, v := range ids {
			if v != m.UserID {
				id = v
				break
			}
		}

		bot, err := ctx.GetBotByBotUserID(id)
		if err != nil {
			return fmt.Errorf("failed to GetBotByBotUserID: %w", err)
		}
		if bot == nil || !bot.SubscribeEvents.Contains(botevent.DirectMessageDeleted) {
			return nil
		}

		if err := ctx.Unicast(
			botevent.DirectMessageDeleted,
			payload.MakeDirectMessageDeleted(datetime, m),
			bot,
		); err != nil {
			return fmt.Errorf("failed to unicast: %w", err)
		}
	} else {
		bots, err := ctx.GetChannelBots(m.ChannelID, botevent.MessageDeleted)
		if err != nil {
			return fmt.Errorf("failed to GetChannelBots: %w", err)
		}

		bots = filterBotUserIDNotEquals(bots, m.UserID)
		if len(bots) == 0 {
			return nil
		}

		if err := ctx.Multicast(
			botevent.MessageDeleted,
			payload.MakeMessageDeleted(datetime, m),
			bots,
		); err != nil {
			return fmt.Errorf("failed to multicast: %w", err)
		}
	}
	return nil
}
