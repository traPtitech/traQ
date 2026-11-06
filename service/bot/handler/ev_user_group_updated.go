package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func UserGroupUpdated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	groupID := event.MustPayload[event.UserGroupUpdatedPayload](fields).GroupID
	bots, err := ctx.GetBots(botevent.UserGroupUpdated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		botevent.UserGroupUpdated,
		payload.MakeUserGroupUpdated(datetime, groupID),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
