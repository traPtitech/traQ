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

func BotLeft(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	p := event.MustPayload[event.BotLeftPayload](fields)
	botID := p.BotID
	channelID := p.ChannelID

	bot, err := ctx.GetBot(botID)
	if err != nil {
		return fmt.Errorf("failed to GetBot: %w", err)
	}

	ch, err := ctx.CM().GetChannel(context.Background(), channelID)
	if err != nil {
		return fmt.Errorf("failed to GetChannel: %w", err)
	}
	user, err := ctx.R().GetUser(context.Background(), ch.CreatorID, false)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to GetUser: %w", err)
	}

	err = ctx.Unicast(
		botevent.Left,
		payload.MakeLeft(datetime, ch, ctx.CM().PublicChannelTree(context.Background()).GetChannelPath(channelID), user),
		bot,
	)
	if err != nil {
		return fmt.Errorf("failed to unicast: %w", err)
	}
	return nil
}
