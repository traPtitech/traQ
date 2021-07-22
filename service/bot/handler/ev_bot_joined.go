package handler

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func BotJoined(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	botID := fields["bot_id"].(uuid.UUID)
	channelID := fields["channel_id"].(uuid.UUID)

	bot, err := ctx.GetBot(botID)
	if err != nil {
		return fmt.Errorf("failed to GetBot: %w", err)
	}

	ch, err := ctx.CM().GetChannel(channelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}
	user, err := ctx.R().GetUser(ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	err = ctx.Unicast(
		event.Joined,
		payload.MakeJoined(datetime, ch, ctx.CM().PublicChannelTree().GetChannelPath(channelID), user),
		bot,
	)
	if err != nil {
		return fmt.Errorf("failed to unicast: %w", err)
	}
	return nil
}
