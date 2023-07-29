package handler

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func UserTagUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	userID := fields["user_id"].(uuid.UUID)
	tagID := fields["tag_id"].(uuid.UUID)

	bot, err := ctx.GetBotByBotUserID(userID)
	if err != nil {
		return fmt.Errorf("failed to GetBotByBotUserID: %w", err)
	}
	if bot == nil || !bot.SubscribeEvents.Contains(event.TagUpdated) {
		return nil
	}

	t, err := ctx.R().GetTagByID(tagID)
	if err != nil {
		return fmt.Errorf("failed to GetTagByID: %w", err)
	}

	if err := ctx.Unicast(
		event.TagUpdated,
		payload.MakeTagUpdated(datetime, t),
		bot,
	); err != nil {
		return fmt.Errorf("failed to unicast: %w", err)
	}
	return nil
}
