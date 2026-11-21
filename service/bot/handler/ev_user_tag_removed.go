package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func UserTagRemoved(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	p := event.MustPayload[event.UserTagRemovedPayload](fields)
	userID := p.UserID
	tagID := p.TagID

	bot, err := ctx.GetBotByBotUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to GetBotByBotUserID: %w", err)
	}
	if bot == nil || !bot.SubscribeEvents.Contains(botevent.TagRemoved) {
		return nil
	}

	t, err := ctx.R().GetTagByID(context.Background(), tagID)
	if err != nil {
		return fmt.Errorf("failed to GetTagByID: %w", err)
	}

	if err := ctx.Unicast(
		botevent.TagRemoved,
		payload.MakeTagRemoved(datetime, t),
		bot,
	); err != nil {
		return fmt.Errorf("failed to unicast: %w", err)
	}
	return nil
}
