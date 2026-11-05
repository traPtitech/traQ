package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"github.com/traPtitech/traQ/service/message"
)

func MessageStampsUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	m := fields["message"].(message.Message)

	bot, err := ctx.GetBotByBotUserID(m.GetUserID())
	if err != nil {
		return fmt.Errorf("failed to GetBotByBotUserID: %w", err)
	}
	if bot == nil || !bot.SubscribeEvents.Contains(botevent.BotMessageStampsUpdated) {
		return nil
	}

	if err := ctx.Unicast(
		botevent.BotMessageStampsUpdated,
		payload.MakeBotMessageStampsUpdated(datetime, m.GetID(), m.GetStamps()),
		bot,
	); err != nil {
		return fmt.Errorf("failed to unicast: %w", err)
	}
	return nil
}
