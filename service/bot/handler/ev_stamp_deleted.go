package handler

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func StampDeleted(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	stampID := fields["stamp_id"].(uuid.UUID)
	bots, err := ctx.GetBots(event.StampDeleted)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		event.StampDeleted,
		payload.MakeStampDeleted(datetime, stampID),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
