package handler

import (
	"fmt"
	"time"

	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	botevent "github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/service/bot/event/payload"
)

func UserGroupMemberRemoved(ctx Context, datetime time.Time, _ string, fields hub.Fields) error {
	p := event.MustPayload[event.UserGroupMemberRemovedPayload](fields)
	groupID := p.GroupID
	userID := p.UserID
	bots, err := ctx.GetBots(botevent.UserGroupMemberRemoved)
	if err != nil {
		return fmt.Errorf("failed to GetBots: %w", err)
	}
	if len(bots) == 0 {
		return nil
	}

	if err := ctx.Multicast(
		botevent.UserGroupMemberRemoved,
		payload.MakeUserGroupMemberRemoved(datetime, groupID, userID),
		bots,
	); err != nil {
		return fmt.Errorf("failed to multicast: %w", err)
	}
	return nil
}
