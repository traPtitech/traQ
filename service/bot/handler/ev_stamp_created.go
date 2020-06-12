package handler

import (
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
	"time"
)

func StampCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	stamp := fields["stamp"].(*model.Stamp)

	bots, err := ctx.GetBots(event.StampCreated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	var user model.UserInfo
	if !stamp.IsSystemStamp() {
		user, err = ctx.R().GetUser(stamp.CreatorID, false)
		if err != nil {
			return fmt.Errorf("failed to GetUser: %w", err)
		}
	}

	if err := ctx.Multicast(
		event.StampCreated,
		payload.MakeStampCreated(datetime, stamp, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
