package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func StampCreated(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	stamp := event.MustPayload[event.StampCreatedPayload](fields).Stamp

	bots, err := ctx.GetBots(botevent.StampCreated)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	var user model.UserInfo
	if !stamp.IsSystemStamp() {
		user, err = ctx.R().GetUser(context.Background(), stamp.CreatorID, false)
		if err != nil {
			return fmt.Errorf("failed to GetUser: %w", err)
		}
	}

	if err := ctx.Multicast(
		botevent.StampCreated,
		payload.MakeStampCreated(datetime, stamp, user),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
