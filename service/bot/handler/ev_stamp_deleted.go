package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func StampDeleted(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	stampID := event.MustPayload[event.StampDeletedPayload](fields).StampID
	bots, err := ctx.GetBots(botevent.StampDeleted)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		botevent.StampDeleted,
		payload.MakeStampDeleted(datetime, stampID),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
