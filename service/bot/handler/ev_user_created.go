package handler

import (
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"time"
)

func UserCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	user := fields["user"].(model.UserInfo)

	bots, err := ctx.GetBots(event.UserCreated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		event.UserCreated,
		payload.MakeUserCreated(datetime, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
