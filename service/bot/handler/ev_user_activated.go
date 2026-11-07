package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func UserActivated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	user := event.MustPayload[event.UserActivatedPayload](fields).User

	bots, err := ctx.GetBots(botevent.UserActivated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		botevent.UserActivated,
		payload.MakeUserActivated(datetime, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
